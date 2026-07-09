import { useEffect, useRef, useState } from "react";

/**
 * Tracks an element's rendered width so SVG charts can be drawn at real pixel
 * geometry (rather than stretched with preserveAspectRatio, which distorts strokes).
 */
export function useElementWidth<T extends HTMLElement>() {
  const ref = useRef<T>(null);
  const [width, setWidth] = useState(0);

  useEffect(() => {
    const element = ref.current;
    if (!element) return;

    const observer = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width));
    observer.observe(element);

    return () => observer.disconnect();
  }, []);

  return [ref, width] as const;
}
