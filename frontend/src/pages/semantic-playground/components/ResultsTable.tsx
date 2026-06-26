// Results Table Component
import React, { useState, useMemo } from "react";
import {
  Box,
  Button,
  Card,
  CardContent,
  CircularProgress,
  Chip,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
  TableSortLabel,
  Typography,
  TextField,
  InputAdornment,
  Alert,
} from "@mui/material";
import SearchIcon from "@mui/icons-material/Search";
import GetAppIcon from "@mui/icons-material/GetApp";
import { QueryExecutionResponse } from "../types";

interface ResultsTableProps {
  results: QueryExecutionResponse | null;
  executionTime: number | null;
  loading: boolean;
  error?: string;
  sql?: string;
}

type SortOrder = "asc" | "desc";

export const ResultsTable: React.FC<ResultsTableProps> = ({
  results,
  executionTime,
  loading,
  error,
  sql,
}) => {
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(20);
  const [sortColumn, setSortColumn] = useState<string | null>(null);
  const [sortOrder, setSortOrder] = useState<SortOrder>("asc");
  const [searchText, setSearchText] = useState("");

  const paginatedRows = useMemo(() => {
    if (!results?.rows) return [];

    let filtered = results.rows;

    // Apply search filter
    if (searchText) {
      filtered = filtered.filter((row) =>
        Object.values(row).some((val) =>
          String(val).toLowerCase().includes(searchText.toLowerCase())
        )
      );
    }

    // Apply sorting
    if (sortColumn) {
      filtered = [...filtered].sort((a, b) => {
        const aVal = a[sortColumn];
        const bVal = b[sortColumn];

        if (aVal === null || aVal === undefined) return 1;
        if (bVal === null || bVal === undefined) return -1;

        let comparison = 0;
        if (typeof aVal === "number" && typeof bVal === "number") {
          comparison = aVal - bVal;
        } else {
          comparison = String(aVal).localeCompare(String(bVal));
        }

        return sortOrder === "asc" ? comparison : -comparison;
      });
    }

    // Apply pagination
    return filtered.slice(page * rowsPerPage, (page + 1) * rowsPerPage);
  }, [results, page, rowsPerPage, sortColumn, sortOrder, searchText]);

  const handleSort = (column: string) => {
    if (sortColumn === column) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortColumn(column);
      setSortOrder("asc");
    }
  };

  const handleChangePage = (_event: unknown, newPage: number) => {
    setPage(newPage);
  };

  const handleChangeRowsPerPage = (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    setRowsPerPage(parseInt(event.target.value, 10));
    setPage(0);
  };

  const handleExportCSV = () => {
    if (!results?.rows || results.rows.length === 0) return;

    const rows = results.rows;
    const columns = results.columns || Object.keys(rows[0]);

    // Create CSV header
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

    // Download
    const blob = new Blob([csvContent], { type: "text/csv" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `query-results-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (loading) {
    return (
      <Card
        sx={{
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: "#1e1e1e",
          border: "1px solid #333",
        }}
      >
        <Stack alignItems="center" spacing={2}>
          <CircularProgress sx={{ color: "#2196F3" }} />
          <Typography sx={{ color: "#999" }}>Executing query...</Typography>
        </Stack>
      </Card>
    );
  }

  if (error) {
    return (
      <Card
        sx={{
          height: "100%",
          backgroundColor: "#1e1e1e",
          border: "1px solid #333",
        }}
      >
        <CardContent>
          <Alert severity="error">{error}</Alert>
        </CardContent>
      </Card>
    );
  }

  if (!results || results.rows.length === 0) {
    return (
      <Card
        sx={{
          height: "100%",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          backgroundColor: "#1e1e1e",
          border: "1px solid #333",
        }}
      >
        <Typography sx={{ color: "#666" }}>
          No results yet. Execute a query to see data.
        </Typography>
      </Card>
    );
  }

  const columns = results.columns || Object.keys(results.rows[0]);
  const totalFiltered = searchText
    ? results.rows.filter((row) =>
        Object.values(row).some((val) =>
          String(val).toLowerCase().includes(searchText.toLowerCase())
        )
      ).length
    : results.row_count;

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
        <Stack direction="row" spacing={2} alignItems="center">
          <Typography variant="h6" sx={{ fontWeight: 600, color: "#fff" }}>
            Results
          </Typography>
          <Chip
            label={`${totalFiltered} rows`}
            size="small"
            sx={{ backgroundColor: "#2d2d2d", color: "#aaa" }}
          />
          {executionTime !== null && (
            <Chip
              label={`${executionTime}ms`}
              size="small"
              sx={{ backgroundColor: "#2d2d2d", color: "#4CAF50" }}
            />
          )}
        </Stack>

        {/* Search */}
        <TextField
          fullWidth
          size="small"
          placeholder="Search results..."
          value={searchText}
          onChange={(e) => {
            setSearchText(e.target.value);
            setPage(0);
          }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ color: "#666" }} />
              </InputAdornment>
            ),
          }}
          sx={{
            "& .MuiOutlinedInput-root": {
              backgroundColor: "#2d2d2d",
              color: "#fff",
              "& fieldset": { borderColor: "#444" },
              "&:hover fieldset": { borderColor: "#666" },
            },
          }}
        />

        {/* Table */}
        <TableContainer
          sx={{
            flex: 1,
            backgroundColor: "#2d2d2d",
            borderRadius: 1,
            overflow: "auto",
          }}
        >
          <Table stickyHeader size="small">
            <TableHead>
              <TableRow sx={{ backgroundColor: "#1e1e1e" }}>
                {columns.map((col) => (
                  <TableCell
                    key={col}
                    sortDirection={
                      sortColumn === col ? sortOrder : false
                    }
                    sx={{
                      backgroundColor: "#1e1e1e",
                      borderBottom: "1px solid #444",
                      color: "#999",
                      fontWeight: 600,
                      fontSize: "12px",
                    }}
                  >
                    <TableSortLabel
                      active={sortColumn === col}
                      direction={sortOrder}
                      onClick={() => handleSort(col)}
                      sx={{
                        color: sortColumn === col ? "#2196F3" : "#999",
                        "&:hover": { color: "#2196F3" },
                      }}
                    >
                      {col}
                    </TableSortLabel>
                  </TableCell>
                ))}
              </TableRow>
            </TableHead>
            <TableBody>
              {paginatedRows.map((row, idx) => (
                <TableRow
                  key={idx}
                  sx={{
                    backgroundColor: idx % 2 === 0 ? "#2d2d2d" : "#333",
                    "&:hover": { backgroundColor: "#3d3d3d" },
                    borderBottom: "1px solid #444",
                  }}
                >
                  {columns.map((col) => (
                    <TableCell
                      key={col}
                      sx={{
                        color: "#ddd",
                        fontSize: "12px",
                        padding: "8px",
                        fontFamily: 'monospace',
                      }}
                    >
                      {row[col] === null || row[col] === undefined
                        ? "NULL"
                        : typeof row[col] === "boolean"
                        ? String(row[col])
                        : typeof row[col] === "object"
                        ? JSON.stringify(row[col])
                        : String(row[col])}
                    </TableCell>
                  ))}
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>

        {/* Pagination */}
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <TablePagination
            rowsPerPageOptions={[10, 20, 50, 100]}
            component="div"
            count={totalFiltered}
            rowsPerPage={rowsPerPage}
            page={page}
            onPageChange={handleChangePage}
            onRowsPerPageChange={handleChangeRowsPerPage}
            sx={{
              color: "#999",
              "& .MuiTablePagination-selectLabel": { margin: 0 },
              "& .MuiTablePagination-displayedRows": { margin: 0 },
              "& .MuiInputBase-root": { color: "#999" },
            }}
          />
          <Button
            size="small"
            variant="outlined"
            startIcon={<GetAppIcon />}
            onClick={handleExportCSV}
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
        </Stack>
      </CardContent>
    </Card>
  );
};
