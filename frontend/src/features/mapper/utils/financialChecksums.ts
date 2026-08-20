/**
 * Financial Industry Pattern & Algorithmic Checksum Verification
 * 
 * Provides ISO 6166 (ISIN), CUSIP Mod-10, SEDOL, ISO 17442 (LEI),
 * and ISO 4217 Currency validation for billion-row sample profiling.
 */

// ISO 4217 Major Currency Codes Set
export const ISO_4217_CURRENCY_CODES = new Set([
  'USD', 'EUR', 'GBP', 'JPY', 'CHF', 'CAD', 'AUD', 'NZD', 'SEK', 'NOK',
  'DKK', 'SGD', 'HKD', 'ZAR', 'CNY', 'INR', 'BRL', 'MXN', 'KRW', 'PLN',
  'TRY', 'TWD', 'THB', 'ILS', 'IDR', 'CZK', 'HUF', 'CLP', 'PHP', 'AED',
  'SAR', 'MYR', 'COP', 'PEN', 'KWD', 'QAR', 'EGP', 'VND', 'ARS', 'NGN'
]);

/**
 * ISO 6166 ISIN (International Securities Identification Number) Checksum Validator
 * Structure: 2 alpha country code + 9 alphanumeric NSIN + 1 check digit (Luhn mod 10)
 */
export function validateISIN(isin: string): boolean {
  if (!isin || typeof isin !== 'string') return false;
  const clean = isin.trim().toUpperCase();
  if (!/^[A-Z]{2}[A-Z0-9]{9}[0-9]$/.test(clean)) return false;

  // Convert letters to numbers: A=10, B=11 ... Z=35
  let digits = '';
  for (let i = 0; i < clean.length - 1; i++) {
    const code = clean.charCodeAt(i);
    if (code >= 65 && code <= 90) {
      digits += (code - 55).toString();
    } else {
      digits += clean[i];
    }
  }

  // Double every second digit from right to left
  let sum = 0;
  let double = true;
  for (let i = digits.length - 1; i >= 0; i--) {
    let d = parseInt(digits[i], 10);
    if (double) {
      d *= 2;
      if (d > 9) d = Math.floor(d / 10) + (d % 10);
    }
    sum += d;
    double = !double;
  }

  const checkDigit = (10 - (sum % 10)) % 10;
  return checkDigit === parseInt(clean[clean.length - 1], 10);
}

/**
 * CUSIP (Committee on Uniform Securities Identification Procedures) Mod-10 Checksum
 * Structure: 9 characters (alphanumeric).
 */
export function validateCUSIP(cusip: string): boolean {
  if (!cusip || typeof cusip !== 'string') return false;
  const clean = cusip.trim().toUpperCase();
  if (!/^[0-9A-Z]{8}[0-9]$/.test(clean)) return false;

  let sum = 0;
  for (let i = 0; i < 8; i++) {
    const ch = clean[i];
    let val: number;
    if (ch >= '0' && ch <= '9') {
      val = parseInt(ch, 10);
    } else if (ch >= 'A' && ch <= 'Z') {
      val = ch.charCodeAt(0) - 55;
    } else {
      return false;
    }

    if (i % 2 !== 0) {
      val *= 2;
    }
    sum += Math.floor(val / 10) + (val % 10);
  }

  const checkDigit = (10 - (sum % 10)) % 10;
  return checkDigit === parseInt(clean[8], 10);
}

/**
 * SEDOL (Stock Exchange Daily Official List) Checksum
 * Structure: 7 alphanumeric characters (B000300) with weighted sum mod 10
 */
export function validateSEDOL(sedol: string): boolean {
  if (!sedol || typeof sedol !== 'string') return false;
  const clean = sedol.trim().toUpperCase();
  if (!/^[0-9B-DF-HJ-NP-TV-Z]{6}[0-9]$/.test(clean)) return false;

  const weights = [1, 3, 1, 7, 3, 9];
  let sum = 0;

  for (let i = 0; i < 6; i++) {
    const ch = clean[i];
    let val: number;
    if (ch >= '0' && ch <= '9') {
      val = parseInt(ch, 10);
    } else {
      val = ch.charCodeAt(0) - 55;
    }
    sum += val * weights[i];
  }

  const checkDigit = (10 - (sum % 10)) % 10;
  return checkDigit === parseInt(clean[6], 10);
}

/**
 * ISO 17442 Legal Entity Identifier (LEI) Structure Validator
 * Structure: 20 alphanumeric characters (4 LOU + 2 reserved + 12 entity + 2 check)
 */
export function validateLEI(lei: string): boolean {
  if (!lei || typeof lei !== 'string') return false;
  const clean = lei.trim().toUpperCase();
  return /^[0-9A-Z]{18}[0-9]{2}$/.test(clean);
}

/**
 * Check if string is an ISO 4217 Currency Code
 */
export function validateCurrencyCode(val: string): boolean {
  if (!val || typeof val !== 'string') return false;
  return ISO_4217_CURRENCY_CODES.has(val.trim().toUpperCase());
}

export type InferredFinancialPattern =
  | 'ISIN'
  | 'CUSIP'
  | 'SEDOL'
  | 'LEI'
  | 'ISO_CURRENCY'
  | 'FINANCIAL_AMOUNT'
  | 'YIELD_PERCENTAGE'
  | 'DATE'
  | 'EMAIL'
  | 'GENERIC_STRING';

/**
 * Analyzes a stream/array of sampled values (e.g. 10 to 500 samples) and determines
 * the definitive financial pattern with ground-truth confidence.
 */
export function analyzeSampledValues(sampleValues: any[]): {
  pattern: InferredFinancialPattern;
  confidence: number;
  description: string;
  bloombergCandidate?: string;
} {
  const nonNulls = sampleValues
    .filter(v => v !== null && v !== undefined && v !== '')
    .map(v => String(v).trim());

  if (nonNulls.length === 0) {
    return { pattern: 'GENERIC_STRING', confidence: 0, description: 'Empty or all-null sample' };
  }

  // 1. ISIN check
  const isinMatches = nonNulls.filter(validateISIN);
  if (isinMatches.length / nonNulls.length >= 0.8) {
    return {
      pattern: 'ISIN',
      confidence: 100,
      description: `ISO 6166 ISIN checksum verified (${Math.round((isinMatches.length / nonNulls.length) * 100)}% valid)`,
      bloombergCandidate: 'ID_ISIN',
    };
  }

  // 2. CUSIP check
  const cusipMatches = nonNulls.filter(validateCUSIP);
  if (cusipMatches.length / nonNulls.length >= 0.8) {
    return {
      pattern: 'CUSIP',
      confidence: 100,
      description: `CUSIP mod-10 checksum verified (${Math.round((cusipMatches.length / nonNulls.length) * 100)}% valid)`,
      bloombergCandidate: 'ID_CUSIP',
    };
  }

  // 3. SEDOL check
  const sedolMatches = nonNulls.filter(validateSEDOL);
  if (sedolMatches.length / nonNulls.length >= 0.8) {
    return {
      pattern: 'SEDOL',
      confidence: 100,
      description: `SEDOL weighted checksum verified (${Math.round((sedolMatches.length / nonNulls.length) * 100)}% valid)`,
      bloombergCandidate: 'ID_SEDOL1',
    };
  }

  // 4. ISO 4217 Currency check
  const ccyMatches = nonNulls.filter(validateCurrencyCode);
  if (ccyMatches.length / nonNulls.length >= 0.8) {
    return {
      pattern: 'ISO_CURRENCY',
      confidence: 100,
      description: `ISO 4217 3-letter currency standard verified (${Math.round((ccyMatches.length / nonNulls.length) * 100)}% valid)`,
      bloombergCandidate: 'CRNCY',
    };
  }

  // 5. LEI check
  const leiMatches = nonNulls.filter(validateLEI);
  if (leiMatches.length / nonNulls.length >= 0.8) {
    return {
      pattern: 'LEI',
      confidence: 100,
      description: `ISO 17442 Legal Entity Identifier format verified`,
      bloombergCandidate: 'LEI_CODE',
    };
  }

  // 6. Yield / Rate percentage check (numbers between 0.0001 and 0.50 or 0% to 50%)
  const numericValues = nonNulls
    .map(v => parseFloat(v))
    .filter(v => !isNaN(v));

  if (numericValues.length / nonNulls.length >= 0.8 && numericValues.length > 0) {
    const isSmallRate = numericValues.every(n => n >= 0 && n <= 0.40);
    const hasDecimals = numericValues.some(n => n % 1 !== 0);

    if (isSmallRate && hasDecimals) {
      return {
        pattern: 'YIELD_PERCENTAGE',
        confidence: 95,
        description: `Fixed income yield / rate distribution (0% to 40% range)`,
        bloombergCandidate: 'YLD_YTM_MID',
      };
    }

    if (hasDecimals) {
      return {
        pattern: 'FINANCIAL_AMOUNT',
        confidence: 90,
        description: `Financial monetary quantity with decimal precision`,
        bloombergCandidate: 'PX_LAST',
      };
    }
  }

  // 7. Date check
  const dateMatches = nonNulls.filter(v => !isNaN(Date.parse(v)) && (v.includes('-') || v.includes('/')));
  if (dateMatches.length / nonNulls.length >= 0.8) {
    return {
      pattern: 'DATE',
      confidence: 95,
      description: `ISO 8601 / standard temporal timestamp format`,
      bloombergCandidate: 'MATURITY',
    };
  }

  return {
    pattern: 'GENERIC_STRING',
    confidence: 50,
    description: `Sample text strings (${nonNulls.length} samples)`,
  };
}
