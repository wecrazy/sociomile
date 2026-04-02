import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { jsonResponse, installFetchMock, localeRoutes } from "../../test/mock-fetch";
import { agentUser, renderWithProviders, seedAuth } from "../../test/test-utils";
import { TicketDetailPage } from "./TicketDetailPage";

describe("TicketDetailPage", () => {
  it("submits the updated status for admins", async () => {
    const statusPayloads: Array<{ status: string }> = [];

    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/api\/v1\/tickets\/ticket-1$/,
        response: jsonResponse({
          data: {
            id: "ticket-1",
            title: "Refund issue",
            description: "Customer asked for a refund.",
            status: "open",
            priority: "high",
            assigned_agent: { name: "Aaron Agent" },
          },
        }),
      },
      {
        match: (url, init) =>
          url.endsWith("/api/v1/tickets/ticket-1/status") && init?.method === "PATCH",
        response: (_url, init) => {
          statusPayloads.push(JSON.parse(String(init?.body)) as { status: string });

          return jsonResponse({
            data: {
              id: "ticket-1",
              title: "Refund issue",
              description: "Customer asked for a refund.",
              status: "resolved",
              priority: "high",
              assigned_agent: { name: "Aaron Agent" },
            },
          });
        },
      },
    ]);

    seedAuth();

    const user = userEvent.setup();
    renderWithProviders(<TicketDetailPage />, {
      route: "/tickets/ticket-1",
      path: "/tickets/:id",
    });

    await screen.findByText("Refund issue");

    await user.selectOptions(screen.getByRole("combobox", { name: "Status" }), "resolved");
    await user.click(screen.getByRole("button", { name: "Save Status" }));

    await waitFor(() => expect(statusPayloads).toEqual([{ status: "resolved" }]));
    await waitFor(() =>
      expect(document.querySelector(".badge-status span")).toHaveTextContent("Resolved"),
    );
  });

  it("hides the status form from agents", async () => {
    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/api\/v1\/tickets\/ticket-1$/,
        response: jsonResponse({
          data: {
            id: "ticket-1",
            title: "Refund issue",
            description: "Customer asked for a refund.",
            status: "open",
            priority: "high",
            assigned_agent: { name: "Aaron Agent" },
          },
        }),
      },
    ]);

    seedAuth(agentUser);

    renderWithProviders(<TicketDetailPage />, {
      route: "/tickets/ticket-1",
      path: "/tickets/:id",
    });

    await screen.findByText("Refund issue");
    expect(screen.queryByRole("button", { name: "Save Status" })).not.toBeInTheDocument();
    expect(screen.queryByRole("combobox", { name: "Status" })).not.toBeInTheDocument();
  });
});
