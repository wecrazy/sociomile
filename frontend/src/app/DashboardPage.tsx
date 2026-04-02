import {
  faArrowRight,
  faChartLine,
  faComments,
  faSignal,
  faTicket,
} from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiRequest } from "../lib/api";
import { useAuth } from "../lib/auth";
import { useI18n } from "../lib/i18n";
import { WorkspaceStatusBadge } from "../components/WorkspaceStatusBadge";

type Metrics = {
  conversations: number;
  tickets: number;
};

export function DashboardPage() {
  const { token, user } = useAuth();
  const { t } = useI18n();
  const [metrics, setMetrics] = useState<Metrics>({ conversations: 0, tickets: 0 });
  const totalWorkload = metrics.conversations + metrics.tickets;
  const conversationShare =
    totalWorkload === 0 ? 0 : Math.round((metrics.conversations / totalWorkload) * 100);
  const ticketShare = totalWorkload === 0 ? 0 : 100 - conversationShare;

  useEffect(() => {
    let cancelled = false;

    async function load() {
      const [conversationResponse, ticketResponse] = await Promise.all([
        apiRequest<unknown[]>("/conversations?offset=0&limit=1", { token }),
        apiRequest<unknown[]>("/tickets?offset=0&limit=1", { token }),
      ]);

      if (!cancelled) {
        setMetrics({
          conversations: conversationResponse.meta?.total ?? 0,
          tickets: ticketResponse.meta?.total ?? 0,
        });
      }
    }

    load().catch(() => undefined);

    return () => {
      cancelled = true;
    };
  }, [token]);

  return (
    <section className="page-grid dashboard-grid">
      <div className="hero-card hero-card-dashboard">
        <div className="hero-copy">
          <div className="hero-heading">
            <p className="eyebrow">Sociomile</p>
            <h1>{t("dashboard.title")}</h1>
            <p>{t("dashboard.subtitle")}</p>
          </div>
          <div className="hero-actions">
            <Link className="button button-with-icon" to="/conversations">
              <span>{t("nav.conversations")}</span>
              <FontAwesomeIcon icon={faArrowRight} />
            </Link>
            <Link className="button ghost button-with-icon" to="/tickets">
              <span>{t("nav.tickets")}</span>
              <FontAwesomeIcon icon={faArrowRight} />
            </Link>
          </div>
          <div className="hero-strip">
            <article className="hero-strip-card">
              <span className="hero-aside-label">{t("dashboard.totalWorkload")}</span>
              <strong>{totalWorkload}</strong>
              <p>{user?.name ?? "Sociomile"}</p>
            </article>
            <article className="hero-strip-card">
              <span className="hero-aside-label">{t("dashboard.queueMix")}</span>
              <strong>{`${conversationShare}% / ${ticketShare}%`}</strong>
              <p>
                {t("dashboard.openConversations")} / {t("dashboard.openTickets")}
              </p>
            </article>
          </div>
        </div>
        <aside className="hero-aside">
          <div className="hero-aside-head">
            <div className="hero-aside-mark">
              <FontAwesomeIcon icon={faSignal} />
            </div>
            <div>
              <span className="hero-aside-label">{user?.tenant?.name ?? "Sociomile"}</span>
              <WorkspaceStatusBadge className="hero-status-pill" />
            </div>
          </div>
          <strong>{totalWorkload}</strong>
          <p>{t("dashboard.subtitle")}</p>
          <div className="hero-aside-stats">
            <article className="hero-aside-stat">
              <span>{t("dashboard.openConversations")}</span>
              <strong>{metrics.conversations}</strong>
            </article>
            <article className="hero-aside-stat">
              <span>{t("dashboard.openTickets")}</span>
              <strong>{metrics.tickets}</strong>
            </article>
          </div>
        </aside>
      </div>
      <div className="stats-grid stats-grid-dashboard">
        <article className="stat-card stat-card-accent-primary">
          <span className="stat-icon stat-icon-primary">
            <FontAwesomeIcon icon={faComments} />
          </span>
          <span className="stat-card-label">{t("dashboard.openConversations")}</span>
          <strong>{metrics.conversations}</strong>
          <p className="stat-card-meta">{user?.tenant?.name ?? "Sociomile"}</p>
        </article>
        <article className="stat-card stat-card-accent-secondary">
          <span className="stat-icon stat-icon-secondary">
            <FontAwesomeIcon icon={faTicket} />
          </span>
          <span className="stat-card-label">{t("dashboard.openTickets")}</span>
          <strong>{metrics.tickets}</strong>
          <p className="stat-card-meta">{user?.email}</p>
        </article>
        <article className="stat-card stat-card-accent-primary">
          <span className="stat-icon stat-icon-primary">
            <FontAwesomeIcon icon={faSignal} />
          </span>
          <span className="stat-card-label">{t("dashboard.totalWorkload")}</span>
          <strong>{totalWorkload}</strong>
          <p className="stat-card-meta">
            {t("dashboard.openConversations")} + {t("dashboard.openTickets")}
          </p>
        </article>
        <article className="stat-card stat-card-accent-secondary">
          <span className="stat-icon stat-icon-secondary">
            <FontAwesomeIcon icon={faChartLine} />
          </span>
          <span className="stat-card-label">{t("dashboard.queueMix")}</span>
          <strong>{`${conversationShare}% / ${ticketShare}%`}</strong>
          <p className="stat-card-meta">{`${metrics.conversations}:${metrics.tickets}`}</p>
        </article>
      </div>
    </section>
  );
}
