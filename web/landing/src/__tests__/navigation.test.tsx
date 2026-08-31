import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { Footer } from "@/components/layout/Footer";
import { Header } from "@/components/layout/Header";
import { navigationItems } from "@/constants/navigation";
import i18n, { initI18n } from "@/i18n";
import zhCN from "@/i18n/locales/zh-CN";

vi.mock("motion/react", async (importOriginal) => {
  const mod = await importOriginal<typeof import("motion/react")>();
  return {
    ...mod,
    useReducedMotion: vi.fn(() => true),
  };
});

function NavigationFixture() {
  return (
    <>
      <Header />
      <main>
        {navigationItems.map((item) => (
          <section key={item.id} id={item.id} />
        ))}
      </main>
      <Footer />
    </>
  );
}

describe("landing navigation", () => {
  beforeEach(async () => {
    await initI18n();
    await i18n.changeLanguage("zh-CN");
  });

  it("桌面与页尾导航只指向存在的五个区块", () => {
    render(
      <MemoryRouter initialEntries={["/cn"]}>
        <NavigationFixture />
      </MemoryRouter>,
    );

    for (const item of navigationItems) {
      const links = screen.getAllByRole("link", {
        name: zhCN[item.labelKey],
      });
      expect(links).toHaveLength(2);
      expect(
        links.every((link) => link.getAttribute("href") === item.href),
      ).toBe(true);
      links.forEach((link) => {
        expect(link).toHaveClass("min-h-11", "min-w-11");
      });
      expect(document.querySelector(item.href)).toBeInTheDocument();
    }
    expect(screen.getByRole("link", { name: "Pixoma" })).toHaveClass(
      "min-h-11",
      "min-w-11",
    );
    expect(
      screen.queryByRole("link", { name: /tools|pricing|价格|工具集成/i }),
    ).not.toBeInTheDocument();
  });

  it("选择移动导航后关闭菜单", () => {
    render(
      <MemoryRouter initialEntries={["/cn"]}>
        <Header />
      </MemoryRouter>,
    );

    fireEvent.click(
      screen.getByRole("button", { name: zhCN["nav.menu.open"] }),
    );
    const mobileNavigation = screen.getByRole("navigation", {
      name: zhCN["nav.mobile"],
    });
    expect(mobileNavigation.parentElement).not.toHaveStyle("opacity: 0");
    expect(mobileNavigation.parentElement).not.toHaveStyle("height: 0px");
    fireEvent.click(
      within(mobileNavigation).getByRole("link", {
        name: zhCN["nav.faq"],
      }),
    );
    expect(
      screen.queryByRole("navigation", { name: zhCN["nav.mobile"] }),
    ).not.toBeInTheDocument();
  });
});
