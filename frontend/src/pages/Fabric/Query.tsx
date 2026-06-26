import { useQuery } from "@apollo/client";
import { GET_SEMANTIC_MODELS } from "../../graphql/queries/getSemanticModels";
import { Box, Typography, CircularProgress, Alert } from "@mui/material";

export default function SemanticModelsQuery() {
  const { loading, error, data } = useQuery(GET_SEMANTIC_MODELS);

  if (loading) {
    return <CircularProgress />;
  }

  if (error) {
    return <Alert severity="error">Error fetching semantic models: {error.message}</Alert>;
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom>
        Semantic Models
      </Typography>
      <pre>{JSON.stringify(data?.semantic_models, null, 2)}</pre>
    </Box>
  );
}
