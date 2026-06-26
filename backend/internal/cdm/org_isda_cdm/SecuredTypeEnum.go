/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Specifies the type of secured debt.
   */
  
  const (
  /**
   * Debt instrument with periodic income payments and value derived from or backed by a specified pool of underlying assets.
   */
  SecuredTypeEnum_ASSET_BACKED SecuredTypeEnum = iota + 1
  /**
   * Complex product that is backed by a pool of loans and other assets and sold to institutional investors.
   */
  SecuredTypeEnum_COLLATERALIZED_OBLIGATION SecuredTypeEnum = iota + 1
  /**
   * Specifies a debt obligations issued by credit institutions which offer a so-called double-recourse protection to bondholders.
   */
  SecuredTypeEnum_COVERED_BONDS SecuredTypeEnum = iota + 1
  )    
