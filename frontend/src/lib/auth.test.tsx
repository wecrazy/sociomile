import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { jsonResponse, installFetchMock } from "../test/mock-fetch";
import { adminUser } from "../test/test-utils";
import { AuthProvider, useAuth } from "./auth";

describe("AuthProvider", () => {
  it("falls back to a signed-out state when persisted auth is invalid", () => {
    localStorage.setItem("sociomile-auth", "{");

    render(
      <AuthProvider>
        <AuthHarness />
      </AuthProvider>,
    );

    expect(screen.getByTestId("token")).toHaveTextContent("none");
    expect(screen.getByTestId("user")).toHaveTextContent("none");
  });

  it("persists successful login state and clears it on logout", async () => {
    installFetchMock([
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
    render(
      <AuthProvider>
        <AuthHarness />
      </AuthProvider>,
    );

    await user.click(screen.getByRole("button", { name: "Login" }));

    await waitFor(() => expect(screen.getByTestId("token")).toHaveTextContent("fresh-token"));
    expect(screen.getByTestId("user")).toHaveTextContent(adminUser.email);
    expect(localStorage.getItem("sociomile-auth")).toContain("fresh-token");

    await user.click(screen.getByRole("button", { name: "Logout" }));

    await waitFor(() => expect(localStorage.getItem("sociomile-auth")).toBeNull());
    expect(screen.getByTestId("token")).toHaveTextContent("none");
    expect(screen.getByTestId("user")).toHaveTextContent("none");
  });
});

function AuthHarness() {
  const { token, user, login, logout } = useAuth();

  return (
    <div>
      <span data-testid="token">{token ?? "none"}</span>
      <span data-testid="user">{user?.email ?? "none"}</span>
      <button onClick={() => void login("alice.admin@acme.local", "Password123!")} type="button">
        Login
      </button>
      <button onClick={logout} type="button">
        Logout
      </button>
    </div>
  );
}
