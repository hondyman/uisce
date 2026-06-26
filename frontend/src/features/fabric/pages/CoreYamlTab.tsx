// React default import removed (not used as a value)
import { Box, Card, CardContent, Typography, Button } from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import DownloadIcon from '@mui/icons-material/Download';
import { SemanticModelConfig } from './types';
import { copyContent, downloadContent } from './utils';

interface CoreYamlTabProps {
  config: SemanticModelConfig;
  modelName: string;
  generateCoreYAML: (config: SemanticModelConfig, modelName: string) => string;
  toast: (options: { title: string; description: string; variant?: string }) => void;
}

export default function CoreYamlTab({ config, modelName, generateCoreYAML, toast }: CoreYamlTabProps) {
  return (
    <Box sx={{ pt: 2 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6">Core YAML (Base Model)</Typography>
        <Box>
          <Button variant="outlined" onClick={() => copyContent(generateCoreYAML(config, modelName), 'Core YAML', toast)} startIcon={<ContentCopyIcon />} sx={{ mr: 1 }}>Copy YAML</Button>
          <Button variant="contained" onClick={() => downloadContent(generateCoreYAML(config, modelName), `${modelName || 'semantic_model'}_core.yml`, 'YAML', toast)} startIcon={<DownloadIcon />}>Download YAML</Button>
        </Box>
      </Box>
      <Card>
        <CardContent>
          <pre className="fabric-pre">
            {generateCoreYAML(config, modelName)}
          </pre>
        </CardContent>
      </Card>
    </Box>
  );
}