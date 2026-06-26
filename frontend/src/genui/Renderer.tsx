import React, { Suspense } from "react";
import { LayoutSchema, ComponentSchema } from "./schema";

interface GenUIRendererProps {
  layoutJson: unknown;
  fallback?: React.ReactNode;
}

export function GenUIRenderer({ layoutJson, fallback }: GenUIRendererProps) {
  // Validate and parse layout with Zod
  const layout = LayoutSchema.parse(layoutJson);

  return (
    <div className="genui-container">
      {layout.title && <h1 className="text-2xl font-bold mb-4">{layout.title}</h1>}
      {layout.description && <p className="text-gray-600 mb-6">{layout.description}</p>}

      <div className={`genui-layout-${layout.layout || "grid"} gap-4`}>
        {layout.components.map((component) => (
          <Suspense
            key={component.id}
            fallback={fallback || <ComponentSkeleton type={component.type} />}
          >
            <ComponentRenderer component={component} />
          </Suspense>
        ))}
      </div>
    </div>
  );
}

interface ComponentRendererProps {
  component: ComponentSchema;
}

function ComponentRenderer({ component }: ComponentRendererProps) {
  // Check visibility rules
  if (component.visibility) {
    // TODO: Evaluate CEL expression
    // For now, always render
  }

  switch (component.type) {
    case "chart":
      return <ChartWidget def={component} />;
    case "grid":
      return <GridWidget def={component} />;
    case "card":
      return <CardWidget def={component} />;
    case "form":
      return <FormWidget def={component} />;
    case "timeline":
      return <TimelineWidget def={component} />;
    case "disclosure":
      return <DisclosureBannerWidget def={component} />;
    default:
      return <div>Unknown component type: {(component as any).type}</div>;
  }
}

function ComponentSkeleton({ type }: { type: string }) {
  return (
    <div className="animate-pulse bg-gray-200 rounded-lg p-4">
      <div className="h-4 bg-gray-300 rounded w-1/4 mb-2"></div>
      <div className="h-32 bg-gray-300 rounded"></div>
    </div>
  );
}

// Component imports (to be implemented)
import { ChartWidget } from "./components/ChartWidget";
import { GridWidget } from "./components/GridWidget";
import { CardWidget } from "./components/CardWidget";
import { FormWidget } from "./components/FormWidget";
import { TimelineWidget } from "./components/TimelineWidget";
import { DisclosureBannerWidget } from "./components/DisclosureBanner";
