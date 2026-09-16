import React from 'react';
import { Dialog, DialogTitle, DialogContent, Grid, Card, CardActionArea, CardContent, Typography, Box } from '@mui/material';
import { LAYOUT_TEMPLATES, LayoutTemplate } from './layoutTemplates';
import type { PageLayout } from '../../types/pageStudio';

interface TemplatePickerDialogProps {
  open: boolean;
  onClose: () => void;
  onPick: (result: { layout: PageLayout; sectionIds: string[] }) => void;
}

const Thumbnail: React.FC<{ rows: number[][] }> = ({ rows }) => (
  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, height: 56, width: '100%' }}>
    {rows.map((row, i) => (
      <Box key={i} sx={{ display: 'flex', gap: 0.5, flex: 1 }}>
        {row.map((weight, j) => (
          <Box key={j} sx={{ flex: weight, bgcolor: 'action.selected', borderRadius: 0.5 }} />
        ))}
      </Box>
    ))}
  </Box>
);

/**
 * Gallery of standard body-section layouts (single/two/three column,
 * dashboard grid, master-detail, side panel) a page or tab can start from.
 * Only offered at creation time (a new page, or PageEditor.tsx's "Add tab")
 * - swapping an already-populated tab's template isn't supported yet, since
 * doing that safely means reconciling existing widgets into new sections
 * rather than just discarding the old layout tree.
 */
const TemplatePickerDialog: React.FC<TemplatePickerDialogProps> = ({ open, onClose, onPick }) => {
  const handlePick = (template: LayoutTemplate) => {
    onPick(template.build());
    onClose();
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>Choose a layout template</DialogTitle>
      <DialogContent>
        <Grid container spacing={2} sx={{ mt: 0.5 }}>
          {LAYOUT_TEMPLATES.map((template) => (
            <Grid key={template.id} size={{ xs: 12, sm: 6 }}>
              <Card variant="outlined" sx={{ borderRadius: 2, height: '100%' }}>
                <CardActionArea sx={{ p: 2, height: '100%' }} onClick={() => handlePick(template)}>
                  <Thumbnail rows={template.thumbnail} />
                  <CardContent sx={{ p: 0, pt: 1.5, '&:last-child': { pb: 0 } }}>
                    <Typography variant="subtitle2" fontWeight={700}>{template.name}</Typography>
                    <Typography variant="caption" color="text.secondary">{template.description}</Typography>
                  </CardContent>
                </CardActionArea>
              </Card>
            </Grid>
          ))}
        </Grid>
      </DialogContent>
    </Dialog>
  );
};

export default TemplatePickerDialog;
