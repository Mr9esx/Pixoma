import { Component, lazy, Suspense, type ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { act, render, screen } from "@testing-library/react";
import { DeferredDemo } from "@/components/demos/DeferredDemo";
import { DemoBoundary } from "@/components/demos/DemoBoundary";

class BrokenDemo extends Component {
  override render(): ReactNode {
    throw new Error("chunk failed");
  }
}

describe("deferred demos", () => {
  const originalObserver = globalThis.IntersectionObserver;

  afterEach(() => {
    globalThis.IntersectionObserver = originalObserver;
    vi.restoreAllMocks();
  });

  it("接近视口前不挂载演示内容", () => {
    let notify: IntersectionObserverCallback | undefined;
    class ControlledObserver {
      constructor(callback: IntersectionObserverCallback) {
        notify = callback;
      }
      observe() {}
      unobserve() {}
      disconnect() {}
    }
    globalThis.IntersectionObserver =
      ControlledObserver as unknown as typeof IntersectionObserver;

    render(
      <DeferredDemo fallback={<div>加载占位</div>}>
        <div>演示内容</div>
      </DeferredDemo>,
    );
    expect(screen.getByText("加载占位")).toBeInTheDocument();
    expect(screen.queryByText("演示内容")).not.toBeInTheDocument();

    act(() => {
      notify?.(
        [{ isIntersecting: true } as IntersectionObserverEntry],
        {} as IntersectionObserver,
      );
    });
    expect(screen.getByText("演示内容")).toBeInTheDocument();
  });

  it("单个演示失败时保留页面其余内容", () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined);
    render(
      <>
        <p>页面内容</p>
        <DemoBoundary fallback={<div role="alert">演示加载失败</div>}>
          <BrokenDemo />
        </DemoBoundary>
      </>,
    );
    expect(screen.getByText("页面内容")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent("演示加载失败");
  });

  it("动态 import 失败时只替换当前演示", async () => {
    vi.spyOn(console, "error").mockImplementation(() => undefined);
    const RejectedDemo = lazy(() => Promise.reject(new Error("chunk failed")));

    render(
      <>
        <p>页面内容</p>
        <DemoBoundary fallback={<div role="alert">演示加载失败</div>}>
          <Suspense fallback={<div>加载中</div>}>
            <RejectedDemo />
          </Suspense>
        </DemoBoundary>
      </>,
    );

    expect(screen.getByText("页面内容")).toBeInTheDocument();
    expect(await screen.findByRole("alert")).toHaveTextContent("演示加载失败");
  });
});
