import React, { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Paper from "@mui/material/Paper";
import CircularProgress from "@mui/material/CircularProgress";
import { useIncident } from "../hooks/useIncident";
import { useRCA } from "../hooks/useRCA";
import { useIncidentPattern, useSimilarIncidents } from "../hooks/usePatterns";
import { IncidentHeader } from "../components/IncidentHeader";
import { IncidentActions } from "../components/IncidentActions";
import { IncidentRCA } from "../components/IncidentRCA";
import { IncidentActionsPanel } from "../components/IncidentActionsPanel";
import { IntelligentRCAPanel } from "../components/IntelligentRCAPanel";
import { PatternMatchPanel } from "../components/PatternMatchPanel";
import { SmartRCASuggestions } from "../components/SmartRCASuggestions";
import { ExecuteActionModal } from "../components/ExecuteActionModal";
import { Spinner } from "../components/Feedback";
import { Card } from "../components/Card";

const severityColors: Record<string, string> = {
  critical: "#ef4444",
  error: "#f87171",
  warning: "#eab308",
  info: "#3b82f6",
};

function getEventIcon(eventType: string): string {
  const icons: Record<string, string> = {
    alert: "🚨",
    fingerprint: "👆",
    tenant_health: "🏢",
    endpoint_health: "🔗",
    latency_anomaly: "📈",
    incident_opened: "📌",
    incident_closed: "✅",
  };
  return icons[eventType] || "📍";
}

function IncidentEventList({ events }: { events: any[] }) {
  const theme = useTheme();

  if (events.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ textAlign: "center", p: 2 }}>
        No events in this incident
      </Typography>
    );
  }

  return (
    <Box sx={{ display: "flex", flexDirection: "column", gap: 1.5, maxHeight: 600, overflowY: "auto" }}>
      {events.map((event) => (
        <Paper
          key={event.id}
          sx={{
            display: "flex",
            gap: 1.5,
            p: 1.5,
            borderLeft: 3,
            borderLeftColor: severityColors[event.severity] || theme.palette.grey[400],
            borderRadius: 1,
            backgroundColor: "background.paper",
            transition: "all 150ms ease",
            "&:hover": {
              backgroundColor: "action.hover",
              transform: "translateX(4px)",
            },
          }}
        >
          <Box sx={{ fontSize: "1.25rem", flexShrink: 0 }}>{getEventIcon(event.event_type)}</Box>
          <Box sx={{ flex: 1 }}>
            <Typography variant="body2" sx={{ fontWeight: 600, mb: 0.25, wordBreak: "break-word" }}>
              {event.title}
            </Typography>
            <Box sx={{ display: "flex", gap: 1, flexWrap: "wrap" }}>
              <Typography
                variant="caption"
                sx={{
                  color: "text.secondary",
                  px: 0.75,
                  py: 0.25,
                  bgcolor: "action.hover",
                  borderRadius: 0.5,
                }}
              >
                {event.event_type.replace(/_/g, " ")}
              </Typography>
              <Typography
                variant="caption"
                sx={{
                  color: "text.secondary",
                  px: 0.75,
                  py: 0.25,
                  bgcolor: "action.hover",
                  borderRadius: 0.5,
                }}
              >
                {new Date(event.occurred_at).toLocaleTimeString()}
              </Typography>
            </Box>
          </Box>
        </Paper>
      ))}
    </Box>
  );
}

export function IncidentDetailPage() {
  const theme = useTheme();
  const { incidentId } = useParams();
  const navigate = useNavigate();
  const [selectedSmartAction, setSelectedSmartAction] = useState<string | null>(null);
  const incidentQuery = useIncident(incidentId || null);
  const rcaQuery = useRCA(incidentId || "");
  const patternQuery = useIncidentPattern(incidentId || "");
  const similarQuery = useSimilarIncidents(incidentId || "");

  if (!incidentId) {
    return (
      <Box sx={{ p: 3, textAlign: "center" }}>
        <Typography variant="h5" sx={{ mb: 2 }}>Incident Not Found</Typography>
        <Typography variant="body2" color="text.secondary">
          No incident ID provided. Please select an incident from the timeline.
        </Typography>
      </Box>
    );
  }

  if (incidentQuery.isLoading) {
    return (
      <Box
        sx={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          minHeight: "60vh",
          gap: 2,
        }}
      >
        <CircularProgress size="large" />
        <Typography variant="h6" color="text.secondary">Loading incident details...</Typography>
      </Box>
    );
  }

  if (incidentQuery.isError || !incidentQuery.data) {
    return (
      <Box
        sx={{
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          minHeight: "60vh",
          gap: 2,
          textAlign: "center",
        }}
      >
        <Typography variant="h5">Error Loading Incident</Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          Failed to load incident details. Please try again.
        </Typography>
        <Button variant="outlined" onClick={() => navigate("/admin/operations")}>
          ← Back to Timeline
        </Button>
      </Box>
    );
  }

  const { incident, events } = incidentQuery.data;

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: "flex", alignItems: "center", gap: 2, mb: 3 }}>
        <Button variant="outlined" onClick={() => navigate(-1)} size="small">
          ← Back
        </Button>
        <Typography variant="h5" sx={{ flex: 1, fontWeight: 600, m: 0 }}>
          {incident.title}
        </Typography>
      </Box>

      <Box sx={{ display: "grid", gridTemplateColumns: { xs: "1fr", lg: "1fr 300px" }, gap: 3 }}>
        <Box sx={{ display: "flex", flexDirection: "column", gap: 3 }}>
          <IncidentHeader incident={incident} />
          <IncidentActions incident={incident} />
          <IncidentRCA incident={incident} events={events} />

          {rcaQuery.data && (
            <IntelligentRCAPanel rca={rcaQuery.data} isLoading={rcaQuery.isLoading} />
          )}

          {rcaQuery.data && (
            <SmartRCASuggestions
              rca={rcaQuery.data}
              pattern={patternQuery.data}
              similarities={similarQuery.data?.similarities || []}
              onSuggestedActionClick={setSelectedSmartAction}
              isLoading={rcaQuery.isLoading || patternQuery.isLoading}
            />
          )}

          {patternQuery.data && (
            <PatternMatchPanel
              pattern={patternQuery.data}
              similarities={similarQuery.data?.similarities || []}
              isLoading={patternQuery.isLoading}
            />
          )}

          <IncidentActionsPanel incidentId={incident.id} incidentStatus={incident.status} />

          {selectedSmartAction && (
            <ExecuteActionModal
              incidentId={incident.id}
              actionType={selectedSmartAction}
              isOpen={!!selectedSmartAction}
              onClose={() => setSelectedSmartAction(null)}
              onSuccess={() => {
                setSelectedSmartAction(null);
                incidentQuery.refetch();
              }}
            />
          )}
        </Box>

        <Box sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
          <Card title={`Timeline (${events.length} events)`}>
            <IncidentEventList events={events} />
          </Card>
        </Box>
      </Box>
    </Box>
  );
}
