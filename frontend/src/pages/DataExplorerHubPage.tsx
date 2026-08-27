import React from 'react';
import { useNavigate } from 'react-router-dom';
import { Box } from '@mui/material';
import { QueryLibrary, LibraryQueryItem } from '../features/data-explorer/components/QueryLibrary';

export const DataExplorerHubPage: React.FC = () => {
  const navigate = useNavigate();

  const handleOpenQuery = (query: LibraryQueryItem) => {
    navigate('/data-explorer/builder', { state: { query } });
  };

  const handleCreateNew = () => {
    navigate('/data-explorer/builder');
  };

  return (
    <Box sx={{ height: 'calc(100vh - 64px)', overflow: 'hidden' }}>
      <QueryLibrary onOpenQuery={handleOpenQuery} onCreateNew={handleCreateNew} />
    </Box>
  );
};

export default DataExplorerHubPage;
