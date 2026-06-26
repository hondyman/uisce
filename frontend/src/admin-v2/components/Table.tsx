// Table component
import React from "react";
import "./Table.css";

export function Table({
  columns,
  rows,
  loading = false,
  empty = "No data"
}: {
  columns: string[];
  rows: (React.ReactNode | string | number | boolean)[][];
  loading?: boolean;
  empty?: string;
}) {
  if (loading) {
    return <div className="table-loading">Loading…</div>;
  }

  if (rows.length === 0) {
    return <div className="table-empty">{empty}</div>;
  }

  return (
    <div className="table-container">
      <table className="table">
        <thead>
          <tr>
            {columns.map((col) => (
              <th key={col}>{col}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={i}>
              {row.map((cell, j) => (
                <td key={j} className={typeof cell === "number" ? "numeric" : ""}>
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
