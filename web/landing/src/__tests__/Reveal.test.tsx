import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { Reveal } from "@/components/motion/Reveal";

vi.mock("motion/react", async (importOriginal) => {
  const mod = await importOriginal<typeof import("motion/react")>();
  return {
    ...mod,
    useReducedMotion: vi.fn(() => true),
  };
});

describe("Reveal", () => {
  it("reduced motion 下渲染内容且不依赖 opacity 动画", () => {
    render(<Reveal>你好</Reveal>);
    expect(screen.getByText("你好")).toBeInTheDocument();
  });
});
