import React, { useState } from "react";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Tabs from "@mui/material/Tabs";
import Tab from "@mui/material/Tab";
import Paper from "@mui/material/Paper";
import { SourcePreferenceEditor } from "../components/SourcePreference/SourcePreferenceEditor";
import { SourceAnalyticsDashboard } from "../components/SourcePreference/SourceAnalyticsDashboard";
import { ExceptionManager } from "../components/SourcePreference/ExceptionManager";

type Tab = "preferences" | "analytics" | "exceptions";

export function SourcePreferencePage() {
  const theme = useTheme();
  const [tab, setTab] = useState<Tab>("preferences");

  const tabIndex: Record<Tab, number> = {
    preferences: 0,
    analytics: 1,
    exceptions: 2,
  };

  const handleTabChange = (_: React.SyntheticEvent, newValue: number) => {
    const tabs: Tab[] = ["preferences", "analytics", "exceptions"];
    setTab(tabs[newValue]);
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ mb: 3 }}>
        <Typography variant="h4" component="h1" sx={{ fontWeight: 600, mb: 1 }}>
          Source Preference Management
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Manage preferred data sources, view analytics, and handle exceptions across your semantic layer.
        </Typography>
      </Box>

      <Paper sx={{ mb: 3 }}>
        <Tabs
          value={tabIndex[tab]}
          onChange={handleTabChange}
          sx={{
            borderBottom: 1,
            borderColor: "divider",
            "& .MuiTab-root": {
              textTransform: "none",
              fontWeight: 500,
            },
          }}
        >
          <Tab label="📋 Preferences" />
          <Tab label="📊 Analytics" />
          <Tab label="⚡ Exceptions" />
        </Tabs>
      </Paper>

      <Box>
        {tab === "preferences" && <SourcePreferenceEditor />}
        {tab === "analytics" && <SourceAnalyticsDashboard />}
        {tab === "exceptions" && <ExceptionManager />}
      </Box>
    </Box>
  );
}
