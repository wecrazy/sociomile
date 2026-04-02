import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { AuthProvider } from "../../lib/auth";
import { I18nProvider } from "../../lib/i18n";
import { ThemeProvider } from "../../lib/theme";
import { jsonResponse, installFetchMock, localeRoutes } from "../../test/mock-fetch";
import { adminUser } from "../../test/test-utils";
import { LoginPage } from "./LoginPage";

describe("LoginPage", () => {
  it("switches between the seeded admin and agent demo accounts", async () => {
    installFetchMock([...localeRoutes()]);

    const user = userEvent.setup();
    renderLoginPage();

    const emailInput = await screen.findByLabelText("Email");
    const passwordInput = screen.getByLabelText("Password");

    expect(emailInput).toHaveValue("alice.admin@acme.local");
    expect(passwordInput).toHaveValue("Password123!");

    await user.click(screen.getByRole("button", { name: "Use Agent login" }));

    expect(emailInput).toHaveValue("aaron.agent@acme.local");
    expect(passwordInput).toHaveValue("Password123!");
  });

  it("logs in successfully and navigates home", async () => {
    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/api\/v1\/auth\/login$/,
        response: jsonResponse({
          data: {
            access_token: "fresh-token",
            user: adminUser,
          },
        }),
      },
    ]);

    const user = userEvent.setup();
    renderLoginPage();

    await screen.findByText("Sign in to your tenant workspace");
    await user.click(screen.getByRole("button", { name: "Sign In" }));

    await screen.findByText("Home Screen");
    expect(localStorage.getItem("sociomile-auth")).toContain("fresh-token");
  });

  it("shows the API error message when login fails", async () => {
    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/api\/v1\/auth\/login$/,
        response: jsonResponse(
          {
            error: {
              message: "Invalid credentials.",
            },
          },
          { status: 401 },
        ),
      },
    ]);

    const user = userEvent.setup();
    renderLoginPage();

    await screen.findByText("Sign in to your tenant workspace");
    await user.click(screen.getByRole("button", { name: "Sign In" }));

    await waitFor(() => expect(screen.getByText("Invalid credentials.")).toBeInTheDocument());
    expect(localStorage.getItem("sociomile-auth")).toBeNull();
  });
});

function renderLoginPage() {
  return render(
    <MemoryRouter initialEntries={["/login"]}>
      <ThemeProvider>
        <I18nProvider>
          <AuthProvider>
            <Routes>
              <Route path="/login" element={<LoginPage />} />
              <Route path="/" element={<div>Home Screen</div>} />
            </Routes>
          </AuthProvider>
        </I18nProvider>
      </ThemeProvider>
    </MemoryRouter>,
  );
}
