import React from 'react';
import { Link as MuiLink } from '@mui/material';
import { useNavigate } from 'react-router-dom';
import { ComponentDefinition } from '../../types/pageStudio';

interface HyperlinkWidgetProps {
  component: ComponentDefinition;
}

/**
 * A page-to-page or external link. `props.targetPageSlug` wins over
 * `props.href` when both are set - internal navigation goes through
 * react-router (no full reload, keeps the app's auth/session state)
 * while an external href falls through to a plain anchor.
 */
const HyperlinkWidget: React.FC<HyperlinkWidgetProps> = ({ component }) => {
  const navigate = useNavigate();
  const label = (component.props?.label as string) || 'Link';
  const targetPageSlug = component.props?.targetPageSlug as string | undefined;
  const href = component.props?.href as string | undefined;

  if (targetPageSlug) {
    return (
      <MuiLink component="button" underline="hover" onClick={() => navigate(`/pages/${targetPageSlug}`)}>
        {label}
      </MuiLink>
    );
  }

  return (
    <MuiLink href={href || '#'} target={href ? '_blank' : undefined} rel="noopener noreferrer" underline="hover">
      {label}
    </MuiLink>
  );
};

export default HyperlinkWidget;
