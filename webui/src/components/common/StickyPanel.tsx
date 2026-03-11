import React, { useCallback, useEffect, useRef } from "react";

/**
 * StickyPanel keeps its content visible within the nearest scrollable ancestor's
 * visible bounds.
 *
 * - Content shorter than visible area: stays at the top of the visible area.
 * - Content taller than visible area: scrolling down pins the bottom to the
 *   visible bottom; scrolling up pins the top to the visible top.
 *
 * Uses a constraint-based approach: on each frame, the current offset is nudged
 * only when content has drifted out of the visible region, then clamped to stay
 * within the container. No explicit scroll-direction tracking is needed.
 */
export const StickyPanel = ({ children }: { children: React.ReactNode }) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const innerRef = useRef<HTMLDivElement>(null);
  const currentOffset = useRef(0);
  const rafId = useRef(0);

  const findScrollParent = useCallback((el: HTMLElement) => {
    let parent = el.parentElement;
    while (parent) {
      const style = getComputedStyle(parent);
      if (
        ["auto", "scroll"].includes(style.overflowY) ||
        ["auto", "scroll"].includes(style.overflow)
      ) {
        return parent;
      }
      parent = parent.parentElement;
    }
    return null;
  }, []);

  const update = useCallback(() => {
    rafId.current = 0;
    const container = containerRef.current;
    const inner = innerRef.current;
    if (!container || !inner) return;

    const containerRect = container.getBoundingClientRect();
    const innerHeight = inner.scrollHeight;

    // Visible bounds from nearest scrollable ancestor, or the viewport.
    const scrollParent = findScrollParent(container);
    let visibleTop = 0;
    let visibleBottom = window.innerHeight;
    if (scrollParent) {
      const rect = scrollParent.getBoundingClientRect();
      visibleTop = rect.top;
      visibleBottom = rect.bottom;
    }

    let offset = currentOffset.current;

    // Constraint: if bottom of content is above visible bottom, push it down.
    const innerBottom = containerRect.top + offset + innerHeight;
    if (innerBottom < visibleBottom) {
      offset += visibleBottom - innerBottom;
    }

    // Constraint: if top of content is below visible top, pull it up.
    const innerTop = containerRect.top + offset;
    if (innerTop > visibleTop) {
      offset = visibleTop - containerRect.top;
    }

    // Clamp: keep content within the container.
    const maxOffset = Math.max(0, containerRect.height - innerHeight);
    offset = Math.max(0, Math.min(offset, maxOffset));

    currentOffset.current = offset;
    inner.style.transform = `translateY(${offset}px)`;
  }, [findScrollParent]);

  const scheduleUpdate = useCallback(() => {
    if (!rafId.current) {
      rafId.current = requestAnimationFrame(update);
    }
  }, [update]);

  // Attach scroll/resize listeners.
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;

    // Collect all scrollable ancestors.
    const scrollParents: (HTMLElement | Window)[] = [window];
    let parent = container.parentElement;
    while (parent) {
      const style = getComputedStyle(parent);
      if (
        ["auto", "scroll"].includes(style.overflowY) ||
        ["auto", "scroll"].includes(style.overflow)
      ) {
        scrollParents.push(parent);
      }
      parent = parent.parentElement;
    }

    for (const sp of scrollParents) {
      sp.addEventListener("scroll", scheduleUpdate, { passive: true });
    }
    window.addEventListener("resize", scheduleUpdate, { passive: true });

    // Run once to set initial position.
    update();

    return () => {
      if (rafId.current) cancelAnimationFrame(rafId.current);
      for (const sp of scrollParents) {
        sp.removeEventListener("scroll", scheduleUpdate);
      }
      window.removeEventListener("resize", scheduleUpdate);
    };
  }, [scheduleUpdate, update]);

  // Reset offset and reposition when children change.
  useEffect(() => {
    currentOffset.current = 0;
    if (innerRef.current) {
      innerRef.current.style.transform = "translateY(0px)";
    }
    // Re-run positioning so content snaps into view immediately.
    update();
  }, [children, update]);

  return (
    <div
      ref={containerRef}
      style={{ width: "100%", height: "100%", position: "relative", overflow: "hidden" }}
    >
      <div ref={innerRef}>{children}</div>
    </div>
  );
};
