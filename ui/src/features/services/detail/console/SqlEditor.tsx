import { PostgreSQL, sql } from "@codemirror/lang-sql";
import CodeMirror, {
  EditorView,
  keymap,
  Prec,
} from "@uiw/react-codemirror";
import { useMemo, useRef, type MutableRefObject } from "react";

import { useTheme } from "@/components/theme/ThemeProvider";

import { columnCompletionSource } from "./sqlCompletion";

interface SqlEditorProps {
  value: string;
  onChange: (value: string) => void;
  /**
   * Tables/columns for autocomplete, keyed by table name (schema-qualified
   * outside `public`). Absent while the schema loads or when introspection
   * failed — the editor then offers keyword completion only.
   */
  schema?: Record<string, string[]>;
  /** Invoked on Ctrl/Cmd-Enter; the same action as the Run button. */
  onRun: () => void;
  /**
   * Receives the live editor view so the parent can read the cursor/selection
   * when deciding which statement to run.
   */
  viewRef?: MutableRefObject<EditorView | null>;
  /**
   * Fixed editor height in pixels. When set, the editor stays this tall and
   * scrolls its content internally (the parent owns the drag-to-resize). Falls
   * back to the default min/max growth range when omitted.
   */
  height?: number;
}

/**
 * CodeMirror-based SQL editor: PostgreSQL syntax highlighting, keyword
 * completion out of the box, and schema-aware table/column suggestions once
 * the schema is known. Presentational — the parent owns the text and the run
 * action.
 */
export function SqlEditor({
  value,
  onChange,
  schema,
  onRun,
  viewRef,
  height,
}: SqlEditorProps) {
  const { resolvedTheme } = useTheme();

  // The keymap extension captures its handler once; route it through a ref so
  // rebuilding extensions isn't needed when the parent re-renders with a new
  // closure.
  const runRef = useRef(onRun);
  runRef.current = onRun;

  const extensions = useMemo(() => {
    const exts = [
      sql({ dialect: PostgreSQL, schema, upperCaseKeywords: true }),
      // Highest precedence so Mod-Enter beats the default newline binding.
      Prec.highest(
        keymap.of([
          {
            key: "Mod-Enter",
            run: () => {
              runRef.current();
              return true;
            },
          },
        ]),
      ),
    ];
    // lang-sql suggests table names (and qualified `alias.column`) but not bare
    // column names in a WHERE/SELECT context; this fills that gap from the
    // statement's FROM clause. Only meaningful once the schema is known.
    if (schema) {
      exts.push(
        PostgreSQL.language.data.of({
          autocomplete: columnCompletionSource(schema),
        }),
      );
    }
    return exts;
  }, [schema]);

  return (
    <CodeMirror
      value={value}
      onChange={onChange}
      onCreateEditor={(view) => {
        if (viewRef) viewRef.current = view;
      }}
      theme={resolvedTheme}
      extensions={extensions}
      {...(height !== undefined
        ? { height: `${height}px`, minHeight: "6rem" }
        : { minHeight: "10rem", maxHeight: "24rem" })}
      placeholder="SELECT * FROM …"
      aria-label="SQL editor"
      className="overflow-hidden rounded-md border border-border text-sm [&_.cm-editor]:bg-background [&_.cm-focused]:outline-none"
      basicSetup={{
        lineNumbers: true,
        foldGutter: false,
        autocompletion: true,
        highlightActiveLine: false,
      }}
    />
  );
}
