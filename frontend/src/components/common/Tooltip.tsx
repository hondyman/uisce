// React default import removed — using automatic JSX runtime
import './Tooltip.css';

interface TooltipProps {
  children: React.ReactNode;
  content: React.ReactNode;
  placement?: 'top' | 'bottom' | 'left' | 'right';
  className?: string;
}

const Tooltip: React.FC<TooltipProps> = ({ children, content, placement = 'top', className = '' }) => {
  return (
    <span className={`tooltip-wrapper ${className} tooltip-${placement}`}>
      {children}
      <span className="tooltip-content" role="tooltip">{content}</span>
    </span>
  );
};

export default Tooltip;
