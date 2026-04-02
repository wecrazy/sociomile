import {
  faArrowDown,
  faArrowRotateRight,
  faArrowUp,
  faCircle,
  faCircleCheck,
  faCircleDot,
  faHeadset,
  faLock,
  faShieldHalved,
  faUserCheck,
} from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

type BadgeTone = "priority" | "role" | "status";

type BadgeProps = {
  tone: BadgeTone;
  value: string;
};

type BadgeConfig = {
  className: string;
  icon: typeof faCircleDot;
};

function getBadgeConfig(tone: BadgeTone, value: string): BadgeConfig {
  const normalizedValue = value.toLowerCase();

  if (tone === "status") {
    switch (normalizedValue) {
      case "open":
        return { className: "is-open", icon: faCircle };
      case "assigned":
        return { className: "is-assigned", icon: faUserCheck };
      case "closed":
        return { className: "is-closed", icon: faLock };
      case "resolved":
        return { className: "is-resolved", icon: faCircleCheck };
      case "in_progress":
        return { className: "is-progress", icon: faArrowRotateRight };
      default:
        return { className: "is-neutral", icon: faCircleDot };
    }
  }

  if (tone === "priority") {
    switch (normalizedValue) {
      case "low":
        return { className: "is-low", icon: faArrowDown };
      case "medium":
        return { className: "is-medium", icon: faCircleDot };
      case "high":
        return { className: "is-high", icon: faArrowUp };
      default:
        return { className: "is-neutral", icon: faCircleDot };
    }
  }

  switch (normalizedValue) {
    case "admin":
      return { className: "is-admin", icon: faShieldHalved };
    case "agent":
      return { className: "is-agent", icon: faHeadset };
    default:
      return { className: "is-neutral", icon: faCircleDot };
  }
}

export function formatTokenLabel(value: string) {
  return value
    .split("_")
    .filter(Boolean)
    .map((segment) => segment.charAt(0).toUpperCase() + segment.slice(1))
    .join(" ");
}

export function StatusBadge({ tone, value }: BadgeProps) {
  const config = getBadgeConfig(tone, value);

  return (
    <span className={`badge badge-${tone} ${config.className}`}>
      <FontAwesomeIcon fixedWidth icon={config.icon} />
      <span>{formatTokenLabel(value)}</span>
    </span>
  );
}
