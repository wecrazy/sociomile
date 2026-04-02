import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { AuthProvider } from "../lib/auth";
import { ProtectedRoute } from "./ProtectedRoute";

describe("ProtectedRoute", () => {
  it("redirects unauthenticated users to login", async () => {
    render(
      <MemoryRouter initialEntries={["/private"]}>
        <AuthProvider>
          <Routes>
            <Route path="/login" element={<div>Login Route</div>} />
            <Route
              path="/private"
              element={
                <ProtectedRoute>
                  <div>Private Route</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>,
    );

    expect(await screen.findByText("Login Route")).toBeInTheDocument();
  });

  it("renders the child route for authenticated users", async () => {
    localStorage.setItem(
      "sociomile-auth",
      JSON.stringify({ token: "test-token", user: { id: "admin-1" } }),
    );

    render(
      <MemoryRouter initialEntries={["/private"]}>
        <AuthProvider>
          <Routes>
            <Route path="/login" element={<div>Login Route</div>} />
            <Route
              path="/private"
              element={
                <ProtectedRoute>
                  <div>Private Route</div>
                </ProtectedRoute>
              }
            />
          </Routes>
        </AuthProvider>
      </MemoryRouter>,
    );

    expect(await screen.findByText("Private Route")).toBeInTheDocument();
  });
});
