// React default import removed (not used as a value)
import { Card, CardContent, Typography, Grid, Box } from '@mui/material';
import renderCoreCustomChips from '../../../components/common/semanticChips';
import { SemanticModelConfig } from './types';


interface BuilderTabProps {
  config: SemanticModelConfig;
}

export default function BuilderTab({ config }: BuilderTabProps) {
  return (
    <Box sx={{ pt: 2 }}>
      <Grid container spacing={2}>
        <Grid item xs={12} md={6}>
          <Typography variant="h6" gutterBottom>Core Dimensions</Typography>
          {config.core.dimensions.map((dimension) => (
            <Card key={dimension.id} sx={{ mb: 1 }}>
              <CardContent>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div>
                    <Typography variant="subtitle1">{dimension.title || dimension.name}</Typography>
                    <Typography variant="body2" color="text.secondary">{dimension.type} - {dimension.sql}</Typography>
                  </div>
                  {renderCoreCustomChips({ is_core: true })}
                </Box>
              </CardContent>
            </Card>
          ))}
          
          <Typography variant="h6" gutterBottom sx={{ mt: 2 }}>Custom Dimensions</Typography>
          {config.custom.dimensions.map((dimension) => (
            <Card key={dimension.id} sx={{ mb: 1 }}>
              <CardContent>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div>
                    <Typography variant="subtitle1">{dimension.title || dimension.name}</Typography>
                    <Typography variant="body2" color="text.secondary">{dimension.type} - {dimension.sql}</Typography>
                  </div>
                  {renderCoreCustomChips({ is_custom: true })}
                </Box>
              </CardContent>
            </Card>
          ))}
        </Grid>
        
        <Grid item xs={12} md={6}>
          <Typography variant="h6" gutterBottom>Core Measures</Typography>
          {config.core.measures.map((measure) => (
            <Card key={measure.id} sx={{ mb: 1 }}>
              <CardContent>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div>
                    <Typography variant="subtitle1">{measure.title || measure.name}</Typography>
                    <Typography variant="body2" color="text.secondary">{measure.type} of {measure.sql}</Typography>
                  </div>
                  {renderCoreCustomChips({ is_core: true })}
                </Box>
              </CardContent>
            </Card>
          ))}
          
          <Typography variant="h6" gutterBottom sx={{ mt: 2 }}>Custom Measures</Typography>
          {config.custom.measures.map((measure) => (
            <Card key={measure.id} sx={{ mb: 1 }}>
              <CardContent>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <div>
                    <Typography variant="subtitle1">{measure.title || measure.name}</Typography>
                    <Typography variant="body2" color="text.secondary">{measure.type} of {measure.sql}</Typography>
                  </div>
                  {renderCoreCustomChips({ is_custom: true })}
                </Box>
              </CardContent>
            </Card>
          ))}
        </Grid>
      </Grid>
    </Box>
  );
}