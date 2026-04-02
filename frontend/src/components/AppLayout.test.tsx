import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { AuthProvider } from "../lib/auth";
import { I18nProvider } from "../lib/i18n";
import { ThemeProvider } from "../lib/theme";
import { installFetchMock, localeRoutes } from "../test/mock-fetch";
import { AppLayout } from "./AppLayout";

describe("AppLayout", () => {
  it("renders navigation, switches locale, and logs out", async () => {
    installFetchMock(localeRoutes());
    localStorage.setItem(
      "sociomile-auth",
      JSON.stringify({
        token: "test-token",
        user: {
          id: "admin-1",
          tenant_id: "tenant-1",
          name: "Alice Admin",
          email: "alice.admin@acme.local",
          role: "admin",
          tenant: { id: "tenant-1", name: "Acme Inc" },
        },
      }),
    );

    const user = userEvent.setup();
    render(
      <MemoryRouter initialEntries={["/"]}>
        <ThemeProvider>
          <I18nProvider>
            <AuthProvider>
              <Routes>
                <Route path="/" element={<AppLayout />}>
                  <Route index element={<div>Outlet Content</div>} />
                </Route>
              </Routes>
            </AuthProvider>
          </I18nProvider>
        </ThemeProvider>
      </MemoryRouter>,
    );

    await screen.findByText("Dashboard");
    expect(screen.getByText("Acme Inc")).toBeInTheDocument();
    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    expect(screen.getByText("Conversations")).toBeInTheDocument();
    expect(screen.getByText("Tickets")).toBeInTheDocument();
    expect(screen.getByText("Settings")).toBeInTheDocument();
    expect(screen.getByText("Outlet Content")).toBeInTheDocument();

    await user.selectOptions(screen.getByRole("combobox", { name: "Language" }), "id");
    await screen.findByRole("button", { name: "Keluar" });
    expect(localStorage.getItem("sociomile-locale")).toBe("id");

    await user.click(screen.getByRole("button", { name: "Keluar" }));
    await waitFor(() => expect(localStorage.getItem("sociomile-auth")).toBeNull());
  });
});
