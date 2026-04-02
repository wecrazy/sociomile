import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { AuthProvider } from "../lib/auth";
import { I18nProvider } from "../lib/i18n";
import { ThemeProvider } from "../lib/theme";
import { installFetchMock, localeRoutes } from "../test/mock-fetch";
import { seedAuth } from "../test/test-utils";
import { App } from "./App";

describe("App", () => {
  it("redirects unknown unauthenticated routes to login", async () => {
    installFetchMock(localeRoutes());

    renderApp("/unknown");

    expect(await screen.findByText("Sign in to your tenant workspace")).toBeInTheDocument();
  });

  it("renders a protected nested route for authenticated users", async () => {
    installFetchMock(localeRoutes());
    seedAuth();

    renderApp("/settings");

    expect(await screen.findByText("Workspace settings")).toBeInTheDocument();
  });
});

function renderApp(route: string) {
  return render(
    <MemoryRouter initialEntries={[route]}>
      <ThemeProvider>
        <I18nProvider>
          <AuthProvider>
            <App />
          </AuthProvider>
        </I18nProvider>
      </ThemeProvider>
    </MemoryRouter>,
  );
}
