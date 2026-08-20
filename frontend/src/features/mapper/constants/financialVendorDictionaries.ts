/**
 * Financial Vendor Data Dictionaries Registry
 * 
 * Provides official vendor field definitions, mnemonics, categories, and alias patterns
 * for capital markets and financial institutions (Bloomberg, FactSet, LSEG / Refinitiv).
 */

export type FinancialVendor = 'BLOOMBERG' | 'FACTSET' | 'LSEG' | 'SP_CAPITAL_IQ' | 'MSCI' | 'MORNINGSTAR';

export interface VendorFieldDefinition {
  vendor: FinancialVendor;
  mnemonic: string;
  canonicalTermName: string;
  category: 'Pricing & Valuation' | 'Symbology & Identifiers' | 'Security Master & Reference' | 'Fixed Income & Rates' | 'Equities & Fundamentals' | 'Corporate Actions' | 'Risk & ESG' | 'Entities & Counterparties';
  description: string;
  dataType: 'numeric' | 'string' | 'date' | 'boolean' | 'timestamp';
  universalArchetype?: 'FinancialAmount' | 'AuditTimestamp' | 'ContactCommunication' | 'Address' | 'PersonName';
  aliases: string[];
  feedType?: 'Data License' | 'B-PIPE' | 'Back Office' | 'Real-Time';
}

/**
 * Official Bloomberg Data Dictionary (Data License / B-PIPE / Per Security)
 */
export const BLOOMBERG_DATA_DICTIONARY: VendorFieldDefinition[] = [
  // ── 1. Pricing & Valuation
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'PX_LAST',
    canonicalTermName: 'LastPrice',
    category: 'Pricing & Valuation',
    description: 'Last price or official closing price at which the security traded.',
    dataType: 'numeric',
    universalArchetype: 'FinancialAmount',
    aliases: ['px_last', 'last_px', 'last_price', 'close_px', 'close_price', 'market_price', 'px', 'unit_price', 'unitprice', 'price'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'PX_BID',
    canonicalTermName: 'BidPrice',
    category: 'Pricing & Valuation',
    description: 'Current or closing bid price for the security.',
    dataType: 'numeric',
    universalArchetype: 'FinancialAmount',
    aliases: ['px_bid', 'bid_px', 'bid_price', 'bid'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'PX_ASK',
    canonicalTermName: 'AskPrice',
    category: 'Pricing & Valuation',
    description: 'Current or closing ask / offer price for the security.',
    dataType: 'numeric',
    universalArchetype: 'FinancialAmount',
    aliases: ['px_ask', 'ask_px', 'ask_price', 'ask', 'offer_px', 'offer_price'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'PX_MID',
    canonicalTermName: 'MidPrice',
    category: 'Pricing & Valuation',
    description: 'Midpoint between bid and ask prices.',
    dataType: 'numeric',
    universalArchetype: 'FinancialAmount',
    aliases: ['px_mid', 'mid_px', 'mid_price', 'mid'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'PX_OPEN',
    canonicalTermName: 'OpenPrice',
    category: 'Pricing & Valuation',
    description: 'Opening price of the security on the exchange for the current trading day.',
    dataType: 'numeric',
    universalArchetype: 'FinancialAmount',
    aliases: ['px_open', 'open_px', 'open_price', 'open'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'PX_HIGH',
    canonicalTermName: 'HighPrice',
    category: 'Pricing & Valuation',
    description: 'Highest price at which the security traded during the current trading day.',
    dataType: 'numeric',
    universalArchetype: 'FinancialAmount',
    aliases: ['px_high', 'high_px', 'high_price', 'high', 'day_high'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'PX_LOW',
    canonicalTermName: 'LowPrice',
    category: 'Pricing & Valuation',
    description: 'Lowest price at which the security traded during the current trading day.',
    dataType: 'numeric',
    universalArchetype: 'FinancialAmount',
    aliases: ['px_low', 'low_px', 'low_price', 'low', 'day_low'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'PX_VOLUME',
    canonicalTermName: 'TradingVolume',
    category: 'Pricing & Valuation',
    description: 'Total number of shares or units traded during the day.',
    dataType: 'numeric',
    aliases: ['px_volume', 'volume', 'trading_volume', 'vol', 'total_volume', 'shares_traded'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'VWAP',
    canonicalTermName: 'VolumeWeightedAveragePrice',
    category: 'Pricing & Valuation',
    description: 'Volume Weighted Average Price for the trading session.',
    dataType: 'numeric',
    universalArchetype: 'FinancialAmount',
    aliases: ['vwap', 'px_vwap', 'vol_weighted_avg_px'],
    feedType: 'Data License',
  },

  // ── 2. Symbology & Identifiers
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'ID_ISIN',
    canonicalTermName: 'ISIN',
    category: 'Symbology & Identifiers',
    description: 'International Securities Identification Number (12-character alpha-numeric code).',
    dataType: 'string',
    aliases: ['id_isin', 'isin', 'isin_code', 'isin_id', 'isin_number', 'security_isin'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'ID_CUSIP',
    canonicalTermName: 'CUSIP',
    category: 'Symbology & Identifiers',
    description: 'Committee on Uniform Securities Identification Procedures 9-character identifier.',
    dataType: 'string',
    aliases: ['id_cusip', 'cusip', 'cusip_code', 'cusip_id', 'cusip_number', 'security_cusip'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'ID_SEDOL1',
    canonicalTermName: 'SEDOL',
    category: 'Symbology & Identifiers',
    description: 'Stock Exchange Daily Official List 7-character UK & European identifier.',
    dataType: 'string',
    aliases: ['id_sedol1', 'id_sedol', 'sedol', 'sedol_code', 'sedol_id', 'sedol_number'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'ID_BB_GLOBAL',
    canonicalTermName: 'FIGI',
    category: 'Symbology & Identifiers',
    description: 'Bloomberg Global Identifier (Financial Instrument Global Identifier / FIGI).',
    dataType: 'string',
    aliases: ['id_bb_global', 'bbg_id', 'figi', 'figi_code', 'figi_id', 'bb_global_id', 'composite_id_bb_global'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'TICKER',
    canonicalTermName: 'PrimaryTicker',
    category: 'Symbology & Identifiers',
    description: 'Primary exchange ticker symbol of the security.',
    dataType: 'string',
    aliases: ['ticker', 'symbol', 'primary_ticker', 'stock_symbol', 'ticker_symbol', 'id_exch_symbol'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'ID_MIC_PRIM_EXCH',
    canonicalTermName: 'MarketIdentifierCode',
    category: 'Symbology & Identifiers',
    description: 'ISO 10383 Market Identifier Code (MIC) for primary trading exchange.',
    dataType: 'string',
    aliases: ['id_mic_prim_exch', 'mic', 'mic_code', 'exchange_mic', 'primary_mic'],
    feedType: 'Data License',
  },

  // ── 3. Security Master & Reference
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'SECURITY_DES',
    canonicalTermName: 'SecurityDescription',
    category: 'Security Master & Reference',
    description: 'Standard descriptive text and instrument moniker for the security.',
    dataType: 'string',
    aliases: ['security_des', 'sec_desc', 'security_description', 'instrument_description', 'security_name', 'instrument_name', 'sec_name'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'CRNCY',
    canonicalTermName: 'PricingCurrency',
    category: 'Security Master & Reference',
    description: 'ISO 4217 currency code in which the security price is quoted.',
    dataType: 'string',
    aliases: ['crncy', 'currency', 'ccy', 'pricing_currency', 'trade_currency', 'settle_currency', 'iso_currency'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'CNTRY_ISSUE_ISO',
    canonicalTermName: 'CountryOfIssuance',
    category: 'Security Master & Reference',
    description: 'Two-character ISO 3166 country code where security was issued.',
    dataType: 'string',
    aliases: ['cntry_issue_iso', 'country_of_issue', 'issue_country', 'domicile_country', 'country_iso'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'SECURITY_TYP',
    canonicalTermName: 'SecurityType',
    category: 'Security Master & Reference',
    description: 'Classification of security type (Common Stock, Corporate Bond, Treasury, Option, Future).',
    dataType: 'string',
    aliases: ['security_typ', 'security_type', 'sec_type', 'instrument_type', 'asset_class', 'asset_type'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'GICS_SECTOR_NAME',
    canonicalTermName: 'GICSSector',
    category: 'Security Master & Reference',
    description: 'Global Industry Classification Standard (GICS) Sector Name.',
    dataType: 'string',
    aliases: ['gics_sector_name', 'gics_sector', 'sector_name', 'industry_sector', 'sector', 'gics_sec'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'GICS_INDUSTRY_GROUP_NAME',
    canonicalTermName: 'GICSIndustryGroup',
    category: 'Security Master & Reference',
    description: 'Global Industry Classification Standard (GICS) Industry Group Name.',
    dataType: 'string',
    aliases: ['gics_industry_group_name', 'gics_industry_group', 'industry_group', 'gics_grp'],
    feedType: 'Data License',
  },

  // ── 4. Fixed Income & Rates Analytics
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'YLD_YTM_MID',
    canonicalTermName: 'YieldToMaturity',
    category: 'Fixed Income & Rates',
    description: 'Yield to Maturity calculated from midpoint price of bond.',
    dataType: 'numeric',
    aliases: ['yld_ytm_mid', 'ytm', 'yield_to_maturity', 'ytm_mid', 'bond_yield', 'gross_yield'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'DUR_ADJ_MID',
    canonicalTermName: 'ModifiedDuration',
    category: 'Fixed Income & Rates',
    description: 'Modified duration calculated at mid price (percentage price change per 100bp yield shift).',
    dataType: 'numeric',
    aliases: ['dur_adj_mid', 'mod_dur', 'modified_duration', 'duration_mid', 'adj_duration', 'duration'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'CONVEXITY_MID',
    canonicalTermName: 'Convexity',
    category: 'Fixed Income & Rates',
    description: 'Bond convexity calculated at mid price.',
    dataType: 'numeric',
    aliases: ['convexity_mid', 'convexity', 'bond_convexity'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'CPN',
    canonicalTermName: 'CouponRate',
    category: 'Fixed Income & Rates',
    description: 'Annual interest rate paid by fixed income security expressed as a percentage.',
    dataType: 'numeric',
    aliases: ['cpn', 'coupon', 'coupon_rate', 'interest_rate', 'cpn_rate'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'CPN_TYP',
    canonicalTermName: 'CouponType',
    category: 'Fixed Income & Rates',
    description: 'Type of coupon payment (Fixed, Floating, Zero Coupon, Step-Up).',
    dataType: 'string',
    aliases: ['cpn_typ', 'coupon_type', 'rate_type'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'MATURITY',
    canonicalTermName: 'MaturityDate',
    category: 'Fixed Income & Rates',
    description: 'Date on which principal amount of fixed income security becomes due.',
    dataType: 'date',
    universalArchetype: 'AuditTimestamp',
    aliases: ['maturity', 'maturity_date', 'mat_dt', 'redemption_date', 'final_maturity'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'ISSUE_DT',
    canonicalTermName: 'IssueDate',
    category: 'Fixed Income & Rates',
    description: 'Date when security was originally issued.',
    dataType: 'date',
    universalArchetype: 'AuditTimestamp',
    aliases: ['issue_dt', 'issue_date', 'issuance_date', 'origination_date'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'RATING_SP',
    canonicalTermName: 'StandardAndPoorsRating',
    category: 'Fixed Income & Rates',
    description: 'Current credit rating assigned by S&P Global Ratings.',
    dataType: 'string',
    aliases: ['rating_sp', 'sp_rating', 'sandp_rating', 'credit_rating_sp'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'RATING_MOODY',
    canonicalTermName: 'MoodysRating',
    category: 'Fixed Income & Rates',
    description: 'Current credit rating assigned by Moody’s Investors Service.',
    dataType: 'string',
    aliases: ['rating_moody', 'moodys_rating', 'moody_rating', 'credit_rating_moodys'],
    feedType: 'Data License',
  },

  // ── 5. Equities & Fundamentals
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'CUR_MKT_CAP',
    canonicalTermName: 'MarketCapitalization',
    category: 'Equities & Fundamentals',
    description: 'Current market value of the company’s total outstanding shares.',
    dataType: 'numeric',
    universalArchetype: 'FinancialAmount',
    aliases: ['cur_mkt_cap', 'mkt_cap', 'market_cap', 'market_capitalization', 'total_mkt_cap'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'PE_RATIO',
    canonicalTermName: 'PriceEarningsRatio',
    category: 'Equities & Fundamentals',
    description: 'Price to Earnings ratio (Trailing 12-Month Price / EPS).',
    dataType: 'numeric',
    aliases: ['pe_ratio', 'pe', 'p_e_ratio', 'price_to_earnings', 'trailing_pe'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'PX_TO_BOOK_RATIO',
    canonicalTermName: 'PriceToBookRatio',
    category: 'Equities & Fundamentals',
    description: 'Price to Book Value per Share ratio.',
    dataType: 'numeric',
    aliases: ['px_to_book_ratio', 'pb_ratio', 'price_to_book', 'pb', 'p_b_ratio'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'EQY_DVD_YLD_EST',
    canonicalTermName: 'DividendYield',
    category: 'Equities & Fundamentals',
    description: 'Indicated gross annual dividend yield expressed as a percentage.',
    dataType: 'numeric',
    aliases: ['eqy_dvd_yld_est', 'dvd_yield', 'div_yield', 'dividend_yield', 'yield_dividend'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'TRAIL_12M_EPS',
    canonicalTermName: 'EarningsPerShareTrailing',
    category: 'Equities & Fundamentals',
    description: 'Trailing 12-Month Earnings Per Share before extraordinary items.',
    dataType: 'numeric',
    universalArchetype: 'FinancialAmount',
    aliases: ['trail_12m_eps', 'eps', 'trailing_eps', 'earnings_per_share'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'EBITDA',
    canonicalTermName: 'EBITDA',
    category: 'Equities & Fundamentals',
    description: 'Earnings Before Interest, Taxes, Depreciation, and Amortization.',
    dataType: 'numeric',
    universalArchetype: 'FinancialAmount',
    aliases: ['ebitda', 'operating_ebitda'],
    feedType: 'Data License',
  },

  // ── 6. Corporate Actions & Ownership
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'DVD_RECORD_DT',
    canonicalTermName: 'DividendRecordDate',
    category: 'Corporate Actions',
    description: 'Record date on which shareholders are entitled to receive dividend.',
    dataType: 'date',
    universalArchetype: 'AuditTimestamp',
    aliases: ['dvd_record_dt', 'div_record_date', 'record_date', 'dvd_record_date'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'DVD_PAY_DT',
    canonicalTermName: 'DividendPayDate',
    category: 'Corporate Actions',
    description: 'Payment date on which declared dividend is distributed.',
    dataType: 'date',
    universalArchetype: 'AuditTimestamp',
    aliases: ['dvd_pay_dt', 'div_pay_date', 'pay_date', 'payment_date', 'distribution_date'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'SPLIT_RATIO',
    canonicalTermName: 'StockSplitRatio',
    category: 'Corporate Actions',
    description: 'Ratio of new shares distributed to old shares in a stock split.',
    dataType: 'string',
    aliases: ['split_ratio', 'stock_split_ratio', 'share_split_ratio'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'HOLDINGS_SHARES',
    canonicalTermName: 'PortfolioHoldingShares',
    category: 'Corporate Actions',
    description: 'Quantity of shares or units held in institutional position.',
    dataType: 'numeric',
    aliases: ['holdings_shares', 'shares_held', 'quantity', 'position_qty', 'holding_units'],
    feedType: 'Data License',
  },

  // ── 7. Entities & Counterparties
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'LEI_CODE',
    canonicalTermName: 'LegalEntityIdentifier',
    category: 'Entities & Counterparties',
    description: 'ISO 17442 Legal Entity Identifier (20-character alphanumeric global LEI).',
    dataType: 'string',
    aliases: ['lei_code', 'lei', 'legal_entity_identifier', 'issuer_lei', 'counterparty_lei'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'ISSUER',
    canonicalTermName: 'IssuerLegalName',
    category: 'Entities & Counterparties',
    description: 'Official legal entity name of the issuing company or institution.',
    dataType: 'string',
    universalArchetype: 'PersonName',
    aliases: ['issuer', 'issuer_name', 'issuing_entity', 'counterparty_name', 'obligor_name'],
    feedType: 'Data License',
  },
  {
    vendor: 'BLOOMBERG',
    mnemonic: 'ULT_PARENT_CNTRY_OF_RISK',
    canonicalTermName: 'CountryOfRisk',
    category: 'Entities & Counterparties',
    description: 'ISO two-letter country code of the ultimate parent entity where primary credit risk resides.',
    dataType: 'string',
    aliases: ['ult_parent_cntry_of_risk', 'country_of_risk', 'risk_country', 'ultimate_risk_country'],
    feedType: 'Data License',
  }
];

/**
 * Fast Lookup Map: Normalized Alias / Mnemonic -> Bloomberg Field Definition
 */
const BLOOMBERG_LOOKUP_MAP = new Map<string, VendorFieldDefinition>();

// Build fast lookup index
BLOOMBERG_DATA_DICTIONARY.forEach(field => {
  const normMnemonic = field.mnemonic.toLowerCase().replace(/[^a-z0-9]/g, '');
  BLOOMBERG_LOOKUP_MAP.set(normMnemonic, field);

  field.aliases.forEach(alias => {
    const normAlias = alias.toLowerCase().replace(/[^a-z0-9]/g, '');
    if (normAlias) {
      BLOOMBERG_LOOKUP_MAP.set(normAlias, field);
    }
  });
});

/**
 * Look up a column name or term name against the official Bloomberg Data Dictionary.
 * Performs exact, normalized, and token-stripped matching.
 */
export function lookupBloombergField(name: string): VendorFieldDefinition | undefined {
  if (!name) return undefined;
  const cleaned = name.trim().toLowerCase();
  const normalized = cleaned.replace(/[^a-z0-9]/g, '');

  // 1. Direct normalized lookup
  if (BLOOMBERG_LOOKUP_MAP.has(normalized)) {
    return BLOOMBERG_LOOKUP_MAP.get(normalized);
  }

  // 2. Try removing common table prefixes (e.g. "prices_px_last" -> "px_last")
  const parts = cleaned.split(/[._\s-]+/);
  for (let i = 0; i < parts.length; i++) {
    const sub = parts.slice(i).join('');
    if (BLOOMBERG_LOOKUP_MAP.has(sub)) {
      return BLOOMBERG_LOOKUP_MAP.get(sub);
    }
  }

  // 3. Try finding exact mnemonic inside string
  for (const field of BLOOMBERG_DATA_DICTIONARY) {
    const mn = field.mnemonic.toLowerCase();
    if (cleaned === mn || cleaned.endsWith(`_${mn}`) || cleaned.endsWith(`.${mn}`)) {
      return field;
    }
  }

  return undefined;
}

/**
 * Lookup generic vendor field across supported vendor dictionaries
 */
export function lookupVendorField(vendor: FinancialVendor, token: string): VendorFieldDefinition | undefined {
  switch (vendor) {
    case 'BLOOMBERG':
      return lookupBloombergField(token);
    // Extensible placeholders for future vendor additions
    case 'FACTSET':
    case 'LSEG':
    default:
      return undefined;
  }
}
