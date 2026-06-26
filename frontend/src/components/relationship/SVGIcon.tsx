import type { FC, SVGProps } from 'react';

interface SVGIconProps {
  name: string;
  className?: string;
  ariaLabel?: string;
}
const SVGIcon: FC<SVGIconProps> = ({ name, className = '', ariaLabel }) => {
  // Ensure SVGs inherit currentColor for fill and stroke and allow Tailwind classes to control color/size.
  const composedClass = `${className || ''} fill-current stroke-current`.trim();

  const common: SVGProps<SVGSVGElement> & { role?: string; 'aria-label'?: string } = {
    className: composedClass,
    role: ariaLabel ? 'img' : 'presentation',
    'aria-label': ariaLabel || undefined,
    viewBox: '0 0 24 24',
    xmlns: 'http://www.w3.org/2000/svg',
  };

  switch (name) {
    case 'more_horiz':
      return (
        <svg {...common} fill="currentColor">
          <circle cx="6" cy="12" r="1.5" />
          <circle cx="12" cy="12" r="1.5" />
          <circle cx="18" cy="12" r="1.5" />
        </svg>
      );
    case 'grid_view':
      return (
        <svg {...common} fill="currentColor">
          <rect x="3" y="3" width="8" height="8" />
          <rect x="13" y="3" width="8" height="8" />
          <rect x="3" y="13" width="8" height="8" />
          <rect x="13" y="13" width="8" height="8" />
        </svg>
      );
    case 'schema':
      return (
        <svg {...common} stroke="currentColor" strokeWidth={1.5} fill="none">
          <circle cx="12" cy="6" r="2" />
          <circle cx="6" cy="18" r="2" />
          <circle cx="18" cy="18" r="2" />
          <path d="M12 8v6" stroke="currentColor" strokeWidth={1.5} />
          <path d="M12 14l4 4" stroke="currentColor" strokeWidth={1.5} />
          <path d="M12 14l-4 4" stroke="currentColor" strokeWidth={1.5} />
        </svg>
      );
    case 'ios_share':
      return (
        <svg {...common} fill="none" stroke="currentColor" strokeWidth={1.5}>
          <path d="M12 2v13" stroke="currentColor" />
          <path d="M5 9l7-7 7 7" stroke="currentColor" />
          <rect x="4" y="15" width="16" height="6" rx="2" stroke="currentColor" />
        </svg>
      );
    case 'search':
      return (
        <svg {...common} stroke="currentColor" strokeWidth={1.5} fill="none">
          <circle cx="11" cy="11" r="6" />
          <path d="M21 21l-4.35-4.35" stroke="currentColor" />
        </svg>
      );
    case 'refresh':
      return (
        <svg {...common} stroke="currentColor" strokeWidth={1.5} fill="none">
          <path d="M21 12a9 9 0 11-3.19-6.6" stroke="currentColor" />
          <path d="M21 3v6h-6" stroke="currentColor" />
        </svg>
      );
    case 'error':
      return (
        <svg {...common} fill="currentColor">
          <path d="M11.001 2.0001c.69 0 1.32.28 1.77.73l7.5 7.5c.95.95.95 2.49 0 3.44l-7.5 7.5c-.45.45-1.08.73-1.77.73-.69 0-1.32-.28-1.77-.73l-7.5-7.5c-.95-.95-.95-2.49 0-3.44l7.5-7.5c.45-.45 1.08-.73 1.77-.73z" />
        </svg>
      );
    case 'person':
      return (
        <svg {...common} fill="currentColor">
          <circle cx="12" cy="8" r="3" />
          <path d="M5 20c1.5-4 5.5-6 7-6s5.5 2 7 6" />
        </svg>
      );
    case 'school':
      return (
        <svg {...common} fill="currentColor">
          <path d="M12 2L2 7l10 5 10-5-10-5z" />
          <path d="M2 17l10 5 10-5" />
        </svg>
      );
    case 'corporate_fare':
      return (
        <svg {...common} fill="currentColor">
          <path d="M3 13h18v8H3z" />
          <path d="M7 13V6l5-3 5 3v7" />
        </svg>
      );
    case 'verified':
      return (
        <svg {...common} fill="currentColor">
          <path d="M12 2l1.9 3.9L18 7l-2.1 2.1L16 15l-4-2-4 2 .1-5.9L4 7l4.1-.1L12 2z" />
        </svg>
      );
    case 'link':
      return (
        <svg {...common} viewBox="0 0 24 24" fill="currentColor">
          <path d="M9.5 7c0-1.933 1.567-3.5 3.5-3.5s3.5 1.567 3.5 3.5c0 1.566-1.036 2.889-2.45 3.37.276.523.45 1.104.45 1.73 0 2.485-2.015 4.5-4.5 4.5s-4.5-2.015-4.5-4.5c0-1.25.506-2.386 1.324-3.214-.05-.31-.074-.628-.074-.956V7zm.5 6.5c.828 0 1.5-.672 1.5-1.5s-.672-1.5-1.5-1.5-1.5.672-1.5 1.5.672 1.5 1.5 1.5zm6-3c.828 0 1.5-.672 1.5-1.5S16.828 7.5 16 7.5s-1.5.672-1.5 1.5.672 1.5 1.5 1.5z" />
        </svg>
      );
    case 'link_off':
      return (
        <svg {...common} viewBox="0 0 24 24" fill="currentColor">
          <path d="M17 8.5c-.828 0-1.5.672-1.5 1.5s.672 1.5 1.5 1.5 1.5-.672 1.5-1.5-.672-1.5-1.5-1.5zM3.584 2.515l1.407 1.407C3.734 5.075 3 6.404 3 7.88c0 2.59 2.1 4.7 4.69 4.7 1.476 0 2.804-.734 3.958-1.991l1.407 1.407C11.695 18.283 10.25 19 8.69 19 5.417 19 3 16.583 3 13.31c0-1.56.717-3.005 1.867-3.98L3 8.19l.584-1.268 16.416 16.416 1.064-1.064L4.648 3.58l-.584-.585z" />
        </svg>
      );
    case 'check_circle':
      return (
        <svg {...common} viewBox="0 0 24 24" fill="currentColor">
          <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z" />
        </svg>
      );
    case 'close':
      return (
        <svg {...common} viewBox="0 0 24 24" fill="currentColor">
          <path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12 19 6.41z" />
        </svg>
      );
    case 'arrow_forward':
      return (
        <svg {...common} stroke="currentColor" strokeWidth={1.5} fill="none">
          <path d="M5 12h14" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
          <path d="M12 5l7 7-7 7" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      );
    case 'download':
      return (
        <svg {...common} stroke="currentColor" strokeWidth={1.5} fill="none">
          <path d="M12 3v12" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
          <path d="M8 11l4 4 4-4" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
          <path d="M4 21h16" stroke="currentColor" strokeWidth={1.5} strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      );
    case 'zoom_in':
      return (
        <svg {...common} stroke="currentColor" strokeWidth={1.5} fill="none">
          <circle cx="11" cy="11" r="6" stroke="currentColor" />
          <path d="M21 21l-4.35-4.35" stroke="currentColor" />
          <path d="M11 8v6" stroke="currentColor" strokeLinecap="round" />
          <path d="M8 11h6" stroke="currentColor" strokeLinecap="round" />
        </svg>
      );
    case 'zoom_out':
      return (
        <svg {...common} stroke="currentColor" strokeWidth={1.5} fill="none">
          <circle cx="11" cy="11" r="6" stroke="currentColor" />
          <path d="M21 21l-4.35-4.35" stroke="currentColor" />
          <path d="M8 11h6" stroke="currentColor" strokeLinecap="round" />
        </svg>
      );
    case 'fit_screen':
      return (
        <svg {...common} stroke="currentColor" strokeWidth={1.5} fill="none">
          <rect x="4" y="4" width="16" height="16" rx="1" stroke="currentColor" />
          <path d="M8 4v4M16 4v4M8 20v-4M16 20v-4" stroke="currentColor" strokeLinecap="round" />
        </svg>
      );
    default:
      return (
        <svg {...common} viewBox="0 0 24 24" fill="currentColor">
          <path d="M9.5 7c0-1.933 1.567-3.5 3.5-3.5s3.5 1.567 3.5 3.5c0 1.566-1.036 2.889-2.45 3.37.276.523.45 1.104.45 1.73 0 2.485-2.015 4.5-4.5 4.5s-4.5-2.015-4.5-4.5c0-1.25.506-2.386 1.324-3.214-.05-.31-.074-.628-.074-.956V7zm.5 6.5c.828 0 1.5-.672 1.5-1.5s-.672-1.5-1.5-1.5-1.5.672-1.5 1.5.672 1.5 1.5 1.5zm6-3c.828 0 1.5-.672 1.5-1.5S16.828 7.5 16 7.5s-1.5.672-1.5 1.5.672 1.5 1.5 1.5z" />
        </svg>
      );
  }
};

export default SVGIcon;
