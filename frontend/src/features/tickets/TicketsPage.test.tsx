import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { jsonResponse, installFetchMock, localeRoutes } from "../../test/mock-fetch";
import { renderWithProviders, seedAuth } from "../../test/test-utils";
import { TicketsPage } from "./TicketsPage";

describe("TicketsPage", () => {
  it("applies status, priority, and agent filters while resetting offset", async () => {
    const requestedTicketUrls: string[] = [];

    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/api\/v1\/users\/agents$/,
        response: jsonResponse({
          data: [{ id: "agent-1", name: "Aaron Agent" }],
        }),
      },
      {
        match: /\/api\/v1\/tickets\?/,
        response: (url) => {
          requestedTicketUrls.push(url);
          return jsonResponse({
            data: [],
            meta: { total: 50, offset: 0, limit: 10 },
          });
        },
      },
    ]);

    seedAuth();

    const user = userEvent.setup();
    renderWithProviders(<TicketsPage />, {
      route: "/tickets?offset=30&limit=10",
      path: "/tickets",
    });

    await screen.findByText("Ticket queue");
    await waitFor(() =>
      expect(
        requestedTicketUrls.some((url) => url.includes("offset=30") && url.includes("limit=10")),
      ).toBe(true),
    );

    await user.click(screen.getByRole("button", { name: "Next" }));

    await waitFor(() =>
      expect(
        requestedTicketUrls.some((url) => url.includes("offset=40") && url.includes("limit=10")),
      ).toBe(true),
    );

    await user.selectOptions(screen.getByRole("combobox", { name: "Status" }), "resolved");
    await waitFor(() =>
      expect(
        requestedTicketUrls.some(
          (url) => url.includes("offset=0") && url.includes("status=resolved"),
        ),
      ).toBe(true),
    );

    await user.selectOptions(screen.getByRole("combobox", { name: "Priority" }), "high");
    await waitFor(() =>
      expect(
        requestedTicketUrls.some(
          (url) => url.includes("status=resolved") && url.includes("priority=high"),
        ),
      ).toBe(true),
    );

    await user.selectOptions(screen.getByRole("combobox", { name: "Assigned Agent" }), "agent-1");
    await waitFor(() =>
      expect(
        requestedTicketUrls.some(
          (url) => url.includes("priority=high") && url.includes("assigned_agent_id=agent-1"),
        ),
      ).toBe(true),
    );

    await user.selectOptions(screen.getByRole("combobox", { name: "Priority" }), "");
    await waitFor(() =>
      expect(
        requestedTicketUrls.some(
          (url) =>
            url.includes("status=resolved") &&
            url.includes("assigned_agent_id=agent-1") &&
            !url.includes("priority="),
        ),
      ).toBe(true),
    );
  });
});
