import React from "react";
import { useTheme, Box, Typography } from "@mui/material";
import type { CardComponent } from "../schema";
import { TrendingUp, TrendingDown, Minus, AlertCircle, Info } from "lucide-react";

interface CardWidgetProps {
  def: CardComponent;
}

export function CardWidget({ def }: CardWidgetProps) {
  const theme = useTheme();
  const Icon = getIconComponent(def.icon);

  return (
    <Box
      sx={{
        bgcolor: 'background.paper',
        borderRadius: 2,
        boxShadow: 1,
        p: 3,
        ...getVariantStyles(def.variant || "", theme),
        ...(def.className ? { className: def.className } : {}),
      }}
    >
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
        <Box sx={{ flex: 1 }}>
          {def.title && (
            <Typography variant="body2" sx={{ color: 'grey.600', mb: 0.5 }}>
              {def.title}
            </Typography>
          )}
          
          <Typography variant="h4" sx={{ fontWeight: 700, color: 'grey.900', mb: 1 }}>
            {def.value}
          </Typography>

          {def.metric && (
            <Typography variant="caption" sx={{ color: 'grey.500' }}>
              {def.metric}
            </Typography>
          )}

          {def.trend && (
            <Box sx={{ display: 'flex', alignItems: 'center', mt: 1, fontSize: '0.875rem', color: getTrendColor(def.trend.direction, theme) }}>
              <TrendIcon direction={def.trend.direction} />
              <Typography variant="body2" sx={{ ml: 0.5, color: 'inherit' }}>
                {def.trend.value}
              </Typography>
            </Box>
          )}
        </Box>

        {Icon && (
          <Box sx={{ ml: 2 }}>
            <Icon size={48} style={{ color: theme.palette.grey[400] }} />
          </Box>
        )}
      </Box>
    </Box>
  );
}

function TrendIcon({ direction }: { direction: "up" | "down" | "flat" }) {
  switch (direction) {
    case "up":
      return <TrendingUp size={16} />;
    case "down":
      return <TrendingDown size={16} />;
    case "flat":
      return <Minus size={16} />;
  }
}

function getTrendColor(direction: "up" | "down" | "flat", theme: any) {
  switch (direction) {
    case "up":
      return theme.palette.success.main;
    case "down":
      return theme.palette.error.main;
    case "flat":
      return theme.palette.grey[600];
  }
}

function getVariantStyles(variant: string, theme: any) {
  switch (variant) {
    case "kpi":
      return { borderLeft: `4px solid ${theme.palette.primary.main}` };
    case "alert":
      return { borderLeft: `4px solid ${theme.palette.error.main}`, bgcolor: theme.palette.error.light };
    case "info":
      return { borderLeft: `4px solid ${theme.palette.grey[300]}` };
    default:
      return {};
  }
}

function getIconComponent(iconName?: string) {
  switch (iconName) {
    case "alert":
      return AlertCircle;
    case "info":
      return Info;
    default:
      return null;
  }
}
