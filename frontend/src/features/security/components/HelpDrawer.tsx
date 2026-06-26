import React, { useState } from 'react';
import {
  Drawer,
  Box,
  Typography,
  IconButton,
  Divider,
  List,
  ListItem,
  ListItemText,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Paper,
  Chip,
} from '@mui/material';
import {
  Close as CloseIcon,
  ExpandMore as ExpandMoreIcon,
  HelpOutline as HelpIcon,
} from '@mui/icons-material';

interface HelpDrawerProps {
  open: boolean;
  onClose: () => void;
}

export const HelpDrawer: React.FC<HelpDrawerProps> = ({ open, onClose }) => {
  return (
    <Drawer anchor="right" open={open} onClose={onClose}>
      <Box sx={{ width: 400, p: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h6" sx={{ fontWeight: 700 }}>
            Access Rules Help
          </Typography>
          <IconButton onClick={onClose}>
            <CloseIcon />
          </IconButton>
        </Box>

        <Divider sx={{ mb: 3 }} />

        {/* Quick Start */}
        <Accordion defaultExpanded>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              Quick Start Guide
            </Typography>
          </AccordionSummary>
          <AccordionDetails>
            <List dense>
              <ListItem>
                <ListItemText
                  primary="1. Click 'Create New Rule'"
                  secondary="Start the guided wizard"
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary="2. Select a team/group"
                  secondary="Choose who this rule applies to"
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary="3. Choose data type"
                  secondary="Select what data they can access"
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary="4. Set access level"
                  secondary="Read, Write, or None"
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary="5. Add filters (optional)"
                  secondary="Restrict rows and mask fields"
                />
              </ListItem>
            </List>
          </AccordionDetails>
        </Accordion>

        {/* Row Filters */}
        <Accordion>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              Row Filter Examples
            </Typography>
          </AccordionSummary>
          <AccordionDetails>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Paper elevation={0} sx={{ p: 2, bgcolor: 'grey.50' }}>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                  Single Condition
                </Typography>
                <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                  region = 'EMEA'
                </Typography>
              </Paper>
              <Paper elevation={0} sx={{ p: 2, bgcolor: 'grey.50' }}>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                  Multiple Conditions (AND)
                </Typography>
                <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                  region = 'EMEA' AND status = 'active'
                </Typography>
              </Paper>
              <Paper elevation={0} sx={{ p: 2, bgcolor: 'grey.50' }}>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1 }}>
                  Numeric Comparison
                </Typography>
                <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                  amount {'>'} 1000000
                </Typography>
              </Paper>
            </Box>
          </AccordionDetails>
        </Accordion>

        {/* Field Masks */}
        <Accordion>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              Field Masking Types
            </Typography>
          </AccordionSummary>
          <AccordionDetails>
            <List dense>
              <ListItem>
                <ListItemText
                  primary={<Chip label="HIDE" size="small" color="error" />}
                  secondary="Completely removes the field from results"
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary={<Chip label="MASK" size="small" color="warning" />}
                  secondary="Shows partial data (e.g., ***-**-1234)"
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary={<Chip label="NONE" size="small" color="success" />}
                  secondary="Shows full data (no masking)"
                />
              </ListItem>
            </List>
          </AccordionDetails>
        </Accordion>

        {/* Best Practices */}
        <Accordion>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
              Best Practices
            </Typography>
          </AccordionSummary>
          <AccordionDetails>
            <List dense>
              <ListItem>
                <ListItemText
                  primary="Start with DRAFT status"
                  secondary="Test rules before approving"
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary="Use specific filters"
                  secondary="Avoid overly broad access"
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary="Review impact analysis"
                  secondary="Check affected users and systems"
                />
              </ListItem>
              <ListItem>
                <ListItemText
                  primary="Document rule purpose"
                  secondary="Add clear descriptions"
                />
              </ListItem>
            </List>
          </AccordionDetails>
        </Accordion>
      </Box>
    </Drawer>
  );
};

export default HelpDrawer;
