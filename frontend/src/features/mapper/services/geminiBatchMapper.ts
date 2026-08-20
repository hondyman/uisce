import type { CatalogNodeItem, MatchSuggestion, TargetNodeDraft, EdgePropertiesDraft, RelatedItemCandidate, HierarchicalEdgeDraft, VendorAlignment } from '../types';
import type { EdgeType } from '../../types/edgeTypes';
import { ABBREVIATIONS, UNIVERSAL_VALUE_TYPES } from '../utils/graphMatcher';
import { lookupBloombergField } from '../constants/financialVendorDictionaries';
import apiClient from '../../../utils/apiClient';

export interface BatchMappingResult {
  sourceId: string;
  suggestion: MatchSuggestion;
}

/**
 * Run Gemini AI Batch Mapping on a chunk of source items
 */
export async function runGeminiBatchMapping(
  sourceItems: CatalogNodeItem[],
  targetItems: CatalogNodeItem[],
  edgeType: EdgeType,
  sourceNodeType: any,
  targetNodeType: any,
  tenantId: string,
  onProgress?: (processed: number, total: number) => void
): Promise<Map<string, MatchSuggestion>> {
  const results = new Map<string, MatchSuggestion>();
  const CHUNK_SIZE = 20;

  for (let i = 0; i < sourceItems.length; i += CHUNK_SIZE) {
    const chunk = sourceItems.slice(i, i + CHUNK_SIZE);

    try {
      // Build batch prompt
      const prompt = `You are a world-class financial semantic layer and capital markets data catalog architect.
Analyze the following source items of type "${sourceNodeType?.catalog_type_name || 'Node'}" and suggest the optimal mapping relationship "${edgeType.edge_type_name}" to target items of type "${targetNodeType?.catalog_type_name || 'Node'}".

EXISTING TARGET NODES IN CATALOG:
${JSON.stringify(targetItems.slice(0, 80).map(t => ({ id: t.id, name: t.node_name, description: t.description })), null, 2)}

FINANCIAL SERVICES VENDOR DICTIONARY (BLOOMBERG DATA LICENSE / B-PIPE FIELDS):
- Pricing & Valuation: PX_LAST (Last/Close Price), PX_BID (Bid), PX_ASK (Ask), PX_MID (Mid), PX_OPEN (Open), PX_VOLUME (Volume), VWAP
- Symbology & Security Master: ID_ISIN (ISIN), ID_CUSIP (CUSIP), ID_SEDOL1 (SEDOL), ID_BB_GLOBAL (FIGI), TICKER, CRNCY (ISO Currency)
- Fixed Income & Rates: YLD_YTM_MID (Yield to Maturity), DUR_ADJ_MID (Modified Duration), CONVEXITY_MID (Convexity), CPN (Coupon Rate), MATURITY (Maturity Date)
- Equities & Fundamentals: CUR_MKT_CAP (Market Cap), PE_RATIO (P/E Ratio), PX_TO_BOOK_RATIO (P/B), EQY_DVD_YLD_EST (Dividend Yield), EBITDA
- Entity & Risk: LEI_CODE (Legal Entity Identifier), ISSUER (Issuer Legal Name), ULT_PARENT_CNTRY_OF_RISK (Country of Risk)

COMMON ABBREVIATIONS:
${JSON.stringify(Object.entries(ABBREVIATIONS).slice(0, 40), null, 2)}

SOURCE ITEMS TO MAP:
${JSON.stringify(chunk.map(s => ({ id: s.id, name: s.node_name, path: s.qualified_path, description: s.description, dataType: s.properties?.data_type })), null, 2)}

INSTRUCTIONS:
For each source item, determine:
1. If it matches an EXISTING target node, provide "target_id".
2. If NO existing target node matches, suggest a NEW target node with "new_target_name", "new_target_description", and "new_target_properties" (e.g. data_type, category, classification).
3. FINANCIAL & BLOOMBERG ALIGNMENT: If the field corresponds to a financial metric or market data element (pricing, yield, duration, market cap, dividend, ISIN, CUSIP, currency, etc.), identify the corresponding "bloomberg_mnemonic" (e.g. "PX_LAST", "ID_ISIN", "YLD_YTM_MID").
4. CONTEXTUAL DISAMBIGUATION: If the source column name is generic (such as "address", "name", "id", "status", "date", "price", "phone", "city", etc.), incorporate the parent table/entity name into the suggested term (e.g., table "vendor" and column "address" -> "VendorAddress") to prevent cross-table collisions.
5. Provide a "confidence" score (0 to 100) and a concise human-readable "reason".
6. Identify any related "see_also_names" (e.g. if source is CUSIP, related items might be SEDOL, ISIN, Ticker).

Return ONLY valid JSON array with objects matching:
[
  {
    "source_id": "string",
    "target_id": "string or null",
    "new_target_name": "string or null",
    "new_target_description": "string or null",
    "bloomberg_mnemonic": "string or null",
    "new_target_properties": { "data_type": "string", "category": "string" },
    "confidence": number,
    "reason": "string",
    "edge_properties": { "transformation": "string", "notes": "string" },
    "see_also_names": ["string"]
  }
]`;

      // Try calling LLM backend gateway
      const aiResponse = await apiClient<any>(`/api/llm/query`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          datasource: 'catalog',
          prompt,
          mode: 'exploratory',
          tenant_id: tenantId,
        })
      }).catch(() => null);

      if (aiResponse && (aiResponse.response || aiResponse.text || Array.isArray(aiResponse))) {
        const rawContent = aiResponse.response || aiResponse.text || JSON.stringify(aiResponse);
        const jsonMatch = rawContent.match(/\[\s*\{[\s\S]*\}\s*\]/);
        if (jsonMatch) {
          const parsed = JSON.parse(jsonMatch[0]);
          for (const item of parsed) {
            if (!item.source_id) continue;

            const existingTarget = item.target_id ? targetItems.find(t => t.id === item.target_id) : null;
            let targetNode: CatalogNodeItem;
            let targetDraft: TargetNodeDraft | undefined;

            if (existingTarget) {
              targetNode = existingTarget;
            } else if (item.new_target_name) {
              const newName = item.new_target_name.trim();
              targetNode = {
                id: `ai-draft-${item.source_id}`,
                node_name: newName,
                description: item.new_target_description || `AI suggested ${targetNodeType?.catalog_type_name || 'term'}`,
                catalog_type: targetNodeType?.catalog_type_name || 'semantic_term',
                properties: item.new_target_properties || {},
              };
              targetDraft = {
                isNew: true,
                node_name: newName,
                description: item.new_target_description || `AI suggested term for ${item.source_id}`,
                catalog_type: targetNodeType?.catalog_type_name || 'semantic_term',
                node_type_id: targetNodeType?.id || '',
                properties: item.new_target_properties || {},
              };
            } else {
              continue;
            }

            const bbgLookup = lookupBloombergField(item.bloomberg_mnemonic || item.new_target_name || '');
            let vendorAlignment: VendorAlignment | undefined;
            const hierarchicalEdges: HierarchicalEdgeDraft[] = [];

            if (bbgLookup) {
              vendorAlignment = {
                vendor: bbgLookup.vendor,
                mnemonic: bbgLookup.mnemonic,
                canonicalTermName: bbgLookup.canonicalTermName,
                category: bbgLookup.category,
                description: bbgLookup.description,
                feedType: bbgLookup.feedType,
              };

              hierarchicalEdges.push({
                edge_type_name: 'LICENSED_BY',
                target_node_name: `Bloomberg:${bbgLookup.mnemonic}`,
                target_catalog_type: 'vendor_field',
                properties: {
                  vendor: 'BLOOMBERG',
                  mnemonic: bbgLookup.mnemonic,
                  category: bbgLookup.category,
                  description: bbgLookup.description,
                  feed_type: bbgLookup.feedType,
                }
              });

              if (bbgLookup.universalArchetype) {
                hierarchicalEdges.push({
                  edge_type_name: 'SPECIALIZES',
                  target_node_name: bbgLookup.universalArchetype,
                  target_catalog_type: 'semantic_term',
                  properties: {
                    standard: UNIVERSAL_VALUE_TYPES[bbgLookup.universalArchetype]?.standard || 'ISO 4217 / FIBO',
                    category: bbgLookup.category,
                  }
                });
              }
            }

            const seeAlsoCandidates: RelatedItemCandidate[] = (item.see_also_names || []).map((name: string, sIdx: number) => ({
              id: `see-also-${sIdx}-${name}`,
              node_name: name,
              relation_type: 'see_also' as const,
              description: `Related identifier / peer term`,
            }));

            results.set(item.source_id, {
              targetNode,
              targetDraft,
              edgeDraft: {
                transformation: item.edge_properties?.transformation || 'direct',
                confidence: item.confidence || 85,
                mapping_notes: item.edge_properties?.notes || item.reason || 'Gemini AI recommendation',
              },
              confidence: item.confidence || 85,
              matchReason: `✨ Gemini AI: ${item.reason || 'Optimal semantic mapping'}${bbgLookup ? ` [Aligned to Bloomberg ${bbgLookup.mnemonic}]` : ''}`,
              matchType: bbgLookup ? 'vendor_aligned' : 'gemini_ai',
              relatedItems: seeAlsoCandidates,
              vendorAlignment,
              hierarchicalEdgesToCreate: hierarchicalEdges.length > 0 ? hierarchicalEdges : undefined,
              universalParentName: bbgLookup?.universalArchetype,
              universalStandard: bbgLookup?.universalArchetype ? UNIVERSAL_VALUE_TYPES[bbgLookup.universalArchetype]?.standard : undefined,
            });
          }
        }
      }
    } catch (err) {
      console.warn('Gemini batch suggestion warning (falling back to heuristics):', err);
    }

    if (onProgress) {
      onProgress(Math.min(i + CHUNK_SIZE, sourceItems.length), sourceItems.length);
    }
  }

  return results;
}
