import { PostgreSQL, sql } from "@codemirror/lang-sql";
import CodeMirror, { keymap, Prec } from "@uiw/react-codemirror";
import { useMemo, useRef } from "react";

import { useTheme } from "@/components/theme/ThemeProvider";

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
}

/**
 * CodeMirror-based SQL editor: PostgreSQL syntax highlighting, keyword
 * completion out of the box, and schema-aware table/column suggestions once
 * the schema is known. Presentational — the parent owns the text and the run
 * action.
 */
export function SqlEditor({ value, onChange, schema, onRun }: SqlEditorProps) {
  const { resolvedTheme } = useTheme();

  // The keymap extension captures its handler once; route it through a ref so
  // rebuilding extensions isn't needed when the parent re-renders with a new
  // closure.
  const runRef = useRef(onRun);
  runRef.current = onRun;

  const extensions = useMemo(
    () => [
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
    ],
    [schema],
  );

  return (
    <CodeMirror
      value={value}
      onChange={onChange}
      theme={resolvedTheme}
      extensions={extensions}
      minHeight="10rem"
      maxHeight="24rem"
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
