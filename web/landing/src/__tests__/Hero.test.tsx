import { beforeAll, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { HeroSection } from "@/components/sections/HeroSection";
import { initI18n } from "@/i18n";

describe("HeroSection", () => {
  beforeAll(async () => {
    await initI18n();
  });

  it("渲染主标题与 CTA", () => {
    render(<HeroSection />);
    expect(screen.getByRole("heading", { level: 1 })).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: /下载|Download/i }),
    ).toBeInTheDocument();
  });
});
