import { beforeEach, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FaqSection } from "@/components/sections/FaqSection";
import i18n, { initI18n } from "@/i18n";
import zhCN from "@/i18n/locales/zh-CN";

describe("FaqSection", () => {
  beforeEach(async () => {
    await initI18n();
    await i18n.changeLanguage("zh-CN");
  });

  it("点击同一标题可展开和收起并保持焦点", async () => {
    const user = userEvent.setup();
    render(<FaqSection />);
    const trigger = screen.getByRole("button", {
      name: zhCN["faq.install.question"],
    });

    expect(trigger).toHaveAttribute("aria-expanded", "true");
    await user.click(trigger);
    expect(trigger).toHaveAttribute("aria-expanded", "false");
    await user.click(trigger);
    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(trigger).toHaveAttribute("aria-controls");
    expect(
      document.getElementById(trigger.getAttribute("aria-controls")!),
    ).toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it.each(["{Enter}", " "])("%s 激活 FAQ 得到与鼠标一致的结果", async (key) => {
    const user = userEvent.setup();
    render(<FaqSection />);
    const trigger = screen.getByRole("button", {
      name: zhCN["faq.telegram.question"],
    });

    trigger.focus();
    await user.keyboard(key);

    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(trigger).toHaveFocus();
  });
});
