/**
 * Icon Mapping Guide: Antd → Lucide React + MUI Icons Material
 * 
 * This file documents all antd icon replacements used across the codebase.
 * Prefer lucide-react for most cases (lighter, more consistent with modern design)
 * Use @mui/icons-material for MUI-specific components or when lucide lacks the icon
 */

// Mapping of antd icons to replacements
export const ICON_MAPPING = {
  // Common Actions
  'PlusOutlined': { lucide: 'Plus', mui: 'AddIcon' },
  'DeleteOutlined': { lucide: 'Trash2', mui: 'DeleteIcon' },
  'EditOutlined': { lucide: 'Edit', mui: 'EditIcon' },
  'SearchOutlined': { lucide: 'Search', mui: 'SearchIcon' },
  'SaveOutlined': { lucide: 'Save', mui: 'SaveIcon' },
  'CloseOutlined': { lucide: 'X', mui: 'CloseIcon' },
  'CheckOutlined': { lucide: 'Check', mui: 'CheckIcon' },
  'CopyOutlined': { lucide: 'Copy', mui: 'ContentCopyIcon' },
  'EyeOutlined': { lucide: 'Eye', mui: 'VisibilityIcon' },
  'EyeInvisibleOutlined': { lucide: 'EyeOff', mui: 'VisibilityOffIcon' },
  
  // Navigation
  'ArrowLeftOutlined': { lucide: 'ArrowLeft', mui: 'ArrowBackIcon' },
  'ArrowRightOutlined': { lucide: 'ArrowRight', mui: 'ArrowForwardIcon' },
  'ArrowUpOutlined': { lucide: 'ArrowUp', mui: 'ArrowUpwardIcon' },
  'ArrowDownOutlined': { lucide: 'ArrowDown', mui: 'ArrowDownwardIcon' },
  'MenuOutlined': { lucide: 'Menu', mui: 'MenuIcon' },
  'ExpandOutlined': { lucide: 'Maximize2', mui: 'FullscreenIcon' },
  'ShrinkOutlined': { lucide: 'Minimize2', mui: 'FullscreenExitIcon' },
  
  // Status & Alerts
  'CheckCircleOutlined': { lucide: 'CheckCircle', mui: 'CheckCircleIcon' },
  'CloseCircleOutlined': { lucide: 'XCircle', mui: 'CancelIcon' },
  'ExclamationCircleOutlined': { lucide: 'AlertCircle', mui: 'WarningIcon' },
  'InfoCircleOutlined': { lucide: 'Info', mui: 'InfoIcon' },
  'WarningOutlined': { lucide: 'AlertTriangle', mui: 'WarningIcon' },
  'WarningFilled': { lucide: 'AlertTriangle', mui: 'WarningIcon' },
  'ErrorOutlined': { lucide: 'AlertCircle', mui: 'ErrorIcon' },
  'QuestionCircleOutlined': { lucide: 'HelpCircle', mui: 'HelpIcon' },
  
  // Editing & Content
  'FormOutlined': { lucide: 'FileText', mui: 'AssignmentIcon' },
  'CodeOutlined': { lucide: 'Code', mui: 'CodeIcon' },
  'ClearOutlined': { lucide: 'Trash', mui: 'DeleteIcon' },
  'DownloadOutlined': { lucide: 'Download', mui: 'DownloadIcon' },
  'UploadOutlined': { lucide: 'Upload', mui: 'UploadIcon' },
  'PrinterOutlined': { lucide: 'Printer', mui: 'PrintIcon' },
  'BgColorsOutlined': { lucide: 'Palette', mui: 'PaletteIcon' },
  'FontSizeOutlined': { lucide: 'Type', mui: 'TextFieldsIcon' },
  
  // Time & Schedule
  'ClockCircleOutlined': { lucide: 'Clock', mui: 'ScheduleIcon' },
  'CalendarOutlined': { lucide: 'Calendar', mui: 'DateRangeIcon' },
  'HistoryOutlined': { lucide: 'History', mui: 'HistoryIcon' },
  'RollbackOutlined': { lucide: 'RotateCcw', mui: 'RestoreIcon' },
  'RedoOutlined': { lucide: 'Redo2', mui: 'RedoIcon' },
  'UndoOutlined': { lucide: 'Undo2', mui: 'UndoIcon' },
  
  // Database & Data
  'DatabaseOutlined': { lucide: 'Database', mui: 'StorageIcon' },
  'TableOutlined': { lucide: 'Table', mui: 'TableChartIcon' },
  'BarChartOutlined': { lucide: 'BarChart3', mui: 'BarChartIcon' },
  'LineChartOutlined': { lucide: 'LineChart', mui: 'ShowChartIcon' },
  'PieChartOutlined': { lucide: 'PieChart', mui: 'PieChartIcon' },
  'FundOutlined': { lucide: 'TrendingUp', mui: 'TrendingUpIcon' },
  'SortAscendingOutlined': { lucide: 'ArrowUp', mui: 'SortIcon' },
  'SortDescendingOutlined': { lucide: 'ArrowDown', mui: 'SortIcon' },
  
  // Settings & Configuration
  'SettingOutlined': { lucide: 'Settings', mui: 'SettingsIcon' },
  'ToolOutlined': { lucide: 'Wrench', mui: 'BuildIcon' },
  'FilterOutlined': { lucide: 'Filter', mui: 'FilterListIcon' },
  'ShakeOutlined': { lucide: 'Shuffle', mui: 'ShuffleIcon' },
  'ThunderboltOutlined': { lucide: 'Zap', mui: 'FlashOnIcon' },
  'PlayCircleOutlined': { lucide: 'Play', mui: 'PlayCircleFilledIcon' },
  'PauseCircleOutlined': { lucide: 'Pause', mui: 'PauseCircleFilledIcon' },
  'StopOutlined': { lucide: 'Square', mui: 'StopIcon' },
  'PoweroffOutlined': { lucide: 'Power', mui: 'PowerSettingsNewIcon' },
  'ReloadOutlined': { lucide: 'RefreshCw', mui: 'RefreshIcon' },
  
  // Integration & Connections
  'LinkOutlined': { lucide: 'Link', mui: 'LinkIcon' },
  'UnlinkOutlined': { lucide: 'Unlink', mui: 'LinkOffIcon' },
  'ApiOutlined': { lucide: 'Zap', mui: 'WebIcon' },
  'InsertRowAboveOutlined': { lucide: 'ArrowUp', mui: 'ArrowUpwardIcon' },
  'InsertRowBelowOutlined': { lucide: 'ArrowDown', mui: 'ArrowDownwardIcon' },
  'DeleteRowOutlined': { lucide: 'X', mui: 'DeleteIcon' },
  
  // Organization & Structure
  'FolderOutlined': { lucide: 'Folder', mui: 'FolderIcon' },
  'FolderOpenOutlined': { lucide: 'FolderOpen', mui: 'FolderOpenIcon' },
  'FileOutlined': { lucide: 'File', mui: 'DescriptionIcon' },
  'FileTextOutlined': { lucide: 'FileText', mui: 'ArticleIcon' },
  'HomeOutlined': { lucide: 'Home', mui: 'HomeIcon' },
  'BranchesOutlined': { lucide: 'GitBranch', mui: 'GitHubIcon' },
  
  // Users & Permissions
  'UserOutlined': { lucide: 'User', mui: 'PersonIcon' },
  'UserAddOutlined': { lucide: 'UserPlus', mui: 'PersonAddIcon' },
  'UserDeleteOutlined': { lucide: 'UserX', mui: 'PersonRemoveIcon' },
  'TeamOutlined': { lucide: 'Users', mui: 'GroupIcon' },
  'LockOutlined': { lucide: 'Lock', mui: 'LockIcon' },
  'UnlockOutlined': { lucide: 'Unlock', mui: 'LockOpenIcon' },
  'SecurityScanOutlined': { lucide: 'Shield', mui: 'SecurityIcon' },
  'RobotOutlined': { lucide: 'Bot', mui: 'SmartToyIcon' },
  'SafeOutlined': { lucide: 'Shield', mui: 'VaultIcon' },
  
  // Money & Commerce
  'DollarOutlined': { lucide: 'DollarSign', mui: 'AttachMoneyIcon' },
  'ShoppingCartOutlined': { lucide: 'ShoppingCart', mui: 'ShoppingCartIcon' },
  'ShoppingOutlined': { lucide: 'ShoppingBag', mui: 'ShoppingBagIcon' },
  'CreditCardOutlined': { lucide: 'CreditCard', mui: 'CreditCardIcon' },
  'WalletOutlined': { lucide: 'Wallet', mui: 'WalletIcon' },
  
  // Miscellaneous
  'FlagOutlined': { lucide: 'Flag', mui: 'FlagIcon' },
  'StarOutlined': { lucide: 'Star', mui: 'StarIcon' },
  'HeartOutlined': { lucide: 'Heart', mui: 'FavoriteBorderIcon' },
  'SmileOutlined': { lucide: 'Smile', mui: 'EmojiEmotionsIcon' },
  'BellOutlined': { lucide: 'Bell', mui: 'NotificationsIcon' },
  'MailOutlined': { lucide: 'Mail', mui: 'MailIcon' },
  'PhoneOutlined': { lucide: 'Phone', mui: 'PhoneIcon' },
  'GlobalOutlined': { lucide: 'Globe', mui: 'PublicIcon' },
  'CarOutlined': { lucide: 'Car', mui: 'DirectionsCarIcon' },
  'BoxPlotOutlined': { lucide: 'Box', mui: 'ViewInArIcon' },
};

/**
 * Usage Examples:
 * 
 * // Before (antd):
 * import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
 * <PlusOutlined />
 * <DeleteOutlined />
 * 
 * // After (lucide-react):
 * import { Plus, Trash2 } from 'lucide-react';
 * <Plus className="w-5 h-5" />
 * <Trash2 className="w-5 h-5" />
 * 
 * // Alternative (MUI Icons):
 * import { AddIcon, DeleteIcon } from '@mui/icons-material';
 * <AddIcon />
 * <DeleteIcon />
 */

export const LUCIDE_ICON_CLASSES = 'w-5 h-5'; // Standard icon size
export const MUI_ICON_SIZE = 'medium'; // MUI standard size
