import { faGlobe, faIdBadge, faPalette, faUserLarge } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useAuth } from "../../lib/auth";
import { useI18n } from "../../lib/i18n";
import { ThemeToggle } from "../../components/ThemeToggle";
import { StatusBadge } from "../../components/StatusBadge";
import { useNotifications } from "../../lib/notifications";

export function SettingsPage() {
  const { user } = useAuth();
  const { locale, setLocale, t } = useI18n();
  const notifications = useNotifications();
  const localeLabel = locale === "en" ? "English" : "Bahasa Indonesia";

  function handleLocaleChange(nextLocale: string) {
    setLocale(nextLocale);
    notifications.languageChanged(nextLocale);
  }

  return (
    <section className="page-grid settings-grid">
      <article className="panel settings-hero">
        <div className="settings-hero-content">
          <div className="title-block">
            <span className="title-icon">
              <FontAwesomeIcon icon={faPalette} />
            </span>
            <div>
              <p className="eyebrow">{t("nav.settings")}</p>
              <h1>{t("settings.title")}</h1>
              <p>{t("settings.subtitle")}</p>
            </div>
          </div>
        </div>
        <div className="settings-hero-metrics">
          <article className="settings-metric">
            <span>{t("shell.tenant")}</span>
            <strong>{user?.tenant?.name ?? user?.tenant_id}</strong>
          </article>
          <article className="settings-metric">
            <span>{t("settings.language")}</span>
            <strong>{localeLabel}</strong>
          </article>
        </div>
      </article>
      <article className="panel profile-panel">
        <div className="panel-heading">
          <h2>{t("settings.profile")}</h2>
          {user?.role ? <StatusBadge tone="role" value={user.role} /> : null}
        </div>
        <div className="profile-grid">
          <div className="profile-item">
            <span className="profile-icon">
              <FontAwesomeIcon icon={faUserLarge} />
            </span>
            <div>
              <span className="profile-label">{t("shell.operator")}</span>
              <strong>{user?.name}</strong>
            </div>
          </div>
          <div className="profile-item">
            <span className="profile-icon">
              <FontAwesomeIcon icon={faIdBadge} />
            </span>
            <div>
              <span className="profile-label">Email</span>
              <strong>{user?.email}</strong>
            </div>
          </div>
        </div>
      </article>
      <article className="panel preference-panel form-stack">
        <div className="panel-heading">
          <h2>{t("settings.preferences")}</h2>
          <span className="panel-chip">
            <FontAwesomeIcon icon={faGlobe} />
            <span>{locale === "en" ? "English" : "Bahasa Indonesia"}</span>
          </span>
        </div>
        <ThemeToggle />
        <select
          aria-label={t("settings.language")}
          value={locale}
          onChange={(event) => handleLocaleChange(event.target.value)}
        >
          <option value="en">English</option>
          <option value="id">Bahasa Indonesia</option>
        </select>
      </article>
    </section>
  );
}
