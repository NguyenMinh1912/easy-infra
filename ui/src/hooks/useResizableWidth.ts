import { useCallback, useEffect, useState } from "react";

/** localStorage key prefix for persisted sidebar/panel widths. */
const STORAGE_PREFIX = "easy-infra:width:";

function clamp(value: number, min: number, max: number): number {
  return Math.min(max, Math.max(min, value));
}

/** Read a persisted width, clamped to the allowed range, else the fallback. */
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
  /** Stable identifier scoping the width in localStorage. */
  key: string;
  /** Width used before the user has dragged (and when nothing is stored). */
  initial: number;
  min: number;
  max: number;
}

/**
 * A horizontally draggable width persisted to localStorage. Returns the current
 * width (in px) and a pointer-down handler to attach to a drag handle on the
 * element's right edge; dragging right grows the element. The width survives
 * reloads, scoped by `key`.
 */
export function useResizableWidth({ key, initial, min, max }: Options) {
  const [width, setWidth] = useState(() => load(key, initial, min, max));

  // Re-read when the key changes (e.g. a different connection's sidebar).
  useEffect(() => {
    setWidth(load(key, initial, min, max));
  }, [key, initial, min, max]);

  // Persist on every change.
  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_PREFIX + key, String(width));
    } catch {
      // Storage unavailable (private mode, quota) — keep working in memory.
    }
  }, [key, width]);

  const onResizeStart = useCallback(
    (event: React.PointerEvent) => {
      event.preventDefault();
      const startX = event.clientX;
      const startWidth = width;

      const onMove = (e: PointerEvent) => {
        setWidth(clamp(startWidth + (e.clientX - startX), min, max));
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
      // Keep a col-resize cursor and suppress text selection for the whole drag,
      // even when the pointer leaves the thin handle.
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";
    },
    [width, min, max],
  );

  return { width, onResizeStart };
}
