import { fireEvent, render, screen } from "@testing-library/react";
import { ThemeProvider, useTheme } from "./theme-provider";
import { ThemeSwitcher } from "@/components/kibo-ui/theme-switcher";

function ThemeSwitcherUnderProvider() {
  const { theme, setTheme } = useTheme();

  return <ThemeSwitcher value={theme} onChange={setTheme} />;
}

describe("ThemeProvider", () => {
  beforeEach(() => {
    document.cookie = "vite-ui-theme=; path=/; max-age=0";
    document.documentElement.classList.remove("light", "dark");
  });

  it("switches between light and dark mode", () => {
    render(
      <ThemeProvider>
        <ThemeSwitcherUnderProvider />
      </ThemeProvider>,
    );

    expect(document.documentElement.classList.contains("light")).toBe(true);

    fireEvent.click(screen.getByRole("switch", { name: "Toggle theme" }));
    expect(document.documentElement.classList.contains("dark")).toBe(true);
    expect(document.cookie).toContain("vite-ui-theme=dark");

    fireEvent.click(screen.getByRole("switch", { name: "Toggle theme" }));
    expect(document.documentElement.classList.contains("light")).toBe(true);
    expect(document.cookie).toContain("vite-ui-theme=light");
  });
});
