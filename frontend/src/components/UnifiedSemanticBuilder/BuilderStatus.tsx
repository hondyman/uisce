// React default import removed — using automatic JSX runtime
import * as TablerIcons from '@tabler/icons-react';

export const LoadingView: React.FC<{message?:string}> = ({ message }) => (
  <div className="builder-container">
    <div className="builder-loading">
      <div className="loading-spinner">
        <TablerIcons.IconSettings size={24} className="spinner-icon" />
      </div>
  <h3>{message || 'Loading Model Builder'}</h3>
      <p>Preparing your workspace...</p>
    </div>
  </div>
);

export const ErrorView: React.FC<{error?: string | null; onClose?: () => void}> = ({ error, onClose }) => (
  <div className="builder-container">
    <div className="builder-error">
      <div className="error-content">
  <h3>Unable to Load Model Builder</h3>
        <p>Error: {error || 'Unknown error'}</p>
        {onClose && (
          <button onClick={onClose} className="btn btn-secondary">Close Model Builder</button>
        )}
      </div>
    </div>
  </div>
);

export default LoadingView;
