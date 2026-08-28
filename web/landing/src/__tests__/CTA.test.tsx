import { beforeAll, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { CTASection } from "@/components/sections/CTASection";
import { initI18n } from "@/i18n";

describe("CTASection", () => {
  beforeAll(async () => {
    await initI18n();
  });

  it("渲染主行动入口", () => {
    render(<CTASection />);
    expect(
      screen.getByRole("link", { name: /下载|Download/i }),
    ).toBeInTheDocument();
  });
});
