import {
  faChartLine,
  faComments,
  faGlobe,
  faRightFromBracket,
  faSliders,
  faTicket,
} from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { NavLink, useLocation } from "react-router-dom";
import { AnimatedOutlet } from "./AnimatedOutlet";
import { ThemeToggle } from "./ThemeToggle";
import { useI18n } from "../lib/i18n";
import { useAuth } from "../lib/auth";
import { StatusBadge } from "./StatusBadge";
import { WorkspaceStatusBadge } from "./WorkspaceStatusBadge";
import { useNotifications } from "../lib/notifications";

export function AppLayout() {
  const { t, locale, setLocale } = useI18n();
  const { user, logout } = useAuth();
  const notifications = useNotifications();
  const location = useLocation();
  const localeLabel = locale === "en" ? "English" : "Bahasa Indonesia";
  const workspaceReference = formatWorkspaceReference(user?.tenant_id, user?.tenant?.slug);
  const navItems = [
    { icon: faChartLine, label: t("nav.dashboard"), to: "/" },
    { icon: faComments, label: t("nav.conversations"), to: "/conversations" },
    { icon: faTicket, label: t("nav.tickets"), to: "/tickets" },
    { icon: faSliders, label: t("nav.settings"), to: "/settings" },
  ];
  const currentItem =
    navItems.find((item) =>
      item.to === "/"
        ? location.pathname === "/"
        : location.pathname === item.to || location.pathname.startsWith(`${item.to}/`),
    ) ?? navItems[0];
  const initials =
    user?.name
      ?.split(" ")
      .filter(Boolean)
      .slice(0, 2)
      .map((part) => part.charAt(0).toUpperCase())
      .join("") ?? "SO";

  function handleLocaleChange(nextLocale: string) {
    setLocale(nextLocale);
    notifications.languageChanged(nextLocale);
  }

  function handleLogout() {
    logout();
    notifications.logoutSuccess();
  }

  return (
    <div className="shell">
      <aside className="sidebar">
        <div className="sidebar-stack">
          <div className="brand-card">
            <div aria-hidden="true" className="brand-mark">
              <img alt="" className="sociomile-mark" height="56" src="/favicon.svg" width="56" />
            </div>
            <div className="brand-copy">
              <p className="eyebrow">Sociomile</p>
              <h2>{user?.tenant?.name ?? user?.tenant_id}</h2>
              <WorkspaceStatusBadge className="brand-ribbon" />
              <p className="sidebar-copy">{user?.email}</p>
            </div>
          </div>
          <div className="workspace-card">
            <div className="workspace-row">
              <div>
                <span className="workspace-label">{t("shell.operator")}</span>
                <strong>{user?.name}</strong>
              </div>
              {user?.role ? <StatusBadge tone="role" value={user.role} /> : null}
            </div>
            <div className="workspace-grid">
              <article className="workspace-stat">
                <span>{t("shell.tenant")}</span>
                <strong className="workspace-value" title={user?.tenant_id}>
                  {workspaceReference}
                </strong>
              </article>
              <article className="workspace-stat">
                <span>{t("settings.language")}</span>
                <strong>{localeLabel}</strong>
              </article>
            </div>
          </div>
          <nav className="nav-links">
            {navItems.map((item) => (
              <NavLink
                className={({ isActive }) => (isActive ? "active" : undefined)}
                end={item.to === "/"}
                key={item.to}
                to={item.to}
              >
                <span className="nav-icon">
                  <FontAwesomeIcon icon={item.icon} />
                </span>
                <span>{item.label}</span>
              </NavLink>
            ))}
          </nav>
        </div>
        <div className="sidebar-footer">
          <div className="control-card">
            <span className="control-label">
              <FontAwesomeIcon icon={faGlobe} />
              <span>{t("settings.language")}</span>
            </span>
            <select
              aria-label={t("settings.language")}
              value={locale}
              onChange={(event) => handleLocaleChange(event.target.value)}
            >
              <option value="en">English</option>
              <option value="id">Bahasa Indonesia</option>
            </select>
          </div>
          <ThemeToggle />
          <button
            className="button ghost full-width button-with-icon sidebar-signout"
            onClick={handleLogout}
            type="button"
          >
            <FontAwesomeIcon icon={faRightFromBracket} />
            <span>{t("auth.logout")}</span>
          </button>
        </div>
      </aside>
      <main className="content">
        <header className="topbar">
          <div className="topbar-intro">
            <p className="eyebrow">{t("shell.eyebrow")}</p>
            <h1>{t("shell.headline")}</h1>
            <p>{`${currentItem.label} - ${user?.tenant?.name ?? user?.tenant_id}`}</p>
          </div>
          <div className="topbar-actions">
            <WorkspaceStatusBadge className="topbar-signal" />
            <div className="topbar-card">
              <span className="profile-avatar" aria-hidden="true">
                {initials}
              </span>
              <div className="topbar-card-copy">
                <strong>{user?.name}</strong>
                <span>{user?.email}</span>
              </div>
              {user?.role ? <StatusBadge tone="role" value={user.role} /> : null}
            </div>
          </div>
        </header>
        <AnimatedOutlet />
      </main>
    </div>
  );
}

function formatWorkspaceReference(tenantID?: string, tenantSlug?: string) {
  if (tenantSlug) {
    return tenantSlug;
  }

  if (!tenantID) {
    return "sociomile";
  }

  if (tenantID.length <= 18) {
    return tenantID;
  }

  return `${tenantID.slice(0, 8)}...${tenantID.slice(-4)}`;
}
