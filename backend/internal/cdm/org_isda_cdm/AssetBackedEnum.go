/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Specifies the underlying asset or asset pool securing an Asset Backed debt instrument.
   */
  
  const (
  /**
   * When the asset(s) backing the debt are auto loans.
   */
  AssetBackedEnum_AUTO_LOAN AssetBackedEnum = iota + 1
  /**
   * When the asset(s) backing the debt are credit card loans.
   */
  AssetBackedEnum_CREDIT_CARD AssetBackedEnum = iota + 1
  /**
   * When the asset(s) backing the debt are home equity loans, in which the borrower uses the equity of their home as collateral.
   */
  AssetBackedEnum_HOME_EQUITY AssetBackedEnum = iota + 1
  /**
   * When the asset(s) backing the debt instrument is a pool of mortgage loans, e.g. for a mortgage backed security.
   */
  AssetBackedEnum_MORTGAGE AssetBackedEnum = iota + 1
  /**
   * Any other asset which generates receivables for an Asset Backed Security.
   */
  AssetBackedEnum_OTHER AssetBackedEnum = iota + 1
  /**
   * When the asset(s) backing the debt are property, typically when the debt is a mortgage loan.
   */
  AssetBackedEnum_PROPERTY AssetBackedEnum = iota + 1
  /**
   * When the asset(s) backing the debt are student loans.
   */
  AssetBackedEnum_STUDENT_LOAN AssetBackedEnum = iota + 1
  )    
