import type { FC } from 'react';
import styles from './Gauge.module.css';

/**
 * Custom Gauge Component
 * Renders a circular gauge chart for visual representation of metrics
 */
interface GaugeProps {
  value: number;
  max?: number;
  color?: string;
  size?: 'small' | 'medium' | 'large';
  label?: string;
}

const Gauge: FC<GaugeProps> = ({ value, max = 100, color = '#00875A', size = 'medium', label }) => {
  const percentage = (value / max) * 100;
  const circumference = 282.6;
  const offset = circumference - (circumference * percentage) / 100;

  // size impacts CSS class only; numeric mapping unused in current implementation

  return (
    <div className="flex flex-col items-center">
      {label && <p className="text-sm font-medium text-slate-600 dark:text-slate-400 mb-2">{label}</p>}
      <div className={`${styles.gaugeContainer} ${styles[`gauge-${size}`]} relative`}>
        <svg
          className={`${styles.gaugeRotate} w-full h-full`}
          viewBox="0 0 100 100"
        >
          <circle cx="50" cy="50" r="45" fill="none" strokeWidth="10" stroke="#f0f0f0" />
          <circle
            cx="50"
            cy="50"
            r="45"
            fill="none"
            strokeWidth="10"
            stroke={color}
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            strokeLinecap="round"
          />
        </svg>
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="text-lg font-bold text-slate-900 dark:text-white">{value.toFixed(2)}</span>
        </div>
      </div>
    </div>
  );
};

export default Gauge;
