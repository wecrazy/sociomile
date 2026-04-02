import { fireEvent, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { I18nProvider } from "../lib/i18n";
import { ThemeProvider } from "../lib/theme";
import { WorkspaceStatusProvider } from "../lib/workspace-status";
import { jsonResponse, installFetchMock, localeRoutes } from "../test/mock-fetch";
import { WorkspaceStatusBadge } from "./WorkspaceStatusBadge";
import { WorkspaceServiceStatusList } from "./WorkspaceServiceStatusList";

const { toastSuccessSpy } = vi.hoisted(() => ({
  toastSuccessSpy: vi.fn(),
}));

vi.mock("sonner", async () => {
  const actual = await vi.importActual<typeof import("sonner")>("sonner");

  return {
    ...actual,
    toast: {
      ...actual.toast,
      success: toastSuccessSpy,
    },
  };
});

describe("WorkspaceStatusBadge", () => {
  beforeEach(() => {
    toastSuccessSpy.mockClear();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("switches from live to offline when health checks start failing", async () => {
    let healthChecks = 0;

    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/health$/,
        response: () => {
          healthChecks += 1;

          if (healthChecks === 1) {
            return new Response(JSON.stringify({ status: "ok", port: 8080 }), {
              headers: { "Content-Type": "application/json" },
              status: 200,
            });
          }

          return Promise.reject(new Error("backend offline"));
        },
      },
    ]);

    const { container } = render(
      <ThemeProvider>
        <I18nProvider>
          <WorkspaceStatusProvider>
            <WorkspaceStatusBadge className="login-pill" />
          </WorkspaceStatusProvider>
        </I18nProvider>
      </ThemeProvider>,
    );

    await waitFor(() =>
      expect(container.querySelector('[role="status"][data-status="online"]')).not.toBeNull(),
    );

    fireEvent(window, new Event("online"));

    await waitFor(() =>
      expect(container.querySelector('[role="status"][data-status="offline"]')).not.toBeNull(),
    );
  });

  it("shows backend and worker health and announces reconnection", async () => {
    let healthChecks = 0;

    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/health$/,
        response: () => {
          healthChecks += 1;

          if (healthChecks === 1) {
            return Promise.reject(new Error("backend offline"));
          }

          return jsonResponse({
            status: "degraded",
            port: 8080,
            services: {
              api: { status: "online" },
              worker: { status: "offline" },
            },
          });
        },
      },
    ]);

    const { container } = render(
      <ThemeProvider>
        <I18nProvider>
          <WorkspaceStatusProvider>
            <WorkspaceStatusBadge className="login-pill" />
            <WorkspaceServiceStatusList className="login-service-list" />
          </WorkspaceStatusProvider>
        </I18nProvider>
      </ThemeProvider>,
    );

    await waitFor(() =>
      expect(container.querySelector('[role="status"][data-status="offline"]')).not.toBeNull(),
    );

    fireEvent(window, new Event("online"));

    await waitFor(() =>
      expect(container.querySelector('[role="status"][data-status="online"]')).not.toBeNull(),
    );
    await waitFor(() =>
      expect(container.querySelector('[data-service="api"][data-status="online"]')).not.toBeNull(),
    );

    expect(
      container.querySelector('[data-service="worker"][data-status="offline"]'),
    ).not.toBeNull();
    expect(toastSuccessSpy).toHaveBeenCalledWith("Workspace connection restored.");
  });
});
