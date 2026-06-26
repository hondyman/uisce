/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the day count fraction.
   */
  
  const (
  /**
   * Per 2021 ISDA Definitions, Section 4.6.1 Day Count Fraction, paragraph (v), per 2006 ISDA Definitions, Section 4.16. Day Count Fraction, paragraph (e) or Annex to the 2000 ISDA Definitions (June 2000 Version), Section 4.16. Day Count Fraction, paragraph (d).
   */
  DayCountFractionEnum_ACT_360 DayCountFractionEnum = iota + 1
  /**
   * Per CFTC definitions.
   */
  DayCountFractionEnum_ACT_364 DayCountFractionEnum = iota + 1
  /**
   * Per 2021 ISDA Definitions, Section 4.6.1 Day Count Fraction, paragraph (ix), per 2006 ISDA Definitions, Section 4.16. Day Count Fraction, paragraph (i).
   */
  DayCountFractionEnum_ACT_365L DayCountFractionEnum = iota + 1
  /**
   * Per 2021 ISDA Definitions, Section 4.6.1 Day Count Fraction, paragraph (iv), per 2006 ISDA Definitions, Section 4.16. Day Count Fraction, paragraph (d) or Annex to the 2000 ISDA Definitions (June 2000 Version), Section 4.16. Day Count Fraction, paragraph (c).
   */
  DayCountFractionEnum_ACT_365_FIXED DayCountFractionEnum = iota + 1
  /**
   * The Fixed/Floating Amount will be calculated in accordance with the 'BASE EXACT/EXACT' day count fraction, as defined in the 'Definitions Communes plusieurs Additifs Techniques' published by the Association Francaise des Banques in September 1994.
   */
  DayCountFractionEnum_ACT_ACT_AFB DayCountFractionEnum = iota + 1
  /**
   * Per 2021 ISDA Definitions, Section 4.6.1 Day Count Fraction, paragraph (iii), per 2006 ISDA Definitions, Section 4.16. Day Count Fraction, paragraph (c). This day count fraction code is applicable for transactions booked under the 2021 ISDA Definitions and the 2006 ISDA Definitions. Transactions under the 2000 ISDA Definitions should use the ACT/ACT.ISMA code instead.
   */
  DayCountFractionEnum_ACT_ACT_ICMA DayCountFractionEnum = iota + 1
  /**
   * Per 2021 ISDA Definitions, Section 4.6.1 Day Count Fraction, paragraph (ii), per 2006 ISDA Definitions, Section 4.16. Day Count Fraction, paragraph (b) or Annex to the 2000 ISDA Definitions (June 2000 Version), Section 4.16. Day Count Fraction, paragraph (b). Note that going from FpML 2.0 Recommendation to the FpML 3.0 Trial Recommendation the code in FpML 2.0 'ACT/365.ISDA' became 'ACT/ACT.ISDA'.
   */
  DayCountFractionEnum_ACT_ACT_ISDA DayCountFractionEnum = iota + 1
  /**
   * The Fixed/Floating Amount will be calculated in accordance with Rule 251 of the statutes, by-laws, rules and recommendations of the International Securities Market Association, as published in April 1999, as applied to straight and convertible bonds issued after December 31, 1998, as though the Fixed/Floating Amount were the interest coupon on such a bond. This day count fraction code is applicable for transactions booked under the 2000 ISDA Definitions. Transactions under the 2021 ISDA Definitions and the 2006 ISDA Definitions should use the ACT/ACT.ICMA code instead.
   */
  DayCountFractionEnum_ACT_ACT_ISMA DayCountFractionEnum = iota + 1
  /**
   * Per 2021 ISDA Definitions, Section 4.6.1 Day Count Fraction, paragraph (x). Supercedes BUS/252, the number of Business Days in the Calculation Period or Compounding Period in respect of which payment is being made divided by 252.
   */
  DayCountFractionEnum_CAL_252 DayCountFractionEnum = iota + 1
  /**
   * Per 2021 ISDA Definitions, Section 4.6.1 Day Count Fraction, paragraph (xi), the calculation mechanics are driven deterministically by the Calculation Period Frequency (i.e. 0.25 if it is three months, 0.5 if it is 6 months, 1 if it is a year), except that if the first Calculation Period or the final Calculation Period is less than the Calculation Period Frequency, Actual/Actual (ISDA) shall apply to that Calculation Period
   */
  DayCountFractionEnum_RBA_BOND_BASIS DayCountFractionEnum = iota + 1
  /**
   * Per 2006 ISDA Definitions, Section 4.16. Day Count Fraction, paragraph (a) or Annex to the 2000 ISDA Definitions (June 2000 Version), Section 4.16. Day Count Fraction, paragraph (a).
   */
  DayCountFractionEnum__1_1 DayCountFractionEnum = iota + 1
  /**
   * Per 2021 ISDA Definitions, Section 4.6.1 Day Count Fraction, paragraph (vii), per 2006 ISDA Definitions, Section 4.16. Day Count Fraction, paragraph (g) or Annex to the 2000 ISDA Definitions (June 2000 Version), Section 4.16. Day Count Fraction, paragraph (f). Note that the algorithm defined for this day count fraction has changed between the 2000 ISDA Definitions and 2006 ISDA Definitions. See Introduction to the 2006 ISDA Definitions for further information relating to this change.
   */
  DayCountFractionEnum__30E_360 DayCountFractionEnum = iota + 1
  /**
   * Per 2021 ISDA Definitions, Section 4.6.1 Day Count Fraction, paragraph (viii), per 2006 ISDA Definitions, Section 4.16. Day Count Fraction, paragraph (h). Note the algorithm for this day count fraction under the 2006 ISDA Definitions is designed to yield the same results in practice as the version of the 30E/360 day count fraction defined in the 2000 ISDA Definitions. See Introduction to the 2006 ISDA Definitions for further information relating to this change.
   */
  DayCountFractionEnum__30E_360_ISDA DayCountFractionEnum = iota + 1
  /**
   * Per 2021 ISDA Definitions, Section 4.6.1 Day Count Fraction, paragraph (vi), per 2006 ISDA Definitions, Section 4.16. Day Count Fraction, paragraph (f) or Annex to the 2000 ISDA Definitions (June 2000 Version), Section 4.16. Day Count Fraction, paragraph (e).
   */
  DayCountFractionEnum__30_360 DayCountFractionEnum = iota + 1
  )    
