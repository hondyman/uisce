import React from 'react';
import { JobRun } from '../../api/schedulerApi';
import { CheckCircle, XCircle, Clock, AlertTriangle } from 'lucide-react';

interface JobRunTimelineProps {
  runs: JobRun[];
  loading?: boolean;
}

export const JobRunTimeline: React.FC<JobRunTimelineProps> = ({ runs, loading }) => {
  if (loading) {
    return <div className="timeline-loading">Loading timeline...</div>;
  }

  if (!runs || runs.length === 0) {
    return <div className="timeline-empty">No run history</div>;
  }

  // Sort runs by created_at desc
  const sortedRuns = [...runs].sort((a, b) => 
    new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  );

  return (
    <div className="job-run-timeline">
      <h4 className="timeline-header">Run History</h4>
      <div className="timeline-list">
        {sortedRuns.map((run, index) => (
          <div key={run.id} className="timeline-item">
            <div className="timeline-connector">
              <div className={`timeline-dot ${run.status}`} />
              {index < sortedRuns.length - 1 && <div className="timeline-line" />}
            </div>
            
            <div className="timeline-content">
              <div className="run-header">
                <StatusIcon status={run.status} sloBreached={run.slo_breached} />
                <span className="run-time">
                  {new Date(run.created_at).toLocaleString()}
                </span>
                {run.slo_breached && (
                  <span className="slo-breach-badge">
                    <AlertTriangle size={12} /> SLO Breached
                  </span>
                )}
              </div>
              
              <div className="run-details">
                <span className="run-duration">
                  <Clock size={12} /> {run.duration_ms ? `${run.duration_ms}ms` : 'Running'}
                </span>
                {run.trigger_type && (
                  <span className="run-trigger">
                    via {run.trigger_type}
                  </span>
                )}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

const StatusIcon: React.FC<{ status: string; sloBreached: boolean }> = ({ status, sloBreached }) => {
  if (status === 'completed') {
    return sloBreached ? 
      <AlertTriangle className="status-icon warning" size={16} /> :
      <CheckCircle className="status-icon success" size={16} />;
  }
  if (status === 'failed') {
    return <XCircle className="status-icon error" size={16} />;
  }
  return <div className="status-icon pending" />;
};

export default JobRunTimeline;
