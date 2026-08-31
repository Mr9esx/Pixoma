import { beforeEach, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { AudienceSection } from "@/components/sections/AudienceSection";
import { CrossDeviceSection } from "@/components/sections/CrossDeviceSection";
import { audienceIds } from "@/constants/content";
import i18n, { initI18n } from "@/i18n";
import en from "@/i18n/locales/en";
import zhCN from "@/i18n/locales/zh-CN";

describe("landing content sections", () => {
  beforeEach(async () => {
    await initI18n();
    await i18n.changeLanguage("zh-CN");
  });

  it.each([
    ["zh-CN", zhCN],
    ["en", en],
  ])("跨设备区在 %s 下说明手机发起和桌面管理", async (locale, copy) => {
    await i18n.changeLanguage(locale);
    render(<CrossDeviceSection />);
    expect(
      screen.getByRole("heading", {
        name: copy["crossDevice.mobile.title"],
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(copy["crossDevice.desktop.desc"]),
    ).toBeInTheDocument();
    expect(
      screen.queryByText(/原生客户端|native mobile app/i),
    ).not.toBeInTheDocument();
  });

  it("适用人群区渲染三类独立标题、收益和用法", () => {
    render(<AudienceSection />);
    for (const id of audienceIds) {
      expect(
        screen.getByRole("heading", {
          name: zhCN[`audience.${id}.title`],
        }),
      ).toBeInTheDocument();
      expect(
        screen.getByText(zhCN[`audience.${id}.benefit`]),
      ).toBeInTheDocument();
      expect(
        screen.getByText(zhCN[`audience.${id}.usage`]),
      ).toBeInTheDocument();
    }
  });
});
