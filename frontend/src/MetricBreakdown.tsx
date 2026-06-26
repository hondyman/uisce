// React default import removed (not used as a value)

interface Props {
  certified: number;
  claims?: any;
  usage?: number;
  audit?: number;
  risk?: number;
}

export default function MetricBreakdown({ certified, claims: _claims, usage, audit, risk }: Props) {
  return (
    <div className="metric-breakdown">
      <div className="mb-item">Certified: {certified}%</div>
      <div className="mb-item">Usage Coverage: {usage ?? '-'}</div>
      <div className="mb-item">Audit: {audit ?? '-'}</div>
      <div className="mb-item">Risk: {risk ?? '-'}</div>
    </div>
  );
}