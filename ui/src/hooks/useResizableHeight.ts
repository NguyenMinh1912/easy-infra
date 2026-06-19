import { useCallback, useEffect, useState } from "react";

/** localStorage key prefix for persisted panel heights. */
const STORAGE_PREFIX = "easy-infra:height:";

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

/** Read a persisted height, clamped to the allowed range, else the fallback. */
function load(key: string, fallback: number, min: number, max: number): number {
  try {
    const raw = localStorage.getItem(STORAGE_PREFIX + key);
    if (raw === null) return fallback;
    const value = Number(raw);
    return Number.isFinite(value) ? clamp(value, min, max) : fallback;
  } catch {
    return fallback;
  }
}

interface Options {
  /** Stable identifier scoping the height in localStorage. */
  key: string;
  /** Height used before the user has dragged (and when nothing is stored). */
  initial: number;
  min: number;
  max: number;
}

/**
 * A vertically draggable height persisted to localStorage. Returns the current
 * height (in px) and a pointer-down handler to attach to a drag handle on the
 * element's bottom edge; dragging down grows the element. The height survives
 * reloads, scoped by `key`.
 */
export function useResizableHeight({ key, initial, min, max }: Options) {
  const [height, setHeight] = useState(() => load(key, initial, min, max));

  // Re-read when the key changes (e.g. a different console's editor).
  useEffect(() => {
    setHeight(load(key, initial, min, max));
  }, [key, initial, min, max]);

  // Persist on every change.
  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_PREFIX + key, String(height));
    } catch {
      // Storage unavailable (private mode, quota) — keep working in memory.
    }
  }, [key, height]);

  const onResizeStart = useCallback(
    (event: React.PointerEvent) => {
      event.preventDefault();
      const startY = event.clientY;
      const startHeight = height;

      const onMove = (e: PointerEvent) => {
        setHeight(clamp(startHeight + (e.clientY - startY), min, max));
      };
      const onUp = () => {
        window.removeEventListener("pointermove", onMove);
        window.removeEventListener("pointerup", onUp);
        // Restore the cursor/selection suppressed during the drag.
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
      };

      window.addEventListener("pointermove", onMove);
      window.addEventListener("pointerup", onUp);
      // Keep a row-resize cursor and suppress text selection for the whole drag,
      // even when the pointer leaves the thin handle.
      document.body.style.cursor = "row-resize";
      document.body.style.userSelect = "none";
    },
    [height, min, max],
  );

  return { height, onResizeStart };
}
