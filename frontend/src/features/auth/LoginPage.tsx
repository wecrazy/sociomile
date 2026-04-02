import { faArrowRight, faComments, faMoon, faTicket } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../lib/auth";
import { useI18n } from "../../lib/i18n";
import { useNotifications } from "../../lib/notifications";
import { WorkspaceStatusBadge } from "../../components/WorkspaceStatusBadge";

const demoAccounts = [
  {
    id: "admin",
    roleKey: "auth.roleAdmin",
    tenant: "Acme Support",
    name: "Alice Admin",
    email: "alice.admin@acme.local",
    password: "Password123!",
  },
  {
    id: "agent",
    roleKey: "auth.roleAgent",
    tenant: "Acme Support",
    name: "Aaron Agent",
    email: "aaron.agent@acme.local",
    password: "Password123!",
  },
] as const;

export function LoginPage() {
  const { login } = useAuth();
  const { t } = useI18n();
  const notifications = useNotifications();
  const navigate = useNavigate();
  const [email, setEmail] = useState<string>(demoAccounts[0].email);
  const [password, setPassword] = useState<string>(demoAccounts[0].password);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  function applyDemoAccount(account: (typeof demoAccounts)[number]) {
    setEmail(account.email);
    setPassword(account.password);
    setError("");
  }

  async function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setLoading(true);
    setError("");

    try {
      await notifications.login(login(email, password));
      navigate("/");
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : t("auth.invalidCredentials"));
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="login-screen">
      <section className="login-shell">
        <aside className="login-showcase">
          <div className="login-showcase-header">
            <p className="eyebrow">Sociomile</p>
            <h1>Sociomile</h1>
            <p>{t("shell.headline")}</p>
            <p>{t("auth.subtitle")}</p>
          </div>
          <section className="login-demo-panel" aria-label={t("shell.demoCredentials")}>
            <div className="login-demo-header">
              <span className="control-label">{t("shell.demoCredentials")}</span>
              <p>{t("auth.demoHint")}</p>
            </div>
            <div className="login-demo-grid">
              {demoAccounts.map((account) => {
                const roleLabel = t(account.roleKey);

                return (
                  <article className="login-demo-card" key={account.id}>
                    <div className="login-demo-meta">
                      <span className={`login-demo-role login-demo-role-${account.id}`}>
                        {roleLabel}
                      </span>
                      <span className="login-demo-tenant">{account.tenant}</span>
                    </div>
                    <strong>{account.name}</strong>
                    <span className="login-demo-email">{account.email}</span>
                    <button
                      className="button ghost login-demo-action"
                      onClick={() => applyDemoAccount(account)}
                      type="button"
                    >
                      {t("auth.useDemoAccount", { role: roleLabel })}
                    </button>
                  </article>
                );
              })}
            </div>
            <p className="login-demo-note">
              {t("auth.demoPasswordHint")}{" "}
              <span className="login-demo-secret">{demoAccounts[0].password}</span>
            </p>
          </section>
          <div className="login-highlights">
            <article className="login-highlight-card">
              <span className="login-highlight-icon">
                <FontAwesomeIcon icon={faComments} />
              </span>
              <div>
                <strong>{t("nav.conversations")}</strong>
                <p>{t("conversation.title")}</p>
              </div>
            </article>
            <article className="login-highlight-card">
              <span className="login-highlight-icon">
                <FontAwesomeIcon icon={faTicket} />
              </span>
              <div>
                <strong>{t("nav.tickets")}</strong>
                <p>{t("ticket.titlePlural")}</p>
              </div>
            </article>
            <article className="login-highlight-card">
              <span className="login-highlight-icon">
                <FontAwesomeIcon icon={faMoon} />
              </span>
              <div>
                <strong>{t("nav.settings")}</strong>
                <p>{t("settings.subtitle")}</p>
              </div>
            </article>
          </div>
        </aside>
        <section className="login-panel">
          <div className="login-panel-header">
            <div>
              <p className="eyebrow">Sociomile</p>
              <h1>{t("auth.title")}</h1>
              <p>{t("auth.subtitle")}</p>
            </div>
            <WorkspaceStatusBadge className="login-pill" />
          </div>
          <form className="form-stack" onSubmit={handleSubmit}>
            <label className="field-stack">
              <span>{t("auth.email")}</span>
              <input
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
              />
            </label>
            <label className="field-stack">
              <span>{t("auth.password")}</span>
              <input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </label>
            {error ? <p className="error-text">{error}</p> : null}
            <button className="button full-width button-with-icon" disabled={loading} type="submit">
              <span>{loading ? t("common.loading") : t("auth.submit")}</span>
              <FontAwesomeIcon icon={faArrowRight} />
            </button>
          </form>
        </section>
      </section>
    </main>
  );
}
