/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumeration values to specify the transfer status.
   */
  
  const (
  /**
   * The transfer is disputed.
   */
  TransferStatusEnum_DISPUTED TransferStatusEnum = iota + 1
  /**
   * The transfer has been instructed.
   */
  TransferStatusEnum_INSTRUCTED TransferStatusEnum = iota + 1
  /**
   * The transfer has been netted into a separate Transfer.
   */
  TransferStatusEnum_NETTED TransferStatusEnum = iota + 1
  /**
   * The transfer is pending instruction.
   */
  TransferStatusEnum_PENDING TransferStatusEnum = iota + 1
  /**
   * The transfer has been settled.
   */
  TransferStatusEnum_SETTLED TransferStatusEnum = iota + 1
  )    
