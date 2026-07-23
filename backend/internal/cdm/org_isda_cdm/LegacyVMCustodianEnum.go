/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Details of how the Custodian is determined.
   */
  
  const (
  /**
   * The Custodian as advised by Party 1 to Party 2.
   */
  LegacyVMCustodianEnum_AS_ADVISED_1_TO_2 LegacyVMCustodianEnum = iota + 1
  /**
   * The Custodian as advised by Party 2 to Party 1.
   */
  LegacyVMCustodianEnum_AS_ADVISED_2_TO_1 LegacyVMCustodianEnum = iota + 1
  /**
   * A named entity is specified as the Custodian.
   */
  LegacyVMCustodianEnum_NAMED_ENTITY LegacyVMCustodianEnum = iota + 1
  /**
   * Not Applicable.
   */
  LegacyVMCustodianEnum_NOT_APPLICABLE LegacyVMCustodianEnum = iota + 1
  /**
   * Party 1 will notify Party 2 of the custodian.
   */
  LegacyVMCustodianEnum_PARTY_1_TO_PARTY_2 LegacyVMCustodianEnum = iota + 1
  /**
   * Party 2 will notify Party 1 of the custodian.
   */
  LegacyVMCustodianEnum_PARTY_2_TO_PARTY_1 LegacyVMCustodianEnum = iota + 1
  )    
