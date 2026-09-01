import React from "react";
import { useTheme } from "@mui/material/styles";
import Box from "@mui/material/Box";
import Typography from "@mui/material/Typography";
import Paper from "@mui/material/Paper";

export function Card({
  title,
  subtitle,
  children,
}: {
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
  children: React.ReactNode;
}) {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  return (
    <Paper
      elevation={0}
      sx={{
        border: 1,
        borderColor: 'divider',
        overflow: 'hidden',
      }}
    >
      {(title || subtitle) && (
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider', bgcolor: isDark ? '#1f2937' : '#f9fafb' }}>
          {title && (
            typeof title === 'string' ? (
              <Typography variant="h6" sx={{ fontWeight: 600 }}>{title}</Typography>
            ) : (
              <Box>{title}</Box>
            )
          )}
          {subtitle && (
            typeof subtitle === 'string' ? (
              <Typography variant="body2" color="text.secondary">{subtitle}</Typography>
            ) : (
              <Box>{subtitle}</Box>
            )
          )}
        </Box>
      )}
      <Box sx={{ p: 2 }}>{children}</Box>
    </Paper>
  );
}
