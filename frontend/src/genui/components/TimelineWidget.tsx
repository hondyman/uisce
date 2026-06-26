import React from "react";
import type { TimelineComponent } from "../schema";
import { CheckCircle, AlertCircle, Info, XCircle } from "lucide-react";

interface TimelineWidgetProps {
  def: TimelineComponent;
}

export function TimelineWidget({ def }: TimelineWidgetProps) {
  const isVertical = def.orientation !== "horizontal";

  return (
    <div className="bg-white rounded-lg shadow p-6">
      {def.title && <h3 className="text-lg font-semibold mb-4">{def.title}</h3>}

      <div className={isVertical ? "space-y-4" : "flex space-x-4 overflow-x-auto"}>
        {def.events.map((event, idx) => (
          <div
            key={event.id}
            className={`flex ${isVertical ? "flex-row" : "flex-col"} gap-4`}
          >
            <div className="flex-shrink-0">
              <EventIcon type={event.type || "info"} />
            </div>

            <div className="flex-1">
              <div className="text-sm text-gray-500">
                {new Date(event.timestamp).toLocaleString()}
              </div>
              <div className="font-medium text-gray-900 mt-1">{event.title}</div>
              {event.description && (
                <div className="text-sm text-gray-600 mt-1">{event.description}</div>
              )}
            </div>

            {isVertical && idx < def.events.length - 1 && (
              <div className="border-l-2 border-gray-200 h-8 ml-4"></div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}

function EventIcon({ type }: { type: string }) {
  const iconClass = "w-8 h-8";

  switch (type) {
    case "success":
      return <CheckCircle className={`${iconClass} text-green-500`} />;
    case "error":
      return <XCircle className={`${iconClass} text-red-500`} />;
    case "warning":
      return <AlertCircle className={`${iconClass} text-yellow-500`} />;
    default:
      return <Info className={`${iconClass} text-blue-500`} />;
  }
}
