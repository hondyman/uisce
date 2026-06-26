/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to define the supplements to a base set of ISDA Definitions that are applicable to the transaction.
   */
  
  const (
  /**
   * Standard Terms Supplement for ABX Transactions.
   */
  ContractualSupplementTypeEnum_ABX ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Asset-Backed Tranche Transactions.
   */
  ContractualSupplementTypeEnum_ABX_TRANCHE ContractualSupplementTypeEnum = iota + 1
  /**
   * ISDA Standard Terms Supplement for use with Credit Derivative Transactions on Leveraged Loans.
   */
  ContractualSupplementTypeEnum_CD_SON_LEVERAGED_LOANS ContractualSupplementTypeEnum = iota + 1
  /**
   * ISDA Standard Terms Supplement for use with Credit Derivative Transactions on Mortgage-backed Security with Pay-As-You-Go or Physical Settlement.
   */
  ContractualSupplementTypeEnum_CD_SON_MBS ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for CDX Untranched Transactions.
   */
  ContractualSupplementTypeEnum_CDX ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for CDX Emerging Markets Untranched Transactions.
   */
  ContractualSupplementTypeEnum_CDX_EMERGING_MARKETS ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for CDX Emerging Markets Diversified Untranched Transactions..
   */
  ContractualSupplementTypeEnum_CDX_EMERGING_MARKETS_DIVERSIFIED ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for CDX Swaption Transactions.
   */
  ContractualSupplementTypeEnum_CDX_SWAPTION ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Dow Jones CDX Tranche Transactions.
   */
  ContractualSupplementTypeEnum_CDX_TRANCHE ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for CMBX Transactions.
   */
  ContractualSupplementTypeEnum_CMBX ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Single Name European CMBS Transactions.
   */
  ContractualSupplementTypeEnum_EUROPEAN_CMBS ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Single Name European RMBS Transactions.
   */
  ContractualSupplementTypeEnum_EUROPEAN_RMBS ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for IOS Transactions.
   */
  ContractualSupplementTypeEnum_IOS ContractualSupplementTypeEnum = iota + 1
  /**
   * Supplement to the 1999 ISDA Credit Derivatives Definitions Relating to Convertible, Exchangeable or Accreting Obligations dated November 9, 2001.
   */
  ContractualSupplementTypeEnum_ISDA_1999_CREDIT_CONVERTIBLE_EXCHANGEABLE_ACCRETING_OBLIGATIONS ContractualSupplementTypeEnum = iota + 1
  /**
   * Restructuring Supplement to the 1999 ISDA Credit Derivatives Definitions dated May 11, 2001.
   */
  ContractualSupplementTypeEnum_ISDA_1999_CREDIT_RESTRUCTURING ContractualSupplementTypeEnum = iota + 1
  /**
   * Supplement Relating to Successor and Credit Events to the 1999 ISDA Credit Derivatives Definitions dated November 28, 2001.
   */
  ContractualSupplementTypeEnum_ISDA_1999_CREDIT_SUCCESSOR_AND_CREDIT_EVENTS ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for LPN dated December 6, 2007.
   */
  ContractualSupplementTypeEnum_ISDA_2003_ADDITIONAL_PROVISIONS_LPN ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for Contingent Credit Spread Transactions dated August 15, 2008.
   */
  ContractualSupplementTypeEnum_ISDA_2003_CONTINGENT_CREDIT_SPREAD_TRANSACTION ContractualSupplementTypeEnum = iota + 1
  /**
   * 2005 Matrix Supplement to the 2003 ISDA Credit Derivatives.
   */
  ContractualSupplementTypeEnum_ISDA_2003_CREDIT_2005_MATRIX_SUPPLEMENT ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for the Argentine Republic: Excluded Obligations and Excluded Deliverable Obligations dated December 21, 2005.
   */
  ContractualSupplementTypeEnum_ISDA_2003_CREDIT_ARGENTINE_REPUBLIC ContractualSupplementTypeEnum = iota + 1
  /**
   * ISDA Credit Derivatives Determinations Committees and Auction Settlement Supplement to the 2003 ISDA Credit Derivatives Definitions (published on [TBD]).
   */
  ContractualSupplementTypeEnum_ISDA_2003_CREDIT_AUCTION_SUPPLEMENT ContractualSupplementTypeEnum = iota + 1
  /**
   * May 2003 Supplement to the 2003 ISDA Credit Derivatives Definitions.
   */
  ContractualSupplementTypeEnum_ISDA_2003_CREDIT_MAY_2003 ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for Physically Settled Default Swaps Monoline Insurer as Reference Entity dated May 9, 2003.
   */
  ContractualSupplementTypeEnum_ISDA_2003_CREDIT_MONOLINE_INSURERS ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for Physically Settled Default Swaps Monoline Insurer as Reference Entity dated January 21, 2005.
   */
  ContractualSupplementTypeEnum_ISDA_2003_CREDIT_MONOLINE_INSURERS_2005 ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for the Republic of Hungary: Obligation Characteristics and Deliverable Obligation Characteristics dated August 13, 2004.
   */
  ContractualSupplementTypeEnum_ISDA_2003_CREDIT_REPUBLIC_OF_HUNGARY ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for the Republic of Hungary: Obligation Characteristics and Deliverable Obligation Characteristics dated February 14, 2005. 
   */
  ContractualSupplementTypeEnum_ISDA_2003_CREDIT_REPUBLIC_OF_HUNGARY_2005 ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for the Russian Federation: Obligation Characteristics and Deliverable Obligation Characteristics dated August 13, 2004.
   */
  ContractualSupplementTypeEnum_ISDA_2003_CREDIT_RUSSIAN_FEDERATION ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for Credit Derivative Transactions - U.S. Municipal Entity as Reference Entity dated September 17, 2004.
   */
  ContractualSupplementTypeEnum_ISDA_2003_CREDIT_US_MUNICIPALS ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for STMicroelectronics NV dated December 6, 2007.
   */
  ContractualSupplementTypeEnum_ISDA_2003_ST_MICROELECTRONICS_NV ContractualSupplementTypeEnum = iota + 1
  /**
   * 2007 Full Lookthrough Depository Receipt Supplement to the 2002 Equity Derivatives Definitions.
   */
  ContractualSupplementTypeEnum_ISDA_2007_FULL_LOOKTHROUGH_DEPOSITORY_RECEIPT_SUPPLEMENT ContractualSupplementTypeEnum = iota + 1
  /**
   * 2007 Partial Lookthrough Depository Receipt Supplement to the 2002 ISDA Equity Derivatives Definitions.
   */
  ContractualSupplementTypeEnum_ISDA_2007_PARTIAL_LOOKTHROUGH_DEPOSITORY_RECEIPT_SUPPLEMENT ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for Physically Settled Default Swaps Monoline Insurer.
   */
  ContractualSupplementTypeEnum_ISDA_CREDIT_MONOLINE_INSURERS ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for Fixed Recovery Credit Default Swap Transactions
   */
  ContractualSupplementTypeEnum_ISDA_DELIVERY_RESTRICTIONS ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for Fixed Recovery Credit Default Swap Transactions.
   */
  ContractualSupplementTypeEnum_ISDA_FIXED_RECOVERY ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for LPN Reference Entities.
   */
  ContractualSupplementTypeEnum_ISDALPN_REFERENCE_ENTITIES ContractualSupplementTypeEnum = iota + 1
  /**
   * Canadian Supplement to the 2004 Americas Interdealer Master Equity Derivatives Confirmation Agreement dated March 29, 2004.
   */
  ContractualSupplementTypeEnum_ISDA_MARCH_2004_EQUITY_CANADIAN_SUPPLEMENT ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for Recovery Lock Credit Default Swap Transactions.
   */
  ContractualSupplementTypeEnum_ISDA_RECOVERY_LOCK ContractualSupplementTypeEnum = iota + 1
  /**
   * Additional Provisions for Secured Deliverable Obligation Characteristic.
   */
  ContractualSupplementTypeEnum_ISDA_SECURED_DELIVERABLE_OBLIGATION_CHARACTERISTIC ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Syndicated Secured Loan Credit Default Swap Index Transactions.
   */
  ContractualSupplementTypeEnum_LCDX ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Syndicated Secured Loan Credit Default Swap Index Tranche Transactions.
   */
  ContractualSupplementTypeEnum_LCDX_TRANCHE ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for MBX Transactions.
   */
  ContractualSupplementTypeEnum_MBX ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Municipal CDX Untranched Transactions.
   */
  ContractualSupplementTypeEnum_MCDX ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for PO Index Transactions.
   */
  ContractualSupplementTypeEnum_PO ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for PrimeX Transactions.
   */
  ContractualSupplementTypeEnum_PRIME_X ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Standard CDX Tranche Transactions.
   */
  ContractualSupplementTypeEnum_STANDARD_CDX_TRANCHE ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Syndicated Secured Loan Credit Default Swap Standard Terms Supplement.
   */
  ContractualSupplementTypeEnum_STANDARD_LCDS ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Standard Syndicated Secured Loan Credit Default Swap Bullet Transactions.
   */
  ContractualSupplementTypeEnum_STANDARD_LCDS_BULLET ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Standard Syndicated Secured Loan Credit Default Swap Index Bullet Transactions.
   */
  ContractualSupplementTypeEnum_STANDARD_LCDX_BULLET ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Standard Syndicated Secured Loan Credit Default Swap Index Bullet Tranche Transactions.
   */
  ContractualSupplementTypeEnum_STANDARD_LCDX_BULLET_TRANCHE ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for Standard iTraxx Europe Tranched Transactions.
   */
  ContractualSupplementTypeEnum_STANDARDI_TRAXX_EUROPE_TRANCHE ContractualSupplementTypeEnum = iota + 1
  /**
   * Syndicated Secured Loan Credit Default Swap Standard Terms Supplement.
   */
  ContractualSupplementTypeEnum_SYNDICATED_SECURED_LOAN_CDS ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for TRX Transactions.
   */
  ContractualSupplementTypeEnum_TRX ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for TRX.II Transactions.
   */
  ContractualSupplementTypeEnum_TRX_II ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Asia Excluding Japan.
   */
  ContractualSupplementTypeEnum_I_TRAXX_ASIA_EX_JAPAN ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Asia Ex-Japan Swaption Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_ASIA_EX_JAPAN_SWAPTION ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Asia Excluding Japan Tranched Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_ASIA_EX_JAPAN_TRANCHE ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Australia.
   */
  ContractualSupplementTypeEnum_I_TRAXX_AUSTRALIA ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Australia Swaption Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_AUSTRALIA_SWAPTION ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Australia Tranched Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_AUSTRALIA_TRANCHE ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx CJ.
   */
  ContractualSupplementTypeEnum_I_TRAXX_CJ ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx CJ Tranched Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_CJ_TRANCHE ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Europe Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_EUROPE ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Europe Dealer Form.
   */
  ContractualSupplementTypeEnum_I_TRAXX_EUROPE_DEALER ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Europe Non-Dealer Form.
   */
  ContractualSupplementTypeEnum_I_TRAXX_EUROPE_NON_DEALER ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Europe Swaption Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_EUROPE_SWAPTION ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Europe Tranched Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_EUROPE_TRANCHE ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Japan.
   */
  ContractualSupplementTypeEnum_I_TRAXX_JAPAN ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Japan Swaption Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_JAPAN_SWAPTION ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx Japan Tranched Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_JAPAN_TRANCHE ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx LevX.
   */
  ContractualSupplementTypeEnum_I_TRAXX_LEV_X ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx SDI 75 Dealer Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_SDI_75_DEALER ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx SDI 75 Non-Dealer Transactions.
   */
  ContractualSupplementTypeEnum_I_TRAXX_SDI_75_NON_DEALER ContractualSupplementTypeEnum = iota + 1
  /**
   * Standard Terms Supplement for iTraxx SovX.
   */
  ContractualSupplementTypeEnum_I_TRAXX_SOV_X ContractualSupplementTypeEnum = iota + 1
  )    
