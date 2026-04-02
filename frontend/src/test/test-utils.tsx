import type { ReactElement } from "react";
import { render } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { AuthProvider, type User } from "../lib/auth";
import { I18nProvider } from "../lib/i18n";
import { ThemeProvider } from "../lib/theme";

export const adminUser: User = {
  id: "admin-1",
  tenant_id: "tenant-1",
  name: "Alice Admin",
  email: "alice.admin@acme.local",
  role: "admin",
  tenant: {
    id: "tenant-1",
    name: "Acme Inc",
  },
};

export const agentUser: User = {
  id: "agent-1",
  tenant_id: "tenant-1",
  name: "Aaron Agent",
  email: "aaron.agent@acme.local",
  role: "agent",
  tenant: {
    id: "tenant-1",
    name: "Acme Inc",
  },
};

export function seedAuth(user: User = adminUser, token = "test-token") {
  localStorage.setItem("sociomile-auth", JSON.stringify({ token, user }));
  return { token, user };
}

export function renderWithProviders(
  ui: ReactElement,
  options: { route?: string; path?: string } = {},
) {
  const { route = "/", path = "/" } = options;

  return render(
    <MemoryRouter initialEntries={[route]}>
      <ThemeProvider>
        <I18nProvider>
          <AuthProvider>
            <Routes>
              <Route path={path} element={ui} />
            </Routes>
          </AuthProvider>
        </I18nProvider>
      </ThemeProvider>
    </MemoryRouter>,
  );
}
