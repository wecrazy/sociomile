import { faAngleLeft, faAngleRight, faTableCellsLarge } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import type { ReactNode } from "react";
import { useI18n } from "../lib/i18n";

type Column<Row> = {
  key: string;
  header: string;
  render: (row: Row) => ReactNode;
};

type DataTableProps<Row> = {
  columns: Column<Row>[];
  rows: Row[];
  total: number;
  offset: number;
  limit: number;
  loading: boolean;
  onPageChange: (nextOffset: number) => void;
};

export function DataTable<Row>({
  columns,
  rows,
  total,
  offset,
  limit,
  loading,
  onPageChange,
}: DataTableProps<Row>) {
  const { t } = useI18n();
  const statusText = loading ? t("common.loading") : total === 0 ? t("common.empty") : null;

  return (
    <div className="table-card">
      <div className="table-toolbar">
        <div className="table-toolbar-meta">
          <span className="table-count-pill">
            <FontAwesomeIcon icon={faTableCellsLarge} />
            {t("table.showing", {
              from: String(total === 0 ? 0 : offset + 1),
              to: String(Math.min(offset + limit, total)),
              total: String(total),
            })}
          </span>
          {statusText ? (
            <span className={`table-status${loading ? " is-loading" : ""}`}>{statusText}</span>
          ) : null}
        </div>
        <div className="pagination-actions">
          <button
            className="button ghost button-with-icon"
            disabled={offset === 0 || loading}
            onClick={() => onPageChange(Math.max(0, offset - limit))}
            type="button"
          >
            <FontAwesomeIcon icon={faAngleLeft} />
            <span>{t("table.previous")}</span>
          </button>
          <button
            className="button ghost button-with-icon"
            disabled={offset + limit >= total || loading}
            onClick={() => onPageChange(offset + limit)}
            type="button"
          >
            <span>{t("table.next")}</span>
            <FontAwesomeIcon icon={faAngleRight} />
          </button>
        </div>
      </div>
      <div className="table-wrap">
        <table className="data-table">
          <thead>
            <tr>
              {columns.map((column) => (
                <th key={column.key}>{column.header}</th>
              ))}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td className="table-empty-state" colSpan={columns.length}>
                  {t("common.loading")}
                </td>
              </tr>
            ) : rows.length === 0 ? (
              <tr>
                <td className="table-empty-state" colSpan={columns.length}>
                  {t("common.empty")}
                </td>
              </tr>
            ) : (
              rows.map((row, index) => (
                <tr key={index}>
                  {columns.map((column) => (
                    <td key={column.key}>{column.render(row)}</td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
