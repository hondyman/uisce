/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the option exercise style. i.e., European, Bermuda or American.
   */
  
  const (
  /**
   * Continuous exercise over a range of dates
   */
  OptionExerciseStyleEnum_AMERICAN OptionExerciseStyleEnum = iota + 1
  /**
   * Multiple specified exercise dates
   */
  OptionExerciseStyleEnum_BERMUDA OptionExerciseStyleEnum = iota + 1
  /**
   * Single Exercise
   */
  OptionExerciseStyleEnum_EUROPEAN OptionExerciseStyleEnum = iota + 1
  )    
