import type { FC, MouseEvent } from 'react';
import LinkIcon from '@mui/icons-material/Link';
import LinkOffIcon from '@mui/icons-material/LinkOff';
import ActionButton from '../ui/ActionButton';

type Variant = 'link' | 'unlink' | 'status';

interface Props {
  variant: Variant;
  label?: string;
  ariaLabel?: string;
  pending?: boolean;
  onClick?: (e: MouseEvent<HTMLButtonElement>) => void;
}

const RelationshipActionButton: FC<Props> = ({ variant, label, ariaLabel, pending, onClick }) => {
  if (variant === 'status') {
    // status chip (Linked) - already rendered in RelationshipCard
    return null;
  }

  // Map local variants to ActionButton variants and icon components
  const map = {
    link: { variant: 'primary' as const, Icon: LinkIcon },
    unlink: { variant: 'danger' as const, Icon: LinkOffIcon },
  };

  const { Icon, ...mapped } = map[variant];

  // Wrap onClick to stop propagation so clicks inside lists don't trigger parent handlers
  const handleClick = (e: MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    onClick && onClick(e);
  };

  return (
    <ActionButton
      variant={mapped.variant}
      size="sm"
      pending={pending}
      onClick={handleClick}
      title={ariaLabel || (variant === 'unlink' ? 'Unlink relationship' : 'Link relationship')}
      aria-label={ariaLabel || (variant === 'unlink' ? 'Unlink relationship' : 'Link relationship')}
      className="inline-flex items-center"
    >
      <Icon className="h-4 w-4 mr-2" />
      {label || (variant === 'unlink' ? 'Unlink' : 'Link')}
    </ActionButton>
  );
};

export default RelationshipActionButton;
