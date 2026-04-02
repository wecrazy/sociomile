import { useI18n } from "../lib/i18n";
import { type WorkspaceStatus, useWorkspaceStatus } from "../lib/workspace-status";

type WorkspaceStatusBadgeProps = {
  className?: string;
};

export function WorkspaceStatusBadge({ className }: WorkspaceStatusBadgeProps) {
  const { t } = useI18n();
  const { status } = useWorkspaceStatus();

  return (
    <span
      aria-live="polite"
      className={["workspace-status-chip", className, `is-${status}`].filter(Boolean).join(" ")}
      data-status={status}
      role="status"
    >
      <span aria-hidden="true" className={`signal-dot workspace-status-dot is-${status}`} />
      <span>{t(getWorkspaceStatusLabel(status))}</span>
    </span>
  );
}

function getWorkspaceStatusLabel(status: WorkspaceStatus) {
  switch (status) {
    case "online":
      return "shell.workspaceLive";
    case "offline":
      return "shell.workspaceOffline";
    default:
      return "shell.workspaceChecking";
  }
}
