import { RebalanceProposal } from "./schema";

export const mockRebalanceProposal: RebalanceProposal = {
    proposal_id: "prop_2025_11_22_001",
    portfolio_id: "pf_892",
    generated_at: "2025-11-22T16:12:03Z",
    advisor_view: {
        title: "Reduce US Equity overweight; harvest BND losses to offset gains",
        summary: "Sell 50 IVV to lower drift; sell 100 BND (loss) to offset ~$2,000 gain; buy AGG to maintain fixed-income exposure.",
        tracking_error_before: 2.0,
        tracking_error_after: 1.4,
        tax_impact_usd: -1800.0,
        disclosures: [
            "Rebalancing may trigger transaction costs and tax events",
            "Loss harvesting subject to wash‑sale rules"
        ]
    },
    orders: [
        {
            side: "SELL",
            symbol: "IVV",
            qty: 50,
            est_value_usd: 22500,
            lots: [],
            reason: "reduce_overweight"
        },
        {
            side: "SELL",
            symbol: "BND",
            qty: 100,
            est_value_usd: 7000,
            lots: [
                { lot_id: "bnd_l1", term: "long", unrealized_pnl: -1200 }
            ],
            reason: "harvest_loss"
        },
        {
            side: "BUY",
            symbol: "AGG",
            qty: 100,
            est_value_usd: 10000,
            lots: [],
            reason: "replacement_buy"
        }
    ],
    citations: [
        {
            id: "C1",
            source: "snap_positions_001",
            snapshot_id: "snap_20251122_positions",
            excerpt: "Current weights: IVV 35%, target 30%."
        },
        {
            id: "C2",
            source: "tax_rules_2025-11-22",
            snapshot_id: "tax_rules_snap_33",
            excerpt: "Wash-sale period: 30 days; long-term losses preferred."
        }
    ],
    actions: {
        approve: { label: "Approve and execute" },
        reject: { label: "Reject" },
        clarify: { label: "Request clarification" }
    }
};
