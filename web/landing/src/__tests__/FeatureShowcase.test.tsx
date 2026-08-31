import fs from "node:fs";
import path from "node:path";
import { beforeEach, describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { FeatureShowcaseSection } from "@/components/sections/FeatureShowcaseSection";
import { HeroSection } from "@/components/sections/HeroSection";
import i18n, { initI18n } from "@/i18n";
import zhCN from "@/i18n/locales/zh-CN";

describe("FeatureShowcaseSection", () => {
  beforeEach(async () => {
    await initI18n();
    await i18n.changeLanguage("zh-CN");
  });

  it("按 Bot、Case、后台顺序渲染三组说明与演示", async () => {
    render(<FeatureShowcaseSection />);
    const headings = screen.getAllByRole("heading", { level: 3 });
    expect(headings.map((heading) => heading.textContent)).toEqual([
      zhCN["features.bot.title"],
      zhCN["features.case.title"],
      zhCN["features.admin.title"],
    ]);
    expect(await screen.findByText("都市夜景工作流")).toBeInTheDocument();
    expect(await screen.findByText("节点 1")).toBeInTheDocument();
  });

  it("首屏 Bot 与功能区 Bot 的前进状态互不影响", async () => {
    render(
      <>
        <HeroSection />
        <FeatureShowcaseSection />
      </>,
    );
    const nextButtons = await screen.findAllByRole("button", {
      name: /前进|Next/i,
    });

    fireEvent.click(nextButtons[0]);

    expect(screen.getAllByText(/赛博朋克/)).toHaveLength(2);
    expect(screen.getAllByText(/GPU 节点 1/)).toHaveLength(1);
  });

  it("功能区复用首屏 Bot 模块并继续惰性加载 Case 与后台", () => {
    const source = fs.readFileSync(
      path.resolve(import.meta.dirname, "../components/demos/LazyDemos.tsx"),
      "utf8",
    );
    expect(source).toContain("lazy(() =>");
    expect(source).toContain(
      'import { BotChatDemo } from "@/components/demos/BotChatDemo"',
    );
    expect(source).not.toContain('import("@/components/demos/BotChatDemo")');
    expect(source).toContain('import("@/components/demos/CaseDirDemo")');
    expect(source).toContain('import("@/components/demos/AdminDemo")');
    expect(source).toContain("min-h-");
  });
});
