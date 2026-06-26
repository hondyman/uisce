/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Defines the different cash settlement methods for a product where cash settlement is applicable.
   */
  
  const (
  /**
   * An ISDA defined cash settlement method used for the determination of the applicable cash settlement amount. The method is defined in the 2006 ISDA Definitions, Section 18.3. Cash Settlement Methods, paragraph (b).
   */
  CashSettlementMethodEnum_CASH_PRICE_ALTERNATE_METHOD CashSettlementMethodEnum = iota + 1
  /**
   * An ISDA defined cash settlement method used for the determination of the applicable cash settlement amount. The method is defined in the 2006 ISDA Definitions, Section 18.3. Cash Settlement Methods, paragraph (a).
   */
  CashSettlementMethodEnum_CASH_PRICE_METHOD CashSettlementMethodEnum = iota + 1
  /**
   * An ISDA defined cash settlement method (yield curve) used for the determination of the applicable cash settlement amount. The method is defined in the 2006 ISDA Definitions, Section 18.3. Cash Settlement Methods, paragraph (g) (published in Supplement number 28). The method is defined in the 2021 ISDA Definitions, section 18.2.6.
   */
  CashSettlementMethodEnum_COLLATERALIZED_CASH_PRICE_METHOD CashSettlementMethodEnum = iota + 1
  /**
   * An ISDA defined cash settlement method used for the determination of the applicable cash settlement amount. The method is defined in the 2006 ISDA Definitions, Section 18.3. Cash Settlement Methods, paragraph (f) (published in Supplement number 23).
   */
  CashSettlementMethodEnum_CROSS_CURRENCY_METHOD CashSettlementMethodEnum = iota + 1
  /**
   * An ISDA defined cash settlement method used for the determination of the applicable cash settlement amount. The method is defined in the 2021 ISDA Definitions, Section 18.2.3.
   */
  CashSettlementMethodEnum_MID_MARKET_CALCULATION_AGENT_DETERMINATION CashSettlementMethodEnum = iota + 1
  /**
   * An ISDA defined cash settlement method used for the determination of the applicable cash settlement amount. The method is defined in the 2021 ISDA Definitions, Section 18.2.1.
   */
  CashSettlementMethodEnum_MID_MARKET_INDICATIVE_QUOTATIONS CashSettlementMethodEnum = iota + 1
  /**
   * An ISDA defined cash settlement method used for the determination of the applicable cash settlement amount. The method is defined in the 2021 ISDA Definitions, Section 18.2.2.
   */
  CashSettlementMethodEnum_MID_MARKET_INDICATIVE_QUOTATIONS_ALTERNATE CashSettlementMethodEnum = iota + 1
  /**
   * An ISDA defined cash settlement method used for the determination of the applicable cash settlement amount. The method is defined in the 2006 ISDA Definitions, Section 18.3. Cash Settlement Methods, paragraph (c).
   */
  CashSettlementMethodEnum_PAR_YIELD_CURVE_ADJUSTED_METHOD CashSettlementMethodEnum = iota + 1
  /**
   * An ISDA defined cash settlement method used for the determination of the applicable cash settlement amount. The method is defined in the 2006 ISDA Definitions, Section 18.3. Cash Settlement Methods, paragraph (e).
   */
  CashSettlementMethodEnum_PAR_YIELD_CURVE_UNADJUSTED_METHOD CashSettlementMethodEnum = iota + 1
  /**
   * An ISDA defined cash settlement method used for the determination of the applicable cash settlement amount. The method is defined in the 2021 ISDA Definitions, Section 18.2.5
   */
  CashSettlementMethodEnum_REPLACEMENT_VALUE_CALCULATION_AGENT_DETERMINATION CashSettlementMethodEnum = iota + 1
  /**
   * An ISDA defined cash settlement method used for the determination of the applicable cash settlement amount. The method is defined in the 2021 ISDA Definitions, Section 18.2.4.
   */
  CashSettlementMethodEnum_REPLACEMENT_VALUE_FIRM_QUOTATIONS CashSettlementMethodEnum = iota + 1
  /**
   * An ISDA defined cash settlement method used for the determination of the applicable cash settlement amount. The method is defined in the 2006 ISDA Definitions, Section 18.3. Cash Settlement Methods, paragraph (d).
   */
  CashSettlementMethodEnum_ZERO_COUPON_YIELD_ADJUSTED_METHOD CashSettlementMethodEnum = iota + 1
  )    
