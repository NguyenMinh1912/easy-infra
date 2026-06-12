import type {
  Completion,
  CompletionContext,
  CompletionResult,
  CompletionSource,
} from "@codemirror/autocomplete";
import { syntaxTree } from "@codemirror/language";
import type { Text } from "@uiw/react-codemirror";

/**
 * Minimal structural view of a Lezer syntax node — enough for the FROM-clause
 * walk below without taking a direct dependency on `@lezer/common`.
 */
interface SqlNode {
  name: string;
  from: number;
  to: number;
  parent: SqlNode | null;
  firstChild: SqlNode | null;
  nextSibling: SqlNode | null;
}

// Keywords that end the FROM clause; no table references appear past them.
const FROM_END = new Set(
  "where group having order union intersect except all distinct limit offset fetch for".split(
    " ",
  ),
);

// Matches `@codemirror/lang-sql`'s own completion span so options filter the
// same way as the built-in keyword/table suggestions they merge with.
const SPAN = /^\w*$/;

function isIdentifier(name: string): boolean {
  // Identifier | QuotedIdentifier | CompositeIdentifier (e.g. `schema.table`).
  return /Identifier$/.test(name);
}

/** Strip surrounding identifier quotes (`"`, `` ` ``, `'`, `[ ]`) as lang-sql does. */
function unquote(text: string): string {
  const m = /^([`'"[])(.*)([`'"\]])$/.exec(text);
  return m ? m[2] : text;
}

/** Dotted name of an identifier node, e.g. `audit.logs` for a CompositeIdentifier. */
function nodePath(node: SqlNode, doc: Text): string {
  if (node.name === "CompositeIdentifier") {
    const parts: string[] = [];
    for (let ch = node.firstChild; ch; ch = ch.nextSibling) {
      if (isIdentifier(ch.name)) parts.push(unquote(doc.sliceString(ch.from, ch.to)));
    }
    return parts.join(".");
  }
  return unquote(doc.sliceString(node.from, node.to));
}

/**
 * The statement the cursor is editing. `resolveInner` lands on the enclosing
 * `Statement` while inside a token, but in the whitespace after `WHERE ` — the
 * very spot this completion targets — it returns the top `Script` node instead.
 * So fall back to the last top-level statement that starts at or before the
 * cursor (statements are ordered, separated by `;`).
 */
function statementAt(node: SqlNode, pos: number): SqlNode | null {
  let stmt: SqlNode | null = node;
  while (stmt && stmt.name !== "Statement") stmt = stmt.parent;
  if (stmt) return stmt;

  let top: SqlNode = node;
  while (top.parent) top = top.parent;
  let best: SqlNode | null = null;
  for (let ch = top.firstChild; ch; ch = ch.nextSibling) {
    if (ch.name !== "Statement") continue;
    if (ch.from <= pos) best = ch;
    else break;
  }
  return best;
}

/**
 * Tables referenced in the statement's FROM/JOIN clause, by the key they were
 * written as (the alias' target is folded in, so `employee e` yields
 * `employee`). Mirrors lang-sql's own scan: start collecting after `FROM`, stop
 * at the first clause keyword. JOIN tables sit between the two, so they're
 * included; identifiers inside `ON` conditions are qualified or follow `ON`, so
 * they aren't mistaken for tables.
 */
function fromTables(stmt: SqlNode, doc: Text): string[] {
  const tables: string[] = [];
  let sawFrom = false;
  let expectTable = false;
  let lastTable: string | null = null;

  const add = (key: string) => {
    if (key && !tables.includes(key)) tables.push(key);
  };

  for (let scan = stmt.firstChild; scan; scan = scan.nextSibling) {
    const kw =
      scan.name === "Keyword"
        ? doc.sliceString(scan.from, scan.to).toLowerCase()
        : null;

    if (!sawFrom) {
      if (kw === "from") {
        sawFrom = true;
        expectTable = true;
      }
      continue;
    }
    if (kw && FROM_END.has(kw)) break;

    if (kw === "from" || kw === "join") {
      expectTable = true;
      lastTable = null;
      continue;
    }
    if (kw === "as") continue; // the next identifier is an alias, not a table
    if (doc.sliceString(scan.from, scan.to) === ",") {
      expectTable = true;
      lastTable = null;
      continue;
    }
    if (!isIdentifier(scan.name)) continue; // on / using / operators, etc.

    if (expectTable) {
      const key = nodePath(scan, doc);
      add(key);
      lastTable = key;
      expectTable = false;
    } else if (
      lastTable &&
      (scan.name === "Identifier" || scan.name === "QuotedIdentifier")
    ) {
      // Implicit alias (`employee e`); the table itself is already recorded.
      lastTable = null;
    }
  }

  return tables;
}

/** Columns for a referenced table, tolerating schema-qualified references. */
function columnsFor(
  table: string,
  schema: Record<string, string[]>,
): string[] {
  if (schema[table]) return schema[table];
  // `public.employee` written explicitly still resolves to the `employee` key
  // when that table lives in the connection's current schema.
  const last = table.split(".").pop();
  return (last && schema[last]) || [];
}

/**
 * Completion source that suggests column names at unqualified positions (most
 * importantly inside a WHERE clause) drawn from the tables in the statement's
 * FROM/JOIN clause. lang-sql already handles qualified `alias.`/`table.`
 * completions and offers table/alias names at the top level; this only fills
 * the gap it leaves — bare column names — and merges with those results.
 */
export function columnCompletionSource(
  schema: Record<string, string[]>,
): CompletionSource {
  return (context: CompletionContext): CompletionResult | null => {
    const word = context.matchBefore(/[\w$]*/);
    if (!word) return null;
    // Don't pop the menu on a bare separator unless the user asked for it.
    if (word.from === word.to && !context.explicit) return null;
    // Qualified completions (`alias.col`) belong to lang-sql's own source.
    if (
      word.from > 0 &&
      context.state.sliceDoc(word.from - 1, word.from) === "."
    ) {
      return null;
    }

    const doc = context.state.doc;
    const node = syntaxTree(context.state).resolveInner(
      context.pos,
      -1,
    ) as unknown as SqlNode;
    const stmt = statementAt(node, context.pos);
    if (!stmt) return null;

    const tables = fromTables(stmt, doc);
    if (tables.length === 0) return null;

    const seen = new Set<string>();
    const options: Completion[] = [];
    for (const table of tables) {
      for (const col of columnsFor(table, schema)) {
        if (seen.has(col)) continue;
        seen.add(col);
        options.push({ label: col, type: "property" });
      }
    }
    if (options.length === 0) return null;

    return { from: word.from, options, validFor: SPAN };
  };
}
