import React from "react";
import type { CardComponent } from "../schema";
import { TrendingUp, TrendingDown, Minus, AlertCircle, Info } from "lucide-react";

interface CardWidgetProps {
  def: CardComponent;
}

export function CardWidget({ def }: CardWidgetProps) {
  const Icon = getIconComponent(def.icon);

  return (
    <div
      className={`bg-white rounded-lg shadow p-6 ${getVariantStyles(def.variant)} ${
        def.className || ""
      }`}
    >
      <div className="flex items-center justify-between">
        <div className="flex-1">
          {def.title && <h4 className="text-sm font-medium text-gray-600 mb-1">{def.title}</h4>}
          
          <div className="text-3xl font-bold text-gray-900 mb-2">{def.value}</div>

          {def.metric && <p className="text-xs text-gray-500">{def.metric}</p>}

          {def.trend && (
            <div className={`flex items-center mt-2 text-sm ${getTrendColor(def.trend.direction)}`}>
              <TrendIcon direction={def.trend.direction} />
              <span className="ml-1">{def.trend.value}</span>
            </div>
          )}
        </div>

        {Icon && (
          <div className="ml-4">
            <Icon className="w-12 h-12 text-gray-400" />
          </div>
        )}
      </div>
    </div>
  );
}

function TrendIcon({ direction }: { direction: "up" | "down" | "flat" }) {
  switch (direction) {
    case "up":
      return <TrendingUp className="w-4 h-4" />;
    case "down":
      return <TrendingDown className="w-4 h-4" />;
    case "flat":
      return <Minus className="w-4 h-4" />;
  }
}

function getTrendColor(direction: "up" | "down" | "flat") {
  switch (direction) {
    case "up":
      return "text-green-600";
    case "down":
      return "text-red-600";
    case "flat":
      return "text-gray-600";
  }
}

function getVariantStyles(variant: string) {
  switch (variant) {
    case "kpi":
      return "border-l-4 border-blue-500";
    case "alert":
      return "border-l-4 border-red-500 bg-red-50";
    case "info":
      return "border-l-4 border-gray-300";
    default:
      return "";
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
