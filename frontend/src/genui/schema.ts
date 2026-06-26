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
