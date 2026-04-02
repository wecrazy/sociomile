import { useEffect, useRef } from "react";
import { toast } from "sonner";
import {
  APP_VERSION,
  clearRefreshVersionParam,
  fetchLatestAppVersion,
  refreshToLatestVersion,
} from "../lib/app-version";
import { useI18n } from "../lib/i18n";

const UPDATE_CHECK_INTERVAL_MS = 60_000;
const UPDATE_TOAST_ID = "app-update-available";

export function AppUpdateMonitor() {
  const { t } = useI18n();
  const notifiedVersionRef = useRef<string | null>(null);
  const updateAvailableLabel = t("toast.updateAvailable");
  const refreshActionLabel = t("toast.refreshAction");

  useEffect(() => {
    clearRefreshVersionParam();
  }, []);

  useEffect(() => {
    if (
      import.meta.env.MODE === "development" ||
      updateAvailableLabel === "toast.updateAvailable" ||
      refreshActionLabel === "toast.refreshAction"
    ) {
      return;
    }

    let active = true;

    async function checkForUpdates() {
      try {
        const latest = await fetchLatestAppVersion();
        if (
          !active ||
          !latest.version ||
          latest.version === APP_VERSION ||
          notifiedVersionRef.current === latest.version
        ) {
          return;
        }

        notifiedVersionRef.current = latest.version;
        toast.info(updateAvailableLabel, {
          id: UPDATE_TOAST_ID,
          duration: Number.POSITIVE_INFINITY,
          action: {
            label: refreshActionLabel,
            onClick: () => {
              refreshToLatestVersion(latest.version);
            },
          },
        });
      } catch {
        return;
      }
    }

    function handleVisibilityChange() {
      if (document.visibilityState === "visible") {
        void checkForUpdates();
      }
    }

    function handleFocus() {
      void checkForUpdates();
    }

    void checkForUpdates();

    const intervalID = window.setInterval(() => {
      if (document.visibilityState !== "hidden") {
        void checkForUpdates();
      }
    }, UPDATE_CHECK_INTERVAL_MS);

    window.addEventListener("focus", handleFocus);
    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      active = false;
      window.clearInterval(intervalID);
      window.removeEventListener("focus", handleFocus);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [refreshActionLabel, updateAvailableLabel]);

  return null;
}
