import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { AuthProvider } from "../../lib/auth";
import { I18nProvider } from "../../lib/i18n";
import { ThemeProvider } from "../../lib/theme";
import { jsonResponse, installFetchMock, localeRoutes } from "../../test/mock-fetch";
import { agentUser, renderWithProviders, seedAuth } from "../../test/test-utils";
import { ConversationDetailPage } from "./ConversationDetailPage";

describe("ConversationDetailPage", () => {
  it("submits the selected agent during admin assignment", async () => {
    const assignmentPayloads: Array<{ agent_id: string }> = [];

    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/api\/v1\/users\/agents$/,
        response: jsonResponse({
          data: [{ id: "agent-1", name: "Aaron Agent" }],
        }),
      },
      {
        match: /\/api\/v1\/conversations\/conv-1$/,
        response: jsonResponse({
          data: {
            id: "conv-1",
            status: "open",
            customer: { name: "Lena Hart" },
            channel: { name: "WhatsApp" },
            messages: [],
            assigned_agent_id: null,
          },
        }),
      },
      {
        match: (url, init) =>
          url.endsWith("/api/v1/conversations/conv-1/assign") && init?.method === "PATCH",
        response: (_url, init) => {
          assignmentPayloads.push(JSON.parse(String(init?.body)) as { agent_id: string });

          return jsonResponse({
            data: {
              id: "conv-1",
              status: "assigned",
              customer: { name: "Lena Hart" },
              channel: { name: "WhatsApp" },
              messages: [],
              assigned_agent_id: "agent-1",
              assigned_agent: { name: "Aaron Agent" },
            },
          });
        },
      },
    ]);

    seedAuth();

    const user = userEvent.setup();
    renderWithProviders(<ConversationDetailPage />, {
      route: "/conversations/conv-1",
      path: "/conversations/:id",
    });

    await screen.findByText("Lena Hart");

    await user.selectOptions(screen.getByRole("combobox", { name: "Assigned Agent" }), "agent-1");
    await user.click(screen.getByRole("button", { name: "Save Assignment" }));

    await waitFor(() => expect(assignmentPayloads).toEqual([{ agent_id: "agent-1" }]));
    await waitFor(() => expect(screen.getAllByText("Aaron Agent").length).toBeGreaterThan(0));
  });

  it("lets an agent reply, escalate, and open the created ticket from the toast action", async () => {
    const replyPayloads: Array<{ message: string }> = [];
    const escalatePayloads: Array<{ title: string; description: string; priority: string }> = [];

    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/api\/v1\/users\/agents$/,
        response: jsonResponse({
          data: [{ id: "agent-1", name: "Aaron Agent" }],
        }),
      },
      {
        match: /\/api\/v1\/conversations\/conv-1$/,
        response: jsonResponse({
          data: {
            id: "conv-1",
            status: "assigned",
            customer: { name: "Lena Hart" },
            channel: { name: "WhatsApp" },
            assigned_agent_id: "agent-1",
            assigned_agent: { name: "Aaron Agent" },
            messages: [],
          },
        }),
      },
      {
        match: (url, init) =>
          url.endsWith("/api/v1/conversations/conv-1/messages") && init?.method === "POST",
        response: (_url, init) => {
          replyPayloads.push(JSON.parse(String(init?.body)) as { message: string });
          return jsonResponse({
            data: {
              id: "conv-1",
              status: "assigned",
              customer: { name: "Lena Hart" },
              channel: { name: "WhatsApp" },
              assigned_agent_id: "agent-1",
              assigned_agent: { name: "Aaron Agent" },
              messages: [{ id: "message-1", sender_type: "agent", message: "Handled by agent" }],
            },
          });
        },
      },
      {
        match: (url, init) =>
          url.endsWith("/api/v1/conversations/conv-1/escalate") && init?.method === "POST",
        response: (_url, init) => {
          escalatePayloads.push(
            JSON.parse(String(init?.body)) as {
              title: string;
              description: string;
              priority: string;
            },
          );
          return jsonResponse({ data: { id: "ticket-1" } });
        },
      },
    ]);

    seedAuth(agentUser);
    const user = userEvent.setup();
    renderConversationDetailWithRoutes();

    await screen.findByText("Lena Hart");
    expect(screen.queryByRole("button", { name: "Save Assignment" })).not.toBeInTheDocument();

    const replyForm = screen
      .getByRole("heading", { name: /^(Reply|conversation\.reply)$/ })
      .closest("form");
    expect(replyForm).not.toBeNull();
    await user.type(within(replyForm as HTMLFormElement).getByRole("textbox"), "Handled by agent");
    await user.click(
      within(replyForm as HTMLFormElement).getByRole("button", {
        name: /^(Send Reply|conversation\.sendReply)$/,
      }),
    );

    await waitFor(() => expect(replyPayloads).toEqual([{ message: "Handled by agent" }]));
    await screen.findByText("Handled by agent");

    const escalateForm = screen
      .getByRole("heading", { name: /^(Escalate to ticket|ticket\.escalate)$/ })
      .closest("form");
    expect(escalateForm).not.toBeNull();
    const textboxes = within(escalateForm as HTMLFormElement).getAllByRole("textbox");
    await user.type(textboxes[0], "Need escalation");
    await user.type(textboxes[1], "Customer needs a specialist.");
    await user.selectOptions(within(escalateForm as HTMLFormElement).getByRole("combobox"), "high");
    await user.click(
      within(escalateForm as HTMLFormElement).getByRole("button", {
        name: /^(Create Ticket|ticket\.create)$/,
      }),
    );

    await waitFor(() =>
      expect(escalatePayloads).toEqual([
        { title: "Need escalation", description: "Customer needs a specialist.", priority: "high" },
      ]),
    );
    await screen.findByText("Tickets Page");

    const viewTicketAction = await screen.findByRole("button", { name: "View ticket" });
    await user.click(viewTicketAction);

    await screen.findByText("Ticket Detail Page");
  });

  it("shows a linked ticket and hides the escalate form when a ticket already exists", async () => {
    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/api\/v1\/users\/agents$/,
        response: jsonResponse({ data: [{ id: "agent-1", name: "Aaron Agent" }] }),
      },
      {
        match: /\/api\/v1\/conversations\/conv-1$/,
        response: jsonResponse({
          data: {
            id: "conv-1",
            status: "assigned",
            customer: { name: "Lena Hart" },
            channel: { name: "WhatsApp" },
            assigned_agent_id: "agent-1",
            ticket: { id: "ticket-1" },
            messages: [],
          },
        }),
      },
    ]);

    seedAuth(agentUser);
    renderConversationDetailWithRoutes();

    await screen.findByRole("link", { name: /^(View linked ticket|conversation\.viewTicket)$/ });
    expect(
      screen.queryByRole("heading", { name: /^(Escalate to ticket|ticket\.escalate)$/ }),
    ).not.toBeInTheDocument();
  });

  it("navigates back to the conversation list when the detail request fails", async () => {
    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/api\/v1\/users\/agents$/,
        response: jsonResponse({ data: [] }),
      },
      {
        match: /\/api\/v1\/conversations\/conv-1$/,
        response: jsonResponse({ error: { message: "not found" } }, { status: 404 }),
      },
    ]);

    seedAuth();
    renderConversationDetailWithRoutes();

    await screen.findByText("Conversation List");
  });
});

function renderConversationDetailWithRoutes() {
  return render(
    <MemoryRouter initialEntries={["/conversations/conv-1"]}>
      <ThemeProvider>
        <I18nProvider>
          <AuthProvider>
            <Routes>
              <Route path="/conversations/:id" element={<ConversationDetailPage />} />
              <Route path="/conversations" element={<div>Conversation List</div>} />
              <Route path="/tickets" element={<div>Tickets Page</div>} />
              <Route path="/tickets/:id" element={<div>Ticket Detail Page</div>} />
            </Routes>
          </AuthProvider>
        </I18nProvider>
      </ThemeProvider>
    </MemoryRouter>,
  );
}
