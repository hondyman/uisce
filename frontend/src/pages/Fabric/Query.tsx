import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "../../lib/apiClient";
import { Box, Typography, CircularProgress, Alert } from "@mui/material";

export default function SemanticModelsQuery() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['semantic-models'],
    queryFn: () => apiFetch('/api/rest/semantic-models').then(r => r.json()),
  });

  if (isLoading) {
    return <CircularProgress />;
  }

  if (error) {
    return <Alert severity="error">Error fetching semantic models: {(error as Error).message}</Alert>;
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom>
        Semantic Models
      </Typography>
      <pre>{JSON.stringify(data, null, 2)}</pre>
    </Box>
  );
}
