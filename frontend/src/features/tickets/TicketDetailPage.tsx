import { faTicket } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { apiRequest } from "../../lib/api";
import { useAuth } from "../../lib/auth";
import { StatusBadge } from "../../components/StatusBadge";
import { useI18n } from "../../lib/i18n";
import { useNotifications } from "../../lib/notifications";

type Ticket = {
  id: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  assigned_agent?: { name: string };
};

export function TicketDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { token, user } = useAuth();
  const { t } = useI18n();
  const notifications = useNotifications();
  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [status, setStatus] = useState("open");
  const [isSavingStatus, setIsSavingStatus] = useState(false);

  useEffect(() => {
    if (!id) {
      return;
    }

    apiRequest<Ticket>(`/tickets/${id}`, { token })
      .then((result) => {
        setTicket(result.data);
        setStatus(result.data.status);
      })
      .catch(() => {
        notifications.errorKey("toast.ticketLoadFailed");
        navigate("/tickets");
      });
  }, [id, navigate, notifications, token]);

  async function updateStatus(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!id) {
      return;
    }

    setIsSavingStatus(true);

    try {
      const result = await notifications.ticketStatus(
        apiRequest<Ticket>(`/tickets/${id}/status`, {
          method: "PATCH",
          body: { status },
          token,
        }),
      );

      setTicket(result.data);
    } catch {
      return;
    } finally {
      setIsSavingStatus(false);
    }
  }

  if (!ticket) {
    return (
      <section className="page-grid">
        <div className="panel loading-panel">{t("common.loading")}</div>
      </section>
    );
  }

  return (
    <section className="page-grid detail-grid">
      <article className="detail-card">
        <div className="detail-heading">
          <div className="detail-heading-copy">
            <p className="eyebrow">
              <FontAwesomeIcon icon={faTicket} /> {t("nav.tickets")}
            </p>
            <h1>{ticket.title}</h1>
          </div>
          <StatusBadge tone="status" value={ticket.status} />
        </div>
        <p>{ticket.description}</p>
        <div className="detail-meta-grid">
          <article className="meta-tile">
            <span>{t("ticket.priority")}</span>
            <strong>
              <StatusBadge tone="priority" value={ticket.priority} />
            </strong>
          </article>
          <article className="meta-tile">
            <span>{t("ticket.assignedAgent")}</span>
            <strong>{ticket.assigned_agent?.name ?? "-"}</strong>
          </article>
        </div>
      </article>
      {user?.role === "admin" ? (
        <aside className="detail-sidebar">
          <form className="form-stack panel" onSubmit={updateStatus}>
            <h2>{t("ticket.updateStatus")}</h2>
            <select
              aria-label={t("ticket.status")}
              disabled={isSavingStatus}
              value={status}
              onChange={(event) => setStatus(event.target.value)}
            >
              <option value="open">Open</option>
              <option value="in_progress">In Progress</option>
              <option value="resolved">Resolved</option>
              <option value="closed">Closed</option>
            </select>
            <button className="button full-width" disabled={isSavingStatus} type="submit">
              {t("ticket.saveStatus")}
            </button>
          </form>
        </aside>
      ) : null}
    </section>
  );
}
