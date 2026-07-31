import { z } from "zod";

// Basic component schema
export const ComponentSchema = z.object({
    id: z.string(),
    type: z.enum(["grid", "rebalance_proposal", "metric_card", "chart"]),
    title: z.string().optional(),
    subtitle: z.string().optional(),
    binding: z.object({
        gql: z.string(),
        variables: z.record(z.any()).optional(),
        dataPath: z.string(),
    }).optional(),
    // Specific component props can be unioned here or kept loose
    columns: z.array(z.any()).optional(),
    actions: z.array(z.any()).optional(),
    pagination: z.any().optional(),
});

export const LayoutItemSchema = z.object({
    w: z.number(), // width in 12-col grid
    component: ComponentSchema,
});

export const LayoutRowSchema = z.object({
    items: z.array(LayoutItemSchema),
    height: z.number().optional(),
});

export const LayoutSchema = z.object({
    title: z.string(),
    rows: z.array(LayoutRowSchema),
});

export type Layout = z.infer<typeof LayoutSchema>;
export type LayoutRow = z.infer<typeof LayoutRowSchema>;
export type LayoutItem = z.infer<typeof LayoutItemSchema>;
export type GridComponent = z.infer<typeof ComponentSchema>;

export const CardComponentSchema = z.object({
    icon: z.string().optional(),
    variant: z.string().optional(),
    className: z.string().optional(),
    title: z.string().optional(),
    value: z.union([z.string(), z.number()]).optional(),
    metric: z.string().optional(),
    trend: z.object({
        direction: z.enum(["up", "down", "flat"]),
        value: z.string(),
    }).optional(),
});
export type CardComponent = z.infer<typeof CardComponentSchema>;

export const ChartComponentSchema = z.object({
    binding: z.object({
        endpoint: z.string().optional(),
        method: z.string().optional(),
        variables: z.record(z.any()).optional(),
        gql: z.string().optional(),
        dataPath: z.string().optional(),
    }).optional(),
    type: z.enum(["line", "bar", "area"]).optional(),
    chartType: z.string().optional(),
    title: z.string().optional(),
    subtitle: z.string().optional(),
    colors: z.array(z.string()).optional(),
    xField: z.string().optional(),
    yFields: z.array(z.string()).optional(),
    legend: z.boolean().optional(),
});
export type ChartComponent = z.infer<typeof ChartComponentSchema>;

export const DisclosureBannerSchema = z.object({
    variant: z.string().optional(),
    dismissible: z.boolean().optional(),
    content: z.string(),
});
export type DisclosureBanner = z.infer<typeof DisclosureBannerSchema>;

export const FormComponentSchema = z.object({
    title: z.string().optional(),
    submitAction: z.object({
        endpoint: z.string().optional(),
        mutation: z.string().optional(),
        method: z.string().optional(),
        queryKey: z.string().optional(),
        successMessage: z.string().optional(),
    }).optional(),
    fields: z.array(z.object({
        name: z.string(),
        label: z.string(),
        type: z.string(),
        required: z.boolean().optional(),
    })).optional(),
});
export type FormComponent = z.infer<typeof FormComponentSchema>;

export const TimelineComponentSchema = z.object({
    orientation: z.enum(["horizontal", "vertical"]).optional(),
    title: z.string().optional(),
    events: z.array(z.object({
        id: z.string(),
        type: z.string().optional(),
    })).optional(),
});
export type TimelineComponent = z.infer<typeof TimelineComponentSchema>;

export const RebalanceProposalSchema = z.object({
    proposal_id: z.string(),
    portfolio_id: z.string(),
    generated_at: z.string().datetime(),
    advisor_view: z.object({
        title: z.string(),
        summary: z.string(),
        tracking_error_before: z.number(),
        tracking_error_after: z.number(),
        tax_impact_usd: z.number(),
        disclosures: z.array(z.string()).default([]),
        monte_carlo: z.object({
            mean: z.number(),
            median: z.number(),
            pct05: z.number(),
            pct95: z.number(),
            confidence80_min: z.number(),
            confidence80_max: z.number(),
            runs: z.number(),
        }).optional(),
    }),
    orders: z.array(z.object({
        side: z.enum(["BUY", "SELL"]),
        symbol: z.string(),
        qty: z.number(),
        est_value_usd: z.number(),
        lots: z.array(z.object({
            lot_id: z.string(),
            term: z.enum(["short", "long"]),
            unrealized_pnl: z.number(),
        })).default([]),
        reason: z.string(),
    })),
    citations: z.array(z.object({
        id: z.string(),
        source: z.string(),
        snapshot_id: z.string(),
        excerpt: z.string(),
    })),
    actions: z.object({
        approve: z.object({ label: z.string() }),
        reject: z.object({ label: z.string() }),
        clarify: z.object({ label: z.string() }),
    }),
});

export type RebalanceProposal = z.infer<typeof RebalanceProposalSchema>;
