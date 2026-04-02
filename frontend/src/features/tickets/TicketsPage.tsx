import { faTicket, faUserGroup } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { useEffect, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { DataTable } from "../../components/DataTable";
import { StatusBadge } from "../../components/StatusBadge";
import { apiRequest } from "../../lib/api";
import { useAuth } from "../../lib/auth";
import { useI18n } from "../../lib/i18n";
import { useTenantAgents } from "../../lib/useTenantAgents";

type Ticket = {
  id: string;
  title: string;
  status: string;
  priority: string;
  assigned_agent?: { name: string };
};

export function TicketsPage() {
  const { token } = useAuth();
  const { t } = useI18n();
  const { agents } = useTenantAgents(token);
  const [searchParams, setSearchParams] = useSearchParams();
  const [rows, setRows] = useState<Ticket[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  const offset = Number(searchParams.get("offset") ?? 0);
  const limit = Number(searchParams.get("limit") ?? 10);
  const status = searchParams.get("status") ?? "";
  const priority = searchParams.get("priority") ?? "";
  const assignedAgentId = searchParams.get("assigned_agent_id") ?? "";

  useEffect(() => {
    let cancelled = false;

    async function load() {
      setLoading(true);
      try {
        const query = new URLSearchParams({ offset: String(offset), limit: String(limit) });
        if (status) query.set("status", status);
        if (priority) query.set("priority", priority);
        if (assignedAgentId) query.set("assigned_agent_id", assignedAgentId);

        const result = await apiRequest<Ticket[]>(`/tickets?${query.toString()}`, { token });
        if (!cancelled) {
          setRows(result.data);
          setTotal(result.meta?.total ?? 0);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    load().catch(() => undefined);

    return () => {
      cancelled = true;
    };
  }, [assignedAgentId, limit, offset, priority, status, token]);

  function updateFilter(name: string, value: string) {
    const next = new URLSearchParams(searchParams);
    if (value) {
      next.set(name, value);
    } else {
      next.delete(name);
    }
    next.set("offset", "0");
    setSearchParams(next);
  }

  return (
    <section className="page-grid">
      <div className="section-header">
        <div className="title-block">
          <span className="title-icon">
            <FontAwesomeIcon icon={faTicket} />
          </span>
          <div>
            <p className="eyebrow">{t("nav.tickets")}</p>
            <h1>{t("ticket.titlePlural")}</h1>
          </div>
        </div>
        <div className="filters-row">
          <label className="filter-field">
            <span className="filter-label">{t("ticket.status")}</span>
            <select
              aria-label={t("ticket.status")}
              value={status}
              onChange={(event) => updateFilter("status", event.target.value)}
            >
              <option value="">{t("filters.allStatuses")}</option>
              <option value="open">Open</option>
              <option value="in_progress">In Progress</option>
              <option value="resolved">Resolved</option>
              <option value="closed">Closed</option>
            </select>
          </label>
          <label className="filter-field">
            <span className="filter-label">{t("ticket.priority")}</span>
            <select
              aria-label={t("ticket.priority")}
              value={priority}
              onChange={(event) => updateFilter("priority", event.target.value)}
            >
              <option value="">{t("filters.allPriorities")}</option>
              <option value="low">Low</option>
              <option value="medium">Medium</option>
              <option value="high">High</option>
            </select>
          </label>
          <label className="filter-field">
            <span className="filter-label">
              <FontAwesomeIcon icon={faUserGroup} />
              <span>{t("ticket.assignedAgent")}</span>
            </span>
            <select
              aria-label={t("ticket.assignedAgent")}
              value={assignedAgentId}
              onChange={(event) => updateFilter("assigned_agent_id", event.target.value)}
            >
              <option value="">{t("filters.allAgents")}</option>
              {agents.map((agent) => (
                <option key={agent.id} value={agent.id}>
                  {agent.name}
                </option>
              ))}
            </select>
          </label>
        </div>
      </div>
      <DataTable
        columns={[
          {
            key: "title",
            header: t("ticket.title"),
            render: (row) => (
              <div className="table-stack">
                <Link className="table-link" to={`/tickets/${row.id}`}>
                  {row.title}
                </Link>
                <span className="table-meta">{row.id.slice(0, 8)}</span>
              </div>
            ),
          },
          {
            key: "status",
            header: t("ticket.status"),
            render: (row) => <StatusBadge tone="status" value={row.status} />,
          },
          {
            key: "priority",
            header: t("ticket.priority"),
            render: (row) => <StatusBadge tone="priority" value={row.priority} />,
          },
          {
            key: "agent",
            header: t("ticket.assignedAgent"),
            render: (row) => row.assigned_agent?.name ?? "-",
          },
        ]}
        rows={rows}
        total={total}
        offset={offset}
        limit={limit}
        loading={loading}
        onPageChange={(nextOffset) => {
          const next = new URLSearchParams(searchParams);
          next.set("offset", String(nextOffset));
          next.set("limit", String(limit));
          setSearchParams(next);
        }}
      />
    </section>
  );
}
