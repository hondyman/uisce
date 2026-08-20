import type {
  CatalogNodeItem,
  MatchSuggestion,
  EdgeCardinality,
  RelatedItemCandidate,
  TargetNodeDraft,
  EdgePropertiesDraft,
  UniversalValueType,
  CompositeCluster,
  CompositeClusterMember,
  HierarchicalEdgeDraft,
  GovernanceTier,
  VendorAlignment
} from '../types';
import type { EdgeType } from '../../types/edgeTypes';
import { isSuggestionRejected } from '../services/rejectionStore';
import { lookupBloombergField, VendorFieldDefinition } from '../constants/financialVendorDictionaries';

// Universal Value Types Registry (Standardized Semantic Archetypes)
export const UNIVERSAL_VALUE_TYPES: Record<string, UniversalValueType> = {
  Address: {
    name: 'Address',
    category: 'Geography & Spatial',
    standard: 'ISO 19160-1 / UPU S42',
    description: 'Physical or postal delivery location containing street, city, region, postal code, and country.',
    subProperties: ['Street', 'City', 'Region', 'PostalCode', 'Country'],
    validationRule: 'Postal address format',
    isPii: true,
  },
  PersonName: {
    name: 'PersonName',
    category: 'Party & Identity',
    standard: 'ISO/IEC 5218 / OASIS CIQ',
    description: 'Legal or preferred designation of an individual human being.',
    subProperties: ['FirstName', 'LastName', 'MiddleName', 'Title', 'Suffix'],
    isPii: true,
  },
  ContactCommunication: {
    name: 'ContactCommunication',
    category: 'Communication Channels',
    standard: 'ITU-T E.164 / RFC 5322',
    description: 'Electronic or telecommunication contact points.',
    subProperties: ['Phone', 'Fax', 'Mobile', 'Email', 'WebsiteUrl'],
    isPii: true,
  },
  FinancialAmount: {
    name: 'FinancialAmount',
    category: 'Financial Economics',
    standard: 'ISO 4217 / FIBO',
    description: 'Monetary quantity with decimal precision and currency code.',
    subProperties: ['Amount', 'CurrencyCode', 'ExchangeRate'],
    isPii: false,
  },
  AuditTimestamp: {
    name: 'AuditTimestamp',
    category: 'Governance & Provenance',
    standard: 'ISO 8601 / W3C PROV',
    description: 'Lifecycle audit tracking records creation and modification timestamps with actors.',
    subProperties: ['CreatedAt', 'CreatedBy', 'UpdatedAt', 'UpdatedBy'],
    isPii: false,
  }
};

export function resolveUniversalParent(columnName: string): UniversalValueType | null {
  const norm = normalizeName(columnName).toLowerCase();

  if (['address', 'addr', 'street', 'city', 'state', 'province', 'zip', 'zip_code', 'postal_code', 'country', 'region'].some(k => norm.includes(k))) {
    return UNIVERSAL_VALUE_TYPES.Address;
  }
  if (['first_name', 'last_name', 'middle_name', 'surname', 'given_name', 'contact_name', 'full_name'].some(k => norm.includes(k))) {
    return UNIVERSAL_VALUE_TYPES.PersonName;
  }
  if (['phone', 'telephone', 'fax', 'mobile', 'cell', 'email', 'website', 'url'].some(k => norm.includes(k))) {
    return UNIVERSAL_VALUE_TYPES.ContactCommunication;
  }
  if (['amount', 'amt', 'price', 'unit_price', 'fee', 'cost', 'currency', 'balance'].some(k => norm.includes(k))) {
    return UNIVERSAL_VALUE_TYPES.FinancialAmount;
  }
  if (['created_at', 'updated_at', 'created_by', 'updated_by', 'deleted_at', 'modified_at'].some(k => norm.includes(k))) {
    return UNIVERSAL_VALUE_TYPES.AuditTimestamp;
  }

  return null;
}

export function detectCompositeClusters(
  sourceNodes: CatalogNodeItem[]
): CompositeCluster[] {
  const tableMap = new Map<string, CatalogNodeItem[]>();

  for (const node of sourceNodes) {
    const entity = extractEntityFromSourceNode(node) || 'General';
    if (!tableMap.has(entity)) {
      tableMap.set(entity, []);
    }
    tableMap.get(entity)!.push(node);
  }

  const clusters: CompositeCluster[] = [];

  tableMap.forEach((nodes, entityName) => {
    const addressMembers: CompositeClusterMember[] = [];
    const nameMembers: CompositeClusterMember[] = [];
    const contactMembers: CompositeClusterMember[] = [];
    const financialMembers: CompositeClusterMember[] = [];

    for (const node of nodes) {
      const colNorm = normalizeName(node.node_name || '').toLowerCase();
      
      // Address matchers
      if (colNorm.includes('address') || colNorm.includes('street')) {
        addressMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'Street', suggestedTermName: `${entityName}Street`, confidence: 95 });
      } else if (colNorm.includes('city')) {
        addressMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'City', suggestedTermName: `${entityName}City`, confidence: 95 });
      } else if (colNorm.includes('region') || colNorm.includes('state') || colNorm.includes('province')) {
        addressMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'Region', suggestedTermName: `${entityName}Region`, confidence: 95 });
      } else if (colNorm.includes('postal') || colNorm.includes('zip')) {
        addressMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'PostalCode', suggestedTermName: `${entityName}PostalCode`, confidence: 95 });
      } else if (colNorm.includes('country')) {
        addressMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'Country', suggestedTermName: `${entityName}Country`, confidence: 95 });
      }

      // Name matchers
      if (colNorm.includes('first_name') || colNorm.includes('given_name')) {
        nameMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'FirstName', suggestedTermName: `${entityName}FirstName`, confidence: 95 });
      } else if (colNorm.includes('last_name') || colNorm.includes('surname')) {
        nameMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'LastName', suggestedTermName: `${entityName}LastName`, confidence: 95 });
      } else if (colNorm.includes('contact_name')) {
        nameMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'ContactName', suggestedTermName: `${entityName}ContactName`, confidence: 95 });
      }

      // Contact matchers
      if (colNorm.includes('phone') || colNorm.includes('telephone')) {
        contactMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'Phone', suggestedTermName: `${entityName}Phone`, confidence: 95 });
      } else if (colNorm.includes('fax')) {
        contactMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'Fax', suggestedTermName: `${entityName}Fax`, confidence: 95 });
      } else if (colNorm.includes('email')) {
        contactMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'Email', suggestedTermName: `${entityName}Email`, confidence: 95 });
      }

      // Financial matchers
      if (colNorm.includes('price') || colNorm.includes('amount') || colNorm.includes('cost')) {
        financialMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'Amount', suggestedTermName: `${entityName}Amount`, confidence: 95 });
      } else if (colNorm.includes('currency') || colNorm.includes('curr')) {
        financialMembers.push({ sourceColumn: node.node_name, sourceNodeId: node.id, subProperty: 'CurrencyCode', suggestedTermName: `${entityName}CurrencyCode`, confidence: 95 });
      }
    }

    const tableName = nodes[0]?.table || nodes[0]?.properties?.table_name || entityName.toLowerCase();

    if (addressMembers.length >= 2) {
      clusters.push({
        clusterId: `cluster-address-${entityName}`,
        clusterType: 'Address',
        entityName,
        tableName,
        universalParent: 'Address',
        standard: UNIVERSAL_VALUE_TYPES.Address.standard,
        compositeTermName: `${entityName}Address`,
        members: addressMembers,
      });
    }

    if (nameMembers.length >= 2) {
      clusters.push({
        clusterId: `cluster-name-${entityName}`,
        clusterType: 'PersonName',
        entityName,
        tableName,
        universalParent: 'PersonName',
        standard: UNIVERSAL_VALUE_TYPES.PersonName.standard,
        compositeTermName: `${entityName}Name`,
        members: nameMembers,
      });
    }

    if (contactMembers.length >= 2) {
      clusters.push({
        clusterId: `cluster-contact-${entityName}`,
        clusterType: 'ContactCommunication',
        entityName,
        tableName,
        universalParent: 'ContactCommunication',
        standard: UNIVERSAL_VALUE_TYPES.ContactCommunication.standard,
        compositeTermName: `${entityName}ContactInfo`,
        members: contactMembers,
      });
    }

    if (financialMembers.length >= 2) {
      clusters.push({
        clusterId: `cluster-fin-${entityName}`,
        clusterType: 'FinancialAmount',
        entityName,
        tableName,
        universalParent: 'FinancialAmount',
        standard: UNIVERSAL_VALUE_TYPES.FinancialAmount.standard,
        compositeTermName: `${entityName}MonetaryAmount`,
        members: financialMembers,
      });
    }
  });

  return clusters;
}

// Common business, enterprise & financial abbreviations
export const ABBREVIATIONS: Record<string, string> = {
  cust: 'customer',
  nbr: 'number',
  num: 'number',
  no: 'number',
  amt: 'amount',
  qty: 'quantity',
  dt: 'date',
  ts: 'timestamp',
  tm: 'time',
  acct: 'account',
  acc: 'account',
  addr: 'address',
  dob: 'date of birth',
  rev: 'revenue',
  desc: 'description',
  txt: 'text',
  str: 'string',
  curr: 'currency',
  ccy: 'currency',
  prod: 'product',
  prd: 'product',
  tx: 'transaction',
  txn: 'transaction',
  pos: 'position',
  bal: 'balance',
  sec: 'security',
  prc: 'price',
  px: 'price',
  vol: 'volume',
  comm: 'commission',
  pct: 'percentage',
  rate: 'rate',
  flg: 'flag',
  ind: 'indicator',
  cd: 'code',
  val: 'value',
  stat: 'status',
  sts: 'status',
  org: 'organization',
  dept: 'department',
  div: 'division',
  emp: 'employee',
  mgr: 'manager',
  id: 'identifier',
  cat: 'category',
  grp: 'group',
  src: 'source',
  tgt: 'target',
  dst: 'destination',
  cnt: 'count',
  tot: 'total',
  avg: 'average',
  min: 'minimum',
  max: 'maximum',
  std: 'standard',
  curr_val: 'current value',
  mkt_val: 'market value',
  nav: 'net asset value',
  fx: 'foreign exchange',
  pnl: 'profit and loss',
  gl: 'general ledger',
  ytd: 'year to date',
  mtd: 'month to date',
  qtd: 'quarter to date',
  isin: 'international securities identification number',
  cusip: 'committee on uniform securities identification procedures',
  sedol: 'stock exchange daily official list',
  figi: 'financial instrument global identifier',
  lei: 'legal entity identifier',
  duns: 'data universal numbering system',
  iban: 'international bank account number',
  bic: 'business identifier code',
  swift: 'society for worldwide interbank financial telecommunication',
};

// Domain & Identifier Clusters for "See Also" Discovery
export const DOMAIN_KNOWLEDGE_CLUSTERS: string[][] = [
  ['cusip', 'sedol', 'isin', 'figi', 'lei', 'ticker', 'ric', 'valor', 'security_id', 'instrument_id'],
  ['first_name', 'last_name', 'middle_name', 'full_name', 'preferred_name', 'given_name', 'surname'],
  ['street', 'street_address', 'address_line_1', 'address_line_2', 'city', 'state', 'province', 'zip', 'zip_code', 'postal_code', 'country'],
  ['ssn', 'ein', 'tin', 'tax_id', 'national_id', 'vat_number'],
  ['nav', 'aum', 'market_value', 'book_value', 'carrying_value', 'fair_value'],
  ['gross_revenue', 'net_revenue', 'operating_income', 'ebitda', 'net_income', 'profit_margin'],
  ['iban', 'bic', 'swift_code', 'routing_number', 'account_number'],
  ['email', 'email_address', 'phone', 'phone_number', 'mobile_number', 'fax'],
  ['order_id', 'order_date', 'order_status', 'shipping_address', 'billing_address', 'order_total'],
  ['invoice_id', 'invoice_date', 'due_date', 'payment_terms', 'invoice_amount', 'balance_due'],
];

// Hyper-Generic Column Tokens that require Contextual Table Disambiguation (e.g. vendor.address -> VendorAddress)
export const GENERIC_COLUMN_TOKENS = new Set([
  'address', 'addr', 'street', 'city', 'state', 'country', 'zip', 'zip_code', 'postal_code', 'region',
  'name', 'first_name', 'last_name', 'full_name', 'title', 'desc', 'description', 'comment', 'comments', 'notes', 'note', 'text', 'txt',
  'id', 'identifier', 'code', 'cd', 'num', 'number', 'nbr', 'no', 'key', 'ref', 'reference', 'seq', 'sequence',
  'status', 'stat', 'sts', 'state', 'stage', 'phase', 'type', 'cat', 'category', 'kind', 'class', 'flag', 'flg', 'ind', 'indicator',
  'date', 'dt', 'time', 'tm', 'timestamp', 'ts', 'created_at', 'updated_at', 'deleted_at', 'created_date', 'modified_date',
  'amount', 'amt', 'total', 'tot', 'balance', 'bal', 'price', 'prc', 'px', 'rate', 'cost', 'fee', 'val', 'value', 'qty', 'quantity', 'count', 'cnt',
  'phone', 'tel', 'fax', 'mobile', 'email', 'url', 'website', 'web',
  'is_active', 'active', 'enabled', 'deleted'
]);

export function isGenericColumn(name: string): boolean {
  if (!name) return false;
  const clean = name.toLowerCase().replace(/^[col_]+/, '').replace(/[^a-z0-9_]/g, '');
  if (GENERIC_COLUMN_TOKENS.has(clean)) return true;
  const tokens = tokenize(clean);
  if (tokens.length === 1 && GENERIC_COLUMN_TOKENS.has(tokens[0])) return true;
  return false;
}

export function singularize(word: string): string {
  if (!word) return '';
  const lower = word.toLowerCase();
  if (lower.endsWith('ies') && lower.length > 4) return lower.slice(0, -3) + 'y';
  if (lower.endsWith('ses') && lower.length > 4) return lower.slice(0, -2);
  if (lower.endsWith('s') && !lower.endsWith('ss') && lower.length > 3) return lower.slice(0, -1);
  return lower;
}

export function toPascalCase(str: string): string {
  if (!str) return '';
  return str
    .replace(/[._\-/\\]+/g, ' ')
    .split(/\s+/)
    .filter(Boolean)
    .map(w => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
    .join('');
}

export function extractEntityFromSourceNode(sourceNode: CatalogNodeItem): string {
  if (sourceNode.table) return toPascalCase(singularize(sourceNode.table));
  if (sourceNode.properties?.table_name) return toPascalCase(singularize(sourceNode.properties.table_name));
  if (sourceNode.properties?.table) return toPascalCase(singularize(sourceNode.properties.table));
  if (sourceNode.properties?.parent_name) return toPascalCase(singularize(sourceNode.properties.parent_name));
  if (sourceNode.parent_name) return toPascalCase(singularize(sourceNode.parent_name));

  const qPath = sourceNode.qualified_path || '';
  if (qPath) {
    const parts = qPath.split(/[./\\]+/).filter(Boolean);
    if (parts.length >= 2) {
      const tableCandidate = parts[parts.length - 2];
      if (tableCandidate && !['public', 'dbo', 'default', 'main', 'root'].includes(tableCandidate.toLowerCase())) {
        return toPascalCase(singularize(tableCandidate));
      }
    }
  }

  return '';
}

export function buildContextualTermName(entityName: string, columnName: string): string {
  const normCol = normalizeName(columnName);
  const pascalCol = toPascalCase(normCol) || toPascalCase(columnName);
  if (!entityName) return pascalCol;
  const pascalEntity = toPascalCase(entityName);
  
  if (pascalCol.toLowerCase().startsWith(pascalEntity.toLowerCase())) {
    return pascalCol;
  }
  return `${pascalEntity}${pascalCol}`;
}

const STRIPPABLE_PREFIXES = ['col_', 'tbl_', 'vw_', 'dim_', 'fact_', 'stg_', 'fct_', 'hub_', 'sat_', 'lnk_'];

export function tokenize(str: string): string[] {
  if (!str) return [];
  const cleaned = str
    .replace(/([a-z])([A-Z])/g, '$1 $2')
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1 $2')
    .replace(/[._\-/\\]+/g, ' ')
    .toLowerCase();
  return cleaned.split(/\s+/).filter(Boolean);
}

export function expandTokens(tokens: string[]): string[] {
  return tokens.map(t => ABBREVIATIONS[t] || t);
}

export function normalizeName(name: string): string {
  if (!name) return '';
  let lower = name.toLowerCase().trim();
  for (const prefix of STRIPPABLE_PREFIXES) {
    if (lower.startsWith(prefix)) {
      lower = lower.slice(prefix.length);
      break;
    }
  }
  const tokens = expandTokens(tokenize(lower));
  return tokens.join(' ');
}

export function levenshteinDistance(s1: string, s2: string): number {
  const m = s1.length;
  const n = s2.length;
  const dp: number[][] = Array.from({ length: m + 1 }, () => Array(n + 1).fill(0));
  for (let i = 0; i <= m; i++) dp[i][0] = i;
  for (let j = 0; j <= n; j++) dp[0][j] = j;

  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      const cost = s1[i - 1] === s2[j - 1] ? 0 : 1;
      dp[i][j] = Math.min(dp[i - 1][j] + 1, dp[i][j - 1] + 1, dp[i - 1][j - 1] + cost);
    }
  }
  return dp[m][n];
}

export function stringSimilarity(s1: string, s2: string): number {
  if (s1 === s2) return 1.0;
  if (!s1 || !s2) return 0;
  const maxLen = Math.max(s1.length, s2.length);
  if (maxLen === 0) return 1.0;
  return 1 - levenshteinDistance(s1, s2) / maxLen;
}

export function tokenJaccardSimilarity(tokens1: string[], tokens2: string[]): number {
  if (!tokens1.length || !tokens2.length) return 0;
  const set1 = new Set(tokens1);
  const set2 = new Set(tokens2);
  let intersectionCount = 0;
  set1.forEach(t => {
    if (set2.has(t)) intersectionCount++;
  });
  const unionCount = set1.size + set2.size - intersectionCount;
  return unionCount === 0 ? 0 : intersectionCount / unionCount;
}

/**
 * Resolves the cardinality of an EdgeType
 */
export function getEdgeTypeCardinality(edgeType: EdgeType | null): EdgeCardinality {
  if (!edgeType) return 'N:1';
  if (edgeType.config?.cardinality) {
    return edgeType.config.cardinality as EdgeCardinality;
  }
  const name = String(edgeType.edge_type_name || '').toLowerCase();
  if (name.includes('one_to_one') || name.includes('1_to_1')) return '1:1';
  if (name.includes('one_to_many') || name.includes('1_to_n')) return '1:N';
  if (name.includes('many_to_many') || name.includes('m_to_m') || name.includes('related') || name.includes('see_also')) return 'N:M';
  return 'N:1'; // Default (e.g. Many columns to One semantic term)
}

/**
 * Evaluates match between source node and target node
 */
export function evaluateMatch(
  sourceNode: CatalogNodeItem,
  targetNode: CatalogNodeItem
): MatchSuggestion | null {
  const sourceRaw = sourceNode.node_name || '';
  const targetRaw = targetNode.node_name || '';
  if (!sourceRaw || !targetRaw) return null;

  const sourceNorm = normalizeName(sourceRaw);
  const targetNorm = normalizeName(targetRaw);
  const sourceTokens = expandTokens(tokenize(sourceRaw));
  const targetTokens = expandTokens(tokenize(targetRaw));

  // 1. Exact Match
  if (sourceRaw.toLowerCase() === targetRaw.toLowerCase()) {
    return {
      targetNode,
      confidence: 100,
      matchReason: 'Exact name match',
      matchType: 'exact_normalized',
      edgeDraft: {
        transformation: 'direct',
        confidence: 100,
        mapping_notes: 'Exact match verified',
      }
    };
  }

  // 2. Normalized Exact Match
  if (sourceNorm === targetNorm && sourceNorm.length > 0) {
    return {
      targetNode,
      confidence: 96,
      matchReason: `Normalized exact match (${sourceNorm})`,
      matchType: 'exact_normalized',
      edgeDraft: {
        transformation: 'direct',
        confidence: 96,
        mapping_notes: `Normalized match on "${sourceNorm}"`,
      }
    };
  }

  // 3. Substring / Phrase Match
  if (sourceNorm.includes(targetNorm) || targetNorm.includes(sourceNorm)) {
    const longer = Math.max(sourceNorm.length, targetNorm.length);
    const shorter = Math.min(sourceNorm.length, targetNorm.length);
    const ratio = shorter / longer;
    if (ratio >= 0.55) {
      return {
        targetNode,
        confidence: Math.round(84 + ratio * 12),
        matchReason: `Direct phrase match (${sourceNorm} ↔ ${targetNorm})`,
        matchType: 'exact_normalized',
        edgeDraft: {
          transformation: 'direct',
          confidence: Math.round(84 + ratio * 12),
          mapping_notes: `Phrase match: ${sourceNorm} in ${targetNorm}`,
        }
      };
    }
  }

  // 4. Token Jaccard Overlap
  const jaccard = tokenJaccardSimilarity(sourceTokens, targetTokens);
  if (jaccard >= 0.45) {
    const sharedTokens = sourceTokens.filter(t => targetTokens.includes(t));
    return {
      targetNode,
      confidence: Math.round(75 + jaccard * 20),
      matchReason: `Shared keywords: [${sharedTokens.join(', ')}]`,
      matchType: 'token_overlap',
      edgeDraft: {
        transformation: 'direct',
        confidence: Math.round(75 + jaccard * 20),
        mapping_notes: `Keyword overlap: ${sharedTokens.join(', ')}`,
      }
    };
  }

  // 5. String Edit Distance (Fuzzy / Abbreviations)
  const similarity = stringSimilarity(sourceNorm, targetNorm);
  if (similarity >= 0.72) {
    return {
      targetNode,
      confidence: Math.round(similarity * 90),
      matchReason: `High fuzzy similarity (${Math.round(similarity * 100)}%)`,
      matchType: 'fuzzy',
      edgeDraft: {
        transformation: 'direct',
        confidence: Math.round(similarity * 90),
        mapping_notes: `Fuzzy similarity match (${Math.round(similarity * 100)}%)`,
      }
    };
  }

  // 6. Description Semantic Check
  const targetDesc = (targetNode.description || '').toLowerCase();
  if (targetDesc && sourceTokens.length > 0) {
    const matchedTokens = sourceTokens.filter(t => t.length > 2 && targetDesc.includes(t));
    if (matchedTokens.length >= Math.ceil(sourceTokens.length / 2) && matchedTokens.length > 0) {
      return {
        targetNode,
        confidence: Math.round(60 + (matchedTokens.length / sourceTokens.length) * 22),
        matchReason: `Description contains terms: [${matchedTokens.join(', ')}]`,
        matchType: 'description',
        edgeDraft: {
          transformation: 'direct',
          confidence: Math.round(60 + (matchedTokens.length / sourceTokens.length) * 22),
          mapping_notes: `Matched description keywords: ${matchedTokens.join(', ')}`,
        }
      };
    }
  }

  return null;
}

/**
 * Discover "See Also" / Related Items for a given source node across the catalog
 */
export function discoverRelatedItems(
  sourceNode: CatalogNodeItem,
  allCatalogNodes: CatalogNodeItem[],
  existingLinkedNodeIds: Set<string>
): RelatedItemCandidate[] {
  const sourceNameLower = (sourceNode.node_name || '').toLowerCase().replace(/[^a-z0-9]/g, '_');
  const sourceTokens = tokenize(sourceNode.node_name || '');
  const relatedCandidates: RelatedItemCandidate[] = [];
  const candidateIds = new Set<string>();

  // 1. Find clusters matching source name
  for (const cluster of DOMAIN_KNOWLEDGE_CLUSTERS) {
    const matchesCluster = cluster.some(term => sourceNameLower.includes(term) || sourceTokens.includes(term));
    if (matchesCluster) {
      // Find all other nodes in the catalog that match any term in this cluster
      for (const peerNode of allCatalogNodes) {
        if (peerNode.id === sourceNode.id) continue;
        const peerNameLower = (peerNode.node_name || '').toLowerCase().replace(/[^a-z0-9]/g, '_');
        const peerTokens = tokenize(peerNode.node_name || '');

        const isPeerInCluster = cluster.some(term => peerNameLower.includes(term) || peerTokens.includes(term));
        if (isPeerInCluster && !candidateIds.has(peerNode.id)) {
          candidateIds.add(peerNode.id);
          relatedCandidates.push({
            id: peerNode.id,
            node_name: peerNode.node_name,
            catalog_type: peerNode.catalog_type || peerNode.catalog_type_name,
            relation_type: 'see_also',
            description: peerNode.description || `Related ${peerNode.node_name}`,
            isAlreadyLinked: existingLinkedNodeIds.has(peerNode.id),
          });
        }
      }
    }
  }

  return relatedCandidates.slice(0, 5);
}

/**
 * Generate intelligent suggestions for a source node, filtering out rejections and respecting constraints
 */
export function generateSuggestionsForNode(
  sourceNode: CatalogNodeItem,
  targetNodes: CatalogNodeItem[],
  tenantId: string,
  minConfidence = 45,
  targetNodeTypeId?: string,
  targetNodeTypeName?: string
): { topSuggestion: MatchSuggestion | null; alternatives: MatchSuggestion[] } {
  const suggestions: MatchSuggestion[] = [];
  const targetType = targetNodeTypeName || 'semantic_term';
  const isGeneric = isGenericColumn(sourceNode.node_name || '');
  const entityName = extractEntityFromSourceNode(sourceNode);
  const contextualTermName = entityName ? buildContextualTermName(entityName, sourceNode.node_name || '') : '';

  for (const target of targetNodes) {
    // 1. Check if user previously rejected this suggestion
    if (isSuggestionRejected(tenantId, sourceNode.id, target.id, target.node_name)) {
      continue;
    }

    const match = evaluateMatch(sourceNode, target);
    if (match && match.confidence >= minConfidence) {
      if (isGeneric && entityName) {
        match.isGenericCollision = true;
        match.contextualEntity = entityName;
        match.suggestedContextualTerm = contextualTermName;
        // If matched an un-prefixed generic term (e.g. "Address" for "vendor.address"), add cautionary note
        const targetNorm = normalizeName(target.node_name || '');
        const sourceNorm = normalizeName(sourceNode.node_name || '');
        if (targetNorm === sourceNorm) {
          match.confidence = Math.min(match.confidence, 72);
          match.matchReason = `Generic Term ("${target.node_name}"): May cause multi-table collision across entities.`;
        }
      }
      suggestions.push(match);
    }
  }

  // ── Bloomberg & Financial Vendor Dictionary Alignment
  const bbgField = lookupBloombergField(sourceNode.node_name || '');
  if (bbgField) {
    const bbgVendorAlignment: VendorAlignment = {
      vendor: bbgField.vendor,
      mnemonic: bbgField.mnemonic,
      canonicalTermName: bbgField.canonicalTermName,
      category: bbgField.category,
      description: bbgField.description,
      feedType: bbgField.feedType,
    };

    const bbgHierarchicalEdges: HierarchicalEdgeDraft[] = [
      {
        edge_type_name: 'LICENSED_BY',
        target_node_name: `Bloomberg:${bbgField.mnemonic}`,
        target_catalog_type: 'vendor_field',
        properties: {
          vendor: 'BLOOMBERG',
          mnemonic: bbgField.mnemonic,
          category: bbgField.category,
          description: bbgField.description,
          feed_type: bbgField.feedType,
          data_type: bbgField.dataType,
        }
      }
    ];

    if (bbgField.universalArchetype) {
      bbgHierarchicalEdges.push({
        edge_type_name: 'SPECIALIZES',
        target_node_name: bbgField.universalArchetype,
        target_catalog_type: 'semantic_term',
        properties: {
          standard: UNIVERSAL_VALUE_TYPES[bbgField.universalArchetype]?.standard || 'ISO 4217 / FIBO',
          category: bbgField.category,
        }
      });
    }

    // Check if an existing target node matches canonical term name or mnemonic
    const existingBbgMatch = targetNodes.find(t => {
      const tNorm = normalizeName(t.node_name);
      return tNorm === normalizeName(bbgField.canonicalTermName) ||
             tNorm === normalizeName(bbgField.mnemonic) ||
             tNorm === normalizeName(sourceNode.node_name || '');
    });

    if (existingBbgMatch && !isSuggestionRejected(tenantId, sourceNode.id, existingBbgMatch.id, existingBbgMatch.node_name)) {
      suggestions.unshift({
        targetNode: existingBbgMatch,
        confidence: 99,
        matchReason: `Bloomberg Data Dictionary: Matched [${bbgField.mnemonic}] "${bbgField.description}". Links LICENSED_BY Bloomberg.`,
        matchType: 'vendor_aligned',
        universalParentName: bbgField.universalArchetype,
        universalStandard: bbgField.universalArchetype ? UNIVERSAL_VALUE_TYPES[bbgField.universalArchetype]?.standard : undefined,
        governanceTier: existingBbgMatch.type === 'core' ? 'gold_certified' : 'custom',
        hierarchicalEdgesToCreate: bbgHierarchicalEdges,
        vendorAlignment: bbgVendorAlignment,
        edgeDraft: {
          transformation: 'direct',
          confidence: 99,
          mapping_notes: `Bloomberg DL field alignment: ${bbgField.mnemonic} (${bbgField.category})`,
        }
      });
    } else if (!isSuggestionRejected(tenantId, sourceNode.id, `new-${bbgField.canonicalTermName}`, bbgField.canonicalTermName)) {
      // Propose NEW draft term aligned to Bloomberg
      const draftBbgNode: CatalogNodeItem = {
        id: `draft-${sourceNode.id}`,
        node_name: bbgField.canonicalTermName,
        description: `${bbgField.description} (Aligned to Bloomberg ${bbgField.mnemonic})`,
        catalog_type: targetType,
        catalog_type_name: targetType,
        properties: {
          data_type: bbgField.dataType,
          bloomberg_mnemonic: bbgField.mnemonic,
          category: bbgField.category,
          vendor: 'BLOOMBERG',
          feed_type: bbgField.feedType,
          standard: bbgField.universalArchetype ? UNIVERSAL_VALUE_TYPES[bbgField.universalArchetype]?.standard : 'Bloomberg DL',
          universal_parent: bbgField.universalArchetype,
          is_vendor_aligned: true,
          is_auto_suggested: true,
        }
      };

      suggestions.unshift({
        targetNode: draftBbgNode,
        targetDraft: {
          isNew: true,
          node_name: bbgField.canonicalTermName,
          description: bbgField.description,
          catalog_type: targetType,
          node_type_id: targetNodeTypeId || '',
          properties: {
            data_type: bbgField.dataType,
            bloomberg_mnemonic: bbgField.mnemonic,
            category: bbgField.category,
            vendor: 'BLOOMBERG',
            feed_type: bbgField.feedType,
            standard: bbgField.universalArchetype ? UNIVERSAL_VALUE_TYPES[bbgField.universalArchetype]?.standard : 'Bloomberg DL',
            universal_parent: bbgField.universalArchetype,
          }
        },
        confidence: 97,
        matchReason: `Bloomberg Data Dictionary: Column "${sourceNode.node_name}" resolves to Bloomberg field [${bbgField.mnemonic}] (${bbgField.category}). Links LICENSED_BY Bloomberg.`,
        matchType: 'vendor_aligned',
        universalParentName: bbgField.universalArchetype,
        universalStandard: bbgField.universalArchetype ? UNIVERSAL_VALUE_TYPES[bbgField.universalArchetype]?.standard : undefined,
        governanceTier: 'draft',
        hierarchicalEdgesToCreate: bbgHierarchicalEdges,
        vendorAlignment: bbgVendorAlignment,
        edgeDraft: {
          transformation: 'direct',
          confidence: 97,
          mapping_notes: `Bloomberg ${bbgField.mnemonic} (${bbgField.feedType || 'Data License'})`,
        }
      });
    }
  }

  // Universal Parent and Standard Resolution
  const universalParent = resolveUniversalParent(sourceNode.node_name || '');

  // If source is a generic column (e.g. vendor.address) and we have an entity name (e.g. Vendor),
  // propose the contextual entity-prefixed term (e.g. VendorAddress)
  if (isGeneric && entityName && contextualTermName) {
    const hierarchicalEdges: HierarchicalEdgeDraft[] = [];
    if (universalParent) {
      hierarchicalEdges.push({
        edge_type_name: 'SPECIALIZES',
        target_node_name: universalParent.name,
        target_catalog_type: 'semantic_term',
        properties: {
          standard: universalParent.standard,
          category: universalParent.category,
        }
      });
    }
    hierarchicalEdges.push({
      edge_type_name: 'BELONGS_TO',
      target_node_name: entityName,
      target_catalog_type: 'business_object',
      properties: {
        entity_name: entityName,
      }
    });

    // Check if an existing target node matches the contextual name
    const existingContextual = targetNodes.find(t => normalizeName(t.node_name) === normalizeName(contextualTermName));
    if (existingContextual && !isSuggestionRejected(tenantId, sourceNode.id, existingContextual.id, existingContextual.node_name)) {
      suggestions.unshift({
        targetNode: existingContextual,
        confidence: 98,
        matchReason: `Contextual Entity Term: Matched existing "${existingContextual.node_name}" for table "${entityName}" (${universalParent ? `specializing ${universalParent.name}` : ''}).`,
        matchType: 'exact_normalized',
        isGenericCollision: true,
        contextualEntity: entityName,
        suggestedContextualTerm: contextualTermName,
        isContextualDisambiguated: true,
        universalParentName: universalParent?.name,
        universalStandard: universalParent?.standard,
        governanceTier: existingContextual.type === 'core' ? 'gold_certified' : 'custom',
        hierarchicalEdgesToCreate: hierarchicalEdges,
        edgeDraft: {
          transformation: 'direct',
          confidence: 98,
          mapping_notes: `Contextually mapped to ${existingContextual.node_name}`,
        }
      });
    } else if (!isSuggestionRejected(tenantId, sourceNode.id, `new-${contextualTermName}`, contextualTermName)) {
      // Propose NEW contextual draft
      const draftNode: CatalogNodeItem = {
        id: `draft-${sourceNode.id}`,
        node_name: contextualTermName,
        description: `Contextual ${targetType} for ${entityName} ${sourceNode.node_name}${universalParent ? ` (Specializes ${universalParent.name})` : ''}`,
        catalog_type: targetType,
        catalog_type_name: targetType,
        properties: {
          data_type: sourceNode.properties?.data_type || 'string',
          source_origin: sourceNode.node_name,
          parent_entity: entityName,
          category: universalParent?.category || 'Contextual Term',
          universal_parent: universalParent?.name,
          standard: universalParent?.standard,
          is_contextual_disambiguated: true,
          is_auto_suggested: true,
        }
      };

      const contextualDraftSuggestion: MatchSuggestion = {
        targetNode: draftNode,
        targetDraft: {
          isNew: true,
          node_name: contextualTermName,
          description: `Contextual ${targetType} for ${entityName} ${sourceNode.node_name}`,
          catalog_type: targetType,
          node_type_id: targetNodeTypeId || '',
          properties: {
            data_type: sourceNode.properties?.data_type || 'string',
            parent_entity: entityName,
            category: universalParent?.category || 'Contextual Term',
            universal_parent: universalParent?.name,
            standard: universalParent?.standard,
          }
        },
        edgeDraft: {
          transformation: 'direct',
          confidence: 95,
          mapping_notes: `Disambiguated with table prefix: ${contextualTermName}${universalParent ? ` -> SPECIALIZES ${universalParent.name}` : ''}`,
        },
        confidence: 95,
        matchReason: `Contextual Entity Term: Prefixes table "${entityName}" to generic column "${sourceNode.node_name}" (${universalParent ? `Specializes Universal ${universalParent.name}` : ''}).`,
        matchType: 'contextual_disambiguated',
        isGenericCollision: true,
        contextualEntity: entityName,
        suggestedContextualTerm: contextualTermName,
        isContextualDisambiguated: true,
        universalParentName: universalParent?.name,
        universalStandard: universalParent?.standard,
        governanceTier: 'draft',
        hierarchicalEdgesToCreate: hierarchicalEdges,
      };

      suggestions.unshift(contextualDraftSuggestion);
    }
  }

  // Sort descending by confidence
  suggestions.sort((a, b) => b.confidence - a.confidence);

  let topSuggestion = suggestions[0] || null;
  const alternatives = suggestions.slice(1, 5);

  // If no existing target matched with high confidence and not generic, generate standard proposed [NEW] Target Node Draft!
  if ((!topSuggestion || topSuggestion.confidence < 70) && (!isGeneric || !entityName)) {
    const rawName = sourceNode.node_name || '';
    const norm = normalizeName(rawName);
    const cleanTitle = norm
      .split(' ')
      .map(w => w.charAt(0).toUpperCase() + w.slice(1))
      .join(' ') || rawName;

    // Only propose draft if not rejected
    if (!isSuggestionRejected(tenantId, sourceNode.id, `new-${cleanTitle}`, cleanTitle)) {
      const draftNode: CatalogNodeItem = {
        id: `draft-${sourceNode.id}`,
        node_name: cleanTitle,
        description: `Suggested ${targetType} for ${sourceNode.node_name}`,
        catalog_type: targetType,
        catalog_type_name: targetType,
        properties: {
          data_type: sourceNode.properties?.data_type || 'string',
          source_origin: sourceNode.node_name,
          category: universalParent?.category || 'Suggested Term',
          universal_parent: universalParent?.name,
          standard: universalParent?.standard,
          is_auto_suggested: true,
        }
      };

      const newDraftSuggestion: MatchSuggestion = {
        targetNode: draftNode,
        targetDraft: {
          isNew: true,
          node_name: cleanTitle,
          description: `Auto-suggested term for ${sourceNode.node_name}`,
          catalog_type: targetType,
          node_type_id: targetNodeTypeId || '',
          properties: {
            data_type: sourceNode.properties?.data_type || 'string',
            category: universalParent?.category || 'Auto Suggested',
            universal_parent: universalParent?.name,
            standard: universalParent?.standard,
          }
        },
        edgeDraft: {
          transformation: 'direct',
          confidence: 88,
          mapping_notes: `Auto-creates new target term "${cleanTitle}"`,
        },
        confidence: 88,
        matchReason: `Create new ${targetType} "${cleanTitle}"`,
        matchType: 'abbreviation',
        universalParentName: universalParent?.name,
        universalStandard: universalParent?.standard,
        governanceTier: 'draft',
      };

      if (!topSuggestion || topSuggestion.confidence < 88) {
        if (topSuggestion) alternatives.unshift(topSuggestion);
        topSuggestion = newDraftSuggestion;
      } else {
        alternatives.unshift(newDraftSuggestion);
      }
    }
  }

  return {
    topSuggestion,
    alternatives,
  };
}
