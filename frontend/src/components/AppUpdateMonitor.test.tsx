import { render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import * as appVersion from "../lib/app-version";
import { I18nProvider } from "../lib/i18n";
import { ThemeProvider } from "../lib/theme";
import { jsonResponse, installFetchMock, localeRoutes } from "../test/mock-fetch";
import { AppUpdateMonitor } from "./AppUpdateMonitor";

const { toastInfoSpy } = vi.hoisted(() => ({
  toastInfoSpy: vi.fn(),
}));

vi.mock("sonner", async () => {
  const actual = await vi.importActual<typeof import("sonner")>("sonner");

  return {
    ...actual,
    toast: {
      ...actual.toast,
      info: toastInfoSpy,
    },
  };
});

describe("AppUpdateMonitor", () => {
  beforeEach(() => {
    toastInfoSpy.mockClear();
    window.history.replaceState({}, "", "/settings");
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("shows a refresh prompt when a newer frontend build is available", async () => {
    installFetchMock([
      ...localeRoutes(),
      {
        match: /\/version\.json/,
        response: jsonResponse({
          version: "forced-next-version",
          builtAt: "2026-04-02T12:00:00.000Z",
        }),
      },
    ]);

    const refreshSpy = vi
      .spyOn(appVersion, "refreshToLatestVersion")
      .mockImplementation(() => undefined);

    render(
      <ThemeProvider>
        <I18nProvider>
          <AppUpdateMonitor />
        </I18nProvider>
      </ThemeProvider>,
    );

    await waitFor(() => expect(toastInfoSpy).toHaveBeenCalledTimes(1));

    expect(toastInfoSpy).toHaveBeenCalledWith(
      "A new version is available.",
      expect.objectContaining({
        action: expect.objectContaining({
          label: "Refresh now",
        }),
      }),
    );

    const refreshAction = toastInfoSpy.mock.calls[0]?.[1]?.action;
    refreshAction.onClick();

    expect(refreshSpy).toHaveBeenCalledWith("forced-next-version");
  });
});
