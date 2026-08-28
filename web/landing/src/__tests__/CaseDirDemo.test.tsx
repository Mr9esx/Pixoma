import { describe, expect, it } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { CaseDirDemo } from "@/components/demos/CaseDirDemo";

describe("CaseDirDemo", () => {
  it("渲染目录列表并响应选择", () => {
    render(<CaseDirDemo />);
    expect(screen.getByText("都市夜景工作流")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "水墨山水" }));
    expect(screen.getByText("国风水墨工作流")).toBeInTheDocument();
  });
});
