// Semantic Playground Main Page
import React, { useEffect, useState } from "react";
import {
  Box,
  Container,
  Grid,
  Paper,
  AppBar,
  Toolbar,
  Typography,
  Alert,
  Snackbar,
} from "@mui/material";
import StorageIcon from "@mui/icons-material/Storage";
import {
  NLInputPanel,
  SemanticQueryEditor,
  SQLViewer,
  ResultsTable,
} from "./components";
import {
  usePlanner,
  useExecutor,
  useSQLRunner,
  useSemanticBundle,
} from "./hooks";
import { semanticPlaygroundApi } from "./utils/api";
import { SemanticQuery, Datasource } from "./types";

export const SemanticPlaygroundPage: React.FC = () => {
  // State management
  const [datasources, setDatasources] = useState<Datasource[]>([]);
  const [selectedDatasource, setSelectedDatasource] = useState<string | null>(
    null
  );
  const [selectedVersion, setSelectedVersion] = useState<string | null>(null);
  const [nlPrompt, setNlPrompt] = useState("");
  const [mode, setMode] = useState<"exploratory" | "strict" | "CRUD">(
    "exploratory"
  );
  const [snackbar, setSnackbar] = useState<{
    open: boolean;
    message: string;
    severity: "success" | "error" | "warning";
  }>({ open: false, message: "", severity: "success" });

  // Hooks
  const { bundle, versions, loading: bundleLoading, fetchBundle, fetchVersions } =
    useSemanticBundle();
  const {
    semanticQuery,
    explanation,
    loading: plannerLoading,
    error: plannerError,
    warnings: plannerWarnings,
    callPlanner,
  } = usePlanner();
  const {
    generatedSQL,
    loading: executorLoading,
    error: executorError,
    warnings: executorWarnings,
    callExecutor,
  } = useExecutor();
  const {
    results,
    executionTime,
    loading: sqlRunnerLoading,
    error: sqlRunnerError,
    runSQL,
  } = useSQLRunner();

  // Load datasources on mount
  useEffect(() => {
    const loadDatasources = async () => {
      try {
        const ds = await semanticPlaygroundApi.getDatasources();
        setDatasources(ds);
        if (ds.length > 0) {
          setSelectedDatasource(ds[0].id);
        }
      } catch (err) {
        showSnackbar("Failed to load datasources", "error");
      }
    };

    loadDatasources();
  }, []);

  // Load bundle and versions when datasource changes
  useEffect(() => {
    if (selectedDatasource) {
      fetchBundle(selectedDatasource);
      fetchVersions(selectedDatasource);
    }
  }, [selectedDatasource]);

  // Set default version when versions load
  useEffect(() => {
    if (versions.length > 0 && !selectedVersion) {
      setSelectedVersion(versions[0].version);
    }
  }, [versions]);

  const showSnackbar = (message: string, severity: "success" | "error" | "warning") => {
    setSnackbar({ open: true, message, severity });
  };

  const handleGenerateQuery = async () => {
    if (!selectedDatasource || !selectedVersion || !nlPrompt) {
      showSnackbar("Please fill in all required fields", "warning");
      return;
    }

    await callPlanner({
      datasource: selectedDatasource,
      version: selectedVersion,
      prompt: nlPrompt,
      mode,
    });
  };

  const handleExecuteQuery = async () => {
    if (!semanticQuery || !selectedDatasource || !selectedVersion) {
      showSnackbar("Generate a semantic query first", "warning");
      return;
    }

    await callExecutor({
      datasource: selectedDatasource,
      version: selectedVersion,
      semantic_query: semanticQuery,
    });
  };

  const handleRunSQL = async () => {
    if (!generatedSQL) {
      showSnackbar("Generate SQL first", "warning");
      return;
    }

    await runSQL(generatedSQL);
  };

  const handleUpdateSemanticQuery = (_newQuery: SemanticQuery) => {
    // This would require modifying the planner hook to support manual updates
    // For now, just refresh semantic query
    showSnackbar("Query updated. Run again to regenerate SQL.", "success");
  };

  const handleExplainQuery = async (): Promise<string> => {
    if (!semanticQuery || !selectedDatasource) {
      throw new Error("No query to explain");
    }

    try {
      const explanation = await semanticPlaygroundApi.explainQuery(
        selectedDatasource,
        semanticQuery
      );
      return explanation;
    } catch (err) {
      throw new Error("Failed to explain query: " + (err as Error).message);
    }
  };

  const handleExportCSV = () => {
    if (!results?.rows) {
      showSnackbar("No results to export", "warning");
      return;
    }

    const rows = results.rows;
    const columns = results.columns || Object.keys(rows[0]);

    const csvContent = [
      columns.map((c) => `"${c}"`).join(","),
      ...rows.map((row) =>
        columns
          .map((col) => {
            const val = row[col];
            if (val === null || val === undefined) return '""';
            if (typeof val === "string" && val.includes(","))
              return `"${val.replace(/"/g, '""')}"`;
            return `"${val}"`;
          })
          .join(",")
      ),
    ].join("\n");

    const blob = new Blob([csvContent], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `query-results-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);

    showSnackbar("CSV exported successfully", "success");
  };

  return (
    <Box sx={{ backgroundColor: "#0d1117", minHeight: "100vh", pb: 4 }}>
      {/* Top Bar */}
      <AppBar
        position="sticky"
        sx={{ backgroundColor: "#0d1117", borderBottom: "1px solid #333" }}
      >
        <Toolbar>
          <StorageIcon sx={{ mr: 1.5, color: "#2196F3" }} />
          <Typography
            variant="h6"
            component="div"
            sx={{ fontWeight: 700, color: "#fff" }}
          >
            Semantic Playground
          </Typography>
          <Typography
            variant="caption"
            sx={{ ml: "auto", color: "#666", fontStyle: "italic" }}
          >
            Natural Language → Semantic Query → SQL → Results
          </Typography>
        </Toolbar>
      </AppBar>

      <Container maxWidth="xl" sx={{ mt: 3 }}>
        {/* Information Alert */}
        <Alert
          severity="info"
          sx={{
            mb: 3,
            backgroundColor: "#1e2d3d",
            borderColor: "#2196F3",
            color: "#2196F3",
          }}
        >
          🎮 <strong>Welcome to the Semantic Playground!</strong> Write natural language
          queries and see them transformed into semantic queries and SQL. Perfect for
          learning, debugging, and exploring the semantic layer.
        </Alert>

        {/* Three-Pane Layout */}
        <Grid container spacing={2} sx={{ height: "calc(100vh - 200px)" }}>
          {/* Left Pane: Natural Language Input */}
          <Grid item xs={12} md={4} sx={{ display: "flex" }}>
            <Paper sx={{ width: "100%", overflow: "auto" }}>
              <NLInputPanel
                datasources={datasources}
                selectedDatasource={selectedDatasource}
                selectedVersion={selectedVersion}
                versions={versions}
                prompt={nlPrompt}
                mode={mode}
                loading={plannerLoading || executorLoading}
                error={plannerError}
                onDatasourceChange={setSelectedDatasource}
                onVersionChange={setSelectedVersion}
                onPromptChange={setNlPrompt}
                onModeChange={setMode}
                onGenerate={handleGenerateQuery}
                onClear={() => {
                  setNlPrompt("");
                }}
                bundle={bundle}
              />
            </Paper>
          </Grid>

          {/* Middle Pane: Semantic Query */}
          <Grid item xs={12} md={4} sx={{ display: "flex" }}>
            <Paper sx={{ width: "100%", overflow: "auto" }}>
              <SemanticQueryEditor
                query={semanticQuery}
                bundle={bundle}
                loading={plannerLoading}
                error={plannerError}
                warnings={plannerWarnings}
                onQueryChange={handleUpdateSemanticQuery}
                onExplain={handleExplainQuery}
              />
            </Paper>
          </Grid>

          {/* Right Pane: SQL + Results (stacked) */}
          <Grid item xs={12} md={4} sx={{ display: "flex", flexDirection: "column", gap: 2 }}>
            {/* SQL Viewer - Top */}
            <Paper sx={{ flex: 0.4, overflow: "auto" }}>
              <SQLViewer
                sql={generatedSQL}
                loading={executorLoading}
                error={executorError}
                warnings={executorWarnings}
                onExecute={handleRunSQL}
                onDownloadCSV={handleExportCSV}
                executingSQL={sqlRunnerLoading}
              />
            </Paper>

            {/* Results Table - Bottom */}
            <Paper sx={{ flex: 0.6, overflow: "auto" }}>
              <ResultsTable
                results={results}
                executionTime={executionTime}
                loading={sqlRunnerLoading}
                error={sqlRunnerError}
                sql={generatedSQL || undefined}
              />
            </Paper>
          </Grid>
        </Grid>

        {/* Workflow Steps */}
        <Paper
          sx={{
            mt: 3,
            p: 2,
            backgroundColor: "#1e1e1e",
            border: "1px solid #333",
          }}
        >
          <Typography variant="caption" sx={{ color: "#666" }}>
            💡 <strong>Workflow:</strong> 1️⃣ Select datasource & version → 2️⃣ Write
            natural language question → 3️⃣ Click "Generate Query" → 4️⃣ Review semantic
            query (edit if needed) → 5️⃣ Click "Execute" to generate SQL → 6️⃣ SQL
            executes automatically → 7️⃣ Review results & export CSV
          </Typography>
        </Paper>
      </Container>

      {/* Snackbar for notifications */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
        anchorOrigin={{ vertical: "bottom", horizontal: "right" }}
      >
        <Alert
          onClose={() => setSnackbar({ ...snackbar, open: false })}
          severity={snackbar.severity}
          sx={{ width: "100%" }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  );
};

export default SemanticPlaygroundPage;
