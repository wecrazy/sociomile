import { useI18n } from "../lib/i18n";
import { type WorkspaceServiceStatus, useWorkspaceStatus } from "../lib/workspace-status";

type WorkspaceServiceStatusListProps = {
  className?: string;
};

export function WorkspaceServiceStatusList({ className }: WorkspaceServiceStatusListProps) {
  const { t } = useI18n();
  const { services } = useWorkspaceStatus();
  const items = [
    { id: "api", label: t("shell.backendService"), status: services.api },
    { id: "worker", label: t("shell.workerService"), status: services.worker },
  ];

  return (
    <div className={["workspace-service-list", className].filter(Boolean).join(" ")}>
      {items.map((item) => (
        <article
          className={`workspace-service-chip is-${item.status}`}
          data-service={item.id}
          data-status={item.status}
          key={item.id}
        >
          <span className="workspace-service-label">{item.label}</span>
          <strong className="workspace-service-value">
            {t(getServiceStatusLabel(item.status))}
          </strong>
        </article>
      ))}
    </div>
  );
}

function getServiceStatusLabel(status: WorkspaceServiceStatus) {
  switch (status) {
    case "online":
      return "shell.statusOnline";
    case "offline":
      return "shell.statusOffline";
    case "checking":
      return "shell.statusChecking";
    default:
      return "shell.statusUnknown";
  }
}
