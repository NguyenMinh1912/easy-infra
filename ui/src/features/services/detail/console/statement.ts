/**
 * Splitting a multi-statement editor buffer into the single statement to run.
 *
 * The console executes one statement per request, so when the buffer holds
 * several `;`-separated statements we pick exactly one: the user's selection if
 * there is one, otherwise the statement under the cursor. The scanner treats
 * `;` as a separator only at the top level — semicolons inside string literals,
 * quoted identifiers, comments, and dollar-quoted blocks (PL/pgSQL bodies) do
 * not split, matching how postgres itself parses the text.
 *
 * Pure and free of editor dependencies so the logic can be reasoned about (and
 * tested) on plain strings; the editor adapter lives at the call site.
 */

/** Half-open `[from, to)` span of one statement within the buffer, excluding
 *  the trailing `;` separator. */
export interface StatementRange {
  from: number;
  to: number;
}

/** Cursor/selection within the buffer, shaped like CodeMirror's
 *  `state.selection.main`. */
export interface Selection {
  from: number;
  to: number;
  head: number;
}

function isTagStart(ch: string): boolean {
  return /[A-Za-z_]/.test(ch);
}

function isTagChar(ch: string): boolean {
  return /[A-Za-z0-9_]/.test(ch);
}

/**
 * If a dollar-quote delimiter opens at `i` (`$$` or `$tag$`), return it;
 * otherwise null. A bare `$1` (positional parameter) is not a delimiter — the
 * tag, when present, must be a valid identifier (no leading digit).
 */
function dollarTagAt(doc: string, i: number): string | null {
  if (doc[i] !== "$") return null;
  let j = i + 1;
  if (j < doc.length && isTagStart(doc[j])) {
    j++;
    while (j < doc.length && isTagChar(doc[j])) j++;
  }
  return doc[j] === "$" ? doc.slice(i, j + 1) : null;
}

/**
 * Split `doc` into top-level statement ranges. Ranges are contiguous and cover
 * the whole buffer; the `;` separators sit in the one-character gaps between
 * them. A buffer with no top-level `;` yields a single range spanning all of it.
 */
export function splitStatements(doc: string): StatementRange[] {
  const ranges: StatementRange[] = [];
  let start = 0;
  let i = 0;

  while (i < doc.length) {
    const ch = doc[i];

    // Line comment: run to end of line.
    if (ch === "-" && doc[i + 1] === "-") {
      i += 2;
      while (i < doc.length && doc[i] !== "\n") i++;
      continue;
    }

    // Block comment: run to the closing */ (postgres block comments do not
    // nest in practice for our purposes).
    if (ch === "/" && doc[i + 1] === "*") {
      i += 2;
      while (i < doc.length && !(doc[i] === "*" && doc[i + 1] === "/")) i++;
      i += 2;
      continue;
    }

    // Single-quoted string, with '' as an embedded quote.
    if (ch === "'") {
      i++;
      while (i < doc.length) {
        if (doc[i] === "'") {
          if (doc[i + 1] === "'") {
            i += 2;
            continue;
          }
          i++;
          break;
        }
        i++;
      }
      continue;
    }

    // Double-quoted identifier, with "" as an embedded quote.
    if (ch === '"') {
      i++;
      while (i < doc.length) {
        if (doc[i] === '"') {
          if (doc[i + 1] === '"') {
            i += 2;
            continue;
          }
          i++;
          break;
        }
        i++;
      }
      continue;
    }

    // Dollar-quoted block: skip to the matching closing delimiter.
    const tag = dollarTagAt(doc, i);
    if (tag) {
      i += tag.length;
      const close = doc.indexOf(tag, i);
      i = close === -1 ? doc.length : close + tag.length;
      continue;
    }

    if (ch === ";") {
      ranges.push({ from: start, to: i });
      start = i + 1;
    }
    i++;
  }

  ranges.push({ from: start, to: doc.length });
  return ranges;
}

/**
 * Whether `text` contains anything runnable, i.e. SQL beyond comments and
 * whitespace. A heuristic strip is enough here: this only decides emptiness,
 * the statement is sent verbatim, and a statement with real content (hence
 * possibly strings containing `--`) is non-empty regardless.
 */
function hasSql(text: string): boolean {
  return (
    text
      .replace(/--[^\n]*/g, " ")
      .replace(/\/\*[\s\S]*?\*\//g, " ")
      .trim() !== ""
  );
}

/**
 * The single statement to execute for the given selection. A non-empty
 * selection runs verbatim (trimmed). With just a cursor, return the statement
 * spanning it; if that lands in a blank or comment-only tail (e.g. just past
 * the final `;`, or a trailing comment), fall back to the preceding statement
 * that has SQL. Returns "" when there is nothing runnable.
 */
export function statementToRun(doc: string, selection: Selection): string {
  if (selection.from !== selection.to) {
    return doc.slice(selection.from, selection.to).trim();
  }

  const head = selection.head;
  const ranges = splitStatements(doc);
  const text = (r: StatementRange) => doc.slice(r.from, r.to).trim();

  let chosen = ranges.find((r) => head >= r.from && head <= r.to);
  if (!chosen || !hasSql(text(chosen))) {
    const prev = [...ranges]
      .reverse()
      .find((r) => r.to <= head && hasSql(text(r)));
    if (prev) chosen = prev;
  }
  return chosen && hasSql(text(chosen)) ? text(chosen) : "";
}
