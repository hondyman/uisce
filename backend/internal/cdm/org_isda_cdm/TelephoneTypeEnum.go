/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values to specify the type of telephone number, e.g. work vs. mobile.
   */
  
  const (
  /**
   * A number used primarily for work-related facsimile transmissions.
   */
  TelephoneTypeEnum_FAX TelephoneTypeEnum = iota + 1
  /**
   * A number on a mobile telephone that is often or usually used for work-related calls. This type of number can be used for urgent work related business when a work number is not sufficient to contact the person or firm.
   */
  TelephoneTypeEnum_MOBILE TelephoneTypeEnum = iota + 1
  /**
   * A number used primarily for non work-related calls. (Normally this type of number would be used only as an emergency backup number, not as a regular course of business).
   */
  TelephoneTypeEnum_PERSONAL TelephoneTypeEnum = iota + 1
  /**
   * A number used primarily for work-related calls. Includes home office numbers used primarily for work purposes.
   */
  TelephoneTypeEnum_WORK TelephoneTypeEnum = iota + 1
  )    
