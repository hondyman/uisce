import React from "react";
import { Card, Table, Spinner } from "../";
import { useQuery } from "@tanstack/react-query";
import { api } from "../../api";

export function TeamManager() {
  const { data: teams, isLoading } = useQuery({
    queryKey: ["teams"],
    queryFn: () => api<any[]>("/teams"),
  });

  const columns = ["Name", "Tier", "Slug", "Created", "Actions"];
  const rows = teams?.map((t: any) => [
    t.name,
    t.subscription_tier || "Standard",
    t.slug,
    new Date(t.created_at).toLocaleDateString(),
    <button className="btn btn-sm btn-outline" key={t.id}>Manage</button>
  ]) || [];

  return (
    <div className="team-manager">
      <div className="section-header">
        <h2 className="text-xl font-semibold">Teams & Workspaces</h2>
        <button className="btn btn-primary">+ Create Team</button>
      </div>

      <Card className="mt-4">
        {isLoading ? (
          <Spinner size="md" />
        ) : (
          <Table
            columns={columns}
            rows={rows}
            empty="No teams found"
          />
        )}
      </Card>
    </div>
  );
}
