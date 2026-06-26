import React, { useState } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Grid,
  Paper,
  Tab,
  Tabs,
  TextField,
  Typography,
  Alert,
  InputAdornment,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import {
  RULE_TEMPLATES as _RULE_TEMPLATES,
  RuleTemplate,
  ValidationRule,
  getTemplatesByCategory,
  getTemplateCategories,
  searchTemplates,
} from '../../data/ruleTemplates';

interface RuleTemplatesSelectorProps {
  onTemplateSelected: (template: RuleTemplate, rule: Partial<ValidationRule>) => void;
  targetEntity?: string;
}

/**
 * Rule Templates Selector Component
 * 
 * Speeds up rule creation by providing:
 * - Pre-built rule templates
 * - Common validation patterns
 * - Quick template selection and customization
 */
const RuleTemplatesSelector: React.FC<RuleTemplatesSelectorProps> = ({
  onTemplateSelected,
  targetEntity,
}) => {
  const [selectedTab, setSelectedTab] = useState(0);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTemplate, setSelectedTemplate] = useState<RuleTemplate | null>(null);
  const [showPreview, setShowPreview] = useState(false);

  const categories = getTemplateCategories();
  const displayedTemplates =
    searchQuery.length > 0 ? searchTemplates(searchQuery) : getTemplatesByCategory(categories[selectedTab]);

  const handleTemplateSelect = (template: RuleTemplate) => {
    setSelectedTemplate(template);
    setShowPreview(true);
  };

  const handleApplyTemplate = () => {
    if (selectedTemplate) {
      const rule: Partial<ValidationRule> = {
        ...selectedTemplate.baseRule,
        name: selectedTemplate.name,
        description: selectedTemplate.description,
        target_entity: targetEntity || '',
      };
      onTemplateSelected(selectedTemplate, rule);
      setShowPreview(false);
      setSelectedTemplate(null);
    }
  };

  const TemplateCard: React.FC<{ template: RuleTemplate }> = ({ template }) => (
    <Card
      sx={{
        cursor: 'pointer',
        transition: 'all 0.3s ease',
        height: '100%',
        '&:hover': {
          boxShadow: 3,
          transform: 'translateY(-2px)',
        },
      }}
      onClick={() => handleTemplateSelect(template)}
    >
      <CardContent>
        <Box sx={{ display: 'flex', alignItems: 'start', gap: 1, mb: 1 }}>
          <Typography variant="h6" sx={{ fontSize: '1.5em' }}>
            {template.icon}
          </Typography>
          <Box sx={{ flex: 1 }}>
            <Typography variant="h6" sx={{ fontSize: '1rem', mb: 0.5 }}>
              {template.name}
            </Typography>
            <Typography variant="caption" sx={{ color: 'text.secondary' }}>
              {template.category.replace('-', ' ')}
            </Typography>
          </Box>
        </Box>

        <Typography variant="body2" sx={{ mb: 1.5, minHeight: 40 }}>
          {template.description}
        </Typography>

        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
          {template.commonUse.slice(0, 2).map((use) => (
            <Chip key={use} label={use} size="small" variant="outlined" />
          ))}
        </Box>
      </CardContent>
    </Card>
  );

  return (
    <Box>
      <Card sx={{ mb: 3 }}>
        <CardHeader title="Rule Templates" subheader="Start with a pre-built template to speed up rule creation" />
        <Divider />
        <CardContent>
          {/* Search Bar */}
          <TextField
            fullWidth
            placeholder="Search templates by name, category, or use case..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            }}
            sx={{ mb: 2 }}
          />

          {/* Category Tabs */}
          {searchQuery.length === 0 && (
            <Tabs
              value={selectedTab}
              onChange={(_, val) => setSelectedTab(val)}
              sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}
              scrollButtons="auto"
              variant="scrollable"
            >
              {categories.map((cat, idx) => (
                <Tab
                  key={idx}
                  label={cat.replace('-', ' ').toUpperCase()}
                  sx={{ textTransform: 'capitalize' }}
                />
              ))}
            </Tabs>
          )}

          {/* Templates Grid */}
          <Grid container spacing={2}>
            {displayedTemplates.map((template) => (
              <Grid item xs={12} sm={6} md={4} key={template.id}>
                <TemplateCard template={template} />
              </Grid>
            ))}
          </Grid>

          {displayedTemplates.length === 0 && (
            <Alert severity="info">
              No templates found. Try a different search query or category.
            </Alert>
          )}
        </CardContent>
      </Card>

      {/* Template Preview Dialog */}
      <Dialog open={showPreview} onClose={() => setShowPreview(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Template Preview</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          {selectedTemplate && (
            <Box>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
                <Typography variant="h6" sx={{ fontSize: '2em' }}>
                  {selectedTemplate.icon}
                </Typography>
                <Box>
                  <Typography variant="h6">{selectedTemplate.name}</Typography>
                  <Chip
                    label={selectedTemplate.category.replace('-', ' ')}
                    size="small"
                    variant="outlined"
                  />
                </Box>
              </Box>

              <Divider sx={{ my: 2 }} />

              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                Description
              </Typography>
              <Typography variant="body2" sx={{ mb: 2, color: 'text.secondary' }}>
                {selectedTemplate.description}
              </Typography>

              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                Help
              </Typography>
              <Typography variant="body2" sx={{ mb: 2, color: 'text.secondary' }}>
                {selectedTemplate.helpText}
              </Typography>

              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                Common Use Cases
              </Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {selectedTemplate.commonUse.map((use) => (
                  <Chip key={use} label={use} size="small" />
                ))}
              </Box>

              <Divider sx={{ my: 2 }} />

              <Typography variant="subtitle2" sx={{ mb: 1 }}>
                Base Rule Configuration
              </Typography>
              <Paper sx={{ p: 1.5, bgcolor: '#f5f5f5', fontFamily: 'monospace' }}>
                <Typography variant="caption">
                  {`Type: ${selectedTemplate.baseRule.rule_type}`}
                  <br />
                  {`Severity: ${selectedTemplate.baseRule.severity}`}
                  <br />
                  {`Condition: ${selectedTemplate.baseRule.rule_condition}`}
                </Typography>
              </Paper>

              <Alert severity="info" sx={{ mt: 2 }}>
                You can customize any part of this rule after applying the template.
              </Alert>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowPreview(false)}>Cancel</Button>
          <Button
            onClick={handleApplyTemplate}
            variant="contained"
            startIcon={<CheckCircleIcon />}
          >
            Use This Template
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default RuleTemplatesSelector;
