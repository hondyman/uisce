/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents the enumerated values to specify taxonomy sources.
   */
  
  const (
  /**
   * Represents the ISO 10962 Classification of Financial Instruments code.
   */
  TaxonomySourceEnum_CFI TaxonomySourceEnum = iota + 1
  /**
   * Represents the Commodity Futures Trading Commission (CFTC) as a taxonomy source.
   */
  TaxonomySourceEnum_CFTC TaxonomySourceEnum = iota + 1
  /**
   * Represents the Canadian Securities Administrators (CSA) as a taxonomy source.
   */
  TaxonomySourceEnum_CSA TaxonomySourceEnum = iota + 1
  /**
   * Represents the EMIR Article 9 Asset Definition Identifier code.
   */
  TaxonomySourceEnum_EMIR TaxonomySourceEnum = iota + 1
  /**
   * Identifies European Union Eligible Collateral Assets classification categories based on EMIR Uncleared Margin Rules.
   */
  TaxonomySourceEnum_EU_EMIR_ELIGIBLE_COLLATERAL_ASSET_CLASS TaxonomySourceEnum = iota + 1
  /**
   * Represents the ISDA Collateral Asset Definition Identifier code.
   */
  TaxonomySourceEnum_ICAD TaxonomySourceEnum = iota + 1
  /**
   * Represents the ISDA product taxonomy.
   */
  TaxonomySourceEnum_ISDA TaxonomySourceEnum = iota + 1
  /**
   * Represents the Monetary Authority of Singapore (MAS) as a taxonomy source.
   */
  TaxonomySourceEnum_MAS TaxonomySourceEnum = iota + 1
  /**
   * Denotes a user-specific scheme or taxonomy or other external sources not listed here.
   */
  TaxonomySourceEnum_OTHER TaxonomySourceEnum = iota + 1
  /**
   * Identifies United Kingdom Eligible Collateral Assets classification categories based on UK Onshored EMIR Uncleared Margin Rules Eligible Collateral asset classes for both initial margin (IM) and variation margin (VM) posted and collected between specified entities.Please note: UK EMIR regulation will detail which eligible collateral assets classes apply to each type of entity pairing (counterparty) and which apply to posting of IM and VM.
   */
  TaxonomySourceEnum_UK_EMIR_ELIGIBLE_COLLATERAL_ASSET_CLASS TaxonomySourceEnum = iota + 1
  /**
   * Identifies US Eligible Collateral Assets classification categories based on Uncleared Margin Rules published by the CFTC and the US Prudential Regulator. Note: While the same basic categories exist in the CFTC and US Prudential Regulators margin rules, the precise definitions or application of those rules could differ between the two rules.
   */
  TaxonomySourceEnum_US_CFTC_PR_ELIGIBLE_COLLATERAL_ASSET_CLASS TaxonomySourceEnum = iota + 1
  )    
