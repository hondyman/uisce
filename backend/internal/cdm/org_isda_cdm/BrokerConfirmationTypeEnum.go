/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the type of Broker Confirm that the FpML trade represents.
   */
  
  const (
  /**
   * Broker Confirmation Type representing ABX index trades.
   */
  BrokerConfirmationTypeEnum_ABX BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Asia Corporate.
   */
  BrokerConfirmationTypeEnum_ASIA_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Asia Sovereign.
   */
  BrokerConfirmationTypeEnum_ASIA_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Australia Corporate.
   */
  BrokerConfirmationTypeEnum_AUSTRALIA_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Australia Sovereign.
   */
  BrokerConfirmationTypeEnum_AUSTRALIA_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for use with Credit Derivative Transactions on Leveraged Loans.
   */
  BrokerConfirmationTypeEnum_CD_SON_LEVERAGED_LOANS BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for use with Credit Derivative Transactions on Mortgage-backed Security with Pay-As-You-Go or Physical Settlement.
   */
  BrokerConfirmationTypeEnum_CD_SON_MBS BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for CDX Emerging Markets Untranched Transactions.
   */
  BrokerConfirmationTypeEnum_CDX_EMERGING_MARKETS BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for CDX Emerging Markets Diversified Untranched Transactions.
   */
  BrokerConfirmationTypeEnum_CDX_EMERGING_MARKETS_DIVERSIFIED BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for CDX Swaption Transactions.
   */
  BrokerConfirmationTypeEnum_CDX_SWAPTION BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for Dow Jones CDX Tranche Transactions.
   */
  BrokerConfirmationTypeEnum_CDX_TRANCHE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type representing CMBX index trades.
   */
  BrokerConfirmationTypeEnum_CMBX BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for CDS Index trades relating to Dow Jones CDX.EM index series.
   */
  BrokerConfirmationTypeEnum_DJ_CDX_EM_ BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for CDS Index trades relating to Dow Jones CDX.NA.IG and Dow Jones CDX.NA.HY index series.
   */
  BrokerConfirmationTypeEnum_DJ_CDX_NA BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Emerging European and Middle Eastern Sovereign.
   */
  BrokerConfirmationTypeEnum_EMERGING_EUROPEAN_AND_MIDDLE_EASTERN_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for EMERGING EUROPEAN CORPORATE.
   */
  BrokerConfirmationTypeEnum_EMERGING_EUROPEAN_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for EMERGING EUROPEAN CORPORATE LPN.
   */
  BrokerConfirmationTypeEnum_EMERGING_EUROPEAN_CORPORATE_LPN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for Single Name European CMBS Transactions.
   */
  BrokerConfirmationTypeEnum_EUROPEAN_CMBS BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of European Corporate.
   */
  BrokerConfirmationTypeEnum_EUROPEAN_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for Single Name European RMBS Transactions.
   */
  BrokerConfirmationTypeEnum_EUROPEAN_RMBS BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Japan Corporate.
   */
  BrokerConfirmationTypeEnum_JAPAN_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Japan Sovereign.
   */
  BrokerConfirmationTypeEnum_JAPAN_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Latin America Corporate.
   */
  BrokerConfirmationTypeEnum_LATIN_AMERICA_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for LATIN AMERICA CORPORATE B.
   */
  BrokerConfirmationTypeEnum_LATIN_AMERICA_CORPORATE_BOND BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for LATIN AMERICA CORPORATE BL.
   */
  BrokerConfirmationTypeEnum_LATIN_AMERICA_CORPORATE_BOND_OR_LOAN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Latin America Sovereign.
   */
  BrokerConfirmationTypeEnum_LATIN_AMERICA_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for MBX Transactions.
   */
  BrokerConfirmationTypeEnum_MBX BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for Municipal CDX Untranched Transactions.
   */
  BrokerConfirmationTypeEnum_MCDX BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of New Zealand Corporate.
   */
  BrokerConfirmationTypeEnum_NEW_ZEALAND_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of New Zealand Sovereign.
   */
  BrokerConfirmationTypeEnum_NEW_ZEALAND_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker ConfirmationType of North American Corporate.
   */
  BrokerConfirmationTypeEnum_NORTH_AMERICAN_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for PO Index Transactions.
   */
  BrokerConfirmationTypeEnum_PO BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Singapore Corporate.
   */
  BrokerConfirmationTypeEnum_SINGAPORE_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Singapore Sovereign.
   */
  BrokerConfirmationTypeEnum_SINGAPORE_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD ASIA CORPORATE.
   */
  BrokerConfirmationTypeEnum_STANDARD_ASIA_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD ASIA SOVEREIGN.
   */
  BrokerConfirmationTypeEnum_STANDARD_ASIA_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD AUSTRALIA CORPORATE.
   */
  BrokerConfirmationTypeEnum_STANDARD_AUSTRALIA_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD AUSTRALIA SOVEREIGN.
   */
  BrokerConfirmationTypeEnum_STANDARD_AUSTRALIA_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for Standard CDX Tranche Transactions.
   */
  BrokerConfirmationTypeEnum_STANDARD_CDX_TRANCHE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD EMERGING EUROPEAN AND MIDDLE EASTERN SOVEREIGN.
   */
  BrokerConfirmationTypeEnum_STANDARD_EMERGING_EUROPEAN_AND_MIDDLE_EASTERN_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD EMERGING EUROPEAN CORPORATE.
   */
  BrokerConfirmationTypeEnum_STANDARD_EMERGING_EUROPEAN_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD EMERGING EUROPEAN CORPORATE LPN.
   */
  BrokerConfirmationTypeEnum_STANDARD_EMERGING_EUROPEAN_CORPORATE_LPN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for STANDARD EUROPEAN CORPORATE.
   */
  BrokerConfirmationTypeEnum_STANDARD_EUROPEAN_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD JAPAN CORPORATE.
   */
  BrokerConfirmationTypeEnum_STANDARD_JAPAN_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD JAPAN SOVEREIGN.
   */
  BrokerConfirmationTypeEnum_STANDARD_JAPAN_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Standard Syndicated Secured Loan Credit Default Swap Broker Confirmation Type.
   */
  BrokerConfirmationTypeEnum_STANDARD_LCDS BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for Standard Syndicated Secured Loan Credit Default Swap Bullet Transactions.
   */
  BrokerConfirmationTypeEnum_STANDARD_LCDS_BULLET BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for Standard Syndicated Secured Loan Credit Default Swap Index Bullet Transactions.
   */
  BrokerConfirmationTypeEnum_STANDARD_LCDX_BULLET BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for Standard Syndicated Secured Loan Credit Default Swap Index Bullet Tranche Transactions.
   */
  BrokerConfirmationTypeEnum_STANDARD_LCDX_BULLET_TRANCHE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD LATIN AMERICA CORPORATE B.
   */
  BrokerConfirmationTypeEnum_STANDARD_LATIN_AMERICA_CORPORATE_BOND BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD LATIN AMERICA CORPORATE BL.
   */
  BrokerConfirmationTypeEnum_STANDARD_LATIN_AMERICA_CORPORATE_BOND_OR_LOAN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD LATIN AMERICA SOVEREIGN.
   */
  BrokerConfirmationTypeEnum_STANDARD_LATIN_AMERICA_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD NEW ZEALAND CORPORATE.
   */
  BrokerConfirmationTypeEnum_STANDARD_NEW_ZEALAND_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD NEW ZEALAND SOVEREIGN.
   */
  BrokerConfirmationTypeEnum_STANDARD_NEW_ZEALAND_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for STANDARD NORTH AMERICAN CORPORATE.
   */
  BrokerConfirmationTypeEnum_STANDARD_NORTH_AMERICAN_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD SINGAPORE CORPORATE.
   */
  BrokerConfirmationTypeEnum_STANDARD_SINGAPORE_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of STANDARD SINGAPORE SOVEREIGN.
   */
  BrokerConfirmationTypeEnum_STANDARD_SINGAPORE_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for STANDARD SUBORDINATED EUROPEAN INSURANCE CORPORATE.
   */
  BrokerConfirmationTypeEnum_STANDARD_SUBORDINATED_EUROPEAN_INSURANCE_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for STANDARD WESTERN EUROPEAN SOVEREIGN.
   */
  BrokerConfirmationTypeEnum_STANDARD_WESTERN_EUROPEAN_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for Standard iTraxx Europe Tranched Transactions.
   */
  BrokerConfirmationTypeEnum_STANDARDI_TRAXX_EUROPE_TRANCHE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Subordinated European Insurance Corporate.
   */
  BrokerConfirmationTypeEnum_SUBORDINATED_EUROPEAN_INSURANCE_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of SUKUK CORPORATE.
   */
  BrokerConfirmationTypeEnum_SUKUK_CORPORATE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of SUKUK SOVEREIGN.
   */
  BrokerConfirmationTypeEnum_SUKUK_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Syndicated Secured Loan Credit Default Swap Broker Confirmation Type.
   */
  BrokerConfirmationTypeEnum_SYNDICATED_SECURED_LOAN_CDS BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for TRX Transactions.
   */
  BrokerConfirmationTypeEnum_TRX BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for TRX.II Transactions.
   */
  BrokerConfirmationTypeEnum_TRX_II BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for U.S. MUNICIPAL FULL FAITH AND CREDIT.
   */
  BrokerConfirmationTypeEnum_US_MUNICIPAL_FULL_FAITH_AND_CREDIT BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for U.S. MUNICIPAL GENERAL FUND.
   */
  BrokerConfirmationTypeEnum_US_MUNICIPAL_GENERAL_FUND BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for U.S. MUNICIPAL REVENUE.
   */
  BrokerConfirmationTypeEnum_US_MUNICIPAL_REVENUE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type of Western European Sovereign.
   */
  BrokerConfirmationTypeEnum_WESTERN_EUROPEAN_SOVEREIGN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Asia Excluding Japan.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_ASIA_EX_JAPAN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Asia Ex-Japan Swaption Transactions.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_ASIA_EX_JAPAN_SWAPTION BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Asia Excluding Japan Tranched Transactions.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_ASIA_EX_JAPAN_TRANCHE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Australia.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_AUSTRALIA BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Australia Swaption Transactions.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_AUSTRALIA_SWAPTION BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Australia Tranched Transactions.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_AUSTRALIA_TRANCHE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx CJ.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_CJ BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx CJ Tranched Transactions.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_CJ_TRANCHE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Europe Transactions
   */
  BrokerConfirmationTypeEnum_I_TRAXX_EUROPE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Europe Swaption Transactions.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_EUROPE_SWAPTION BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Europe Tranched Transactions.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_EUROPE_TRANCHE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Japan.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_JAPAN BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Japan Swaption Transactions.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_JAPAN_SWAPTION BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx Japan Tranched Transactions.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_JAPAN_TRANCHE BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx LevX.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_LEV_X BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx SDI 75 Transactions.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_SDI_75 BrokerConfirmationTypeEnum = iota + 1
  /**
   * Broker Confirmation Type for iTraxx SovX.
   */
  BrokerConfirmationTypeEnum_I_TRAXX_SOV_X BrokerConfirmationTypeEnum = iota + 1
  )    
