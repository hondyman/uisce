import React, { useState } from "react";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Button from "@mui/material/Button";
import Paper from "@mui/material/Paper";
import Chip from "@mui/material/Chip";
import {
  useIncidentActions,
  OPS_ACTIONS,
  formatActionResult,
} from "../hooks/useOpsActions";
import { ExecuteActionModal } from "./ExecuteActionModal";
import { Spinner } from "./Feedback";

export interface IncidentActionsPanelProps {
  incidentId: string;
  incidentStatus: "open" | "closed";
}

export function IncidentActionsPanel({
  incidentId,
  incidentStatus,
}: IncidentActionsPanelProps) {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const [selectedAction, setSelectedAction] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const { data: actions = [], isLoading } = useIncidentActions(incidentId);

  const handleActionClick = (actionId: string) => {
    setSelectedAction(actionId);
    setIsModalOpen(true);
  };

  const isIncidentOpen = incidentStatus === "open";

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical': return 'error';
      case 'warning': return 'warning';
      case 'info': return 'info';
      default: return 'default';
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'success': return 'success';
      case 'failed': return 'error';
      case 'pending': return 'warning';
      default: return 'default';
    }
  };

  return (
    <Box>
      {isIncidentOpen && (
        <Paper sx={{ p: 2, mb: 2 }}>
          <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 1.5 }}>
            ⚡ Available Actions
          </Typography>

          <Box sx={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))', gap: 1 }}>
            {OPS_ACTIONS.map((action) => (
              <Button
                key={action.id}
                variant="outlined"
                onClick={() => handleActionClick(action.id)}
                title={action.description}
                sx={{
                  display: 'flex',
                  flexDirection: 'column',
                  py: 1.5,
                  borderColor: isDark ? '#4b5563' : '#d1d5db',
                  color: isDark ? '#d1d5db' : '#374151',
                  '&:hover': {
                    borderColor: 'primary.main',
                    bgcolor: 'action.hover',
                  },
                }}
              >
                <Typography sx={{ fontSize: '1.25rem', mb: 0.5 }}>{action.icon}</Typography>
                <Typography variant="caption">{action.name}</Typography>
              </Button>
            ))}
          </Box>
        </Paper>
      )}

      <Paper sx={{ p: 2 }}>
        <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 1.5 }}>
          📋 Action History
        </Typography>

        {isLoading ? (
          <Spinner size="sm" />
        ) : actions.length === 0 ? (
          <Typography variant="body2" color="text.secondary" sx={{ fontStyle: 'italic' }}>
            No actions executed yet
          </Typography>
        ) : (
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
            {actions.map((action) => (
              <Paper
                key={action.id}
                variant="outlined"
                sx={{ p: 1.5, borderLeft: 3, borderLeftColor: getStatusColor(action.status) === 'success' ? 'success.main' : getStatusColor(action.status) === 'error' ? 'error.main' : 'warning.main' }}
              >
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                  <Typography variant="body2" sx={{ fontWeight: 500 }}>
                    {action.action_type}
                  </Typography>
                  <Chip
                    label={action.status.toUpperCase()}
                    size="small"
                    color={getStatusColor(action.status)}
                    variant="outlined"
                  />
                </Box>

                <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                  {formatActionResult(action)}
                </Typography>

                {action.result && (
                  <Box sx={{ fontSize: '0.75rem', color: 'text.secondary', mb: 1 }}>
                    {Object.entries(action.result).map(([key, value]) => (
                      <Box key={key} sx={{ display: 'flex', gap: 0.5 }}>
                        <Typography component="span" sx={{ fontWeight: 500 }}>{key}:</Typography>
                        <Typography component="span">
                          {typeof value === "object" ? JSON.stringify(value) : String(value)}
                        </Typography>
                      </Box>
                    ))}
                  </Box>
                )}

                {action.error_msg && (
                  <Typography variant="body2" sx={{ color: 'error.main', mt: 1 }}>
                    <strong>Error:</strong> {action.error_msg}
                  </Typography>
                )}

                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
                  {new Date(action.executed_at).toLocaleString()}
                </Typography>
              </Paper>
            ))}
          </Box>
        )}
      </Paper>

      {selectedAction && (
        <ExecuteActionModal
          incidentId={incidentId}
          actionType={selectedAction}
          isOpen={isModalOpen}
          onClose={() => {
            setIsModalOpen(false);
            setSelectedAction(null);
          }}
          onSuccess={() => {}}
        />
      )}
    </Box>
  );
}
