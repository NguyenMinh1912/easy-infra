import { useLayoutEffect, useRef, useState } from "react";

/**
 * Cap an element to the height left between its own top edge and the bottom of
 * the viewport, so it scrolls within the screen instead of growing the page
 * past it. Returns a ref to attach to the scroll container and the live max
 * height in pixels (null before the first measurement, i.e. no cap yet).
 *
 * Tracks the *remaining* height rather than a fixed slice of the viewport, so
 * it stays correct under whatever chrome (page header, navbar, the SQL editor)
 * happens to sit above the element. Re-measures after every render — cheap, and
 * cascading renders from content above keep the value fresh — and on resize.
 *
 * @param gap pixels to leave between the element's bottom and the viewport edge.
 */
export function useRemainingHeight<T extends HTMLElement>(gap = 24) {
  const ref = useRef<T>(null);
  const [maxHeight, setMaxHeight] = useState<number | null>(null);

  useLayoutEffect(() => {
    function measure() {
      const el = ref.current;
      if (!el) return;
      const top = el.getBoundingClientRect().top;
      const next = Math.max(0, Math.round(window.innerHeight - top - gap));
      setMaxHeight((prev) => (prev === next ? prev : next));
    }
    measure();
    window.addEventListener("resize", measure);
    return () => window.removeEventListener("resize", measure);
  });

  return { ref, maxHeight };
}
