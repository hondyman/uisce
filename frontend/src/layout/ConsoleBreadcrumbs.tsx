import React from 'react';
import { Breadcrumbs as MUIBreadcrumbs, Link, Typography } from '@mui/material';
import NavigateNextIcon from '@mui/icons-material/NavigateNext';

export interface BreadcrumbItem {
  label: string;
  href?: string;
}

export function ConsoleBreadcrumbs({ items }: { items: BreadcrumbItem[] }) {
  return (
    <MUIBreadcrumbs separator={<NavigateNextIcon fontSize="small" />} sx={{ mb: 3 }}>
      {items.map((item, i) => (
        item.href && i < items.length - 1 ?
          (
            <Link key={i} href={item.href} underline="hover" sx={{ cursor: 'pointer' }}>
              {item.label}
            </Link>
          )
          : (
            <Typography key={i} color={i === items.length - 1 ? 'textPrimary' : 'textSecondary'}>
              {item.label}
            </Typography>
          )
      ))}
    </MUIBreadcrumbs>
  );
}
