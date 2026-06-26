// Semantic Query Editor Component
import React, { useEffect, useState } from "react";
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  Dialog,
  DialogContent,
  DialogTitle,
  Stack,
  Tooltip,
  Typography,
} from "@mui/material";
import ContentCopyIcon from "@mui/icons-material/ContentCopy";
import FormatAlignLeftIcon from "@mui/icons-material/FormatAlignLeft";
import InfoIcon from "@mui/icons-material/Info";
import MapIcon from "@mui/icons-material/Map";
import EditIcon from "@mui/icons-material/Edit";
import { SemanticQuery, SemanticBundle } from "../types";

interface SemanticQueryEditorProps {
  query: SemanticQuery | null;
  bundle: SemanticBundle | null;
  loading: boolean;
  error?: string;
  warnings?: string[];
  onQueryChange: (query: SemanticQuery) => void;
  onExplain?: () => Promise<string>;
  onShowLineage?: (fieldId: string) => void;
}

export const SemanticQueryEditor: React.FC<SemanticQueryEditorProps> = ({
  query,
  bundle,
  loading,
  error,
  warnings,
  onQueryChange,
  onExplain,
  onShowLineage,
}) => {
  const [jsonText, setJsonText] = useState("");
  const [editMode, setEditMode] = useState(false);
  const [explanation, setExplanation] = useState<string | null>(null);
  const [showExplanation, setShowExplanation] = useState(false);
  const [isExplaining, setIsExplaining] = useState(false);

  useEffect(() => {
    if (query) {
      setJsonText(JSON.stringify(query, null, 2));
    }
  }, [query]);

  const handleFormatJson = () => {
    try {
      const parsed = JSON.parse(jsonText);
      setJsonText(JSON.stringify(parsed, null, 2));
    } catch (err) {
      alert("Invalid JSON");
    }
  };

  const handleApplyChanges = () => {
    try {
      const parsed = JSON.parse(jsonText);
      onQueryChange(parsed);
      setEditMode(false);
    } catch (err) {
      alert("Invalid JSON: " + (err as Error).message);
    }
  };

  const handleCopyJson = () => {
    navigator.clipboard.writeText(jsonText);
    alert("Copied to clipboard!");
  };

  const handleExplain = async () => {
    if (onExplain) {
      setIsExplaining(true);
      try {
        const exp = await onExplain();
        setExplanation(exp);
        setShowExplanation(true);
      } catch (err) {
        alert("Failed to explain query: " + (err as Error).message);
      } finally {
        setIsExplaining(false);
      }
    }
  };

  return (
    <>
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
          <Stack direction="row" spacing={1} alignItems="center">
            <Typography variant="h6" sx={{ fontWeight: 600, color: "#fff" }}>
              Semantic Query
            </Typography>
            {query && (
              <Chip
                label={`${query.select?.length || 0} fields`}
                size="small"
                sx={{ backgroundColor: "#2d2d2d", color: "#aaa" }}
              />
            )}
            {warnings && warnings.length > 0 && (
              <Tooltip title={warnings.join("\n")}>
                <InfoIcon sx={{ color: "#FF9800", cursor: "help" }} />
              </Tooltip>
            )}
          </Stack>

          {/* JSON Editor */}
          <Box
            sx={{
              flex: 1,
              backgroundColor: "#2d2d2d",
              border: "1px solid #444",
              borderRadius: 1,
              p: 1.5,
              fontFamily: '"Fira Code", monospace',
              fontSize: "12px",
              overflow: "auto",
              color: "#e8e8e8",
              position: "relative",
            }}
          >
            {editMode ? (
              <Box
                component="textarea"
                aria-label="Semantic query JSON editor"
                placeholder="Enter semantic query JSON"
                value={jsonText}
                onChange={(e) => setJsonText(e.target.value)}
                sx={{
                  width: "100%",
                  height: "100%",
                  backgroundColor: "#1e1e1e",
                  color: "#e8e8e8",
                  border: "none",
                  fontFamily: '"Fira Code", monospace',
                  fontSize: "12px",
                  p: 1,
                  resize: "none",
                }}
              />
            ) : (
              <Box
                component="pre"
                aria-label="Semantic query JSON preview"
                sx={{
                  m: 0,
                  p: 0,
                  overflow: "auto",
                  whiteSpace: "pre-wrap",
                  wordWrap: "break-word",
                }}
              >
                {jsonText || (loading ? "Loading..." : "No query yet")}
              </Box>
            )}
          </Box>

          {/* Action buttons */}
          <Stack direction="row" spacing={1} sx={{ flexWrap: "wrap" }}>
            {editMode ? (
              <>
                <Button
                  size="small"
                  variant="contained"
                  onClick={handleApplyChanges}
                  sx={{
                    backgroundColor: "#4CAF50",
                    "&:hover": { backgroundColor: "#45a049" },
                    textTransform: "none",
                  }}
                >
                  Apply
                </Button>
                <Button
                  size="small"
                  variant="outlined"
                  onClick={() => setEditMode(false)}
                  sx={{
                    borderColor: "#555",
                    color: "#999",
                    textTransform: "none",
                  }}
                >
                  Cancel
                </Button>
              </>
            ) : (
              <>
                <Button
                  size="small"
                  variant="text"
                  startIcon={<EditIcon />}
                  onClick={() => setEditMode(true)}
                  disabled={loading}
                  sx={{ color: "#2196F3", textTransform: "none" }}
                >
                  Edit JSON
                </Button>
                <Button
                  size="small"
                  variant="text"
                  startIcon={<FormatAlignLeftIcon />}
                  onClick={handleFormatJson}
                  disabled={loading || !jsonText}
                  sx={{ color: "#666", textTransform: "none" }}
                >
                  Format
                </Button>
                <Button
                  size="small"
                  variant="text"
                  startIcon={<ContentCopyIcon />}
                  onClick={handleCopyJson}
                  disabled={loading || !jsonText}
                  sx={{ color: "#666", textTransform: "none" }}
                >
                  Copy
                </Button>
                {onExplain && (
                  <Button
                    size="small"
                    variant="text"
                    startIcon={
                      isExplaining ? <CircularProgress size={16} /> : <InfoIcon />
                    }
                    onClick={handleExplain}
                    disabled={loading || isExplaining || !query}
                    sx={{ color: "#FF9800", textTransform: "none" }}
                  >
                    Explain
                  </Button>
                )}
                {onShowLineage && query?.select && query.select.length > 0 && (
                  <Button
                    size="small"
                    variant="text"
                    startIcon={<MapIcon />}
                    onClick={() =>
                      query.select && query.select.length > 0 &&
                      onShowLineage(query.select[0])
                    }
                    disabled={loading}
                    sx={{ color: "#9C27B0", textTransform: "none" }}
                  >
                    Lineage
                  </Button>
                )}
              </>
            )}
          </Stack>

          {/* Validation feedback */}
          {error && (
            <Box
              sx={{
                backgroundColor: "#3d1f1f",
                border: "1px solid #cc3333",
                borderRadius: 1,
                p: 1,
                color: "#ff6666",
                fontSize: "12px",
              }}
            >
              ⚠️ {error}
            </Box>
          )}
        </CardContent>
      </Card>

      {/* Explanation Dialog */}
      <Dialog
        open={showExplanation}
        onClose={() => setShowExplanation(false)}
        maxWidth="md"
        fullWidth
        PaperProps={{
          sx: { backgroundColor: "#1e1e1e", color: "#fff" },
        }}
      >
        <DialogTitle sx={{ borderBottom: "1px solid #333" }}>
          Query Explanation
        </DialogTitle>
        <DialogContent sx={{ mt: 2 }}>
          <Box
            sx={{
              backgroundColor: "#2d2d2d",
              p: 2,
              borderRadius: 1,
              fontFamily: 'monospace',
              lineHeight: 1.8,
              color: "#ddd",
              whiteSpace: "pre-wrap",
              wordWrap: "break-word",
            }}
          >
            {explanation}
          </Box>
        </DialogContent>
      </Dialog>
    </>
  );
};
