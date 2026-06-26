/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the settlement rate options as specified in the Annex A to the 1998 FX and Currency Options Definitions.
   */
  
  const (
  /**
   * The Spot Rate for a Rate Calculation Date will be the Argentine Peso/U.S. Dollar Specified Rate, expressed as the amount of Argentine Pesos per one U.S. Dollar, for settlement on the same day (or, if such day is not a Business Day in New York, for settlement on the first succeeding day that is a Business Day in both Buenos Aires and New York) which appears on the Reuters Screen BNAR Page at the close of business in Buenos Aires on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_ARS_BNAR_ARS01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Argentine Peso/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Argentine Pesos per one U.S. Dollar, for settlement on the same day, as published on EMTA's web site (www.emta.org) at approximately 1:00 p.m. (Buenos Aires time), or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate shall be calculated by EMTA (or a service provider EMTA may select in its sole discretion) pursuant to the EMTA ARS Indicative Survey Methodology (which means a methodology, dated as of January 2, 2003, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Argentine Peso/U.S. Dollar markets for the purpose of determining the EMTA ARS Indicative Survey Rate).
   */
  SettlementRateOptionEnum_ARS_EMTA_INDICATIVE_SURVEY_RATE_ARS04 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Argentine Peso/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Argentine Pesos per one U.S. Dollar, for settlement on the same day, as published on EMTA's web site (www.emta.org) at approximately 1:00 p.m. (Buenos Aires time), or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate shall be calculated by EMTA (or a service provider EMTA may select in its sole discretion) pursuant to the EMTA ARS Industry Survey Methodology (which means a methodology, dated as of January 2, 2003, as amended from time to time, for a centralized industry-wide survey of financial institutions in Buenos Aires that are active participants in the Argentine Peso/U.S. Dollar spot markets for the purpose of determining the EMTA ARS Industry Survey Rate).
   */
  SettlementRateOptionEnum_ARS_EMTA_INDUSTRY_SURVEY_RATE_ARS03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the volume weighted average Argentine Peso/U.S. Dollar Rate of all trades executed in the electronic market for a Rate Calculation Day expressed as the amount of Argentine Pesos per one U.S. Dollar, for settlement on the same day, reported by the Mercado Abierto Electronico (the 'MAE') at approximately 3:00 pm, Buenos Aires time, and published on the FOREX-MAE Page as the 'PPN' rate ('Promedio Ponderado Noticiado') on www.mae.com.ar on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_ARS_MAE_ARS05 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Argentine Peso/U.S. Dollar offered rate for U.S. Dollars, expressed as the amount of Argentine Pesos per one U.S. Dollar, for settlement on the same day quoted by Banco de la Nacion (in accordance with the Convertibility Law of March 27, 1991 and Regulatory Decree No. 529/91 of April 1, 1991, as may be amended from time to time) for that Rate Calculation Date.
   */
  SettlementRateOptionEnum_ARS_OFFICIAL_RATE_ARS02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Brazilian Real/U.S. Dollar Specified Rate, expressed as the amount of Brazilian Reais per one U.S. Dollar, for settlement in two Business Days (where such days are Business Days in both Sao Paulo and New York) which appears on the Reuters Screen BRBY Page under the caption 'INTBK FLTING (LAST)' at approximately 11:00 a.m., Sao Paulo time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_BRL_BRBY_BRL01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Brazilian Real/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Brazilian Reais per one U.S. Dollar, for settlement in two Business Days, as published on EMTA's web site (www.emta.org) at approximately 12:00 p.m. (Sao Paulo time), or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate shall be calculated by EMTA (or a service provider EMTA may select in its sole discretion) pursuant to the EMTA BRL Indicative Survey Methodology (which means a methodology, dated as of March 1, 2004, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Brazilian Real/U.S. Dollar markets for the purpose of determining the EMTA BRL Indicative Survey Rate).
   */
  SettlementRateOptionEnum_BRL_EMTA_INDICATIVE_SURVEY_RATE_BRL13 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Brazilian Real/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Brazilian Reais per one U.S. Dollar, for settlement in two Business Days, as published on EMTA's web site (www.emta.org) at approximately 3:45 p.m. (Sao Paulo time), or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate shall be calculated by EMTA (or a service provider EMTA may select in its sole discretion) pursuant to the EMTA BRL Industry Survey Methodology (which means a methodology, dated as of March 1, 2004, as amended from time to time, for a centralized industry-wide survey of financial institutions in Brazil that are active participants in the Brazilian Real/U.S. Dollar spot markets for the purpose of determining the EMTA BRL Industry Survey Rate).
   */
  SettlementRateOptionEnum_BRL_EMTA_INDUSTRY_SURVEY_RATE_BRL12 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Brazilian Real/U.S. Dollar Specified Rate, expressed as the amount of Brazilian Reais per one U.S. Dollar, for settlement in two Business Days (where such days are Business Days in both Sao Paulo and New York) reported by the Banco Central do Brasil in the 'Diario Oficial da Uniao' on the first Business Day following that Rate Calculation Date.
   */
  SettlementRateOptionEnum_BRL_OFFICIAL_RATE_BRL02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Brazilian Real/U.S. Dollar commercial rate, expressed as the amount of Brazilian Reais per one U.S. Dollar, for settlement in two Business Days (where such days are Business Days in both Sao Paulo and New York) reported by the Banco Central do Brasil on SISBACEN Data System under transaction code PCOT- 390, Option 3, at the Specified Time, if any, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_BRL_PCOT_COMMERCIAL_BRL03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Brazilian Real/U.S. Dollar floating rate, expressed as the amount of Brazilian Reais per one U.S. Dollar, for settlement in two Business Days (where such days are Business Days in both Sao Paulo and New York) reported by the Banco Central do Brasil on SISBACEN Data System under transaction code PCOT- 390, Option 3, at the Specified Time, if any, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_BRL_PCOT_FLOATING_BRL04 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Brazilian Real/U.S. Dollar offered rate for U.S. Dollars, expressed as the amount of Brazilian Reais per one U.S. Dollar, for settlement in two Business Days reported by the Banco Central do Brasil on SISBACEN Data System under transaction code PTAX-800 ('Consulta de Cambio' or Exchange Rate Inquiry), Option 5 ('Cotacoes para Contabilidade' or 'Rates for Accounting Purposes') by approximately 6:00 p.m., Sao Paulo time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_BRL_PTAX_BRL09 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Brazilian Real/U.S. Dollar commercial rate, expressed as the amount of Brazilian Reais per one U.S. Dollar, for settlement in two Business Days (where such days are Business Days in both Sao Paulo and New York) reported by the Banco Central do Brasil which appears on the Reuters Screen BRFR Page at PTAX-800 as of 11:00 a.m., Sao Paulo time, on the first Business Day following that Rate Calculation Date. 23
   */
  SettlementRateOptionEnum_BRL_PTAX_COMMERCIAL_BRFR_BRL06 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Brazilian Real/U.S. Dollar commercial rate, expressed as the amount of Brazilian Reais per one U.S. Dollar, for settlement in two Business Days (where such days are Business Days in both Sao Paulo and New York) reported by the Banco Central do Brasil on SISBACEN Data System under transaction code PTAX- 800 ('Consultas de Cambio' or Exchange Rate Inquiry), Option 5 ('Cotacoes para Contabilidad' or Rates for Accounting Purposes) market type 'L' (corresponding to U.S. Dollars traded in the foreign exchange market segment officially denominated 'Livre' and commonly known as 'Comercial') as of 7:30 p.m., Sao Paulo time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_BRL_PTAX_COMMERCIAL_BRL05 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Brazilian Real/U.S. Dollar floating rate, expressed as the amount of Brazilian Reais per one U.S. Dollar, for settlement in two Business Days (where such days are Business Days in both Sao Paulo and New York) reported by the Banco Central do Brasil on the SISBACEN Data System which appears on the Reuters Screen BRFR Page at PTAX-800 as of 11:00 a.m., Sao Paulo time, on the first Business Day following that Rate Calculation Date.
   */
  SettlementRateOptionEnum_BRL_PTAX_FLOATING_BRFR_BRL08 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Brazilian Real/U.S. Dollar floating rate, expressed as the amount of Brazilian Reais per one U.S. Dollar, for settlement in two Business Days (where such days are Business Days in both Sao Paulo and New York) reported by the Banco Central do Brasil on SISBACEN Data System under transaction code PTAX- 800 ('Consultas de Cambio' or Exchange Rate Inquiry), Option 5 ('Cotacoes para Contabilidad' or Rates for Accounting Purposes) market type 'F' (corresponding to U.S. Dollars traded in the foreign exchange market segment officially denominated 'Flutuante') as of 7:30 p.m., Sao Paulo time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_BRL_PTAX_FLOATING_BRL07 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chilean Peso/U.S. Dollar observado rate, expressed as the amount of Chilean Pesos per one U.S. Dollar, for settlement on the same day (or, if such day is not a Business Day in New York, for settlement on the first succeeding day that is a Business Day in both Santiago and New York) reported by the Banco Central de Chile which appears on the Reuters Screen BCCH Page under the caption 'OBSERVADO' at 10:00 a.m., Santiago time, on the first Business Day following that Rate Calculation Date.
   */
  SettlementRateOptionEnum_CLP_BCCH_CLP01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chilean Peso/U.S. Dollar informal rate, expressed as the amount of Chilean Pesos per one U.S. Dollar, for settlement on the same day (or, if such day is not a Business Day in New York, for settlement on the first succeeding day that is a Business Day in both Santiago and New York) of the informal exchange market which appears on the Reuters Screen CHILD Page at the Specified Time, if any, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_CLP_CHILD_INFORMAL_CLP02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chilean Peso/U.S. Dollar interbank rate, expressed as the amount of Chilean Pesos per one U.S. Dollar, for settlement on the same day (or, if such day is not a Business Day in New York, for settlement on the first succeeding day that is a Business Day in both Santiago and New York) reported by the Banco Central de Chile for the formal exchange market which appears on the Reuters Screen CHILD Page at the Specified Time, if any, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_CLP_CHILD_INTERBANK_CLP03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chilean Peso/U.S. Dollar observado rate, expressed as the amount of Chilean Pesos per one U.S. Dollar, for settlement on the same day (or, if such day is not a Business Day in New York, for settlement on the first succeeding day that is a Business Day in both Santiago and New York) reported by the Banco Central de Chile which appears on the Reuters Screen CHILD Page on the first Business Day following that Rate Calculation Date.
   */
  SettlementRateOptionEnum_CLP_CHILD_OBSERVADO_CLP04 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chilean Peso/U.S. Dollar informal rate, expressed as the amount of Chilean Pesos per one U.S. Dollar, for settlement on the same day (or, if such day is not a Business Day in New York, for settlement on the first succeeding day that is a Business Day in both Santiago and New York) of the informal exchange market which appears on the Reuters Screen CHILG Page at the Specified Time, if any, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_CLP_CHILG_INFORMAL_CLP05 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chilean Peso/U.S. Dollar interbank rate, expressed as the amount of Chilean Pesos per one U.S. Dollar, for settlement on the same day (or, if such day is not a Business Day in New York, for settlement on the first succeeding day that is a Business Day in both Santiago and New York) reported by the Banco Central de Chile for the formal exchange market which appears on the Reuters Screen CHILG Page at the Specified Time, if any, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_CLP_CHILG_INTERBANK_CLP06 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chilean Peso/U.S. Dollar observado rate, expressed as the amount of Chilean Pesos per one U.S. Dollar, for settlement on the same day (or, if such day is not a Business Day in New York, for settlement on the first succeeding day that is a Business Day in both Santiago and New York) reported by the Banco Central de Chile which appears on the Reuters Screen CHILG Page under 'OBSERVADO' at the Specified Time, if any, on the first Business Day following that Rate Calculation Date.
   */
  SettlementRateOptionEnum_CLP_CHILG_OBSERVADO_CLP07 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chilean Peso/U.S. Dollar 'observado' rate, expressed as the amount of Chilean Pesos per one U.S. Dollar, for settlement in one Business Day reported by the Banco Central de Chile (www.bcentral.cl) as the 'Dolar Observado' (Dollar Observado) rate by not later than 10:30 a.m., Santiago time, on the first Business Day following that Rate Calculation Date.
   */
  SettlementRateOptionEnum_CLP_DOLAR_OBS_CLP10 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chilean Peso/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Chilean Pesos per one U.S. Dollar, for settlement on the same day, as published on EMTA's web site (www.emta.org) at approximately 11:00 a.m., Santiago time, or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate shall be calculated by EMTA (or a service provider EMTA may select in its sole discretion) pursuant to the EMTA CLP Indicative Survey Methodology (which means a methodology, dated as of August 1, 2006, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Chilean Peso/U.S. Dollar markets for the purpose of determining the EMTA CLP Indicative Survey Rate).
   */
  SettlementRateOptionEnum_CLP_EMTA_INDICATIVE_SURVEY_RATE_CLP11 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chilean Peso/U.S. Dollar Specified Rate, expressed as the amount of Chilean Pesos per one U.S. Dollar (or, if such day is not a Business Day in New York, for settlement on the first succeeding day that is a Business Day in both Santiago and New York), calculated in accordance with Title I, Chapter 1 Number 6 of the Compendium of International Exchange Norms of the Banco Central de Chile and published by the Banco Central de Chile at the Specified Time, if any, on the first Business Day following that Rate Calculation Date.
   */
  SettlementRateOptionEnum_CLP_OFFICIAL_RATE_CLP08 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chilean Peso/U.S. Dollar observado rate, expressed as the amount of Chilean Pesos per one U.S. Dollar, for settlement on the same day (or, if such day is not a Business Day in New York, for settlement on the first succeeding day that is a Business Day in both Santiago and New York) reported by the Banco Central de Chile which appears on the Telerate Page 38942 opposite the caption 'Observado' at the Specified Time, if any, on the first Business Day following the Rate Calculation Date.
   */
  SettlementRateOptionEnum_CLP_TELERATE_38942_CLP09 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chinese Renminbi/U.S. Dollar official fixing rate, expressed as the amount of Chinese Renminbi per one U.S. Dollar, for settlement in two Business Days reported by the People's Bank of China, Beijing, People's Republic of China, which appears on the Reuters Screen 'SAEC' Page opposite the symbol 'USDCNY=' at approximately 9:15 a.m., Beijing time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_CNY_SAEC_CNY01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Chinese Renminbi/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Chinese Renminbi per one U.S. Dollar, for settlement in two Business Days, as published on SFEMC's website (www.sfemc.org) at approximately 3:30 p.m. (Singapore time), or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate will be calculated by SFEMC (or a service provider SFEMC may select in its sole discretion) pursuant to the SFEMC CNY Indicative Survey Methodology (which means a methodology, dated as of December 1, 2004, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Chinese Renminbi/U.S. Dollar markets for the purpose of determining the SFEMC CNY Indicative Survey Rate).
   */
  SettlementRateOptionEnum_CNY_SFEMC_INDICATIVE_SURVEY_RATE_CNY02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Colombian Peso/U.S. Dollar fixing rate, expressed as the amount of Colombian Pesos per one U.S. Dollar, for settlement on the same day (unless such day is not a Business Day in New York, then for settlement on the first succeeding day that is a Business Day in Bogota and New York) reported by the Colombian Banking Superintendency which appears on the Reuters Screen CO/COL03 Page opposite the caption 'TRCM' ('Tasa de Cierre Representative del Mercado' or closing market price) at 12:00 noon, Bogota time, on the first Business Day following that Rate Calculation Date.
   */
  SettlementRateOptionEnum_COP_CO_COL03_COP01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Colombian Peso/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Colombian Pesos per one U.S. Dollar, for settlement on the same day, as published on EMTA's web site (www.emta.org) at approximately 11:30 a.m., Bogota time, or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate shall be calculated by EMTA (or a service provider EMTA may select in its sole discretion) pursuant to the EMTA COP Indicative Survey Methodology (which means a methodology, dated as of August 1, 2006, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Colombian Peso/U.S. Dollar markets for the purpose of determining the EMTA COP Indicative Survey Rate).
   */
  SettlementRateOptionEnum_COP_EMTA_INDICATIVE_SURVEY_RATE_COP03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Colombian Peso/U.S. Dollar fixing rate, expressed as the amount of Colombian Pesos per one U.S. Dollar, for settlement on the same day reported by the Colombian Financial Superintendency (www.banrep.gov.co) as the 'Tasa Representativa del Mercado (TRM)' (also referred to as the 'Tasa de Cambio Representativa del Mercado' (TCRM)) by not later than 10:30 a.m., Bogota time, on the first Business Day following that Rate Calculation Date.
   */
  SettlementRateOptionEnum_COP_TRM_COP02 SettlementRateOptionEnum = iota + 1
  /**
   * the Spot Rate for a Rate Calculation Date will be the Reference Currency/U.S. Dollar exchange rate, expressed as the amount of Reference Currency per one U.S. Dollar, determined on the basis of quotations provided by Reference Dealers on that Rate Calculation Date of that day's price of a Specified Company's American Depositary Receipt or American Depositary Receipts (the 'ADR' or 'ADRs', as appropriate) and the price of the local share or shares of such Specified Company of the same type and in the same quantity represented by such ADR or ADRs, as the case may be (the 'Share' or 'Shares', as appropriate). The Calculation Agent will request each of the Reference Dealers to provide a firm quotation of (A) in the case where one ADR represents less than one Share, its bid and offer price (in the Reference Currency) for one Share and its bid and offer price (in U.S. Dollars) for the number of ADRs which represent such Share and (B) in all other cases, its bid and offer price (in the Reference Currency) for the Share or Shares, as the case may be, and its bid and offer price (in U.S. Dollars) for one ADR. If one or more quotations are provided, the rate for a Rate Calculation Date will equal the ratio of (1) the arithmetic mean of the midpoint of the bid and offer prices quoted in the Reference Currency by each Reference Dealer for such Share or Shares, as the case may be, and (2) the arithmetic mean of the midpoint of the bid and offer prices quoted in U.S. Dollars by each Reference Dealer for such ADR or ADRs, as the case may be, subject to an adjustment, if any, by the Calculation Agent to reduce the effect of momentary disparities in the prices of the Share or Shares and the ADR or ADRs, as appropriate. The quotations used to determine the Spot Rate for a Rate Calculation Date will be determined in each case at the Specified Time on the Rate Calculation Date or, if no such time is specified, the time chosen by the Calculation Agent.
   */
  SettlementRateOptionEnum_CURRENCY_IMPLIED_RATE__ADR__CURA1 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Reference Currency/Settlement Currency exchange rate, expressed as the amount of Reference Currency per one unit of Settlement Currency, determined on the basis of quotations provided by Reference Dealers on that Rate Calculation Date for that day's price of Local Assets. The Calculation Agent will request each of the Reference Dealers to provide a firm quotation of its bid and offer price (in both the Reference Currency and the Settlement Currency) for an amount of Local Assets whose face value equals the Specified Amount. If one or more quotations are provided, the rate for a Rate Calculation Date will equal the ratio of (A) the arithmetic mean of the midpoint of the bid and offer prices quoted in the Reference Currency by each Reference Dealer for such Local Assets and (B) the arithmetic mean of the midpoint of the bid and offer prices quoted in the Settlement Currency by each Reference Dealer for such Local Assets. The quotations used to determine the Spot Rate for a Rate Calculation Date will be determined in each case at the Specified Time on the Rate Calculation Date or, if no such time is specified, the time chosen by the Calculation Agent.
   */
  SettlementRateOptionEnum_CURRENCY_IMPLIED_RATE__LOCAL_ASSET__CURA2 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Reference Currency/Settlement Currency Specified Rate, expressed as the amount of the Reference Currency per one unit of Settlement Currency, for settlement on the Settlement Date agreed upon by the parties on or prior to that Rate Calculation Date (or, if different, the day on which rates for that date would, in the ordinary course, be published or announced).
   */
  SettlementRateOptionEnum_CURRENCY_MUTUAL_AGREEMENT_CURA3 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be determined on the basis of quotations provided by Reference Dealers on that Rate Calculation Date of that day's Specified Rate, expressed as the amount of Reference Currency per one unit of Settlement Currency, for settlement on the Settlement Date. The Calculation Agent will request the Specified Office of each of the Reference Dealers to provide a firm quotation of its Specified Rate for a transaction where the amount of Reference Currency equals the Specified Amount. If four quotations are provided, the rate for a Rate Calculation Date will be the arithmetic mean of the Specified Rates, without regard to the Specified Rates having the highest and lowest value. If exactly three quotations are provided, the rate for a Rate Calculation Date will be the Specified Rate provided by the Reference Dealer that remains after disregarding the Specified Rates having the highest and lowest values. For this purpose, if more than one quotation has the same highest value or lowest value, then the Specified Rate of one of such quotations shall be disregarded. If exactly two quotations are provided, the rate for a Rate Calculation Date will be the arithmetic mean of the Specified Rates. If only one quotation is provided, the rate for a Rate Calculation Date will be the Specified Rate quoted by that Reference Dealer. The quotations used to determine the Spot Rate for a Rate Calculation Date will be determined in each case at the Specified Time on that Rate Calculation Date or, if no such time is specified, the time chosen by the Calculation Agent.
   */
  SettlementRateOptionEnum_CURRENCY_REFERENCE_DEALERS_CURA4 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be determined by the Calculation Agent on the basis of that day's Specified Rate, expressed as the amount of Reference Currency per one unit of Settlement Currency, in a legal and customary wholesale market in which there is no, or minimal, Governmental Authority controls or interference, except as a participant in such market.
   */
  SettlementRateOptionEnum_CURRENCY_WHOLESALE_MARKET_CURA5 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Ecuadorian Sucre/U.S. Dollar Specified Rate, expressed as the amount of Ecuadorian Sucres per one U.S. Dollar, for settlement in one Business Day (where such day is a Business Day in Guayaquil and New York) which appears on Reuters Screen DNRP Page at 12:00 noon, Guayaquil time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_ECS_DNRP_ECS01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Indonesian Rupiah/U.S. Dollar spot rate at 11:00 a.m., Singapore time, expressed as the amount of Indonesian Rupiah per one U.S. Dollar, for settlement in two Business Days, reported by the Association of Banks in Singapore which appears on the Telerate Page 50157 to the right of the caption 'Spot' under the column 'IDR' at approximately 11:30 a.m., Singapore time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_IDR_ABS_IDR01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Indonesian Rupiah/U.S. Dollar weighted average spot rate in the interbank market based on traded IDR/USD spot foreign exchange transactions during a specified time period which are captured on a real time basis, expressed as the amount of Indonesian Rupiah per one U.S. Dollar, for settlement in two Business Days, published by Bank Indonesia at approximately 10:00 a.m., Jakarta time, on that Rate Calculation Date as the Jakarta Interbank Spot Dollar Rate USD - IDR on Bank Indonesia's website or otherwise made available by Bank Indonesia (or its successor as administrator).
   */
  SettlementRateOptionEnum_IDR_JISDOR_IDR04 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Indonesian Rupiah/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Indonesian Rupiah per one U.S. Dollar, for settlement in two Business Days, as published on SFEMC's website (www.sfemc.org) at approximately 3:30 p.m., Singapore time, or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate will be calculated by SFEMC (or a service provider SFEMC may select in its sole discretion) pursuant to the SFEMC IDR Indicative Survey Methodology (which means a methodology, dated as of December 1, 2004, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Indonesian Rupiah/U.S. Dollar markets for the purpose of determining the SFEMC IDR Indicative Survey Rate).
   */
  SettlementRateOptionEnum_IDR_SFEMC_INDICATIVE_SURVEY_RATE_IDR02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Indonesian Rupiah/U.S. Dollar implied spot rate expressed as the amount of Indonesian Rupiah per one U.S. Dollar, for settlement in two Business Days, reported by ABS Benchmarks Administration Co Pte. Ltd. (or its successor as administrator or sponsor of that rate), which appears on Thomson Reuters Screen ABSFIX01 Page at approximately 11:30 a.m., Singapore time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_IDR_VWAP_IDR03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Israeli Shekel/U.S. Dollar Specified Rate, expressed as the amount of Israeli Shekels per one U.S. Dollar, for settlement in two Business Days which appears on the Reuters Screen BOIJ Page as of 1:00 p.m., Tel Aviv time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_ILS_BOIJ_ILS01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Israeli Shekel/U.S. Dollar Specified Rate, expressed as the amount of Israeli Shekels per one U.S. Dollar, for settlement in two Business Days which appears on the Reuters Screen FXIL Page as of 1:00 p.m., Tel Aviv time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_ILS_FXIL_ILS02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Indian Rupee/U.S. Dollar reference rate, expressed as the amount of Indian Rupee per one U.S. Dollar, for settlement in two Business Days, reported by Financial Benchmarks India Pvt. Ltd. (www.fbil.org.in) at approximately 1:30 p.m., Mumbai time, or as soon thereafter as practicable, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_INR_FBIL_INR01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Indian Rupee/U.S. Dollar reference rate, expressed as the amount of Indian Rupee per one U.S. Dollar, for settlement in two Business Days reported by the Reserve Bank of India which appears on the Reuters Screen RBIB Page at approximately 12:30 p.m., Mumbai time, or as soon thereafter as practicable, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_INR_RBIB_INR01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Indian Rupee/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Indian Rupee per one U.S. Dollar, for settlement in two Business Days, as published on SFEMC's website (www.sfemc.org) at approximately 3:30 p.m. (Singapore time), or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate will be calculated by SFEMC (or a service provider SFEMC may select in its sole discretion) pursuant to the SFEMC INR Indicative Survey Methodology (which means a methodology, dated as of December 1, 2004, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Indian Rupee/U.S. Dollar markets for the purpose of determining the SFEMC INR Indicative Survey Rate).
   */
  SettlementRateOptionEnum_INR_SFEMC_INDICATIVE_SURVEY_RATE_INR02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Korean Won/U.S. Dollar Specified Rate, expressed as the amount of Korean Won per one U.S. Dollar, for settlement in two Business Days which appears on the Reuters Screen KEBEY Page at the Specified Time, if any, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_KRW_KEBEY_KRW01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Korean Won/U.S. Dollar market average rate, expressed as the amount of Korean Won per one U.S. Dollar, for settlement in two Business Days reported by the Korea Financial Telecommunications and Clearing Corporation which appears on the Reuters Screen KFTC18 Page to the right of the caption 'USD Today' that is available at approximately 3:30 p.m., Seoul time, on the Rate Calculation Date or as soon thereafter as practicable.
   */
  SettlementRateOptionEnum_KRW_KFTC18_KRW02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Korean Won/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Korean Won per one U.S. Dollar, for settlement in two Business Days, as published on SFEMC's website (www.sfemc.org) at approximately 3:30 p.m., Singapore time, or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate will be calculated by SFEMC (or a service provider SFEMC may select in its sole discretion) pursuant to the SFEMC KRW Indicative Survey Methodology (which means a methodology, dated as of December 1, 2004, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Korean Won/U.S. Dollar markets for the purpose of determining the SFEMC KRW Indicative Survey Rate).
   */
  SettlementRateOptionEnum_KRW_SFEMC_INDICATIVE_SURVEY_RATE_KRW04 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Korean Won/U.S. Dollar market average rate, expressed as the amount of Korean Won per one U.S. Dollar, for settlement in two Business Days reported by the Korea Financial Telecommunications and Clearing Corporation which appears on Telerate Page 45644 to the right of the caption 'USD Today' that is available at approximately 3:30 p.m., Seoul time, on the Rate Calculation Date or as soon thereafter as practicable.
   */
  SettlementRateOptionEnum_KRW_TELERATE_45644_KRW03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Kazakhstan Tenge / U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Kazakhstan Tenge per one U.S. Dollar, for settlement on the same Business Day, as published on EMTA's website (www.emta.org) at approximately 1:00 p.m., Almaty time, or as soon thereafter as practicable, on that Rate Calculation Date. The Spot Rate shall be calculated by EMTA (or a service provider EMTA may select in its sole discretion) pursuant to the EMTA KZT Indicative Survey Methodology (which means a methodology, dated as of March 16, 2009, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Kazakhstan Tenge/U.S. Dollar markets for the purpose of determining the EMTA KZT Indicative Survey Rate).
   */
  SettlementRateOptionEnum_KZT_EMTA_INDICATIVE_SURVEY_RATE_KZT02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Kazakhstan Tenge / U.S. Dollar weighted average rate, expressed as the amount of Kazakhstan Tenge per one U.S. Dollar, for settlement on the same Business Day reported by the Kazakhstan Stock Exchange (www.kase.kz) at approximately 11:00 am, Almaty time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_KZT_KASE_KZT01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Lebanese Pound/U.S. Dollar Specified Rate, expressed as the amount of Lebanese Pounds per one U.S. Dollar, for settlement in two Business Days which appears on the Reuters Screen BDLX Page as of 12:00 noon, Beirut time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_LBP_BDLX_LBP01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Moroccan Dirham/U.S. Dollar Specified Rate, expressed as the amount of Moroccan Dirham per one U.S. Dollar, for settlement in two Business Days reported by the Central Bank of Morocco as of 1:00 p.m., Rabat time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_MAD_OFFICIAL_RATE_MAD01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Mexican Pesos/U.S. Dollar Specified rate, expressed as the amount of Mexican Pesos per one U.S. Dollar, for settlement in two Business Days reported by Banco de Mexico which appears on the Reuters Screen BNMX Page opposite the caption 'Fix' at the close of business in Mexico City on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_MXP_BNMX_MXP01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Mexican Peso/U.S. Dollar fixing rate, expressed as the amount of Mexican Pesos per one U.S. Dollar, for settlement in two Business Days which is published by Banco de Mexico in the Official Gazette of the Federation pursuant to the 'Disposiciones aplicables a la determinacion del tipo de Cambio para solventar obligaciones denominadas en moneda extranjera pagaderas en la Republica Mexicana' (Rules applicable to determine the exchange rate to pay obligations denominated in foreign currency payable in Mexico) on the first Business Day following that Rate Calculation Date.
   */
  SettlementRateOptionEnum_MXP_FIXING_RATE_MXP02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Mexican Peso/U.S. Dollar fixing rate, expressed as the amount of Mexican Pesos per one U.S. Dollar, for settlement in two Business Days reported by Banco de Mexico which appears on Reuters Screen MEX01 Page under the heading 'MXNFIX=RR', at the close of business in Mexico City on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_MXP_MEX01_MXP03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Mexican Peso/U.S. Dollar fixing rate, expressed as the amount of Mexican Pesos per one U.S. Dollar, for settlement in two Business Days which is published by the Bolsa Mexicana de Valores, S.A. de C.V. (as established in Section 2 of the 'Resolution concerning the exchange rate applicable for calculating the Mexican Peso equivalent of principal and interest of Mexican Treasury Notes denominated in foreign currency and payable in Mexican Pesos' published in the Diario Oficial de la Federacion on November 11, 1991) in the Movimiento Diario del Mercado de Valores de la Bolsa Mexicana de Valores, S.A. de C.V. under the heading 'Movimiento Diario del Mercado de Valores' on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_MXP_PUBLISHED_MXP04 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Malaysian Ringgit/U.S. Dollar spot rate at 11:00 a.m., Singapore time, expressed as the amount of Malaysian Ringgit per one U.S. Dollar, for settlement in two Business Days, reported by the Association of Banks in Singapore, which appears on the Telerate Page 50157 to the right of the caption 'Spot' under the column 'MYR' at approximately 11:30 a.m., Singapore time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_MYR_ABS_MYR01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Malaysian Ringgit/U.S. Dollar reference rate, expressed as the amount of Malaysian Ringgit per one U.S. Dollar, for settlement in two Business Days, calculated and reported by Bank Negara Malaysia as its Kuala Lumpur USD/MYR Reference Rate, which appears on Thomson Reuters Screen MYRFIX2 Page at approximately 3:30 p.m., Kuala Lumpur time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_MYR_KL_REF_MYR04 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Malaysian Ringgit/U.S. Dollar spot rate expressed as the amount of Malaysian Ringgit per one U.S. Dollar, for settlement in two Business Days, reported by Persatuan Pasaran Kewangan Malaysia (ACI - Malaysia), which appears on Thomson Reuters Screen MYRFIX2 Page at approximately 11:10 a.m., Kuala Lumpur time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_MYR_PPKM_MYR03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Malaysian Ringgit/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Malaysian Ringgit per one U.S. Dollar, for settlement in two Business Days, as published on SFEMC's website (www.sfemc.org) at approximately 3:30 p.m., Singapore time, or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate will be calculated by SFEMC (or a service provider SFEMC may select in its sole discretion) pursuant to the SFEMC MYR Indicative Survey Methodology (which means a methodology, dated as of July 15, 2005, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Malaysian Ringgit/U.S. Dollar markets for the purpose of determining the SFEMC MYR Indicative Survey Rate).
   */
  SettlementRateOptionEnum_MYR_SFEMC_INDICATIVE_SURVEY_RATE_MYR02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Peruvian Sol/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Peruvian Soles per one U.S. Dollar, for settlement on the same day, as published on EMTA's web site (www.emta.org) at approximately 11:00 a.m., Lima time, or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate shall be calculated by EMTA (or a service provider EMTA may select in its sole discretion) pursuant to the EMTA PEN Indicative Survey Methodology (which means a methodology, dated as of August 1, 2006, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Peruvian Sol/U.S. Dollar markets for the purpose of determining the EMTA PEN Indicative Survey Rate).
   */
  SettlementRateOptionEnum_PEN_EMTA_INDICATIVE_SURVEY_RATE_PEN04 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Peruvian Sol/U.S. Dollar average exchange rate in the interbank market expressed as the amount of Peruvian New Soles per one U.S. Dollar for settlement on the same day reported by the Banco Central de Reserva del Peru (www.bcrp.gob.pe) as the 'Tipo de Cambio Interbancario Promedio' at approximately 2:00 p.m., Lima time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_PEN_INTERBANK_AVE_PEN05 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Peruvian Sol/U.S. Dollar fixing rate (mid market last), expressed as the amount of Peruvian Sols per one U.S. Dollar, for settlement on that same day which appears on the Reuters Screen PDSB Page opposite the caption 'PEN=' as of 12:00 noon, Lima time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_PEN_PDSB_PEN01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the midpoint of the Peruvian Sol/U.S. Dollar closing weighted average bid and offer ('compra y venta') exchange rates expressed as the amount of Peruvian New Soles per one U.S. Dollar for settlement on the same day, reported by the Superintendencia de Banca, Seguros y AFP (www.sbs.gob.pe) of the Republic of Peru at approximately 5:00 p.m., Lima time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_PEN_WT_AVE_PEN03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Philippine Peso/U.S. Dollar morning weighted average rate for that Rate Calculation Date, expressed as the amount of Philippine Pesos per one U.S. Dollar, for settlement in one Business Day, sponsored by Bankers Association of the Philippines (www.bap.org.ph) as its 'BAP AM Weighted Average Rate' at approximately 11:30 a.m., Manila time, or as soon thereafter as practicable, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_PHP_BAPPESO_PHP06 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Philippine Peso/U.S. Dollar morning weighted average rate for that Rate Calculation Date, expressed as the amount of Philippine Pesos per one U.S. Dollar, for settlement in one Business Day reported by the Philippine Dealing System PDEX which appears on the Reuters Screen PDSPESO Page to the right of the caption 'AM WT AVE' at approximately 11:30 a.m., Manila time, or as soon thereafter as practicable, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_PHP_PDSPESO_PHP06 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Philippine Peso/U.S. Dollar tom rate (mid market), expressed as the amount of Philippine Pesos per one U.S. Dollar, for settlement in one Business Day which appears on the Reuters Screen PHPESO Page at approximately 11:00 a.m., Manila time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_PHP_PHPESO_PHP01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Philippine Peso/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Philippine Pesos per one U.S. Dollar, for settlement in one Business Day, as published on SFEMC's website (www.sfemc.org) at approximately 3:30 p.m., Singapore time, or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate will be calculated by SFEMC (or a service provider SFEMC may select in its sole discretion) pursuant to the SFEMC PHP Indicative Survey Methodology (which means a methodology, dated as of December 1, 2004, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Philippine Peso/U.S. Dollar markets for the purpose of determining the SFEMC PHP Indicative Survey Rate).
   */
  SettlementRateOptionEnum_PHP_SFEMC_INDICATIVE_SURVEY_RATE_PHP05 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Philippine Peso/U.S. Dollar tom rate (mid market), expressed as the amount of Philippine Pesos per one U.S. Dollar, for settlement in one Business Day which appears on the Telerate Page 15439 at approximately 11:00 a.m., Manila time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_PHP_TELERATE_15439_PHP03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Philippine Peso/U.S. Dollar Specified Rate, expressed as the amount of Philippine Pesos per one U.S. Dollar, for settlement in one Business Day which appears on the Telerate Page 2920 at the Specified Time, if any, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_PHP_TELERATE_2920_PHP02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Pakistani Rupee/U.S. Dollar reference rate expressed as the amount of Pakistani Rupees per one U.S. Dollar, for settlement in two Business Days reported by the State Bank of Pakistan (www.sbp.org.pk) at approximately 2:30 pm, Karachi time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_PKR_SBPK_PKR01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Pakistani Rupee/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Pakistani Rupees per one U.S. Dollar, for settlement in two Business Days, as published on SFEMC's website (www.sfemc.org) at approximately 3:30 p.m. Singapore time, or as soon thereafter as practicable, on that Rate Calculation Date. The Spot Rate shall be calculated by SFEMC (or a service provider SFEMC may select in its sole discretion) pursuant to the SFEMC PKR Indicative Survey Methodology (which means a methodology, dated as of July 14, 2008, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Pakistani Rupee/U.S. Dollar markets for the purpose of determining the SFEMC PKR Indicative Survey Rate).
   */
  SettlementRateOptionEnum_PKR_SFEMC_INDICATIVE_SURVEY_RATE_PKR02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Polish Zloty/U.S. Dollar Specified Rate, expressed as the amount of Polish Zloty per one U.S. Dollar, for settlement in two Business Days reported by the National Bank of Poland which appears on the Reuters Screen NBPQ Page at the Specified Time, if any, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_PLZ_NBPQ_PLZ01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Polish Zloty/U.S. Dollar fixing rate, expressed as the amount of Polish Zloty per one U.S. Dollar, for settlement in two Business Days reported by the National Bank of Poland which appears on the Reuters Screen NBPR Page at the Specified Time, if any, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_PLZ_NBPR_PLZ02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Russian Ruble/U.S. Dollar Specified Rate, expressed as the amount of Russian Rubles per one U.S. Dollar, for settlement in one Business Day, calculated by the Chicago Mercantile Exchange ('CME') and as published on CME's website, which appears on the Reuters Screen EMTA Page, at approximately 1:30 p.m., Moscow time, on that Rate Calculation Date. The Spot Rate shall be calculated by the CME pursuant to the Chicago Mercantile Exchange / EMTA, Inc. Daily Russian Ruble Per U.S. Dollar Reference Rate Methodology (which means a methodology, effective as of June 16, 2005, as amended from time to time, for a centralized industry-wide survey of financial institutions in Russia that are active participants in the Russian Ruble/U.S. Dollar spot market for the purpose of determining the RUB CME-EMTA Rate).
   */
  SettlementRateOptionEnum_RUB_CME_EMTA_RUB03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Russian Ruble/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Russian Rubles per one U.S. Dollar, for settlement in one Business Day, as published on EMTA's web site (www.emta.org) at approximately 2:45 p.m., Moscow time, or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate shall be calculated by EMTA (or a service provider EMTA may select in its sole discretion) pursuant to the EMTA RUB Indicative Survey Methodology (which means a methodology dated as of June 16, 2005, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Russian Ruble/U.S. Dollar spot market for the purpose of determining the EMTA RUB Indicative Survey Rate).
   */
  SettlementRateOptionEnum_RUB_EMTA_INDICATIVE_SURVEY_RATE_RUB04 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Russian Ruble/U.S. Dollar Specified Rate, expressed as the amount of Russian Rubies per one U.S. Dollar, for settlement on the same day reported by the Moscow Interbank Currency Exchange which appears on the Reuters Screen MICEXFRX Page as of 10:30 a.m., Moscow time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_RUB_MICEXFRX_RUB01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Russian Ruble/U.S. Dollar Specified Rate, expressed as the amount of Russian Rubies per one U.S. Dollar, for settlement on the same day reported by the Moscow Interbank Currency Exchange which appears on the Reuters Screen MMVB Page as of 10:30 a.m., Moscow time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_RUB_MMVB_RUB02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Singapore Dollar/U.S. Dollar spot rate expressed as the amount of Singapore Dollar per one U.S. Dollar for settlement in two Business Days, reported by ABS Benchmarks Administration Co Pte. Ltd. (or its successor as administrator or sponsor of the rate), which appears on Thomson Reuters Screen ABSFIX01 Page at approximately 11:30 a.m., Singapore time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_SGD_VWAP_SGD3 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Slovak Koruna/U.S. Dollar Specified Rate, expressed as the amount of Slovak Koruna per one U.S. Dollar, for settlement in two Business Days reported by the National Bank of Slovakia which appears on the Reuters Screen NBSB Page as of 11:40 a.m., Bratislava time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_SKK_NBSB_SKK01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Thai Baht/U.S. Dollar spot rate at 11:00 a.m., Singapore time, expressed as the amount of Thai Bhaht per one U.S. Dollar, for settlement in two Business Days, reported by the Association of Banks in Singapore which appears on the Reuters Screen ABSIRFIX01 Page to the right of the caption 'Spot' under the column 'THB' at approximately 11:30 a.m., Singapore time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_THB_ABS_THB01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Thai Baht / U.S. Dollar spot rate expressed as the amount of Thai Baht per one U.S. Dollar for settlement in two Business Days, reported by ABS Benchmarks Administration Co Pte. Ltd. (or its successor as administrator or sponsor of the rate), which appears on Thomson Reuters Screen ABSFIX01 Page at approximately 11:30 a.m., Singapore time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_THB_VWAP_THB01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Taiwanese Dollar/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Taiwanese Dollars per one U.S. Dollar, for settlement in two Business Days, as published on SFEMC's website (www.sfemc.org) at approximately 3:30 p.m., Singapore time, or as soon thereafter as practicable, on such Rate Calculation Date. The Spot Rate will be calculated by SFEMC (or a service provider SFEMC may select in its sole discretion) pursuant to the SFEMC TWD Indicative Survey Methodology (which means a methodology, dated as of December 1, 2004, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Taiwanese Dollar/U.S. Dollar markets for the purpose of determining the SFEMC TWD Indicative Survey Rate).
   */
  SettlementRateOptionEnum_TWD_SFEMC_INDICATIVE_SURVEY_RATE_TWD04 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Taiwanese Dollar/U.S. Dollar spot rate, expressed as the amount of Taiwanese Dollars per one U.S. Dollar, for settlement in two Business Days, reported by the Taipei Forex Inc. which appears on the Reuters Screen TAIFX1 Page under the heading 'Spot' as of 11:00 a.m. Taipei time, on that Rate Calculation Date, or if no rate appears as of 11:00 a.m., Taipei time, the rate that first appears in any of the next succeeding 15 minute intervals after such time, up to and including 12:00 noon, Taipei time on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_TWD_TAIFX1_TWD03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Taiwanese Dollar/U.S. Dollar spot rate, expressed as the amount of Taiwanese Dollars per one U.S. Dollar, for settlement in two Business Days, reported by the Taipei Forex Inc. which appears on the Telerate Page 6161 under the heading 'Spot' as of 11:00 a.m., Taipei time, on that Rate Calculation Date, or if no rate appears as of 11:00 a.m., Taipei time, the rate that first appears in any of the next succeeding 15 minute intervals after such time, up to and including 12:00 noon, Taipei time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_TWD_TELERATE_6161_TWD01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Taiwanese Dollar/U.S. Dollar Specified Rate, expressed as the amount of Taiwanese Dollars per one U.S. Dollar, for settlement in two Business Days which appears on the Reuters Screen TFEMA Page as of 11:00 a.m., Taipei time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_TWD_TFEMA_TWD02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Ukrainian Hryvnia/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Ukrainian Hryvnia per one U.S. Dollar, for settlement on the same Business Day, as published on EMTA's website (www.emta.org) at approximately 2:00 p.m., Kiev time, or as soon thereafter as practicable, on that Rate Calculation Date. The Spot Rate shall be calculated by EMTA (or a service provider EMTA may select in its sole discretion) pursuant to the EMTA UAH Indicative Survey Methodology (which means a methodology, dated as of March 16, 2009, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Ukrainian Hryvnia / U.S. Dollar markets for the purpose of determining the EMTA UAH Indicative Survey Rate).
   */
  SettlementRateOptionEnum_UAH_EMTA_INDICATIVE_SURVEY_RATE_UAH03 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Ukrainian Hryvnia/U.S. Dollar Specified Rate for U.S. Dollars expressed as the amount of Ukrainian Hryvnia per one U.S. Dollar, for settlement on the same Business Day calculated by Thomson Reuters pursuant to the EMTA UAH Industry Survey Methodology, which rate appears on EMTA's website (www.emta.org) and on Thomson Reuters Page EMTAUAHFIX at approximately 11:30 am, Kiev time, on that Rate Calculation Date. The 'EMTA UAH Industry Survey Methodology' as used herein means the methodology dated as of March 16, 2009, for a centralized industry wide survey of financial institutions in the Ukrainian Hryvnia/U.S. Dollar spot market for the purposes of determining the EMTA UAH Industry Survey Rate.
   */
  SettlementRateOptionEnum_UAH_EMTA_INDUSTRY_SURVEY_RATE_UAH02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Ukrainian Hryvnia/U.S. Dollar spot rate, expressed as the amount of Ukrainian Hryvnia per one U.S. Dollar, for settlement on the same Business Day reported by GFI Brokers on Thomson Reuters Page GFIU by 9:30 am, London time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_UAH_GFI_UAH01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the midpoint of the Venezuelan Bolivar /U.S. Dollar Tipo de Cambio De Referencia buying and selling rates, expressed as the amount of Venezuelan Bolivar per one U.S. Dollar, for settlement in two Business Days reported by the Banco Central de Venezuela (www.bcv.org.ve) at approximately 5:00 p.m., Caracas time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_VEF_FIX_VEF01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Vietnamese Dong/U.S. Dollar spot rate at 11:00 a.m., Singapore time, expressed as the amount of Vietnamese Dong per one U.S. Dollar, for settlement in two Business Days reported by the Association of Banks in Singapore, which appears on the Reuters Screen ABSIRFIX01 Page to the right of the caption 'Spot' under the column 'VND' at approximately 11:30 a.m., Singapore time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_VND_ABS_VND01 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Vietnamese Dong/U.S. Dollar spot rate expressed as the amount of Vietnamese Dong per one U.S. Dollar, for settlement in two Business Days which appears on Reuters Screen VNDFIX=VN Page under the caption 'Spot' and to the right of the caption 'Average' at approximately 11:00 am, Hanoi time, on that Rate Calculation Date.
   */
  SettlementRateOptionEnum_VND_FX_VND02 SettlementRateOptionEnum = iota + 1
  /**
   * The Spot Rate for a Rate Calculation Date will be the Vietnamese Dong/U.S. Dollar Specified Rate for U.S. Dollars, expressed as the amount of Vietnamese Dong per one U.S. Dollar, for settlement in two Business Days, as published on SFEMC's website (www.sfemc.org) at approximately 3:30 p.m., Singapore time, or as soon as thereafter as practicable, on that Rate Calculation Date. The Spot Rate shall be calculated by SFEMC (or a service provider SFEMC may select in its sole discretion) pursuant to the SFEMC VND Indicative Survey Methodology (which means a methodology, dated as of July 14, 2008, as amended from time to time, for a centralized industry-wide survey of financial institutions that are active participants in the Vietnamese Dong/U.S. Dollar markets for the purpose of determining the SFEMC VND Indicative Survey Rate).
   */
  SettlementRateOptionEnum_VND_SFEMC_INDICATIVE_SURVEY_RATE_VND03 SettlementRateOptionEnum = iota + 1
  )    
