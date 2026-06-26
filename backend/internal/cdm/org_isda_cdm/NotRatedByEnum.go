/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values applicable to define the what conditions apply to trigger a Not Rated condition.
   */
  
  const (
  /**
   * No rating is available for the Party for any of the stated Rating Agencies.
   */
  NotRatedByEnum_ALL NotRatedByEnum = iota + 1
  /**
   * No rating is available for the Party for any one of the stated Rating Agencies.
   */
  NotRatedByEnum_ONE NotRatedByEnum = iota + 1
  /**
   * No rating is available for the Party for any two of the stated Rating Agencies.
   */
  NotRatedByEnum_TWO NotRatedByEnum = iota + 1
  )    
