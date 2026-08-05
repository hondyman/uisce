import React from 'react';
import { Link } from 'react-router-dom';

export type UisceLogoVariant = 'full' | 'mark' | 'wordmark';
export type UisceLogoSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl';

const SIZE_MAP: Record<UisceLogoSize, { drop: number; wordmark: string }> = {
  xs:  { drop: 20, wordmark: '14px' },
  sm:  { drop: 28, wordmark: '18px' },
  md:  { drop: 36, wordmark: '22px' },
  lg:  { drop: 48, wordmark: '28px' },
  xl:  { drop: 72, wordmark: '42px' },
};

interface UisceLogoProps {
  variant?: UisceLogoVariant;
  size?: UisceLogoSize;
  animated?: boolean;
  asLink?: boolean;
  className?: string;
}

export function UisceLogo({
  variant = 'full',
  size = 'md',
  animated = false,
  asLink = false,
  className,
}: UisceLogoProps): JSX.Element {
  const { drop: dropSize, wordmark: wordmarkSize } = SIZE_MAP[size];

  const drop = (
    <svg
      width={dropSize}
      height={dropSize * (150 / 120)}
      viewBox="0 0 120 150"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={`uisce-logo-drop${animated ? ' uisce-logo-drop--animated' : ''}${className ? ` ${className}` : ''}`}
      aria-label="Ishka"
    >
      <defs>
        <linearGradient id="uisce-drop-grad" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%"   stopColor="#0D1B6E" />
          <stop offset="50%"  stopColor="#1565C0" />
          <stop offset="100%" stopColor="#00BCD4" />
        </linearGradient>
        <linearGradient id="uisce-chart-grad" x1="0%" y1="100%" x2="100%" y2="0%">
          <stop offset="0%"   stopColor="#00C9C8" stopOpacity="0.7" />
          <stop offset="100%" stopColor="#ffffff"  stopOpacity="0.95" />
        </linearGradient>
        <clipPath id="uisce-drop-clip">
          <path d="M60 5 C60 5 10 65 10 92 C10 120 33 140 60 140 C87 140 110 120 110 92 C110 65 60 5 60 5 Z" />
        </clipPath>
      </defs>

      <path
        d="M60 5 C60 5 10 65 10 92 C10 120 33 140 60 140 C87 140 110 120 110 92 C110 65 60 5 60 5 Z"
        fill="url(#uisce-drop-grad)"
      />

      <path
        d="M60 5 C60 5 10 65 10 92 C10 120 33 140 60 140 C87 140 110 120 110 92 C110 65 60 5 60 5 Z"
        fill="none"
        stroke="rgba(255,255,255,0.18)"
        strokeWidth="1.5"
      />

      <g clipPath="url(#uisce-drop-clip)">
        <rect x="24" y="102" width="14" height="28" fill="url(#uisce-chart-grad)" opacity="0.85" rx="2" />
        <rect x="42" y="90"  width="14" height="40" fill="url(#uisce-chart-grad)" opacity="0.9"  rx="2" />
        <rect x="60" y="78"  width="14" height="52" fill="url(#uisce-chart-grad)" opacity="0.95" rx="2" />
        <rect x="78" y="65"  width="14" height="65" fill="url(#uisce-chart-grad)"                rx="2" />
        <polyline
          points="31,108 49,94 67,80 85,60"
          stroke="white"
          strokeWidth="3"
          strokeLinecap="round"
          strokeLinejoin="round"
          fill="none"
          opacity="0.95"
        />
        <polyline
          points="78,52 88,60 80,68"
          stroke="white"
          strokeWidth="3"
          strokeLinecap="round"
          strokeLinejoin="round"
          fill="none"
          opacity="0.95"
        />
      </g>

      <ellipse cx="42" cy="65" rx="10" ry="16" fill="rgba(255,255,255,0.10)" />
    </svg>
  );

  const wordmark = (
    <span
      className={`uisce-logo-wordmark${animated ? ' uisce-logo-wordmark--animated' : ''}`}
      style={{ fontSize: wordmarkSize, fontWeight: 800, letterSpacing: '-0.02em', lineHeight: 1 }}
    >
      <span style={{ color: '#00c9c8' }}>i</span>shka
    </span>
  );

  if (variant === 'mark') {
    const el = drop;
    if (asLink) {
      return <Link to="/" style={{ display: 'inline-flex', alignItems: 'center' }}>{el}</Link>;
    }
    return el;
  }

  if (variant === 'wordmark') {
    const el = wordmark;
    if (asLink) {
      return <Link to="/" style={{ display: 'inline-flex', alignItems: 'center' }}>{el}</Link>;
    }
    return el;
  }

  // 'full' — drop + wordmark side by side
  const el = (
    <span
      className="uisce-logo-full"
      style={{ display: 'inline-flex', alignItems: 'center', gap: '0.4em' }}
    >
      {drop}
      {wordmark}
    </span>
  );

  if (asLink) {
    return <Link to="/" style={{ display: 'inline-flex', alignItems: 'center' }}>{el}</Link>;
  }
  return el;
}
