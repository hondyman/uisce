// lib/presentationPolicy.ts

export type ContainerKind = 'modal' | 'panel';
export type SectionType = 'fields' | 'related_list' | 'custom';

export interface ContainerChoiceArgs {
  sectionType: SectionType;
  estimatedRows: number;
  isMobile: boolean;
  role?: string;
  fieldCount?: number;
}

/**
 * chooseContainer: Deterministic policy for selecting modal vs side panel.
 * 
 * Rules (in priority order):
 * 1. Mobile devices always use panel (better UX on small screens)
 * 2. Related lists always use panel (typically long scrollable content)
 * 3. Content > 10 rows → panel (avoids scroll within modal)
 * 4. Default → modal (fits most field edit workflows)
 * 
 * This policy is logged for later optimization and A/B testing.
 */
export function chooseContainer(args: ContainerChoiceArgs): ContainerKind {
  // Mobile first: panels are more touch-friendly
  if (args.isMobile) {
    return 'panel';
  }

  // Related lists are inherently long scrolling content
  if (args.sectionType === 'related_list') {
    return 'panel';
  }

  // Large field sets benefit from dedicated panel
  if (args.estimatedRows > 10) {
    return 'panel';
  }

  // Default to modal for most field edits
  return 'modal';
}

/**
 * logOutcome: Send presentation decision to backend for analytics.
 * Uses sendBeacon (non-blocking) to avoid impacting UI performance.
 */
import { devDebug } from '../utils/devLogger';

export function logOutcome(
  eventType: string,
  data: Record<string, unknown>,
): void {
  try {
    const payload = {
      eventType,
      ...data,
      ts: Date.now(),
    };
    navigator.sendBeacon(
      '/api/analytics/layout',
      new Blob([JSON.stringify(payload)], { type: 'application/json' }),
    );
  } catch (err) {
    // Silently fail - analytics should never break the UI
    devDebug('[analytics] beacon failed', err);
  }
}

/**
 * Example: Using chooseContainer in a layout editor
 * 
 * ```tsx
 * const containerKind = chooseContainer({
 *   sectionType: 'fields',
 *   estimatedRows: fieldIds.length,
 *   isMobile: window.innerWidth < 768,
 * });
 * 
 * logOutcome('container_decision', {
 *   sectionId: section.id,
 *   containerKind,
 *   estimatedRows: fieldIds.length,
 *   device: window.innerWidth < 768 ? 'mobile' : 'desktop',
 * });
 * ```
 */
