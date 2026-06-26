// Enhanced Semantic Matching Utilities for Frontend
// This demonstrates how the backend improvements integrate with the UI

import { getAbbreviationMap } from './abbreviationApi';
import { devDebug, devWarn, devError } from './devLogger';

// Legacy abbreviation map for fallback (will be replaced by database)
export const LEGACY_ABBREVIATION_MAP: Record<string, string> = {
  // Geographic
  'CNTRY': 'COUNTRY',
  'CTRY': 'COUNTRY', 
  'ST': 'STATE',
  'ADDR': 'ADDRESS',
  'ZIP': 'ZIPCODE',
  'POSTAL': 'POSTALCODE',
  'CTY': 'CITY',
  'REGN': 'REGION',
  
  // Financial
  'AMT': 'AMOUNT',
  'VAL': 'VALUE',
  'BAL': 'BALANCE',
  'CURR': 'CURRENCY',
  'FX': 'FOREIGN_EXCHANGE',
  'ACCT': 'ACCOUNT',
  'TXN': 'TRANSACTION',
  'PMT': 'PAYMENT',
  
  // Temporal
  'DT': 'DATE',
  'DTM': 'DATETIME',
  'TS': 'TIMESTAMP',
  'YR': 'YEAR',
  'MON': 'MONTH',
  'WK': 'WEEK',
  'QTR': 'QUARTER',
  
  // Business
  'CUST': 'CUSTOMER',
  'CLNT': 'CLIENT',
  'ORD': 'ORDER',
  'PROD': 'PRODUCT',
  'CATEG': 'CATEGORY',
  'DEPT': 'DEPARTMENT',
  'DIV': 'DIVISION',
  'ORG': 'ORGANIZATION',
  'COMP': 'COMPANY',
  
  // Identity
  'ID': 'IDENTIFIER',
  'NUM': 'NUMBER', 
  'NBR': 'NUMBER',
  'NO': 'NUMBER',
  'CD': 'CODE',
  'KEY': 'KEY',
  'REF': 'REFERENCE',
  
  // Measurements
  'QTY': 'QUANTITY',
  'CNT': 'COUNT',
  'PCT': 'PERCENT',
  'RATE': 'RATE',
  'RATIO': 'RATIO',
  'SCORE': 'SCORE',
  'RANK': 'RANK',
  
  // Status/Flags
  'FLG': 'FLAG',
  'IND': 'INDICATOR',
  'STAT': 'STATUS',
  'TYP': 'TYPE',
  
  // Common prefixes/suffixes
  'DESC': 'DESCRIPTION',
  'NM': 'NAME',
  'TTL': 'TOTAL',
  'AVG': 'AVERAGE',
  'MIN': 'MINIMUM', 
  'MAX': 'MAXIMUM',
  'SUM': 'SUMMARY'
};

// Keep the old constant name for backward compatibility
export const ABBREVIATION_MAP = LEGACY_ABBREVIATION_MAP;

// Enhanced suggestion type that includes profile information
export interface EnhancedSuggestion {
  id: string;
  title: string;
  subtitle: string;
  type: 'semantic-term';
  confidence?: number;
  nameConfidence?: number;
  profileConfidence?: number;
  typeConfidence?: number;
  matchReason?: string;
  abbreviationExpanded?: boolean;
  profileData?: {
    valueOverlap?: number;
    patternOverlap?: number;
    cardinalityMatch?: number;
    dataTypeMatch?: boolean;
  };
}

/**
 * Expands abbreviations in a column name using database-backed abbreviations
 */
export async function expandColumnAbbreviationsDB(columnName: string): Promise<string[]> {
  const normalized = columnName.toUpperCase();
  const variations = [normalized];
  
  try {
    const abbreviationMap = await getAbbreviationMap();
    
    // Split on common separators
    const separators = ['_', '-', '.', ' '];
    let tokens: string[] = [];
    
    for (const sep of separators) {
      if (normalized.includes(sep)) {
        tokens = normalized.split(sep);
        break;
      }
    }
    
    if (tokens.length === 0) {
      tokens = [normalized];
    }
    
    // Check if any token is an abbreviation
    let hasExpansion = false;
    const expandedTokenSets: string[][] = [];
    
    for (const token of tokens) {
      const tokenVariations = [token];
      const expansion = abbreviationMap.get(token);
      if (expansion) {
        tokenVariations.push(expansion);
        hasExpansion = true;
      }
      expandedTokenSets.push(tokenVariations);
    }
    
    // Generate combinations if we have expansions
    if (hasExpansion) {
      const combinations = generateCombinations(expandedTokenSets);
      for (const combo of combinations) {
        variations.push(combo.join('_'));
      }
    }
  } catch (error) {
    devWarn('Failed to fetch abbreviations from database, using fallback:', error);
    // Fall back to legacy method
    return expandColumnAbbreviations(columnName);
  }
  
  return [...new Set(variations)]; // Remove duplicates
}

/**
 * Expands abbreviations in a column name (client-side preview)
 * This mirrors the backend abbreviation expansion logic
 * @deprecated Use expandColumnAbbreviationsDB for database-backed expansion
 */
export function expandColumnAbbreviations(columnName: string): string[] {
  const normalized = columnName.toUpperCase();
  const variations = [normalized];
  
  // Split on common separators
  const separators = ['_', '-', '.', ' '];
  let tokens: string[] = [];
  
  for (const sep of separators) {
    if (normalized.includes(sep)) {
      tokens = normalized.split(sep);
      break;
    }
  }
  
  if (tokens.length === 0) {
    tokens = [normalized];
  }
  
  // Check if any token is an abbreviation
  let hasExpansion = false;
  const expandedTokenSets: string[][] = [];
  
  for (const token of tokens) {
    const tokenVariations = [token];
    if (ABBREVIATION_MAP[token]) {
      tokenVariations.push(ABBREVIATION_MAP[token]);
      hasExpansion = true;
    }
    expandedTokenSets.push(tokenVariations);
  }
  
  // Generate combinations if we have expansions
  if (hasExpansion) {
    const combinations = generateCombinations(expandedTokenSets);
    for (const combo of combinations) {
      variations.push(combo.join('_'));
    }
  }
  
  return [...new Set(variations)]; // Remove duplicates
}

/**
 * Generates all possible combinations from token variations
 */
function generateCombinations(tokenSets: string[][]): string[][] {
  if (tokenSets.length === 0) return [];
  if (tokenSets.length === 1) return tokenSets[0].map(token => [token]);
  
  const result: string[][] = [];
  const restCombos = generateCombinations(tokenSets.slice(1));
  
  for (const token of tokenSets[0]) {
    for (const restCombo of restCombos) {
      result.push([token, ...restCombo]);
    }
  }
  
  return result;
}

/**
 * Enhanced suggestion generator using database-backed abbreviations
 */
export async function generateSemanticSuggestionsDB(
  searchTerm: string
): Promise<EnhancedSuggestion[]> {
  if (!searchTerm || searchTerm.length < 2) return [];
  
  // Expand abbreviations using database
  const expandedTerms = await expandColumnAbbreviationsDB(searchTerm);
  const hasAbbreviation = expandedTerms.length > 1;
  
  const suggestions: EnhancedSuggestion[] = [];
  
  // Show abbreviation expansion as a suggestion
  if (hasAbbreviation) {
    const bestExpansion = expandedTerms.find(term => term !== searchTerm.toUpperCase()) || '';
    if (bestExpansion) {
      const expansions = await getExpandedAbbreviationsDB(searchTerm);
      suggestions.push({
        id: `abbrev-${searchTerm}`,
        title: bestExpansion,
        subtitle: `Expanded from ${searchTerm.toUpperCase()} using database abbreviations`,
        type: 'semantic-term',
        confidence: 0.95,
        nameConfidence: 0.90,
        profileConfidence: 0.0,
        typeConfidence: 0.80,
        matchReason: `Abbreviation expanded: ${expansions}`,
        abbreviationExpanded: true
      });
    }
  }
  
  // Add some example profile-enhanced suggestions
  const profileExamples = generateProfileEnhancedExamples(searchTerm);
  suggestions.push(...profileExamples);
  
  return suggestions.slice(0, 8); // Limit suggestions
}

/**
 * Enhanced suggestion generator that shows abbreviation expansion preview
 * @deprecated Use generateSemanticSuggestionsDB for database-backed suggestions
 */
export function generateSemanticSuggestions(
  searchTerm: string
): EnhancedSuggestion[] {
  if (!searchTerm || searchTerm.length < 2) return [];
  
  // Expand abbreviations for preview
  const expandedTerms = expandColumnAbbreviations(searchTerm);
  const hasAbbreviation = expandedTerms.length > 1;
  
  // Mock enhanced suggestions with profile-aware confidence
  const suggestions: EnhancedSuggestion[] = [];
  
  // Show abbreviation expansion as a suggestion
  if (hasAbbreviation) {
    const bestExpansion = expandedTerms.find(term => term !== searchTerm.toUpperCase()) || '';
    if (bestExpansion) {
      suggestions.push({
        id: `abbrev-${searchTerm}`,
        title: bestExpansion,
        subtitle: `Expanded from ${searchTerm.toUpperCase()} using abbreviation mapping`,
        type: 'semantic-term',
        confidence: 0.95,
        nameConfidence: 0.90,
        profileConfidence: 0.0,
        typeConfidence: 0.80,
        matchReason: `Abbreviation expanded: ${getExpandedAbbreviations(searchTerm)}`,
        abbreviationExpanded: true
      });
    }
  }
  
  // Add some example profile-enhanced suggestions
  const profileExamples = generateProfileEnhancedExamples(searchTerm);
  suggestions.push(...profileExamples);
  
  return suggestions.slice(0, 8); // Limit suggestions
}

/**
 * Generate example suggestions that demonstrate profile-based matching
 */
function generateProfileEnhancedExamples(searchTerm: string): EnhancedSuggestion[] {
  const term = searchTerm.toLowerCase();
  const suggestions: EnhancedSuggestion[] = [];
  
  // Email pattern example
  if (term.includes('email') || term.includes('mail')) {
    suggestions.push({
      id: 'email-enhanced',
      title: 'EMAIL_ADDRESS',
      subtitle: 'High confidence match with profile data',
      type: 'semantic-term',
      confidence: 0.92,
      nameConfidence: 0.85,
      profileConfidence: 0.90,
      typeConfidence: 1.0,
      matchReason: 'Strong name similarity, 90% value overlap (@gmail.com, @yahoo.com), Email pattern detected',
      profileData: {
        valueOverlap: 0.90,
        patternOverlap: 1.0,
        cardinalityMatch: 0.85,
        dataTypeMatch: true
      }
    });
  }
  
  // Customer ID pattern example
  if (term.includes('cust') || term.includes('customer')) {
    suggestions.push({
      id: 'customer-enhanced',
      title: 'CUSTOMER_IDENTIFIER',
      subtitle: 'Enhanced with cardinality analysis',
      type: 'semantic-term',
      confidence: 0.88,
      nameConfidence: 0.80,
      profileConfidence: 0.85,
      typeConfidence: 0.95,
      matchReason: 'Good name similarity, Similar cardinality (50K), Compatible data types (integer)',
      profileData: {
        valueOverlap: 0.0,
        patternOverlap: 0.5,
        cardinalityMatch: 0.95,
        dataTypeMatch: true
      }
    });
  }
  
  // Country code example
  if (term.includes('country') || term.includes('cntry')) {
    suggestions.push({
      id: 'country-enhanced',
      title: 'COUNTRY_CODE',
      subtitle: 'Profile-validated with reference values',
      type: 'semantic-term',
      confidence: 0.94,
      nameConfidence: 0.85,
      profileConfidence: 0.95,
      typeConfidence: 1.0,
      matchReason: 'Strong name similarity, 85% value overlap (US, CA, GB), Perfect cardinality match (195)',
      profileData: {
        valueOverlap: 0.85,
        patternOverlap: 1.0,
        cardinalityMatch: 1.0,
        dataTypeMatch: true
      }
    });
  }
  
  return suggestions;
}

/**
 * Get expanded abbreviations for display using database
 */
async function getExpandedAbbreviationsDB(columnName: string): Promise<string> {
  try {
    const abbreviationMap = await getAbbreviationMap();
    const tokens = columnName.toUpperCase().split(/[_\-\.\s]/);
    const expansions: string[] = [];
    
    for (const token of tokens) {
      const expansion = abbreviationMap.get(token);
      if (expansion) {
        expansions.push(`${token}→${expansion}`);
      }
    }
    
    return expansions.join(', ');
  } catch (error) {
    devWarn('Failed to get expanded abbreviations from database:', error);
    return getExpandedAbbreviations(columnName);
  }
}

/**
 * Get expanded abbreviations for display
 * @deprecated Use getExpandedAbbreviationsDB for database-backed expansion
 */
function getExpandedAbbreviations(columnName: string): string {
  const tokens = columnName.toUpperCase().split(/[_\-\.\s]/);
  const expansions: string[] = [];
  
  for (const token of tokens) {
    if (ABBREVIATION_MAP[token]) {
      expansions.push(`${token}→${ABBREVIATION_MAP[token]}`);
    }
  }
  
  return expansions.join(', ');
}

/**
 * Format confidence score for display
 */
export function formatConfidence(confidence: number): string {
  if (confidence >= 0.9) return 'Very High';
  if (confidence >= 0.8) return 'High';
  if (confidence >= 0.7) return 'Good';
  if (confidence >= 0.6) return 'Moderate';
  return 'Low';
}

/**
 * Get confidence color for UI display
 */
export function getConfidenceColor(confidence: number): string {
  if (confidence >= 0.9) return '#10b981'; // green-500
  if (confidence >= 0.8) return '#3b82f6'; // blue-500  
  if (confidence >= 0.7) return '#f59e0b'; // amber-500
  if (confidence >= 0.6) return '#f97316'; // orange-500
  return '#ef4444'; // red-500
}

/**
 * Enhanced suggestion selection handler
 */
export function handleEnhancedSuggestionSelect(
  suggestion: EnhancedSuggestion,
  onSelect: (suggestion: any) => void
) {
  // Log enhanced matching information
  devDebug('🔍 Enhanced Semantic Match Selected');
  devDebug('Term:', suggestion.title);
  devDebug('Overall Confidence:', suggestion.confidence?.toFixed(3));
  devDebug('Name Confidence:', suggestion.nameConfidence?.toFixed(3));
  devDebug('Profile Confidence:', suggestion.profileConfidence?.toFixed(3));
  devDebug('Type Confidence:', suggestion.typeConfidence?.toFixed(3));
  devDebug('Match Reason:', suggestion.matchReason);
  if (suggestion.abbreviationExpanded) {
    devDebug('✨ Abbreviation Expansion Applied');
  }
  if (suggestion.profileData) {
    devDebug('📊 Profile Data:', suggestion.profileData);
  }
  
  // Call original handler
  onSelect(suggestion);
}

/**
 * Check semantic terms for abbreviation violations
 */
export async function checkSemanticTermsForAbbreviations(termNames: string[]): Promise<{
  violations: Record<string, string[]>;
  suggestions: Record<string, string[]>;
}> {
  try {
    const abbreviationMap = await getAbbreviationMap();
    const violations: Record<string, string[]> = {};
    const suggestions: Record<string, string[]> = {};
    
    for (const termName of termNames) {
      // Split term name and check for abbreviations
      const normalized = termName.toUpperCase();
      const separators = ['_', '-', '.', ' '];
      let tokens: string[] = [];
      
      for (const sep of separators) {
        if (normalized.includes(sep)) {
          tokens = normalized.split(sep);
          break;
        }
      }
      
      if (tokens.length === 0) {
        tokens = [normalized];
      }
      
      const foundAbbreviations: string[] = [];
      const suggestedExpansions: string[] = [];
      
      for (const token of tokens) {
        const expansion = abbreviationMap.get(token);
        if (expansion) {
          foundAbbreviations.push(`${token} -> ${expansion}`);
          suggestedExpansions.push(normalized.replace(token, expansion));
        }
      }
      
      if (foundAbbreviations.length > 0) {
        violations[termName] = foundAbbreviations;
        suggestions[termName] = [...new Set(suggestedExpansions)];
      }
    }
    
    return { violations, suggestions };
  } catch (error) {
    devError('Failed to check semantic terms for abbreviations:', error);
    return { violations: {}, suggestions: {} };
  }
}

/**
 * Demo function to test abbreviation expansion
 */
export function testAbbreviationExpansion() {
  const testCases = [
    'CUST_ID',
    'CNTRY_CD', 
    'TXN_AMT',
    'ORD_DT',
    'EMAIL_ADDR'
  ];
  
  devDebug('🧪 Abbreviation Expansion Test (Legacy)');
  testCases.forEach(testCase => {
    const expanded = expandColumnAbbreviations(testCase);
    devDebug(`${testCase} →`, expanded);
  });
}

/**
 * Demo function to test database-backed abbreviation expansion
 */
export async function testDatabaseAbbreviationExpansion() {
  const testCases = [
    'CUST_ID',
    'CNTRY_CD', 
    'TXN_AMT',
    'ORD_DT',
    'EMAIL_ADDR',
    'ACCT_BAL',
    'INV_QTY'
  ];
  
  devDebug('🧪 Database Abbreviation Expansion Test');
  for (const testCase of testCases) {
    try {
      const expanded = await expandColumnAbbreviationsDB(testCase);
      const expansions = await getExpandedAbbreviationsDB(testCase);
      devDebug(`${testCase} →`, expanded);
      if (expansions) {
        devDebug(`  Expansions: ${expansions}`);
      }
    } catch (error) {
      devError(`  Error expanding ${testCase}:`, error);
    }
  }
  devDebug('🧪 Database Abbreviation Expansion Test (complete)');
}

/**
 * Demo function to test semantic term validation
 */
export async function testSemanticTermValidation() {
  const testTerms = [
    'CUSTOMER_IDENTIFIER',
    'CUST_ID',  // Should be flagged
    'TRANSACTION_AMOUNT', 
    'TXN_AMT', // Should be flagged
    'COUNTRY_CODE',
    'CNTRY_CD' // Should be flagged
  ];
  
  try {
    const result = await checkSemanticTermsForAbbreviations(testTerms);
    devDebug('Violations found:', result.violations);
    devDebug('Suggested expansions:', result.suggestions);
    
    const violationCount = Object.keys(result.violations).length;
    const validCount = testTerms.length - violationCount;
    devDebug(`✅ Valid terms: ${validCount}/${testTerms.length}`);
    devDebug(`⚠️ Terms with abbreviations: ${violationCount}/${testTerms.length}`);
  } catch (error) {
    devError('Validation test failed:', error);
  }
  devDebug('🧪 Semantic Term Validation Test (complete)');
}