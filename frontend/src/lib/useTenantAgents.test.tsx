import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { jsonResponse, installFetchMock } from "../test/mock-fetch";
import { useTenantAgents } from "./useTenantAgents";

describe("useTenantAgents", () => {
  it("loads agents for authenticated users and clears them when the token is removed", async () => {
    let resolveResponse: ((value: Response) => void) | undefined;
    installFetchMock([
      {
        match: /\/api\/v1\/users\/agents$/,
        response: () =>
          new Promise<Response>((resolve) => {
            resolveResponse = resolve;
          }),
      },
    ]);

    const view = render(<AgentsHarness token="test-token" />);

    await waitFor(() => expect(screen.getByTestId("loading")).toHaveTextContent("loading"));
    resolveResponse?.(
      jsonResponse({
        data: [{ id: "agent-1", name: "Aaron Agent" }],
      }),
    );

    expect(await screen.findByText("Aaron Agent")).toBeInTheDocument();
    expect(screen.getByTestId("loading")).toHaveTextContent("idle");
    expect(screen.getByTestId("count")).toHaveTextContent("1");

    view.rerender(<AgentsHarness token={null} />);

    await waitFor(() => expect(screen.getByTestId("count")).toHaveTextContent("0"));
    expect(screen.getByTestId("loading")).toHaveTextContent("idle");
  });

  it("swallows request failures and leaves the list empty", async () => {
    installFetchMock([
      {
        match: /\/api\/v1\/users\/agents$/,
        response: jsonResponse({ error: { message: "failed" } }, { status: 500 }),
      },
    ]);

    render(<AgentsHarness token="test-token" />);

    await waitFor(() => expect(screen.getByTestId("loading")).toHaveTextContent("idle"));
    expect(screen.getByTestId("count")).toHaveTextContent("0");
  });

  it("does not fetch when no token is present", () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);

    render(<AgentsHarness token={null} />);

    expect(screen.getByTestId("count")).toHaveTextContent("0");
    expect(screen.getByTestId("loading")).toHaveTextContent("idle");
    expect(fetchMock).not.toHaveBeenCalled();
  });
});

function AgentsHarness({ token }: { token: string | null }) {
  const { agents, loading } = useTenantAgents(token);

  return (
    <div>
      <span data-testid="loading">{loading ? "loading" : "idle"}</span>
      <span data-testid="count">{String(agents.length)}</span>
      {agents.map((agent) => (
        <span key={agent.id}>{agent.name}</span>
      ))}
    </div>
  );
}
