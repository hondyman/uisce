import type React from 'react';
import SVGIcon from '../../components/relationship/SVGIcon';

type Variant = 'primary' | 'secondary' | 'ghost' | 'warning' | 'danger' | 'success' | 'neutral';

interface Props extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: 'sm' | 'md';
  pending?: boolean;
  iconName?: string;
  iconOnly?: boolean;
  iconPosition?: 'left' | 'right';
}

const variantClass = (v?: Variant) => {
  switch (v) {
    case 'primary':
      return 'btn-primary';
    case 'secondary':
      return 'btn-secondary';
    case 'ghost':
      return 'btn-ghost';
    case 'warning':
      return 'btn-warning';
    case 'danger':
      return 'btn-danger';
    case 'success':
      return 'btn-success';
    default:
      return '';
  }
};

const ActionButton: React.FC<Props> = ({
  variant = 'primary',
  size = 'md',
  pending,
  iconName,
  iconOnly = false,
  iconPosition = 'left',
  children,
  className = '',
  disabled,
  ...rest
}) => {
  const sizeClass = size === 'sm' ? 'btn-sm' : '';
  const base = `btn ${variantClass(variant)} ${sizeClass} ${className}`.trim();

  const renderIcon = () => {
    if (pending) {
      return (
        <svg className={`${iconOnly ? '' : 'mr-2'} animate-spin h-4 w-4`} xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"></path>
        </svg>
      );
    }
    if (iconName) {
      return <SVGIcon name={iconName} className={`${iconOnly ? '' : 'inline-block mr-2 h-4 w-4'}`} ariaLabel={iconName} />;
    }
    return null;
  };

  return (
    <button
      className={base}
      disabled={disabled || pending}
      {...rest}
    >
      {iconPosition === 'left' && (iconOnly ? renderIcon() : renderIcon())}
      {!iconOnly && <span>{children}</span>}
      {iconPosition === 'right' && !iconOnly && renderIcon()}
    </button>
  );
};

export default ActionButton;
