import { useState, type FC } from 'react';
import { Button, Paper } from '@mui/material';
import AbbreviationManager from '../../../components/AbbreviationManager';
import { AbbreviationWizard } from '../../../components/AbbreviationWizard';

const AbbreviationManagementPage: FC = () => {
  const [showWizard, setShowWizard] = useState(false);

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8 flex justify-between items-start">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Abbreviation Management</h1>
          <p className="text-gray-600">
            Manage database-backed abbreviations for enhanced semantic matching and column name expansion.
          </p>
        </div>
        <Button 
          variant="contained" 
          color={showWizard ? "secondary" : "primary"}
          onClick={() => setShowWizard(!showWizard)}
        >
          {showWizard ? "Back to Manager" : "Open Abbreviation Wizard"}
        </Button>
      </div>
      
      {showWizard ? (
        <Paper className="p-4">
          <AbbreviationWizard onCompletion={() => setShowWizard(false)} />
        </Paper>
      ) : (
        <AbbreviationManager />
      )}
    </div>
  );
};

export default AbbreviationManagementPage;