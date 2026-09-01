import React from "react";
import { useTheme } from "@mui/material";
import Box from "@mui/material/Box";
import IconButton from "@mui/material/IconButton";
import Alert from "@mui/material/Alert";
import type { DisclosureBanner as DisclosureBannerType } from "../schema";
import { AlertTriangle, Info, X } from "lucide-react";

interface DisclosureBannerWidgetProps {
  def: DisclosureBannerType;
}

export function DisclosureBannerWidget({ def }: DisclosureBannerWidgetProps) {
  const theme = useTheme();
  const [dismissed, setDismissed] = React.useState(false);

  if (dismissed && def.dismissible) {
    return null;
  }

  const severity = getVariantSeverity(def.variant || "");

  return (
    <Alert
      severity={severity}
      variant="filled"
      sx={{ borderRadius: 1 }}
      icon={def.variant === "warning" ? <AlertTriangle size={20} /> : <Info size={20} />}
      action={
        def.dismissible && (
          <IconButton
            size="small"
            onClick={() => setDismissed(true)}
            sx={{ 
              color: severity === 'warning' ? 'warning.dark' : 'info.dark',
              '&:hover': { color: severity === 'warning' ? 'warning.main' : 'info.main' }
            }}
          >
            <X size={20} />
          </IconButton>
        )
      }
    >
      {def.content}
    </Alert>
  );
}

function getVariantSeverity(variant: string): "warning" | "info" | "error" | "success" {
  switch (variant) {
    case "warning":
      return "warning";
    case "legal":
      return "info";
    default:
      return "info";
  }
}
