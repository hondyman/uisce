/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * Represents an enumeration list to identify which Collateral Criteria type should have priority over others. If set to 'Issuer', the rating in the 
   Issuer Criteria has priority or is used if there is no Asset criteria. If set to 'Asset', the rating in the Asset Criteria has priority or is used if there is no Issuer rating.
   */
  
  const (
  /**
   * Denotes that the Asset Criteria has priority.
   */
  RatingPriorityResolutionEnum_ASSET RatingPriorityResolutionEnum = iota + 1
  /**
   * Denotes that average rating should be used if several criteria apply.
   */
  RatingPriorityResolutionEnum_AVERAGE RatingPriorityResolutionEnum = iota + 1
  /**
   * Denotes that highest rating should be used if several criteria apply.
   */
  RatingPriorityResolutionEnum_HIGHEST RatingPriorityResolutionEnum = iota + 1
  /**
   * Denotes that the Issuer Criteria has priority.
   */
  RatingPriorityResolutionEnum_ISSUER RatingPriorityResolutionEnum = iota + 1
  /**
   * Denotes that lowest rating should be used if several criteria apply.
   */
  RatingPriorityResolutionEnum_LOWEST RatingPriorityResolutionEnum = iota + 1
  )    
