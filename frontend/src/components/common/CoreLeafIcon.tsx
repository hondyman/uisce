import React from 'react';
import SvgIcon, { type SvgIconProps } from '@mui/material/SvgIcon';

/**
 * The platform's "Core" (gold copy) mark - a leaf. Its counterpart for
 * tenant-created items is the stock MUI wrench (BuildIcon). Use these two
 * everywhere an item's core/custom origin is shown.
 */
export const CoreLeafIcon: React.FC<SvgIconProps> = (props) => (
  <SvgIcon viewBox="0 0 24 24" {...props}>
    <path d="M7 17c-2-3-1-9 5-12 4-2 8-1 8 1 0 5-5 11-11 13-2 1-3 0-2-2z" />
    <path d="M7 17c-1 2-2 4-3 5" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
  </SvgIcon>
);

export default CoreLeafIcon;
