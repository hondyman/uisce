
import React, { useEffect, useState } from 'react';
import { Box, Typography, CircularProgress } from '@mui/material';
import { UisceBuilder } from '../features/uisce-builder/UisceBuilder';
import { defaultFilterCategories, FilterCategory, FilterDef } from '../features/uisce-builder/components/Sidebar';
import axios from 'axios';
import HowToRegIcon from '@mui/icons-material/HowToReg';
import SmartToyIcon from '@mui/icons-material/SmartToy';
import ApiIcon from '@mui/icons-material/Api';
import SummarizeIcon from '@mui/icons-material/Summarize';
import TransformIcon from '@mui/icons-material/Transform';

// Map activity names to UI Metadata (Icon, Color, Label)
const activityMetadata: Record<string, Omit<FilterDef, 'type'>> = {
  'ActivityCheckCompliance': { icon: HowToRegIcon, color: '#e53935', label: 'Check Compliance', description: 'Run pre-trade compliance checks' },
  'ActivityValidateGoldenRecord': { icon: ApiIcon, color: '#16a34a', label: 'Validate Golden Record', description: 'Ensure data consistency' },
  'ActivityUserInteraction': { icon: HowToRegIcon, color: '#059669', label: 'User Task', description: 'Request user input/approval' },
  'ActivityGenerateContent': { icon: SmartToyIcon, color: '#9333ea', label: 'AI Content Gen', description: 'Generate content with GenAI' },
  'ApprovalActivity': { icon: HowToRegIcon, color: '#059669', label: 'Generic Approval', description: 'Request approval' },
  'EmailNotificationActivity': { icon: SummarizeIcon, color: '#0284c7', label: 'Send Email', description: 'Email notification' },
  // ... add more mappings as needed
};

const WorkflowStudioPage: React.FC = () => {
  const [loading, setLoading] = useState(true);
  const [safeCategories, setSafeCategories] = useState<FilterCategory[]>([]);

  useEffect(() => {
    const fetchSafeActivities = async () => {
      try {
        setLoading(true);
        // Fetch allowlist from backend (authenticated as client)
        const response = await axios.get('/api/v1/pipelines/activities/safe');
        const allowedActivities: string[] = response.data.activities || [];

        // Build filtered categories based on allowed activities
        const clientCategory: FilterCategory = {
           name: 'Client Safe',
           filters: []
        };

        allowedActivities.forEach(actName => {
             const meta = activityMetadata[actName];
             if (meta) {
                 clientCategory.filters.push({
                     type: actName,
                     ...meta,
                     isCustom: true
                 });
             } else {
                 // Fallback for unknown activities
                  clientCategory.filters.push({
                     type: actName,
                     icon: TransformIcon,
                     color: '#64748b',
                     label: actName,
                     description: 'Custom Activity',
                     isCustom: true
                 });
             }
        });

        // Also include standard Logic nodes (Control Flow) if safe?
        // For now, let's explicit allow specific categories or just use the filtered list.
        // We'll trust the explicitly fetched list.

        setSafeCategories([clientCategory]);
      } catch (err) {
        console.error("Failed to fetch safe activities", err);
        // Fallback for demo if API fails
         setSafeCategories(defaultFilterCategories); 
      } finally {
        setLoading(false);
      }
    };

    fetchSafeActivities();
  }, []);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ height: '100vh' }}>
        {/* Inject our safe filtered categories into the builder */}
        <UisceBuilder filterCategories={safeCategories} />
    </Box>
  );
};

export default WorkflowStudioPage;
