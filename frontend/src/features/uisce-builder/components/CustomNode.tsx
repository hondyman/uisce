import React, { memo } from 'react';
import { Handle, Position, NodeProps } from 'reactflow';
import { Box, Typography, Paper, Tooltip, IconButton } from '@mui/material';
import BlockIcon from '@mui/icons-material/Block';
import MonetizationOnIcon from '@mui/icons-material/MonetizationOn';
import SmartToyIcon from '@mui/icons-material/SmartToy';
import SettingsIcon from '@mui/icons-material/Settings';
import MoreVertIcon from '@mui/icons-material/MoreVert';
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
import CategoryIcon from '@mui/icons-material/Category';

interface NodeStyle {
  icon: any;
  gradient: string;
  iconColor: string;
  shadowColor: string;
}

const nodeStyles: Record<string, NodeStyle> = {
  Sanctions: {
    icon: BlockIcon,
    gradient: 'linear-gradient(135deg, #FFFEFF 0%, #D7FFFE 100%)',
    iconColor: '#e53935',
    shadowColor: 'rgba(229, 57, 53, 0.2)'
  },
  Limit: {
    icon: MonetizationOnIcon,
    gradient: 'linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%)',
    iconColor: '#16a34a',
    shadowColor: 'rgba(22, 163, 74, 0.2)'
  },
  AI_Anomaly: {
    icon: SmartToyIcon,
    gradient: 'linear-gradient(135deg, #f3e8ff 0%, #e9d5ff 100%)',
    iconColor: '#9333ea',
    shadowColor: 'rgba(147, 51, 234, 0.2)'
  },
  List_Lookup: {
    icon: FormatListBulletedIcon,
    gradient: 'linear-gradient(135deg, #e0f2fe 0%, #bae6fd 100%)',
    iconColor: '#0284c7',
    shadowColor: 'rgba(2, 132, 199, 0.2)'
  },
  Cross_Reference: {
    icon: LinkIcon,
    gradient: 'linear-gradient(135deg, #ede9fe 0%, #ddd6fe 100%)',
    iconColor: '#7c3aed',
    shadowColor: 'rgba(124, 58, 237, 0.2)'
  },
  Date_Validator: {
    icon: EventIcon,
    gradient: 'linear-gradient(135deg, #fff7ed 0%, #fed7aa 100%)',
    iconColor: '#ea580c',
    shadowColor: 'rgba(234, 88, 12, 0.2)'
  },
  Formula: {
    icon: FunctionsIcon,
    gradient: 'linear-gradient(135deg, #ecfeff 0%, #a5f3fc 100%)',
    iconColor: '#0891b2',
    shadowColor: 'rgba(8, 145, 178, 0.2)'
  },
  Conditional: {
    icon: CallSplitIcon,
    gradient: 'linear-gradient(135deg, #fdf4ff 0%, #f5d0fe 100%)',
    iconColor: '#c026d3',
    shadowColor: 'rgba(192, 38, 211, 0.2)'
  },
  Approval_Gate: {
    icon: HowToRegIcon,
    gradient: 'linear-gradient(135deg, #ecfdf5 0%, #a7f3d0 100%)',
    iconColor: '#059669',
    shadowColor: 'rgba(5, 150, 105, 0.2)'
  },
  External_API: {
    icon: ApiIcon,
    gradient: 'linear-gradient(135deg, #eef2ff 0%, #c7d2fe 100%)',
    iconColor: '#4f46e5',
    shadowColor: 'rgba(79, 70, 229, 0.2)'
  },
  Transform: {
    icon: TransformIcon,
    gradient: 'linear-gradient(135deg, #fffbeb 0%, #fde68a 100%)',
    iconColor: '#d97706',
    shadowColor: 'rgba(217, 119, 6, 0.2)'
  },
  Aggregation: {
    icon: SummarizeIcon,
    gradient: 'linear-gradient(135deg, #fdf2f8 0%, #fbcfe8 100%)',
    iconColor: '#be185d',
    shadowColor: 'rgba(190, 24, 93, 0.2)'
  },
  Policy_Check: {
    icon: GavelIcon,
    gradient: 'linear-gradient(135deg, #fffbeb 0%, #fcd34d 100%)',
    iconColor: '#f59e0b',
    shadowColor: 'rgba(245, 158, 11, 0.2)'
  },
  Record_Create: {
    icon: PostAddIcon,
    gradient: 'linear-gradient(135deg, #f0fdfa 0%, #ccfbf1 100%)',
    iconColor: '#0f766e',
    shadowColor: 'rgba(15, 118, 110, 0.2)'
  },
  subPipeline: {
    icon: AccountTreeIcon,
    gradient: 'linear-gradient(135deg, #eef2ff 0%, #e0e7ff 100%)',
    iconColor: '#6366f1',
    shadowColor: 'rgba(99, 102, 241, 0.2)'
  },
  // Backward compatibility
  Child_Pipeline: {
    icon: AccountTreeIcon,
    gradient: 'linear-gradient(135deg, #eef2ff 0%, #e0e7ff 100%)',
    iconColor: '#6366f1',
    shadowColor: 'rgba(99, 102, 241, 0.2)'
  },
  parallel: {
    icon: ForkRightIcon,
    gradient: 'linear-gradient(135deg, #e0f2fe 0%, #7dd3fc 100%)',
    iconColor: '#0ea5e9',
    shadowColor: 'rgba(14, 165, 233, 0.2)'
  },
  forEach: {
    icon: RepeatIcon,
    gradient: 'linear-gradient(135deg, #ccfbf1 0%, #5eead4 100%)',
    iconColor: '#14b8a6',
    shadowColor: 'rgba(20, 184, 166, 0.2)'
  },
  wait: {
    icon: TimerIcon,
    gradient: 'linear-gradient(135deg, #ffedd5 0%, #fed7aa 100%)',
    iconColor: '#f97316',
    shadowColor: 'rgba(249, 115, 22, 0.2)'
  },
  waitForEvent: {
    icon: NotificationsActiveIcon,
    gradient: 'linear-gradient(135deg, #ede9fe 0%, #ddd6fe 100%)',
    iconColor: '#8b5cf6',
    shadowColor: 'rgba(139, 92, 246, 0.2)'
  },
  tryCatch: {
    icon: SecurityIcon,
    gradient: 'linear-gradient(135deg, #fee2e2 0%, #fecaca 100%)',
    iconColor: '#ef4444',
    shadowColor: 'rgba(239, 68, 68, 0.2)'
  },
  compensate: {
    icon: UndoIcon,
    gradient: 'linear-gradient(135deg, #fef3c7 0%, #fde68a 100%)',
    iconColor: '#f59e0b',
    shadowColor: 'rgba(245, 158, 11, 0.2)'
  },
  switch: {
    icon: AltRouteIcon,
    gradient: 'linear-gradient(135deg, #d1fae5 0%, #a7f3d0 100%)',
    iconColor: '#10b981',
    shadowColor: 'rgba(16, 185, 129, 0.2)'
  },
  publishEvent: {
    icon: CampaignIcon,
    gradient: 'linear-gradient(135deg, #fce7f3 0%, #fbcfe8 100%)',
    iconColor: '#ec4899',
    shadowColor: 'rgba(236, 72, 153, 0.2)'
  },
  alert: {
    icon: NotificationImportantIcon,
    gradient: 'linear-gradient(135deg, #ffe4e6 0%, #fecdd3 100%)',
    iconColor: '#f43f5e',
    shadowColor: 'rgba(244, 63, 94, 0.2)'
  },
  Interpretation: {
    icon: PsychologyIcon,
    gradient: 'linear-gradient(135deg, #ede9fe 0%, #ddd6fe 100%)',
    iconColor: '#8b5cf6',
    shadowColor: 'rgba(139, 92, 246, 0.2)'
  },
  Classification: {
    icon: CategoryIcon,
    gradient: 'linear-gradient(135deg, #f3e8ff 0%, #e9d5ff 100%)',
    iconColor: '#a855f7',
    shadowColor: 'rgba(168, 85, 247, 0.2)'
  },
  Drafting: {
    icon: EditNoteIcon,
    gradient: 'linear-gradient(135deg, #ede9fe 0%, #c4b5fd 100%)',
    iconColor: '#7c3aed',
    shadowColor: 'rgba(124, 58, 237, 0.2)'
  },
  Recommendation: {
    icon: LightbulbIcon,
    gradient: 'linear-gradient(135deg, #fef3c7 0%, #fde68a 100%)',
    iconColor: '#f59e0b',
    shadowColor: 'rgba(245, 158, 11, 0.2)'
  },
  ExceptionExplanation: {
    icon: HelpOutlineIcon,
    gradient: 'linear-gradient(135deg, #ffedd5 0%, #fed7aa 100%)',
    iconColor: '#f97316',
    shadowColor: 'rgba(249, 115, 22, 0.2)'
  }
};

const defaultStyle: NodeStyle = {
  icon: SettingsIcon,
  gradient: 'linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%)',
  iconColor: '#555',
  shadowColor: 'rgba(0,0,0,0.1)'
};

const CustomNode = ({ data, selected }: NodeProps) => {
  // Get style based on node type (use label to determine type)
  const nodeType = Object.keys(nodeStyles).find(key => data.label.includes(key.replace('_', ' ')) || data.label.includes(key)) || '';
  const style = nodeStyles[nodeType] || defaultStyle;
  const Icon = style.icon;

  // Debug/Trace status styling
  const isPass = data.traceStatus === 'PASS';
  const isFail = data.traceStatus === 'FAIL';

  let borderColor = 'transparent';
  if (selected) borderColor = '#2563eb';
  if (isPass) borderColor = '#22c55e';
  if (isFail) borderColor = '#ef4444';

  return (
    <Paper
      elevation={selected ? 8 : 2}
      sx={{
        minWidth: 180,
        borderRadius: 3,
        background: style.gradient,
        border: '2px solid',
        borderColor: borderColor,
        transition: 'all 0.3s cubic-bezier(0.25, 0.8, 0.25, 1)',
        overflow: 'hidden',
        boxShadow: selected ? `0 10px 25px ${style.shadowColor}` : `0 2px 5px ${style.shadowColor}`,
        '&:hover': {
             transform: 'translateY(-2px)',
             boxShadow: `0 8px 20px -4px ${style.shadowColor}` 
        }
      }}
    >
      {/* Header Area */}
      <Box 
        sx={{ 
          p: 1.5, 
          display: 'flex', 
          alignItems: 'center', 
          justifyContent: 'space-between',
          borderBottom: '1px solid rgba(0,0,0,0.06)',
          backdropFilter: 'blur(4px)'
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ 
                p: 0.5, 
                borderRadius: '50%', 
                bgcolor: 'rgba(255,255,255,0.6)',
                display: 'flex',
                boxShadow: '0 2px 4px rgba(0,0,0,0.05)' 
            }}>
                <Icon sx={{ color: style.iconColor, fontSize: 18 }} />
            </Box>
            <Typography variant="subtitle2" component="div" sx={{ fontWeight: 700, fontSize: '0.85rem', color: '#1e293b' }}>
            {data.label}
            </Typography>
        </Box>
        <IconButton size="small" sx={{ opacity: 0.5, width: 20, height: 20 }}>
            <MoreVertIcon fontSize="inherit" />
        </IconButton>
      </Box>
      
      {/* Body Area */}
      <Box sx={{ p: 1.5, bgcolor: 'rgba(255,255,255,0.4)' }}>
        <Typography variant="caption" color="text.secondary" display="block" sx={{ fontSize: '0.7rem', fontWeight: 500 }}>
            {Object.keys(data.config || {}).length > 0 ? (
                <span style={{ color: '#0f766e' }}>● Configured</span>
            ) : (
                <span style={{ color: '#94a3b8' }}>○ Default</span>
            )}
        </Typography>
        {isFail && (
            <Tooltip title={data.traceError || "Error"}>
                <Typography variant="caption" sx={{ color: '#ef4444', fontWeight: 'bold' }}>
                    ⚠ Validation Failed
                </Typography>
            </Tooltip>
        )}
      </Box>

      {/* Input Handle */}
      <Handle
        type="target"
        position={Position.Left}
        style={{
          background: '#94a3b8',
          width: 8,
          height: 16,
          borderRadius: 4,
          border: '2px solid white',
          left: -6
        }}
      />
      
      {/* Output Handle */}
      <Handle
        type="source"
        position={Position.Right}
        style={{
          background: '#94a3b8',
          width: 8,
          height: 16,
          borderRadius: 4,
          border: '2px solid white',
          right: -6
        }}
      />
    </Paper>
  );
};

export default memo(CustomNode);

