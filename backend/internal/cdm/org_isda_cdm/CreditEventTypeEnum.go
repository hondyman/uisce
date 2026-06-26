/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents the enumerated values to specify a credit event type.
   */
  
  const (
  /**
   * The reference entity has been dissolved or has become insolvent. It also covers events that may be a precursor to insolvency such as instigation of bankruptcy or insolvency proceedings. Sovereign trades are not subject to Bankruptcy as 'technically' a Sovereign cannot become bankrupt. ISDA 2003 Term: Bankruptcy.
   */
  CreditEventTypeEnum_BANKRUPTCY CreditEventTypeEnum = iota + 1
  /**
   * Results from the fact that the rating of the reference obligation is downgraded to a distressed rating level. From a usage standpoint, this credit event is typically not applicable in case of RMBS trades.
   */
  CreditEventTypeEnum_DISTRESSED_RATINGS_DOWNGRADE CreditEventTypeEnum = iota + 1
  /**
   * This credit event triggers, after the expiration of any applicable grace period, if the reference entity fails to make due payments in an aggregrate amount of not less than the payment requirement on one or more obligations (e.g. a missed coupon payment). ISDA 2003 Term: Failure to Pay.
   */
  CreditEventTypeEnum_FAILURE_TO_PAY CreditEventTypeEnum = iota + 1
  /**
   * Corresponds to the failure by the Reference Entity to pay an expected interest amount or the payment of an actual interest amount that is less than the expected interest amount. ISDA 2003 Term: Failure to Pay Interest.
   */
  CreditEventTypeEnum_FAILURE_TO_PAY_INTEREST CreditEventTypeEnum = iota + 1
  /**
   * Corresponds to the failure by the Reference Entity to pay an expected principal amount or the payment of an actual principal amount that is less than the expected principal amount. ISDA 2003 Term: Failure to Pay Principal.
   */
  CreditEventTypeEnum_FAILURE_TO_PAY_PRINCIPAL CreditEventTypeEnum = iota + 1
  /**
   * A governmental intervention is an event resulting from an action by a governmental authority that materially impacts the reference entity's obligations, such as an interest rate reduction, principal reduction, deferral of interest or principal, change in priority ranking, or change in currency or composition of payment. ISDA 2014 Term: Governmental Intervention.
   */
  CreditEventTypeEnum_GOVERNMENTAL_INTERVENTION CreditEventTypeEnum = iota + 1
  /**
   * Results from the fact that losses occur to the underlying instruments that do not result in reductions of the outstanding principal of the reference obligation.
   */
  CreditEventTypeEnum_IMPLIED_WRITEDOWN CreditEventTypeEnum = iota + 1
  /**
   * Results from the fact that the underlier fails to make principal payments as expected.
   */
  CreditEventTypeEnum_MATURITY_EXTENSION CreditEventTypeEnum = iota + 1
  /**
   * One or more of the obligations have been declared due and payable before they would otherwise have been due and payable as a result of, or on the basis of, the occurrence of a default, event of default or other similar condition or event other than failure to pay (preferred by the market over Obligation Default, because more definitive and encompasses the definition of Obligation Default - this is more favorable to the Seller). Subject to the default requirement amount. ISDA 2003 Term: Obligation Acceleration.
   */
  CreditEventTypeEnum_OBLIGATION_ACCELERATION CreditEventTypeEnum = iota + 1
  /**
   * One or more of the obligations have become capable of being declared due and payable before they would otherwise have been due and payable as a result of, or on the basis of, the occurrence of a default, event of default or other similar condition or event other than failure to pay. ISDA 2003 Term: Obligation Default.
   */
  CreditEventTypeEnum_OBLIGATION_DEFAULT CreditEventTypeEnum = iota + 1
  /**
   * The reference entity, or a governmental authority, either refuses to recognise or challenges the validity of one or more obligations of the reference entity, or imposes a moratorium thereby postponing payments on one or more of the obligations of the reference entity. Subject to the default requirement amount. ISDA 2003 Term: Repudiation/Moratorium.
   */
  CreditEventTypeEnum_REPUDIATION_MORATORIUM CreditEventTypeEnum = iota + 1
  /**
   * A restructuring is an event that materially impacts the reference entity's obligations, such as an interest rate reduction, principal reduction, deferral of interest or principal, change in priority ranking, or change in currency or composition of payment. ISDA 2003 Term: Restructuring.
   */
  CreditEventTypeEnum_RESTRUCTURING CreditEventTypeEnum = iota + 1
  /**
   * Results from the fact that the underlier writes down its outstanding principal amount.
   */
  CreditEventTypeEnum_WRITEDOWN CreditEventTypeEnum = iota + 1
  )    
