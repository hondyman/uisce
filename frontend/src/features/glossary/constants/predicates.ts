/**
 * Universal predicate registry for catalog edges.
 *
 * Every relationship in the catalog — across business terms, semantic terms,
 * business objects, columns, etc. — uses one of these predicates. The registry
 * gives every predicate a canonical color, arrow direction, icon, human-readable
 * label, and a one-line description. Components rendering edges anywhere in the
 * app should look up the predicate here so visuals stay consistent.
 */

export type PredicateDirection = 'outbound' | 'inbound' | 'bidirectional';

export interface PredicateMeta {
  /** Stable identifier used by the backend (edge_type_name / relationship_type) */
  key: string;
  /** Short uppercase label for the badge */
  label: string;
  /** Longer human-readable label for tooltips and headers */
  longLabel: string;
  /** Description shown in tooltips / inspector drawers */
  description: string;
  /** Whether the focal node is the source, target, or either */
  direction: PredicateDirection;
  /** Tailwind-style hex color */
  color: string;
  /** Emoji icon for the badge */
  icon: string;
  /** Which entity type pairs this predicate applies to (informational only;
   * rendering code can decide whether to show based on context) */
  appliesTo?: Array<'business_term' | 'semantic_term' | 'business_object' | 'calculation' | 'column' | 'table'>;
}

/**
 * Master list of every catalog-edge predicate we know about. Keep this in
 * sync with the backend's edge taxonomy (see backend/internal/api/glossary_handler.go
 * and BusinessObjectHandler edge-type definitions).
 */
export const PREDICATES: Record<string, PredicateMeta> = {
  // ─── Taxonomy / Inheritance ───────────────────────────────────────────
  IS_SPECIALIZATION_OF: {
    key: 'IS_SPECIALIZATION_OF',
    label: 'Specialization',
    longLabel: 'Is a specialization of',
    description: 'Target term is a specialized subtype of this concept (e.g. Allocation Account Code is a subtype of Account Code).',
    direction: 'outbound',
    color: '#6366F1',
    icon: '🔻',
    appliesTo: ['business_term', 'semantic_term'],
  },
  IS_GENERALIZATION_OF: {
    key: 'IS_GENERALIZATION_OF',
    label: 'Generalization',
    longLabel: 'Is a generalization of',
    description: 'Target term is a generalized supertype of this concept.',
    direction: 'outbound',
    color: '#6366F1',
    icon: '🔺',
    appliesTo: ['business_term', 'semantic_term'],
  },
  SEE_ALSO: {
    key: 'SEE_ALSO',
    label: 'See also',
    longLabel: 'See also',
    description: 'Terms are loosely related within the business domain.',
    direction: 'bidirectional',
    color: '#94A3B8',
    icon: '👀',
    appliesTo: ['business_term', 'semantic_term'],
  },
  RELATES_TO: {
    key: 'RELATES_TO',
    label: 'Relates to',
    longLabel: 'Relates to',
    description: 'Terms share an associative relationship within the business domain.',
    direction: 'bidirectional',
    color: '#94A3B8',
    icon: '🔗',
    appliesTo: ['business_term', 'semantic_term', 'business_object'],
  },

  // ─── Symbology / Identifiers ──────────────────────────────────────────
  IS_PEER_IDENTIFIER_OF: {
    key: 'IS_PEER_IDENTIFIER_OF',
    label: 'Peer identifier',
    longLabel: 'Is a peer identifier of',
    description: 'Target term is an alternate standard symbology or identifier for the same asset domain (e.g. CUSIP <-> ISIN <-> SEDOL).',
    direction: 'bidirectional',
    color: '#A78BFA',
    icon: '🔁',
    appliesTo: ['business_term', 'semantic_term'],
  },
  ALIAS_OF: {
    key: 'ALIAS_OF',
    label: 'Alias',
    longLabel: 'Is an alias of',
    description: 'Terms are different names for the same underlying concept.',
    direction: 'bidirectional',
    color: '#A78BFA',
    icon: '🏷️',
    appliesTo: ['business_term', 'semantic_term'],
  },
  HAS_SYNONYM: {
    key: 'HAS_SYNONYM',
    label: 'Synonym',
    longLabel: 'Has synonym',
    description: 'Target term is a synonym / linguistic variant of this term.',
    direction: 'bidirectional',
    color: '#A78BFA',
    icon: '🗣️',
    appliesTo: ['business_term', 'semantic_term'],
  },
  DIFFERENTIATED_FROM: {
    key: 'DIFFERENTIATED_FROM',
    label: 'Differentiated',
    longLabel: 'Differentiated from',
    description: 'Terms are easily conflated but serve distinct operational and accounting lifecycle stages.',
    direction: 'bidirectional',
    color: '#F59E0B',
    icon: '⚠️',
    appliesTo: ['business_term', 'semantic_term'],
  },

  // ─── Mapping / Realization ────────────────────────────────────────────
  MAPS_TO: {
    key: 'MAPS_TO',
    label: 'Maps to',
    longLabel: 'Maps to',
    description: 'Focal term is realized by the target semantic term (or vice versa).',
    direction: 'bidirectional',
    color: '#2DD4BF',
    icon: '🧠',
    appliesTo: ['business_term', 'semantic_term'],
  },
  HAS_BUSINESS_TERM: {
    key: 'HAS_BUSINESS_TERM',
    label: 'Has business term',
    longLabel: 'Has business term',
    description: 'Business term describes / labels this catalog entity (e.g. a business object field or column).',
    direction: 'inbound',
    color: '#10B981',
    icon: '💼',
    appliesTo: ['business_object', 'semantic_term', 'column', 'table'],
  },
  business_term_to_semantic_term: {
    key: 'business_term_to_semantic_term',
    label: 'Business Term → Semantic Term',
    longLabel: 'Business term to semantic term',
    description: 'Business term is implemented by / maps to a semantic term.',
    direction: 'outbound',
    color: '#2DD4BF',
    icon: '🧠',
    appliesTo: ['business_term', 'semantic_term'],
  },

  // ─── Computation / Dependency ─────────────────────────────────────────
  DEPENDS_ON: {
    key: 'DEPENDS_ON',
    label: 'Depends on',
    longLabel: 'Depends on',
    description: 'Focal term depends on the target calculation/derived term.',
    direction: 'outbound',
    color: '#38BDF8',
    icon: '🧮',
    appliesTo: ['calculation', 'semantic_term'],
  },
  USES_INPUT: {
    key: 'USES_INPUT',
    label: 'Uses input',
    longLabel: 'Uses input',
    description: 'Focal calculation consumes this column/semantic term as an input.',
    direction: 'outbound',
    color: '#38BDF8',
    icon: '⬇',
    appliesTo: ['calculation'],
  },
  TRANSFORMS_TO: {
    key: 'TRANSFORMS_TO',
    label: 'Transforms to',
    longLabel: 'Transforms to',
    description: 'Focal node is transformed into the target node by a pipeline step.',
    direction: 'outbound',
    color: '#0EA5E9',
    icon: '⚡',
    appliesTo: ['column', 'calculation', 'semantic_term'],
  },

  // ─── Business Object ───────────────────────────────────────────────────
  MEMBER_OF: {
    key: 'MEMBER_OF',
    label: 'Member of',
    longLabel: 'Member of',
    description: 'Focal node is a member/field of the target business object.',
    direction: 'outbound',
    color: '#A855F7',
    icon: '🏢',
    appliesTo: ['business_object', 'semantic_term', 'column'],
  },
  BO_RELATIONSHIP: {
    key: 'BO_RELATIONSHIP',
    label: 'BO relationship',
    longLabel: 'Business object relationship',
    description: 'A user-defined relationship between two business objects (e.g. Customer HAS Orders).',
    direction: 'bidirectional',
    color: '#A855F7',
    icon: '🏢',
    appliesTo: ['business_object'],
  },
  HAS_FIELD: {
    key: 'HAS_FIELD',
    label: 'Has field',
    longLabel: 'Has field',
    description: 'Business object has the target field.',
    direction: 'outbound',
    color: '#A855F7',
    icon: '🧩',
    appliesTo: ['business_object'],
  },

  // ─── Physical / Column ──────────────────────────────────────────────────
  HAS_CONTEXT: {
    key: 'HAS_CONTEXT',
    label: 'Has context',
    longLabel: 'Has context',
    description: 'Column has contextual metadata associated with a semantic term.',
    direction: 'bidirectional',
    color: '#60A5FA',
    icon: '🏷️',
    appliesTo: ['column'],
  },
  REALIZED_BY: {
    key: 'REALIZED_BY',
    label: 'Realized by',
    longLabel: 'Realized by',
    description: 'Logical concept is realized by a physical column/table.',
    direction: 'outbound',
    color: '#60A5FA',
    icon: '🗄️',
    appliesTo: ['semantic_term', 'column'],
  },

  // ─── 3-Tier Taxonomy ──────────────────────────────────────────────────
  BELONGS_TO_DOMAIN: {
    key: 'BELONGS_TO_DOMAIN',
    label: 'Domain',
    longLabel: 'Belongs to domain',
    description: 'Term belongs to a Tier-1 data domain.',
    direction: 'outbound',
    color: '#1E40AF',
    icon: '🏛️',
    appliesTo: ['business_term', 'semantic_term', 'business_object'],
  },
  BELONGS_TO_CATEGORY: {
    key: 'BELONGS_TO_CATEGORY',
    label: 'Category',
    longLabel: 'Belongs to category',
    description: 'Term belongs to a Tier-2 data category.',
    direction: 'outbound',
    color: '#2563EB',
    icon: '📂',
    appliesTo: ['business_term', 'semantic_term', 'business_object'],
  },
  BELONGS_TO_L3: {
    key: 'BELONGS_TO_L3',
    label: 'Tier-3 classification',
    longLabel: 'Belongs to Tier-3 classification',
    description: 'Business term connects directly to a Tier-3 taxonomy classification node.',
    direction: 'outbound',
    color: '#3B82F6',
    icon: '🎯',
    appliesTo: ['business_term', 'semantic_term', 'business_object'],
  },
};

/** Look up a predicate by key, falling back to a generic meta. */
export function getPredicate(key: string | null | undefined): PredicateMeta {
  if (!key) return FALLBACK_PREDICATE;
  return PREDICATES[key] ?? { ...FALLBACK_PREDICATE, key };
}

/** Default predicate meta used when the edge's type doesn't match a known key. */
export const FALLBACK_PREDICATE: PredicateMeta = {
  key: 'unknown',
  label: 'Relation',
  longLabel: 'Relationship',
  description: 'A relationship between two catalog entities.',
  direction: 'bidirectional',
  color: '#64748B',
  icon: '🔗',
};
