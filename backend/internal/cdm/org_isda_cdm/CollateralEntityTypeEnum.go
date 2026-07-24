/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents an enumeration list to identify the type of entity issuing or guaranteeing the collateral asset.
   */
  
  const (
  /**
   * Specifies corporate bodies including Banks.
   */
  CollateralEntityTypeEnum_CORPORATE CollateralEntityTypeEnum = iota + 1
  /**
   * Specifies a vehicle (with or without separate legal personality) designed for the purposes of collective investment towards a defined investment goal.
   */
  CollateralEntityTypeEnum_FUND CollateralEntityTypeEnum = iota + 1
  /**
   * Specifies institutions or bodies, typically constituted by statute, with a function mandated by the government and subject to government supervision inclusive of profit- and non-profit making bodies. Includes the US Agencies and GSEs and the EU concept of public sector entities. Excluding any entities which are also Regional Government.
   */
  CollateralEntityTypeEnum_QUASI_GOVERNMENT CollateralEntityTypeEnum = iota + 1
  /**
   * Specifies Regional Governments including states within countries, local authorities and municipalities.
   */
  CollateralEntityTypeEnum_REGIONAL_GOVERNMENT CollateralEntityTypeEnum = iota + 1
  /**
   * Specifies Sovereign, Government Debt Securities including Central Banks.
   */
  CollateralEntityTypeEnum_SOVEREIGN_CENTRAL_BANK CollateralEntityTypeEnum = iota + 1
  /**
   * Specifies a vehicle setup for the purpose of acquisition and financing of specific assets on a limited recourse basis. E.g. asset backed securities, including securitisations.
   */
  CollateralEntityTypeEnum_SPECIAL_PURPOSE_VEHICLE CollateralEntityTypeEnum = iota + 1
  /**
   * Specifies international organisations and multilateral banks, entities constituted by treaties or with multiple sovereign members includes Multilateral development Banks.
   */
  CollateralEntityTypeEnum_SUPRA_NATIONAL CollateralEntityTypeEnum = iota + 1
  )    
