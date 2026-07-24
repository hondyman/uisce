/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Extends product identifiers with additional identifier sources for Assets.
   */
  
  const (
  /**
   * Published by Bloomberg, the BBGID is a 12-digit alphanumeric randomly generated ID covering active and non-active securities.
   */
  AssetIdTypeEnum_BBGID AssetIdTypeEnum = iota + 1
  /**
   * Published by Bloomberg as a short code to identify publicly trades shares of a particular stock on a specific exchange.
   */
  AssetIdTypeEnum_BBGTICKER AssetIdTypeEnum = iota + 1
  /**
   * Derived from the Committee on Uniform Security Identification Procedures, CUSIPs are 9-character identifiers that capture an issue’s important differentiating characteristics for issuers and their financial instruments in the U.S. and Canada.
   */
  AssetIdTypeEnum_CUSIP AssetIdTypeEnum = iota + 1
  /**
   * The identifier follows the symbology set by the clearing house which clears the asset.
   */
  AssetIdTypeEnum_CLEARING_CODE AssetIdTypeEnum = iota + 1
  /**
   * Used to identify the currency of a Cash Asset.
   */
  AssetIdTypeEnum_CURRENCY_CODE AssetIdTypeEnum = iota + 1
  /**
   * The identifier follows the symbology set by the exchange which lists the asset.
   */
  AssetIdTypeEnum_EXCHANGE_CODE AssetIdTypeEnum = iota + 1
  /**
   * Issued under the guidelines of the Object Management Group, the Financial Instrument Global Identifier (FIGI) is a 12 character, alphanumeric, randomly generated ID covering hundreds of millions of active and inactive instruments. The identifier acts as a Uniform Resource Identifier (URI) to link to a set of metadata that uniquely and clearly describes the instrument.
   */
  AssetIdTypeEnum_FIGI AssetIdTypeEnum = iota + 1
  /**
   * Issued by the International Swaps Dealers Association as a string representing a Commodity Reference Price used for purposes of determining a relevant price for an underlying commodity in an OTC derivatives contract.
   */
  AssetIdTypeEnum_ISDACRP AssetIdTypeEnum = iota + 1
  /**
   * Issued by The International Securities Identification Number (ISIN) Organization, the ISIN is a 12-character alpha-numerical code used to uniformly identify a security for trading and settlement purposes. Securities with which ISINs can be used include debt securities, such as notes or bonds as well shares, such as common stock or shares of a fund, options, derivatives, and futures. The ISIN structure is defined in ISO 6166.
   */
  AssetIdTypeEnum_ISIN AssetIdTypeEnum = iota + 1
  /**
   * The name of the product.
   */
  AssetIdTypeEnum_NAME AssetIdTypeEnum = iota + 1
  /**
   * Used when the source is not otherwise in this enumerated list because it is internal or other reasons.  The source can be identified in the scheme which is part of the identifier attribute.
   */
  AssetIdTypeEnum_OTHER AssetIdTypeEnum = iota + 1
  /**
   * Assigned by S&P Global Market Intelligence, REDID (Reference Entity Data Identifier) is a unique alphanumeric code to reference entities, primarily utilized within the credit derivatives market for the standardized identification of underlying entities.
   */
  AssetIdTypeEnum_REDID AssetIdTypeEnum = iota + 1
  /**
   * Issued by Refinitiv (formerly Reuters), the Reuters Instrument Codes(RIC) uniquely identifies financial instruments, including where they are traded.
   */
  AssetIdTypeEnum_RIC AssetIdTypeEnum = iota + 1
  /**
   * Assigned by the London Stock Exchange, the Stock Exchange Daily Official List (SEDOL) is a list of security identifiers used in the United Kingdom and Ireland for clearing purposes.  SEDOLs serve as the National Securities Identifying Number for all securities issued in the United Kingdom and are therefore part of the security's ISIN as well.
   */
  AssetIdTypeEnum_SEDOL AssetIdTypeEnum = iota + 1
  /**
   * Issued by the French Société Interprofessionnelle pour la Compensation des Valeurs Mobilières (SICOVAM) to identify French securities listed on French stock exchanges.
   */
  AssetIdTypeEnum_SICOVAM AssetIdTypeEnum = iota + 1
  /**
   * Assigned by the Derivatives Service Bureau Ltd (DSB), the Unique Product Identifier (UPI) is a unique code to describe an over-the-counter (OTC) derivatives product.  The UPI is used for identifying the product in transaction reporting data.
   */
  AssetIdTypeEnum_UPI AssetIdTypeEnum = iota + 1
  /**
   * Assigned by SIX Financial Information, Valoren is a unique numerical identification number typically ranging from six to nine digits primarily traded in Switzerland, Liechtenstein, and Belgium. This code enables the identification of securities such as stocks, bonds, and investment funds within these markets.
   */
  AssetIdTypeEnum_VALOREN AssetIdTypeEnum = iota + 1
  /**
   * Issued by the Institute for the Issuance and Administration of Securities in Germany (Securities Information), the Wertpapierkennnummer (WKN, WPKN, WPK or simply Wert) consists of six digits or capital letters (excluding I and O), and no check digit. It is used to identify German securities.
   */
  AssetIdTypeEnum_WERTPAPIER AssetIdTypeEnum = iota + 1
  )    
