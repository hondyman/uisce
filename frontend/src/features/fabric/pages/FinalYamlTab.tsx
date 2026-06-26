// React default import removed (not used as a value)
import { Box, Card, CardContent, Typography, Button } from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import DownloadIcon from '@mui/icons-material/Download';
import { SemanticModelConfig } from './types';
import { copyContent, downloadContent } from './utils';

interface FinalYamlTabProps {
  config: SemanticModelConfig;
  modelName: string;
  generateFinalYAML: (config: SemanticModelConfig, modelName: string) => string;
  toast: (options: { title: string; description: string; variant?: string }) => void;
}

export default function FinalYamlTab({ config, modelName, generateFinalYAML, toast }: FinalYamlTabProps) {
  return (
    <Box sx={{ pt: 2 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">Final YAML (Core + Custom)</Typography>
        <Box>
          <Button variant="outlined" onClick={() => copyContent(generateFinalYAML(config, modelName), 'Final YAML', toast)} startIcon={<ContentCopyIcon />} sx={{ mr: 1 }}>Copy YAML</Button>
          <Button variant="contained" onClick={() => downloadContent(generateFinalYAML(config, modelName), `${modelName || 'semantic_model'}.yml`, 'YAML', toast)} startIcon={<DownloadIcon />}>Download YAML</Button>
        </Box>
      </Box>
      <Card>
        <CardContent>
          <pre className="fabric-pre">
            {generateFinalYAML(config, modelName)}
          </pre>
        </CardContent>
      </Card>
    </Box>
  );
}