/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumeration values to specify how the transfer will settle, e.g. DvP.
   */
  
  const (
  /**
   * Simultaneous transfer of two assets, typically securities, as a way to avoid settlement risk.
   */
  TransferSettlementEnum_DELIVERY_VERSUS_DELIVERY TransferSettlementEnum = iota + 1
  /**
   * Settlement in which the transfer of the asset and the cash settlement are simultaneous.
   */
  TransferSettlementEnum_DELIVERY_VERSUS_PAYMENT TransferSettlementEnum = iota + 1
  /**
   * No central settlement.
   */
  TransferSettlementEnum_NOT_CENTRAL_SETTLEMENT TransferSettlementEnum = iota + 1
  /**
   * Simultaneous transfer of cashflows.
   */
  TransferSettlementEnum_PAYMENT_VERSUS_PAYMENT TransferSettlementEnum = iota + 1
  )    
