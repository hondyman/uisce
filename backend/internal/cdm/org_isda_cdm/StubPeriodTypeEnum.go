/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify how to deal with a non standard calculation period within a swap stream.
   */
  
  const (
  /**
   * If there is a non regular period remaining it is placed at the end of the stream and combined with the adjacent calculation period to give a long last calculation period.
   */
  StubPeriodTypeEnum_LONG_FINAL StubPeriodTypeEnum = iota + 1
  /**
   * If there is a non regular period remaining it is placed at the start of the stream and combined with the adjacent calculation period to give a long first calculation period.
   */
  StubPeriodTypeEnum_LONG_INITIAL StubPeriodTypeEnum = iota + 1
  /**
   * If there is a non regular period remaining it is left shorter than the streams calculation period frequency and placed at the end of the stream.
   */
  StubPeriodTypeEnum_SHORT_FINAL StubPeriodTypeEnum = iota + 1
  /**
   * If there is a non regular period remaining it is left shorter than the streams calculation period frequency and placed at the start of the stream.
   */
  StubPeriodTypeEnum_SHORT_INITIAL StubPeriodTypeEnum = iota + 1
  )    
