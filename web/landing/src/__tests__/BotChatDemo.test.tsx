import { beforeAll, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { BotChatDemo } from "@/components/demos/BotChatDemo";
import { initI18n } from "@/i18n";

describe("BotChatDemo", () => {
  beforeAll(async () => {
    await initI18n();
  });

  it("用静态数据渲染消息与前进交互", () => {
    render(<BotChatDemo />);
    expect(screen.getByText(/赛博朋克/)).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: /前进|Next/i }),
    ).toBeInTheDocument();
  });
});
