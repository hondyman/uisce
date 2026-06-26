import { FC } from 'react';
import { devLog } from '../../utils/devLogger';

export interface ExportButtonProps {
  onClick: () => void;
  disabled?: boolean;
}

export const ExportButton: FC<ExportButtonProps> = ({
  onClick,
  disabled = false
}) => {
  const tooltipText = disabled ? 'No data to export' : 'Export';

  const handleClick = () => {
    if (!disabled) {
  devLog('Export button clicked'); // For debugging
      onClick();
    }
  };

  return (
    <div className="export-button-root group">
      <button
        onClick={handleClick}
        disabled={disabled}
        className={`export-button ${disabled ? 'disabled' : 'enabled'}`}
        aria-label={tooltipText}
        title={tooltipText}
      >
        <svg className="export-button-icon" viewBox="0 0 24 24">
          <path d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4" />
        </svg>
      </button>

      <div className="export-tooltip" role="tooltip">
        {tooltipText}
        <div className="export-tooltip-arrow" />
      </div>
    </div>
  );
};