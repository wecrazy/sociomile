import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { jsonResponse, installFetchMock, localeRoutes } from "../test/mock-fetch";
import { renderWithProviders, seedAuth } from "../test/test-utils";
import { DashboardPage } from "./DashboardPage";

describe("DashboardPage", () => {
  it("loads conversation and ticket totals", async () => {
    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/api\/v1\/conversations\?offset=0&limit=1$/,
        response: jsonResponse({
          data: [],
          meta: { total: 12, offset: 0, limit: 1 },
        }),
      },
      {
        match: /\/api\/v1\/tickets\?offset=0&limit=1$/,
        response: jsonResponse({
          data: [],
          meta: { total: 4, offset: 0, limit: 1 },
        }),
      },
    ]);

    seedAuth();
    renderWithProviders(<DashboardPage />, { route: "/", path: "/" });

    await screen.findByText("Operations overview");
    expect(screen.getAllByText("12").length).toBeGreaterThan(0);
    expect(screen.getAllByText("4").length).toBeGreaterThan(0);
  });
});
