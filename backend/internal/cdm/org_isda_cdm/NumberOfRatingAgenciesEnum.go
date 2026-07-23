/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to define the number of Rating Agencies that must be considered to meet the rating condition.
   */
  
  const (
  /**
   * Ratings for all defined Rating Agencies will be considered.
   */
  NumberOfRatingAgenciesEnum_ALL NumberOfRatingAgenciesEnum = iota + 1
  /**
   * Ratings for Any 1 stated Rating Agency will be applicable.
   */
  NumberOfRatingAgenciesEnum_ANY_ONE NumberOfRatingAgenciesEnum = iota + 1
  /**
   * Ratings for Any 2 stated Rating Agencies will be applicable.
   */
  NumberOfRatingAgenciesEnum_ANY_TWO NumberOfRatingAgenciesEnum = iota + 1
  /**
   * Utilised where the clause data structure is not able to capture a material aspect of the clause.
   */
  NumberOfRatingAgenciesEnum_OTHER NumberOfRatingAgenciesEnum = iota + 1
  )    
