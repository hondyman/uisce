// NL Input Panel Component
import React from "react";
import {
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  FormControl,
  FormHelperText,
  InputLabel,
  MenuItem,
  Select,
  Stack,
  TextField,
  Typography,
  Chip,
  Alert,
} from "@mui/material";
import SendIcon from "@mui/icons-material/Send";
import ClearIcon from "@mui/icons-material/Clear";
import { SemanticBundle, Datasource, BundleVersion } from "../types";

interface NLInputPanelProps {
  datasources: Datasource[];
  selectedDatasource: string | null;
  selectedVersion: string | null;
  versions: BundleVersion[];
  prompt: string;
  mode: "exploratory" | "strict" | "CRUD";
  loading: boolean;
  error?: string;
  onDatasourceChange: (datasource: string) => void;
  onVersionChange: (version: string) => void;
  onPromptChange: (prompt: string) => void;
  onModeChange: (mode: "exploratory" | "strict" | "CRUD") => void;
  onGenerate: () => void;
  onClear: () => void;
  bundle?: SemanticBundle | null;
}

export const NLInputPanel: React.FC<NLInputPanelProps> = ({
  datasources,
  selectedDatasource,
  selectedVersion,
  versions,
  prompt,
  mode,
  loading,
  error,
  onDatasourceChange,
  onVersionChange,
  onPromptChange,
  onModeChange,
  onGenerate,
  onClear,
  bundle,
}) => {
  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
      onGenerate();
    }
  };

  return (
    <Card
      sx={{
        height: "100%",
        display: "flex",
        flexDirection: "column",
        backgroundColor: "#1e1e1e",
        border: "1px solid #333",
      }}
    >
      <CardContent
        sx={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          gap: 2,
          overflow: "auto",
        }}
      >
        <Typography variant="h6" sx={{ fontWeight: 600, color: "#fff" }}>
          Natural Language Query
        </Typography>

        <Stack spacing={2}>
          {/* Datasource Selector */}
          <FormControl fullWidth size="small">
            <InputLabel sx={{ color: "#999" }}>Datasource</InputLabel>
            <Select
              value={selectedDatasource || ""}
              label="Datasource"
              onChange={(e) => onDatasourceChange(e.target.value)}
              disabled={loading}
              sx={{
                backgroundColor: "#2d2d2d",
                color: "#fff",
                "& .MuiOutlinedInput-notchedOutline": {
                  borderColor: "#444",
                },
                "&:hover .MuiOutlinedInput-notchedOutline": {
                  borderColor: "#666",
                },
              }}
            >
              {datasources.map((ds) => (
                <MenuItem key={ds.id} value={ds.id}>
                  {ds.name}
                </MenuItem>
              ))}
            </Select>
            <FormHelperText sx={{ color: "#666" }}>
              Select the datasource to query
            </FormHelperText>
          </FormControl>

          {/* Version Selector */}
          <FormControl fullWidth size="small">
            <InputLabel sx={{ color: "#999" }}>Version</InputLabel>
            <Select
              value={selectedVersion || ""}
              label="Version"
              onChange={(e) => onVersionChange(e.target.value)}
              disabled={!selectedDatasource || loading}
              sx={{
                backgroundColor: "#2d2d2d",
                color: "#fff",
                "& .MuiOutlinedInput-notchedOutline": {
                  borderColor: "#444",
                },
                "&:hover .MuiOutlinedInput-notchedOutline": {
                  borderColor: "#666",
                },
              }}
            >
              {versions.map((v) => (
                <MenuItem key={v.version} value={v.version}>
                  {v.version}
                </MenuItem>
              ))}
            </Select>
            <FormHelperText sx={{ color: "#666" }}>
              Select bundle version
            </FormHelperText>
          </FormControl>

          {/* Mode Selector */}
          <FormControl fullWidth size="small">
            <InputLabel sx={{ color: "#999" }}>Query Mode</InputLabel>
            <Select
              value={mode}
              label="Query Mode"
              onChange={(e) =>
                onModeChange(
                  e.target.value as "exploratory" | "strict" | "CRUD"
                )
              }
              disabled={loading}
              sx={{
                backgroundColor: "#2d2d2d",
                color: "#fff",
                "& .MuiOutlinedInput-notchedOutline": {
                  borderColor: "#444",
                },
                "&:hover .MuiOutlinedInput-notchedOutline": {
                  borderColor: "#666",
                },
              }}
            >
              <MenuItem value="exploratory">
                Exploratory (with inference)
              </MenuItem>
              <MenuItem value="strict">Strict (explicit only)</MenuItem>
              <MenuItem value="CRUD">CRUD</MenuItem>
            </Select>
            <FormHelperText sx={{ color: "#666" }}>
              {mode === "exploratory" &&
                "LLM can infer related fields (e.g., customer_type from loyalty_points)"}
              {mode === "strict" &&
                "Only include fields explicitly mentioned in the query"}
              {mode === "CRUD" && "For CREATE/UPDATE/DELETE operations"}
            </FormHelperText>
          </FormControl>

          {/* Mode badges */}
          <Box sx={{ display: "flex", gap: 1, flexWrap: "wrap" }}>
            {mode === "exploratory" && (
              <Chip
                label="Inference Enabled"
                size="small"
                variant="outlined"
                sx={{ color: "#4CAF50", borderColor: "#4CAF50" }}
              />
            )}
            {mode === "strict" && (
              <Chip
                label="Strict Mode"
                size="small"
                variant="outlined"
                sx={{ color: "#2196F3", borderColor: "#2196F3" }}
              />
            )}
            {bundle && (
              <Chip
                label={`${bundle.fields.length} fields`}
                size="small"
                variant="outlined"
                sx={{ color: "#666" }}
              />
            )}
          </Box>

          {/* NL Prompt Input */}
          <TextField
            fullWidth
            multiline
            rows={6}
            placeholder="Enter your natural language query here...

Example:
Show me the 20 most recent retail customers in the US with their id, name, email, and loyalty points, ordered by creation date descending."
            value={prompt}
            onChange={(e) => onPromptChange(e.target.value)}
            onKeyDown={handleKeyDown}
            disabled={!selectedDatasource || !selectedVersion || loading}
            sx={{
              "& .MuiOutlinedInput-root": {
                backgroundColor: "#2d2d2d",
                color: "#fff",
                "& fieldset": { borderColor: "#444" },
                "&:hover fieldset": { borderColor: "#666" },
                "&.Mui-focused fieldset": { borderColor: "#888" },
              },
              "& .MuiOutlinedInput-input::placeholder": {
                color: "#555",
                opacity: 1,
              },
            }}
          />

          {/* Helper text */}
          <FormHelperText sx={{ color: "#666", mt: -1 }}>
            💡 Tip: Be specific about fields, filters, and ordering.
            <br />
            Press Ctrl+Enter to generate query.
          </FormHelperText>

          {/* Error message */}
          {error && (
            <Alert severity="error" sx={{ mt: 1 }}>
              {error}
            </Alert>
          )}

          {/* Action buttons */}
          <Stack direction="row" spacing={1} sx={{ mt: 2 }}>
            <Button
              variant="contained"
              startIcon={loading ? <CircularProgress size={20} /> : <SendIcon />}
              onClick={onGenerate}
              disabled={!selectedDatasource || !selectedVersion || !prompt || loading}
              sx={{
                flex: 1,
                backgroundColor: "#1976D2",
                "&:hover": { backgroundColor: "#1565C0" },
                textTransform: "none",
                fontWeight: 600,
              }}
            >
              {loading ? "Generating..." : "Generate Query"}
            </Button>
            <Button
              variant="outlined"
              startIcon={<ClearIcon />}
              onClick={onClear}
              disabled={!prompt || loading}
              sx={{
                borderColor: "#555",
                color: "#999",
                "&:hover": {
                  borderColor: "#777",
                  color: "#ccc",
                  backgroundColor: "#333",
                },
                textTransform: "none",
              }}
            >
              Clear
            </Button>
          </Stack>
        </Stack>
      </CardContent>
    </Card>
  );
};
