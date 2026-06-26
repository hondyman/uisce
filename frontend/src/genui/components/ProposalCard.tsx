import React from "react";
import { RebalanceProposal } from "../schema";
import ActionButton from "../../components/ui/ActionButton";

type Props = {
  data: RebalanceProposal;
  onApprove: () => void;
  onReject: () => void;
  onClarify: () => void;
};

export function ProposalCard({ data, onApprove, onReject, onClarify }: Props) {
  const { advisor_view, orders, citations } = data;

  return (
    <div className="bg-white dark:bg-gray-800 rounded-lg shadow-md p-6 border border-gray-200 dark:border-gray-700">
      <header className="mb-6">
        <h3 className="text-xl font-bold text-gray-900 dark:text-gray-100 mb-2">{advisor_view.title}</h3>
        <p className="text-gray-600 dark:text-gray-300">{advisor_view.summary}</p>
      </header>

        <div className="grid grid-cols-3 gap-4 mb-4">
          <div className="bg-gray-50 dark:bg-gray-900 p-3 rounded">
            <span className="text-sm text-gray-500 block">Tracking Error</span>
            <span className="font-mono font-medium">{advisor_view.tracking_error_before.toFixed(2)}% → {advisor_view.tracking_error_after.toFixed(2)}%</span>
          </div>
          <div className="bg-gray-50 dark:bg-gray-900 p-3 rounded">
            <span className="text-sm text-gray-500 block">Est. Tax Impact</span>
            <span className={`font-mono font-medium ${advisor_view.tax_impact_usd < 0 ? 'text-green-600' : 'text-red-600'}`}>
              {advisor_view.tax_impact_usd < 0 ? '+' : ''}{(-advisor_view.tax_impact_usd).toLocaleString('en-US', { style: 'currency', currency: 'USD' })}
            </span>
          </div>
        </div>

        {advisor_view.monte_carlo && (
          <div className="mb-6 bg-blue-50 dark:bg-blue-900/20 p-4 rounded border border-blue-100 dark:border-blue-800">
            <h4 className="text-sm font-semibold text-blue-900 dark:text-blue-100 mb-2">Tax Impact Confidence (Monte Carlo)</h4>
            <div className="text-sm text-blue-800 dark:text-blue-200">
              <p>Median Benefit: <span className="font-mono font-bold">{(-advisor_view.monte_carlo.median).toLocaleString('en-US', { style: 'currency', currency: 'USD' })}</span></p>
              <p className="mt-1 text-xs opacity-80">
                80% Confidence Range: {(-advisor_view.monte_carlo.confidence80_min).toLocaleString('en-US', { style: 'currency', currency: 'USD' })} – {(-advisor_view.monte_carlo.confidence80_max).toLocaleString('en-US', { style: 'currency', currency: 'USD' })}
              </p>
            </div>
          </div>
        )}
      <section className="mb-6">
        <h4 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">Proposed Orders</h4>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-800">
              <tr>
                <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Side</th>
                <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Symbol</th>
                <th className="px-3 py-2 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Qty</th>
                <th className="px-3 py-2 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Est. Value</th>
                <th className="px-3 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Reason</th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
              {orders.map((o, i) => (
                <tr key={i}>
                  <td className={`px-3 py-2 whitespace-nowrap text-sm font-medium ${o.side === 'BUY' ? 'text-green-600' : 'text-red-600'}`}>
                    {o.side}
                  </td>
                  <td className="px-3 py-2 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">{o.symbol}</td>
                  <td className="px-3 py-2 whitespace-nowrap text-sm text-gray-500 text-right">{o.qty}</td>
                  <td className="px-3 py-2 whitespace-nowrap text-sm text-gray-500 text-right">{formatUSD(o.est_value_usd)}</td>
                  <td className="px-3 py-2 whitespace-nowrap text-sm text-gray-500">{o.reason}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {orders.some(o => o.lots?.length > 0) && (
        <section className="mb-6">
          <h4 className="text-sm font-semibold text-gray-500 uppercase tracking-wider mb-3">Tax Lots Utilized</h4>
          <div className="space-y-2">
            {orders.filter(o => o.lots?.length > 0).map((o, i) => (
              <div key={i} className="text-sm">
                <span className="font-medium text-gray-700 dark:text-gray-300">{o.symbol}:</span>
                <ul className="list-disc list-inside pl-4 text-gray-600 dark:text-gray-400">
                  {o.lots!.map(l => (
                    <li key={l.lot_id}>
                      ID: {l.lot_id} • {l.term} term • UPNL: <span className={l.unrealized_pnl >= 0 ? "text-green-600" : "text-red-600"}>{formatUSD(l.unrealized_pnl)}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </section>
      )}

      <section className="mb-6 bg-blue-50 dark:bg-blue-900/20 p-4 rounded-md border border-blue-100 dark:border-blue-800">
        <h4 className="text-sm font-semibold text-blue-800 dark:text-blue-300 uppercase tracking-wider mb-2">Evidence & Citations</h4>
        <ul className="space-y-1">
          {citations.map(c => (
            <li key={c.id} className="text-sm text-blue-900 dark:text-blue-100">
              <span className="font-mono text-xs bg-blue-100 dark:bg-blue-800 px-1 rounded mr-2">{c.id}</span>
              <span className="font-medium">{c.source}</span>
              <span className="text-blue-600 dark:text-blue-400 mx-1">•</span>
              <span className="text-xs text-gray-500 dark:text-gray-400">snap {c.snapshot_id}</span>
              <div className="pl-8 text-gray-600 dark:text-gray-300 italic">“{c.excerpt}”</div>
            </li>
          ))}
        </ul>
      </section>

      {advisor_view.disclosures && advisor_view.disclosures.length > 0 && (
        <section className="mb-8">
          <h4 className="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-1">Disclosures</h4>
          <ul className="list-disc list-inside text-xs text-gray-400">
            {advisor_view.disclosures.map((d, i) => <li key={i}>{d}</li>)}
          </ul>
        </section>
      )}

      <footer className="flex gap-3 pt-4 border-t border-gray-200 dark:border-gray-700">
        <ActionButton variant="success" onClick={onApprove}>{data.actions.approve.label}</ActionButton>
        <ActionButton variant="danger" onClick={onReject}>{data.actions.reject.label}</ActionButton>
        <ActionButton variant="secondary" onClick={onClarify}>{data.actions.clarify.label}</ActionButton>
      </footer>
    </div>
  );
}

function Metric({ label, value, valueClass = "text-gray-900 dark:text-gray-100" }: { label: string; value: string; valueClass?: string }) {
  return (
    <div className="flex flex-col">
      <span className="text-xs text-gray-500 uppercase tracking-wider mb-1">{label}</span>
      <span className={`text-lg font-semibold ${valueClass}`}>{value}</span>
    </div>
  );
}

function formatUSD(n: number) {
  const sign = n < 0 ? "-" : "";
  return `${sign}$${Math.abs(n).toLocaleString(undefined, { maximumFractionDigits: 0 })}`;
}
