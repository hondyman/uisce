import React from "react";
import type { DisclosureBanner as DisclosureBannerType } from "../schema";
import { AlertTriangle, Info, X } from "lucide-react";

interface DisclosureBannerWidgetProps {
  def: DisclosureBannerType;
}

export function DisclosureBannerWidget({ def }: DisclosureBannerWidgetProps) {
  const [dismissed, setDismissed] = React.useState(false);

  if (dismissed && def.dismissible) {
    return null;
  }

  return (
    <div className={`rounded-lg p-4 ${getVariantStyles(def.variant)}`}>
      <div className="flex items-start gap-3">
        <div className="flex-shrink-0">
          <VariantIcon variant={def.variant} />
        </div>

        <div className="flex-1 text-sm">{def.content}</div>

        {def.dismissible && (
          <button
            onClick={() => setDismissed(true)}
            className="flex-shrink-0 text-gray-400 hover:text-gray-600"
          >
            <X className="w-5 h-5" />
          </button>
        )}
      </div>
    </div>
  );
}

function VariantIcon({ variant }: { variant: string }) {
  const iconClass = "w-5 h-5";

  switch (variant) {
    case "warning":
      return <AlertTriangle className={`${iconClass} text-yellow-600`} />;
    case "legal":
      return <Info className={`${iconClass} text-blue-600`} />;
    default:
      return <Info className={`${iconClass} text-gray-600`} />;
  }
}

function getVariantStyles(variant: string) {
  switch (variant) {
    case "warning":
      return "bg-yellow-50 border border-yellow-200 text-yellow-900";
    case "legal":
      return "bg-blue-50 border border-blue-200 text-blue-900";
    default:
      return "bg-gray-50 border border-gray-200 text-gray-900";
  }
}
