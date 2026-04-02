import { faMoon, faSun } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useI18n } from "../lib/i18n";
import { useNotifications } from "../lib/notifications";
import { useTheme } from "../lib/theme";

export function ThemeToggle() {
  const { mode, toggleMode } = useTheme();
  const { t } = useI18n();
  const notifications = useNotifications();

  function handleToggle() {
    const nextMode = mode === "light" ? "dark" : "light";
    toggleMode();
    notifications.themeChanged(nextMode);
  }

  return (
    <button
      className="button secondary full-width button-with-icon"
      onClick={handleToggle}
      type="button"
    >
      <FontAwesomeIcon icon={mode === "light" ? faMoon : faSun} />
      <span>{mode === "light" ? t("settings.darkMode") : t("settings.lightMode")}</span>
    </button>
  );
}
