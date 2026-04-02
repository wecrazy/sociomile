import { faComments, faTicket } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useEffect, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { apiRequest } from "../../lib/api";
import { useAuth } from "../../lib/auth";
import { StatusBadge, formatTokenLabel } from "../../components/StatusBadge";
import { useI18n } from "../../lib/i18n";
import { useTenantAgents } from "../../lib/useTenantAgents";
import { useNotifications } from "../../lib/notifications";

type Message = {
  id: string;
  sender_type: string;
  message: string;
  created_at: string;
};

type Conversation = {
  id: string;
  status: string;
  customer?: { name: string };
  channel?: { name: string };
  assigned_agent?: { name: string };
  assigned_agent_id?: string;
  messages?: Message[];
  ticket?: { id: string };
};

export function ConversationDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { token, user } = useAuth();
  const { t } = useI18n();
  const notifications = useNotifications();
  const { agents, loading: agentsLoading } = useTenantAgents(token);
  const [conversation, setConversation] = useState<Conversation | null>(null);
  const [reply, setReply] = useState("");
  const [ticketForm, setTicketForm] = useState({ title: "", description: "", priority: "medium" });
  const [assignedAgentId, setAssignedAgentId] = useState("");
  const [isAssigning, setIsAssigning] = useState(false);
  const [isReplying, setIsReplying] = useState(false);
  const [isEscalating, setIsEscalating] = useState(false);

  useEffect(() => {
    if (!id) {
      return;
    }

    apiRequest<Conversation>(`/conversations/${id}`, { token })
      .then((result) => {
        setConversation(result.data);
        setAssignedAgentId(result.data.assigned_agent_id ?? "");
      })
      .catch(() => {
        notifications.errorKey("toast.conversationLoadFailed");
        navigate("/conversations");
      });
  }, [id, navigate, notifications, token]);

  async function submitReply(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!id || !reply) {
      return;
    }

    setIsReplying(true);

    try {
      const result = await notifications.reply(
        apiRequest<Conversation>(`/conversations/${id}/messages`, {
          method: "POST",
          body: { message: reply },
          token,
        }),
      );

      setConversation(result.data);
      setReply("");
    } catch {
      return;
    } finally {
      setIsReplying(false);
    }
  }

  async function escalate(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!id) {
      return;
    }

    setIsEscalating(true);

    try {
      const result = await notifications.ticketCreation(
        apiRequest<{ id: string }>(`/conversations/${id}/escalate`, {
          method: "POST",
          body: ticketForm,
          token,
        }),
        {
          action: (response) => ({
            label: t("toast.viewTicketAction"),
            onClick: () => navigate(`/tickets/${response.data.id}`),
          }),
        },
      );

      if (!result.data.id) {
        return;
      }

      navigate("/tickets");
    } catch {
      return;
    } finally {
      setIsEscalating(false);
    }
  }

  async function assignConversation(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!id || !assignedAgentId) {
      return;
    }

    setIsAssigning(true);

    try {
      const result = await notifications.assignment(
        apiRequest<Conversation>(`/conversations/${id}/assign`, {
          method: "PATCH",
          body: { agent_id: assignedAgentId },
          token,
        }),
      );

      setConversation(result.data);
      setAssignedAgentId(result.data.assigned_agent_id ?? "");
    } catch {
      return;
    } finally {
      setIsAssigning(false);
    }
  }

  if (!conversation) {
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
              <FontAwesomeIcon icon={faComments} /> {conversation.channel?.name}
            </p>
            <h1>{conversation.customer?.name ?? conversation.id}</h1>
          </div>
          <StatusBadge tone="status" value={conversation.status} />
        </div>
        <div className="detail-meta-grid">
          <article className="meta-tile">
            <span>{t("conversation.assignedAgent")}</span>
            <strong>{conversation.assigned_agent?.name ?? "-"}</strong>
          </article>
          <article className="meta-tile">
            <span>{t("conversation.channel")}</span>
            <strong>{conversation.channel?.name ?? "-"}</strong>
          </article>
        </div>
        {conversation.ticket ? (
          <p className="detail-link-row">
            <Link
              className="button ghost button-with-icon"
              to={`/tickets/${conversation.ticket.id}`}
            >
              <FontAwesomeIcon icon={faTicket} />
              <span>{t("conversation.viewTicket")}</span>
            </Link>
          </p>
        ) : null}
        <div className="message-list">
          {(conversation.messages ?? []).map((message) => (
            <article className={`message-card ${message.sender_type}`} key={message.id}>
              <div className="message-meta">
                <strong>{formatTokenLabel(message.sender_type)}</strong>
                <span>{new Date(message.created_at).toLocaleString()}</span>
              </div>
              <p>{message.message}</p>
            </article>
          ))}
        </div>
      </article>
      <aside className="detail-sidebar">
        {user?.role === "admin" ? (
          <form className="form-stack panel" onSubmit={assignConversation}>
            <h2>{t("conversation.assign")}</h2>
            <select
              aria-label={t("conversation.assignedAgent")}
              disabled={agentsLoading || isAssigning}
              value={assignedAgentId}
              onChange={(event) => setAssignedAgentId(event.target.value)}
            >
              <option value="">
                {agentsLoading ? t("common.loading") : t("conversation.assignToAgent")}
              </option>
              {agents.map((agent) => (
                <option key={agent.id} value={agent.id}>
                  {agent.name}
                </option>
              ))}
            </select>
            <button
              className="button secondary full-width"
              disabled={!assignedAgentId || isAssigning}
              type="submit"
            >
              {t("conversation.saveAssignment")}
            </button>
          </form>
        ) : null}
        {user?.role === "agent" ? (
          <form className="form-stack panel" onSubmit={submitReply}>
            <h2>{t("conversation.reply")}</h2>
            <textarea
              disabled={isReplying}
              rows={4}
              value={reply}
              onChange={(event) => setReply(event.target.value)}
            />
            <button className="button full-width" disabled={isReplying || !reply} type="submit">
              {t("conversation.sendReply")}
            </button>
          </form>
        ) : null}
        {user?.role === "agent" && !conversation.ticket ? (
          <form className="form-stack panel" onSubmit={escalate}>
            <h2>{t("ticket.escalate")}</h2>
            <input
              disabled={isEscalating}
              placeholder={t("ticket.title")}
              value={ticketForm.title}
              onChange={(event) =>
                setTicketForm((current) => ({ ...current, title: event.target.value }))
              }
            />
            <textarea
              disabled={isEscalating}
              rows={4}
              placeholder={t("ticket.description")}
              value={ticketForm.description}
              onChange={(event) =>
                setTicketForm((current) => ({ ...current, description: event.target.value }))
              }
            />
            <select
              disabled={isEscalating}
              value={ticketForm.priority}
              onChange={(event) =>
                setTicketForm((current) => ({ ...current, priority: event.target.value }))
              }
            >
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
            </select>
            <button
              className="button secondary full-width"
              disabled={isEscalating || !ticketForm.title.trim() || !ticketForm.description.trim()}
              type="submit"
            >
              {t("ticket.create")}
            </button>
          </form>
        ) : null}
      </aside>
    </section>
  );
}
