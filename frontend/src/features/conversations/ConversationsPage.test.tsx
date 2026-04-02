import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { jsonResponse, installFetchMock, localeRoutes } from "../../test/mock-fetch";
import { renderWithProviders, seedAuth } from "../../test/test-utils";
import { ConversationsPage } from "./ConversationsPage";

describe("ConversationsPage", () => {
  it("applies server-side status and agent filters while resetting offset", async () => {
    const requestedConversationUrls: string[] = [];

    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/api\/v1\/users\/agents$/,
        response: jsonResponse({
          data: [{ id: "agent-1", name: "Aaron Agent" }],
        }),
      },
      {
        match: /\/api\/v1\/conversations\?/,
        response: (url) => {
          requestedConversationUrls.push(url);
          return jsonResponse({
            data: [],
            meta: { total: 40, offset: 0, limit: 10 },
          });
        },
      },
    ]);

    seedAuth();

    const user = userEvent.setup();
    renderWithProviders(<ConversationsPage />, {
      route: "/conversations?offset=20&limit=10",
      path: "/conversations",
    });

    await screen.findByText("Conversation queue");
    await waitFor(() =>
      expect(
        requestedConversationUrls.some(
          (url) => url.includes("offset=20") && url.includes("limit=10"),
        ),
      ).toBe(true),
    );

    await user.click(screen.getByRole("button", { name: "Next" }));

    await waitFor(() =>
      expect(
        requestedConversationUrls.some(
          (url) => url.includes("offset=30") && url.includes("limit=10"),
        ),
      ).toBe(true),
    );

    await user.selectOptions(screen.getByRole("combobox", { name: "Status" }), "assigned");

    await waitFor(() =>
      expect(
        requestedConversationUrls.some(
          (url) => url.includes("offset=0") && url.includes("status=assigned"),
        ),
      ).toBe(true),
    );

    await user.selectOptions(screen.getByRole("combobox", { name: "Assigned Agent" }), "agent-1");

    await waitFor(() =>
      expect(
        requestedConversationUrls.some(
          (url) => url.includes("status=assigned") && url.includes("assigned_agent_id=agent-1"),
        ),
      ).toBe(true),
    );

    await user.selectOptions(screen.getByRole("combobox", { name: "Status" }), "");

    await waitFor(() =>
      expect(
        requestedConversationUrls.some(
          (url) =>
            url.includes("assigned_agent_id=agent-1") &&
            url.includes("offset=0") &&
            !url.includes("status="),
        ),
      ).toBe(true),
    );
  });
});
