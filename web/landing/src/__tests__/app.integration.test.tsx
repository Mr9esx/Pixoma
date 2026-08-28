import { beforeAll, describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App } from "@/App";
import { initI18n } from "@/i18n";

describe("app integration", () => {
  beforeAll(async () => {
    await initI18n();
  });

  it("渲染 hero + features + cases 场景 + cta", () => {
    render(
      <MemoryRouter initialEntries={["/cn"]}>
        <App />
      </MemoryRouter>,
    );
    expect(screen.getAllByRole("heading").length).toBeGreaterThanOrEqual(4);
  });
});
