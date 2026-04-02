import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { installFetchMock, localeRoutes } from "../../test/mock-fetch";
import { renderWithProviders, seedAuth } from "../../test/test-utils";
import { SettingsPage } from "./SettingsPage";

describe("SettingsPage", () => {
  it("persists theme and locale selections across remounts", async () => {
    installFetchMock(localeRoutes());
    seedAuth();

    const user = userEvent.setup();
    const firstRender = renderWithProviders(<SettingsPage />, {
      route: "/settings",
      path: "/settings",
    });

    await screen.findByText("Workspace settings");
    await waitFor(() => expect(document.documentElement).toHaveAttribute("data-theme", "light"));

    await user.click(screen.getByRole("button", { name: "Switch to dark mode" }));

    await waitFor(() => expect(localStorage.getItem("sociomile-theme")).toBe("dark"));
    expect(document.documentElement).toHaveAttribute("data-theme", "dark");

    await user.selectOptions(screen.getByRole("combobox", { name: "Language" }), "id");

    await screen.findByText("Pengaturan workspace");
    expect(localStorage.getItem("sociomile-locale")).toBe("id");

    firstRender.unmount();

    renderWithProviders(<SettingsPage />, {
      route: "/settings",
      path: "/settings",
    });

    await screen.findByText("Pengaturan workspace");
    await waitFor(() => expect(document.documentElement).toHaveAttribute("data-theme", "dark"));
    expect(screen.getByRole("button", { name: "Ganti ke mode terang" })).toBeInTheDocument();
  });
});
