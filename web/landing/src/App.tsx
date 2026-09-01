import { useEffect, useRef } from "react";
import DuskLandingPage from "@/pages/dusk-landing-page";
import DuskLanding7Page from "@/pages/dusk-landing-7-page";

export function App() {
  const version = new URLSearchParams(window.location.search).get("version");
  const bottomOverlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const footer = document.querySelector("footer");
    const overlay = bottomOverlayRef.current;

    if (!footer || !overlay) return;

    let frame = 0;

    const updateOverlay = () => {
      frame = 0;

      const viewportHeight = window.innerHeight;
      const { top } = footer.getBoundingClientRect();
      const visibleHeight = Math.max(
        0,
        Math.min(viewportHeight - top, viewportHeight),
      );
      const fadeProgress = Math.min(visibleHeight / 320, 1);

      const overlayOpacity = 1 - fadeProgress;

      overlay.style.opacity = String(overlayOpacity);
      overlay.style.backdropFilter = `blur(${overlayOpacity * 4}px)`;
    };

    const scheduleUpdate = () => {
      if (frame === 0) frame = requestAnimationFrame(updateOverlay);
    };

    updateOverlay();
    window.addEventListener("scroll", scheduleUpdate, { passive: true });
    window.addEventListener("resize", scheduleUpdate);

    return () => {
      window.removeEventListener("scroll", scheduleUpdate);
      window.removeEventListener("resize", scheduleUpdate);

      if (frame !== 0) cancelAnimationFrame(frame);
    };
  }, [version]);

  return (
    <div className="relative min-h-dvh">
      {version === "7" ? <DuskLanding7Page /> : <DuskLandingPage />}

      <div
        aria-hidden
        ref={bottomOverlayRef}
        className="pointer-events-none fixed inset-x-0 bottom-0 z-[45] hidden h-[200px] backdrop-blur-sm [mask-image:linear-gradient(to_top,black,transparent)] lg:block"
      >
        <div className="size-full bg-gradient-to-t from-background/80 to-transparent transition-opacity duration-300 ease-out" />
      </div>
    </div>
  );
}
