import { useEffect, useRef, useState, type ReactNode } from "react";

type DeferredDemoProps = {
  children: ReactNode;
  fallback: ReactNode;
};

export function DeferredDemo({ children, fallback }: DeferredDemoProps) {
  const rootRef = useRef<HTMLDivElement>(null);
  const [nearViewport, setNearViewport] = useState(
    () => typeof IntersectionObserver === "undefined",
  );

  useEffect(() => {
    const root = rootRef.current;
    if (!root) return;
    if (typeof IntersectionObserver === "undefined") return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (!entry?.isIntersecting) return;
        setNearViewport(true);
        observer.disconnect();
      },
      { rootMargin: "240px 0px" },
    );
    observer.observe(root);
    return () => observer.disconnect();
  }, []);

  return <div ref={rootRef}>{nearViewport ? children : fallback}</div>;
}
