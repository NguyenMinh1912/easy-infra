import {
  startCompletion,
  type Completion,
  type CompletionContext,
  type CompletionResult,
  type CompletionSource,
} from "@codemirror/autocomplete";
import { EditorView } from "@uiw/react-codemirror";
import { toast } from "sonner";

import { ApiError, getTemplate } from "@/services/api";
import type { TemplateSummary } from "@/types/templates";

// The slash "mention" token: a leading "/" followed by template-name chars.
// matchBefore anchors it to the cursor; validFor keeps the menu open while the
// user narrows the search.
const SLASH_TOKEN = /\/[\w.-]*/;
const SLASH_VALID = /^\/[\w.-]*$/;

/**
 * Completion source for the SQL console's "/" template mentions. Typing "/"
 * (see {@link slashTemplateTrigger}) opens the editor's completion menu listing
 * the workspace's templates; the typed text after the slash filters them.
 * Choosing one replaces the slash token with that template's SQL body.
 *
 * The slash must start a word (preceded by whitespace or the start of a line)
 * so it never fires inside `a/b` division or a `/* comment *​/`.
 */
export function templateCompletionSource(
  templates: TemplateSummary[],
): CompletionSource {
  return (context: CompletionContext): CompletionResult | null => {
    const token = context.matchBefore(SLASH_TOKEN);
    if (!token) return null;
    const before =
      token.from > 0 ? context.state.sliceDoc(token.from - 1, token.from) : "";
    if (before && !/\s/.test(before)) return null;
    if (templates.length === 0) return null;

    const options: Completion[] = templates.map((t) => ({
      label: `/${t.name}`,
      detail: t.description || undefined,
      type: "text",
      // High boost so templates sort above any keyword/table options that may
      // also match the typed text.
      boost: 99,
      apply: (view, _completion, from, to) =>
        insertTemplate(view, t.name, from, to),
    }));

    return { from: token.from, options, validFor: SLASH_VALID };
  };
}

/**
 * Replace the slash token [from, to) with the named template's SQL body. The
 * body is fetched on selection (the list carries no SQL); on success the cursor
 * lands at the end of the inserted text.
 */
function insertTemplate(
  view: EditorView,
  name: string,
  from: number,
  to: number,
): void {
  getTemplate(name)
    .then((t) => {
      view.dispatch({
        changes: { from, to, insert: t.sql },
        selection: { anchor: from + t.sql.length },
      });
      view.focus();
    })
    .catch((cause: unknown) => {
      toast.error("Could not load template", {
        description: cause instanceof ApiError ? cause.message : String(cause),
      });
    });
}

/**
 * Editor extension that opens the template menu the moment "/" is typed.
 * CodeMirror only auto-activates completion on word characters, so this inserts
 * the slash itself and then explicitly starts completion; the source above
 * decides whether any templates apply at that position.
 */
export const slashTemplateTrigger = EditorView.inputHandler.of(
  (view, from, to, text) => {
    if (text !== "/") return false;
    view.dispatch({
      changes: { from, to, insert: "/" },
      selection: { anchor: from + 1 },
    });
    startCompletion(view);
    return true;
  },
);
