import type React from 'react';

/**
 * Domain components: hand-built, domain-owned React components (a golden
 * record's provenance drawer, a side-by-side source matrix, a run wizard)
 * made placeable in Page Studio with a declared contract. This is how the
 * studio reaches hand-built quality without flattening rich domain UI into
 * generic widgets: the page composes them, binds their inputs, and handles
 * their events with ordinary page actions.
 */

export interface DomainComponentInput {
  name: string;
  label?: string;
  type: 'string' | 'number' | 'boolean' | 'object';
  required?: boolean;
  description?: string;
}

export interface DomainComponentEvent {
  name: string;
  label?: string;
  /** What the event's {{event.*}} payload carries. */
  payload?: string[];
  description?: string;
}

export interface DomainComponentDef {
  /** e.g. mastering.RunDialog */
  id: string;
  domain: string;
  label: string;
  description?: string;
  /** Overlays (drawers, dialogs) render nothing inline; design mode shows a chip for them. */
  overlay?: boolean;
  inputs: DomainComponentInput[];
  events: DomainComponentEvent[];
  /**
   * Receives resolved inputs and one callback per declared event; calling a
   * callback runs the page's actions for that event with {{event}} = payload.
   */
  render: React.ComponentType<{ inputs: Record<string, unknown>; emit: (event: string, payload?: Record<string, unknown>) => void }>;
}

const components = new Map<string, DomainComponentDef>();

export function registerDomainComponents(defs: DomainComponentDef[]): void {
  for (const d of defs) components.set(d.id, d);
}

export function getDomainComponent(id: string): DomainComponentDef | undefined {
  return components.get(id);
}

export function listDomainComponents(): DomainComponentDef[] {
  return [...components.values()].sort((a, b) => a.id.localeCompare(b.id));
}
