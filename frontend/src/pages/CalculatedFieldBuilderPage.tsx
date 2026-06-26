import React from 'react';
import { Box } from '@mui/material';
import CalculatedFieldBuilder from '../features/expressions/components/CalculatedFieldBuilder';
import { toast } from 'sonner';

export const CalculatedFieldBuilderPage: React.FC = () => {
    
  const handleSave = async (term: any) => {
    try {
        // TODO: Wire up to real backend POST
        console.log("Saving term:", term);
        toast.success(`Saved calculated field: ${term.node_name}`);
    } catch (err) {
        toast.error("Failed to save field");
    }
  };

  return (
    <Box sx={{ height: '100%', overflow: 'auto', bgcolor: 'background.default' }}>
      <CalculatedFieldBuilder onSave={handleSave} />
    </Box>
  );
};

export default CalculatedFieldBuilderPage;
