import React from 'react';
import HubIcon from '@mui/icons-material/Hub';
import GavelIcon from '@mui/icons-material/Gavel';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import RefreshIcon from '@mui/icons-material/Refresh';
import DownloadIcon from '@mui/icons-material/Download';
import SettingsIcon from '@mui/icons-material/Settings';
import StorageIcon from '@mui/icons-material/Storage';
import DashboardIcon from '@mui/icons-material/Dashboard';
import FactCheckIcon from '@mui/icons-material/FactCheck';
import ScheduleIcon from '@mui/icons-material/Schedule';
import ShowChartIcon from '@mui/icons-material/ShowChart';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import LinkIcon from '@mui/icons-material/Link';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import DeleteIcon from '@mui/icons-material/Delete';
import InsertDriveFileIcon from '@mui/icons-material/InsertDriveFile';
import BusinessIcon from '@mui/icons-material/Business';
import SwapHorizIcon from '@mui/icons-material/SwapHoriz';
import TableChartIcon from '@mui/icons-material/TableChart';
import FileDownloadIcon from '@mui/icons-material/FileDownload';
import SaveIcon from '@mui/icons-material/Save';
import VisibilityIcon from '@mui/icons-material/Visibility';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import type { SvgIconProps } from '@mui/material';

/**
 * The icons a page can name. A curated set, not "any MUI icon": pages are
 * data, and importing icons by string would pull the whole icon library
 * into the bundle. Add here when a page needs another.
 */
export const PAGE_ICONS: Record<string, React.ComponentType<SvgIconProps>> = {
  hub: HubIcon,
  gavel: GavelIcon,
  play: PlayArrowIcon,
  search: SearchIcon,
  add: AddIcon,
  edit: EditIcon,
  refresh: RefreshIcon,
  download: DownloadIcon,
  settings: SettingsIcon,
  storage: StorageIcon,
  dashboard: DashboardIcon,
  review: FactCheckIcon,
  schedule: ScheduleIcon,
  chart: ShowChartIcon,
  warning: WarningAmberIcon,
  link: LinkIcon,
  pipeline: AccountTreeIcon,
  delete: DeleteIcon,
  file: InsertDriveFileIcon,
  business: BusinessIcon,
  swap: SwapHorizIcon,
  table: TableChartIcon,
  export: FileDownloadIcon,
  save: SaveIcon,
  preview: VisibilityIcon,
  assistant: AutoAwesomeIcon,
  back: ArrowBackIcon,
  help: HelpOutlineIcon,
};

export function PageIcon({ name, ...props }: { name?: string } & SvgIconProps) {
  const I = name ? PAGE_ICONS[name] : undefined;
  return I ? <I {...props} /> : null;
}
