import React, { DragEvent, useState, useEffect } from 'react';
import { Paper, Typography, Box, Stack, useTheme, Collapse, IconButton, Divider, CircularProgress, Chip, Tabs, Tab } from '@mui/material';
import BlockIcon from '@mui/icons-material/Block';
import MonetizationOnIcon from '@mui/icons-material/MonetizationOn';
import SmartToyIcon from '@mui/icons-material/SmartToy';
import DragIndicatorIcon from '@mui/icons-material/DragIndicator';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import ExpandLessIcon from '@mui/icons-material/ExpandLess';
// New filter icons
import FormatListBulletedIcon from '@mui/icons-material/FormatListBulleted';
import LinkIcon from '@mui/icons-material/Link';
import EventIcon from '@mui/icons-material/Event';
import FunctionsIcon from '@mui/icons-material/Functions';
import CallSplitIcon from '@mui/icons-material/CallSplit';
import HowToRegIcon from '@mui/icons-material/HowToReg';
import ApiIcon from '@mui/icons-material/Api';
import TransformIcon from '@mui/icons-material/Transform';
import SummarizeIcon from '@mui/icons-material/Summarize';
// Catalog icons
import CategoryIcon from '@mui/icons-material/Category';
import LabelIcon from '@mui/icons-material/Label';
import GavelIcon from '@mui/icons-material/Gavel';
import PostAddIcon from '@mui/icons-material/PostAdd';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import ForkRightIcon from '@mui/icons-material/ForkRight';
import RepeatIcon from '@mui/icons-material/Repeat';
import TimerIcon from '@mui/icons-material/Timer';
import NotificationsActiveIcon from '@mui/icons-material/NotificationsActive';
import SecurityIcon from '@mui/icons-material/Security';
import UndoIcon from '@mui/icons-material/Undo';
import CampaignIcon from '@mui/icons-material/Campaign';
import NotificationImportantIcon from '@mui/icons-material/NotificationImportant';
import AltRouteIcon from '@mui/icons-material/AltRoute';
import PsychologyIcon from '@mui/icons-material/Psychology';
import EditNoteIcon from '@mui/icons-material/EditNote';
import LightbulbIcon from '@mui/icons-material/Lightbulb';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import RateReviewIcon from '@mui/icons-material/RateReview';
import AssignmentIcon from '@mui/icons-material/Assignment';
import FactCheckIcon from '@mui/icons-material/FactCheck';
import axios from '@/utils/axiosClient';

export interface FilterDef {
  type: string;
  icon: any;
  color: string;
  label: string;
  description?: string;
  isCustom?: boolean; // For dynamically loaded activities
}

export interface FilterCategory {
  name: string;
  filters: FilterDef[];
}

interface SemanticTerm {
  id: string;
  name: string;
  display_name: string;
  description?: string;
  data_type?: string;
}

export const defaultFilterCategories: FilterCategory[] = [
  {
    name: 'Validation',
    filters: [
      { type: 'Sanctions', icon: BlockIcon, color: '#e53935', label: 'Sanctions Check', description: 'OFAC SDN list' },
      { type: 'Limit', icon: MonetizationOnIcon, color: '#16a34a', label: 'Value Limit', description: 'Amount thresholds' },
      { type: 'List_Lookup', icon: FormatListBulletedIcon, color: '#0284c7', label: 'List Lookup', description: 'Check reference lists' },
      { type: 'Cross_Reference', icon: LinkIcon, color: '#7c3aed', label: 'Cross-Reference', description: 'Validate relationships' },
      { type: 'Date_Validator', icon: EventIcon, color: '#ea580c', label: 'Date Validator', description: 'Date range/sequence' },
      { type: 'Formula', icon: FunctionsIcon, color: '#0891b2', label: 'Formula', description: 'Custom expressions' },
      { type: 'Policy_Check', icon: GavelIcon, color: '#f59e0b', label: 'Policy Check', description: 'Business Object Rules' },
    ]
  },
  {
    name: 'Control Flow',
    filters: [
      { type: 'Conditional', icon: CallSplitIcon, color: '#c026d3', label: 'Conditional Branch', description: 'IF/ELSE routing' },
      { type: 'subPipeline', icon: AccountTreeIcon, color: '#6366f1', label: 'Sub-Pipeline', description: 'Execute child pipeline' },
      { type: 'parallel', icon: ForkRightIcon, color: '#0ea5e9', label: 'Parallel', description: 'Execute concurrently' },
      { type: 'forEach', icon: RepeatIcon, color: '#14b8a6', label: 'For Each', description: 'Iterate over collection' },
      { type: 'wait', icon: TimerIcon, color: '#f97316', label: 'Wait', description: 'Timer/delay' },
      { type: 'waitForEvent', icon: NotificationsActiveIcon, color: '#8b5cf6', label: 'Wait For Event', description: 'Signal/event listener' },
      { type: 'tryCatch', icon: SecurityIcon, color: '#ef4444', label: 'Try/Catch', description: 'Error handling' },
      { type: 'compensate', icon: UndoIcon, color: '#f59e0b', label: 'Compensate', description: 'Saga rollback' },
      { type: 'switch', icon: AltRouteIcon, color: '#10b981', label: 'Switch', description: 'Multi-way branch' },
    ]
  },
  {
    name: 'Notifications',
    filters: [
      { type: 'publishEvent', icon: CampaignIcon, color: '#ec4899', label: 'Publish Event', description: 'Emit to message bus' },
      { type: 'alert', icon: NotificationImportantIcon, color: '#f43f5e', label: 'Alert', description: 'Send notification' },
    ]
  },
  {
    name: 'LLM-Enhanced',
    filters: [
      { type: 'Interpretation', icon: PsychologyIcon, color: '#8b5cf6', label: 'Interpretation', description: 'Parse unstructured to structured' },
      { type: 'Classification', icon: CategoryIcon, color: '#a855f7', label: 'Classification', description: 'Classify/categorize input' },
      { type: 'Drafting', icon: EditNoteIcon, color: '#7c3aed', label: 'Drafting', description: 'Draft text for review' },
      { type: 'Recommendation', icon: LightbulbIcon, color: '#f59e0b', label: 'Recommendation', description: 'Generate recommendations' },
      { type: 'ExceptionExplanation', icon: HelpOutlineIcon, color: '#f97316', label: 'Explanation', description: 'Explain decisions/errors' },
    ]
  },
  {
    name: 'Human Tasks',
    filters: [
      { type: 'Approval_Gate', icon: HowToRegIcon, color: '#059669', label: 'Approval Gate', description: 'Formal approval decision' },
      { type: 'Review', icon: RateReviewIcon, color: '#10b981', label: 'Review', description: 'Review and edit data' },
      { type: 'ToDo', icon: AssignmentIcon, color: '#14b8a6', label: 'To-Do', description: 'Manual task' },
      { type: 'Acknowledgment', icon: FactCheckIcon, color: '#0ea5e9', label: 'Acknowledgment', description: 'Confirm disclosure' },
    ]
  },
  {
    name: 'Integration',
    filters: [
      { type: 'External_API', icon: ApiIcon, color: '#4f46e5', label: 'External API', description: 'Third-party calls' },
      { type: 'Transform', icon: TransformIcon, color: '#d97706', label: 'Transform', description: 'Data modification' },
      { type: 'Aggregation', icon: SummarizeIcon, color: '#be185d', label: 'Aggregation', description: 'Sum/Count records' },
      { type: 'AI_Anomaly', icon: SmartToyIcon, color: '#9333ea', label: 'AI Watchdog', description: 'ML anomaly detection' },
      { type: 'Record_Create', icon: PostAddIcon, color: '#0f766e', label: 'Create Record', description: 'Insert new DB row' },
    ]
  }
];

interface SidebarProps {
    categories?: FilterCategory[];
}

const Sidebar: React.FC<SidebarProps> = ({ categories = defaultFilterCategories }) => {
  const theme = useTheme();
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set(['Validation']));
  const [catalogExpanded, setCatalogExpanded] = useState(true);
  const [semanticTerms, setSemanticTerms] = useState<SemanticTerm[]>([]);
  const [loadingTerms, setLoadingTerms] = useState(false);

  useEffect(() => {
    fetchSemanticTerms();
  }, []);

  const fetchSemanticTerms = async () => {
    setLoadingTerms(true);
    try {
      const response = await axios.get('/api/semantic-terms');
      setSemanticTerms(response.data || []);
    } catch (err) {
      // Mock data if API not available
      setSemanticTerms([
        { id: 'term-1', name: 'customer_id', display_name: 'Customer ID', description: 'Unique customer identifier', data_type: 'string' },
        { id: 'term-2', name: 'company_name', display_name: 'Company Name', description: 'Business name', data_type: 'string' },
        { id: 'term-3', name: 'contact_name', display_name: 'Contact Name', description: 'Primary contact person', data_type: 'string' },
        { id: 'term-4', name: 'country', display_name: 'Country', description: 'Customer country', data_type: 'string' },
        { id: 'term-5', name: 'phone', display_name: 'Phone', description: 'Contact phone number', data_type: 'string' },
        { id: 'term-6', name: 'order_date', display_name: 'Order Date', description: 'Date of order placement', data_type: 'date' },
        { id: 'term-7', name: 'total_amount', display_name: 'Total Amount', description: 'Order total value', data_type: 'number' },
        { id: 'term-8', name: 'credit_limit', display_name: 'Credit Limit', description: 'Maximum credit allowed', data_type: 'number' },
      ]);
    } finally {
      setLoadingTerms(false);
    }
  };

  const toggleCategory = (name: string) => {
    const newExpanded = new Set(expandedCategories);
    if (newExpanded.has(name)) {
      newExpanded.delete(name);
    } else {
      newExpanded.add(name);
    }
    setExpandedCategories(newExpanded);
  };

  const onDragStart = (event: DragEvent<HTMLDivElement>, nodeType: string, label?: string) => {
    event.dataTransfer.setData('application/reactflow', nodeType);
    if (label) {
      event.dataTransfer.setData('application/reactflow-label', label);
    }
    event.dataTransfer.effectAllowed = 'move';
  };

  const DraggableCard = ({ filter }: { filter: FilterDef }) => (
    <Paper
        elevation={0}
        variant="outlined"
        sx={{
        p: 1.5,
        cursor: 'grab',
        display: 'flex',
        alignItems: 'center',
        gap: 1.5,
        borderRadius: 2,
        transition: 'all 0.2s cubic-bezier(0.4, 0, 0.2, 1)',
        bgcolor: 'background.default',
        '&:hover': { 
            bgcolor: 'background.paper',
            borderColor: filter.color,
            boxShadow: `0 4px 12px ${filter.color}20`,
            transform: 'translateX(4px)'
        },
        '&:active': {
            cursor: 'grabbing',
        }
        }}
        onDragStart={(event) => onDragStart(event, filter.type)}
        draggable
    >
        <Box sx={{ color: 'text.disabled', display: 'flex' }}>
            <DragIndicatorIcon sx={{ fontSize: 16 }} />
        </Box>
        <Box sx={{ 
            p: 0.75, 
            borderRadius: '50%', 
            bgcolor: `${filter.color}15`,
            color: filter.color,
            display: 'flex'
        }}>
            <filter.icon sx={{ fontSize: 16 }} />
        </Box>
        <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography variant="body2" fontWeight="600" color="text.primary" noWrap>
                {filter.label}
            </Typography>
            {filter.description && (
                <Typography variant="caption" color="text.secondary" noWrap display="block">
                    {filter.description}
                </Typography>
            )}
        </Box>
    </Paper>
  );

  const SemanticTermCard = ({ term }: { term: SemanticTerm }) => {
    // Determine filter type based on data type
    const filterType = term.data_type === 'date' ? 'Date_Validator' 
      : term.data_type === 'number' ? 'Limit' 
      : 'List_Lookup';
    
    const typeColor = term.data_type === 'date' ? '#ea580c' 
      : term.data_type === 'number' ? '#16a34a' 
      : '#0284c7';

    return (
      <Paper
        elevation={0}
        variant="outlined"
        sx={{
          p: 1,
          cursor: 'grab',
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          borderRadius: 1.5,
          transition: 'all 0.2s',
          bgcolor: 'background.default',
          '&:hover': { 
            bgcolor: 'background.paper',
            borderColor: 'primary.main',
            transform: 'translateX(4px)'
          },
        }}
        onDragStart={(event) => onDragStart(event, filterType, `Validate ${term.display_name}`)}
        draggable
      >
        <Box sx={{ color: 'text.disabled', display: 'flex' }}>
          <DragIndicatorIcon sx={{ fontSize: 14 }} />
        </Box>
        <LabelIcon sx={{ fontSize: 14, color: 'primary.main' }} />
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography variant="caption" fontWeight="600" color="text.primary" noWrap>
            {term.display_name}
          </Typography>
        </Box>
        <Chip 
          size="small" 
          label={term.data_type || 'text'} 
          sx={{ 
            fontSize: '0.6rem', 
            height: 16, 
            bgcolor: `${typeColor}15`, 
            color: typeColor,
            '& .MuiChip-label': { px: 0.75 }
          }} 
        />
      </Paper>
    );
  };

  const [tabIndex, setTabIndex] = useState(0);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabIndex(newValue);
  };

  return (
    <Box sx={{ p: 2, height: '100%', bgcolor: 'background.default', overflowY: 'auto', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ mb: 2 }}>
        <Typography variant="h6" sx={{ fontWeight: 800, background: `-webkit-linear-gradient(45deg, ${theme.palette.primary.main}, ${theme.palette.secondary.main})`, WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>
            Uisce Toolkit
        </Typography>
      </Box>

      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
        <Tabs value={tabIndex} onChange={handleTabChange} aria-label="toolkit tabs" variant="fullWidth">
            <Tab label="Tiles" sx={{ fontWeight: 600 }} />
            <Tab label="Catalog" sx={{ fontWeight: 600 }} />
        </Tabs>
      </Box>

      {/* Tiles Tab */}
      <div role="tabpanel" hidden={tabIndex !== 0} style={{ flexGrow: 1, overflowY: 'auto' }}>
        {tabIndex === 0 && (
            <Stack spacing={1}>
                {categories.map((category) => (
                <Box key={category.name}>
                    <Box 
                    onClick={() => toggleCategory(category.name)}
                    sx={{ 
                        display: 'flex', 
                        alignItems: 'center', 
                        justifyContent: 'space-between',
                        cursor: 'pointer',
                        py: 1,
                        px: 0.5,
                        borderRadius: 1,
                        '&:hover': { bgcolor: 'action.hover' }
                    }}
                    >
                    <Typography variant="overline" color="text.secondary" fontWeight="bold" sx={{ fontSize: '0.65rem' }}>
                        {category.name}
                    </Typography>
                    <IconButton size="small" sx={{ p: 0 }}>
                        {expandedCategories.has(category.name) ? <ExpandLessIcon fontSize="small" /> : <ExpandMoreIcon fontSize="small" />}
                    </IconButton>
                    </Box>
                    <Collapse in={expandedCategories.has(category.name)}>
                    <Stack spacing={1} sx={{ pb: 1 }}>
                        {category.filters.map((filter) => (
                        <DraggableCard key={filter.type} filter={filter} />
                        ))}
                    </Stack>
                    </Collapse>
                </Box>
                ))}
            </Stack>
        )}
      </div>

      {/* Catalog Tab */}
      <div role="tabpanel" hidden={tabIndex !== 1} style={{ flexGrow: 1, overflowY: 'auto' }}>
        {tabIndex === 1 && (
            <Box>
                 <Box sx={{ mb: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <Typography variant="caption" color="text.secondary">
                        {semanticTerms.length} terms available
                    </Typography>
                    {loadingTerms && <CircularProgress size={12} />}
                 </Box>

                <Stack spacing={1}>
                    {semanticTerms.map((term) => (
                        <SemanticTermCard key={term.id} term={term} />
                    ))}
                </Stack>
            </Box>
        )}
      </div>
    </Box>
  );
};

export default Sidebar;


