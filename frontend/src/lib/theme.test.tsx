import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ThemeProvider, useTheme } from "./theme";

describe("ThemeProvider", () => {
  it("honors a stored theme and toggles it", async () => {
    localStorage.setItem("sociomile-theme", "dark");

    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <ThemeHarness />
      </ThemeProvider>,
    );

    await waitFor(() => expect(document.documentElement).toHaveAttribute("data-theme", "dark"));
    expect(screen.getByTestId("mode")).toHaveTextContent("dark");

    await user.click(screen.getByRole("button", { name: "Toggle Theme" }));

    await waitFor(() => expect(document.documentElement).toHaveAttribute("data-theme", "light"));
    expect(screen.getByTestId("mode")).toHaveTextContent("light");
    expect(localStorage.getItem("sociomile-theme")).toBe("light");
  });

  it("uses the system preference when no stored theme exists", async () => {
    Object.defineProperty(window, "matchMedia", {
      writable: true,
      value: vi.fn().mockImplementation((query: string) => ({
        matches: true,
        media: query,
        onchange: null,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        addListener: vi.fn(),
        removeListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    });

    render(
      <ThemeProvider>
        <ThemeHarness />
      </ThemeProvider>,
    );

    await waitFor(() => expect(document.documentElement).toHaveAttribute("data-theme", "dark"));
    expect(screen.getByTestId("mode")).toHaveTextContent("dark");
  });
});

function ThemeHarness() {
  const { mode, toggleMode } = useTheme();

  return (
    <div>
      <span data-testid="mode">{mode}</span>
      <button onClick={toggleMode} type="button">
        Toggle Theme
      </button>
    </div>
  );
}
