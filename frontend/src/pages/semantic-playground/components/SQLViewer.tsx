// SQL Viewer Component
import React, { useState } from "react";
import {
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Stack,
  Typography,
  Alert,
} from "@mui/material";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import PlayArrowIcon from "@mui/icons-material/PlayArrow";
import GetAppIcon from "@mui/icons-material/GetApp";

interface SQLViewerProps {
  sql: string | null;
  loading: boolean;
  error?: string;
  warnings?: string[];
  onExecute?: () => void;
  onDownloadCSV?: () => void;
  executingSQL?: boolean;
}

export const SQLViewer: React.FC<SQLViewerProps> = ({
  sql,
  loading,
  error,
  warnings,
  onExecute,
  onDownloadCSV,
  executingSQL,
}) => {
  const handleCopySQL = () => {
    if (sql) {
      navigator.clipboard.writeText(sql);
      alert("SQL copied to clipboard!");
    }
  };

  const formatSQL = (sqlText: string): string => {
    // Simple SQL formatting (could use a library like sql-formatter)
    return sqlText
      .replace(/\s+/g, " ")
      .replace(/\bSELECT\b/gi, "\nSELECT")
      .replace(/\bFROM\b/gi, "\nFROM")
      .replace(/\bWHERE\b/gi, "\nWHERE")
      .replace(/\bJOIN\b/gi, "\nJOIN")
      .replace(/\bLEFT JOIN\b/gi, "\nLEFT JOIN")
      .replace(/\bORDER BY\b/gi, "\nORDER BY")
      .replace(/\bLIMIT\b/gi, "\nLIMIT")
      .replace(/\bGROUP BY\b/gi, "\nGROUP BY")
      .trim();
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
          overflow: "hidden",
          p: 2,
        }}
      >
        <Typography variant="h6" sx={{ fontWeight: 600, color: "#fff" }}>
          Generated SQL
        </Typography>

        {/* SQL Display */}
        {sql ? (
          <Box
            sx={{
              flex: 1,
              backgroundColor: "#0d1117",
              border: "1px solid #30363d",
              borderRadius: 1,
              p: 1.5,
              fontFamily: '"Fira Code", monospace',
              fontSize: "12px",
              overflow: "auto",
              color: "#c9d1d9",
              position: "relative",
            }}
          >
            <pre
              style={{
                margin: 0,
                padding: 0,
                whiteSpace: "pre-wrap",
                wordWrap: "break-word",
                lineHeight: "1.6",
              }}
            >
              {formatSQL(sql)}
            </pre>
          </Box>
        ) : loading ? (
          <Box
            sx={{
              flex: 1,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              color: "#666",
            }}
          >
            <Stack alignItems="center" spacing={1}>
              <CircularProgress size={40} sx={{ color: "#2196F3" }} />
              <Typography sx={{ color: "#999" }}>Generating SQL...</Typography>
            </Stack>
          </Box>
        ) : (
          <Box
            sx={{
              flex: 1,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              color: "#666",
              backgroundColor: "#2d2d2d",
              borderRadius: 1,
              border: "2px dashed #444",
            }}
          >
            <Typography sx={{ color: "#999" }}>
              No SQL yet. Generate a query first.
            </Typography>
          </Box>
        )}

        {/* Warnings */}
        {warnings && warnings.length > 0 && (
          <Alert severity="warning" sx={{ backgroundColor: "#3d2d1f" }}>
            {warnings.map((w, i) => (
              <div key={i}>⚠️ {w}</div>
            ))}
          </Alert>
        )}

        {/* Error */}
        {error && <Alert severity="error">{error}</Alert>}

        {/* Action buttons */}
        <Stack direction="row" spacing={1}>
          {sql && (
            <>
              <Button
                size="small"
                variant="contained"
                startIcon={
                  executingSQL ? (
                    <CircularProgress size={16} />
                  ) : (
                    <PlayArrowIcon />
                  )
                }
                onClick={onExecute}
                disabled={executingSQL}
                sx={{
                  backgroundColor: "#4CAF50",
                  "&:hover": { backgroundColor: "#45a049" },
                  textTransform: "none",
                  fontWeight: 600,
                }}
              >
                {executingSQL ? "Executing..." : "Execute"}
              </Button>
              <Button
                size="small"
                variant="outlined"
                startIcon={<ContentCopyIcon />}
                onClick={handleCopySQL}
                sx={{
                  borderColor: "#555",
                  color: "#999",
                  textTransform: "none",
                  "&:hover": {
                    borderColor: "#777",
                  },
                }}
              >
                Copy
              </Button>
              {onDownloadCSV && (
                <Button
                  size="small"
                  variant="outlined"
                  startIcon={<GetAppIcon />}
                  onClick={onDownloadCSV}
                  sx={{
                    borderColor: "#555",
                    color: "#999",
                    textTransform: "none",
                    "&:hover": {
                      borderColor: "#777",
                    },
                  }}
                >
                  Export CSV
                </Button>
              )}
            </>
          )}
        </Stack>
      </CardContent>
    </Card>
  );
};
