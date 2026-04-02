import {
  createContext,
  type PropsWithChildren,
  useContext,
  useEffect,
  useEffectEvent,
  useRef,
  useState,
} from "react";
import { HEALTH_CHECK_URL } from "./api";
import { useNotifications } from "./notifications";

export type WorkspaceStatus = "checking" | "online" | "offline";
export type WorkspaceServiceStatus = "checking" | "online" | "offline" | "unknown";

type WorkspaceServiceState = {
  api: WorkspaceServiceStatus;
  worker: WorkspaceServiceStatus;
};

type WorkspaceStatusValue = {
  status: WorkspaceStatus;
  services: WorkspaceServiceState;
};

type HealthPayload = {
  services?: {
    api?: { status?: string };
    worker?: { status?: string };
  };
};

const WorkspaceStatusContext = createContext<WorkspaceStatusValue | null>(null);
const WORKSPACE_STATUS_POLL_MS = 5000;
const WORKSPACE_STATUS_TIMEOUT_MS = 4000;
const OFFLINE_SERVICES: WorkspaceServiceState = { api: "offline", worker: "unknown" };
const CHECKING_SERVICES: WorkspaceServiceState = { api: "checking", worker: "checking" };

export function WorkspaceStatusProvider({ children }: PropsWithChildren) {
  const value = useWorkspaceStatusState(true, true);

  return (
    <WorkspaceStatusContext.Provider value={value}>{children}</WorkspaceStatusContext.Provider>
  );
}

export function useWorkspaceStatus() {
  const context = useContext(WorkspaceStatusContext);
  const fallback = useWorkspaceStatusState(context === null, false);

  return context ?? fallback;
}

function useWorkspaceStatusState(
  enabled: boolean,
  notifyOnReconnect: boolean,
): WorkspaceStatusValue {
  const notifications = useNotifications();
  const [snapshot, setSnapshot] = useState<WorkspaceStatusValue>({
    status: "checking",
    services: CHECKING_SERVICES,
  });
  const previousStatusRef = useRef<WorkspaceStatus>("checking");

  const setOfflineState = useEffectEvent(() => {
    setSnapshot({ status: "offline", services: OFFLINE_SERVICES });
  });

  const checkWorkspaceStatus = useEffectEvent(async () => {
    if (!enabled) {
      return;
    }

    if (typeof navigator !== "undefined" && navigator.onLine === false) {
      setOfflineState();
      return;
    }

    const controller = new AbortController();
    const timeout = window.setTimeout(() => controller.abort(), WORKSPACE_STATUS_TIMEOUT_MS);

    try {
      const response = await fetch(HEALTH_CHECK_URL, {
        cache: "no-store",
        signal: controller.signal,
      });
      const payload = (await response.json().catch(() => null)) as HealthPayload | null;

      if (!response.ok) {
        throw new Error("Workspace health check failed");
      }

      setSnapshot({
        status: "online",
        services: {
          api: normalizeServiceStatus(payload?.services?.api?.status, "online"),
          worker: normalizeServiceStatus(payload?.services?.worker?.status, "unknown"),
        },
      });
    } catch {
      setOfflineState();
    } finally {
      window.clearTimeout(timeout);
    }
  });

  useEffect(() => {
    if (!enabled) {
      return;
    }

    const previousStatus = previousStatusRef.current;
    if (notifyOnReconnect && previousStatus === "offline" && snapshot.status === "online") {
      notifications.workspaceRestored();
    }

    previousStatusRef.current = snapshot.status;
  }, [enabled, notifications, notifyOnReconnect, snapshot.status]);

  useEffect(() => {
    if (!enabled) {
      return undefined;
    }

    void checkWorkspaceStatus();

    const interval = window.setInterval(() => {
      if (document.visibilityState === "visible") {
        void checkWorkspaceStatus();
      }
    }, WORKSPACE_STATUS_POLL_MS);

    function handleVisibilityChange() {
      if (document.visibilityState === "visible") {
        void checkWorkspaceStatus();
      }
    }

    function handleOnline() {
      void checkWorkspaceStatus();
    }

    function handleOffline() {
      setOfflineState();
    }

    document.addEventListener("visibilitychange", handleVisibilityChange);
    window.addEventListener("online", handleOnline);
    window.addEventListener("offline", handleOffline);

    return () => {
      window.clearInterval(interval);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      window.removeEventListener("online", handleOnline);
      window.removeEventListener("offline", handleOffline);
    };
  }, [checkWorkspaceStatus, enabled]);

  return snapshot;
}

function normalizeServiceStatus(
  value: string | undefined,
  fallback: WorkspaceServiceStatus,
): WorkspaceServiceStatus {
  switch (value) {
    case "checking":
    case "online":
    case "offline":
    case "unknown":
      return value;
    default:
      return fallback;
  }
}
