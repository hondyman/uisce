/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumeration values associated with person identifier sources.
   */
  
  const (
  /**
   * Alien Registration Number, number assigned by a social security agency to identify a non-resident person.
   */
  PersonIdentifierTypeEnum_ARNU PersonIdentifierTypeEnum = iota + 1
  /**
   * Passport Number, number assigned by an authority to identify the passport number of a person.
   */
  PersonIdentifierTypeEnum_CCPT PersonIdentifierTypeEnum = iota + 1
  /**
   * Customer Identification Number, number assigned by an issuer to identify a customer.
   */
  PersonIdentifierTypeEnum_CUST PersonIdentifierTypeEnum = iota + 1
  /**
   * Drivers License Number, number assigned by an authority to identify a driver's license.
   */
  PersonIdentifierTypeEnum_DRLC PersonIdentifierTypeEnum = iota + 1
  /**
   * Employee Identification Number, number assigned by a registration authority to an employee.
   */
  PersonIdentifierTypeEnum_EMPL PersonIdentifierTypeEnum = iota + 1
  /**
   * National Identity Number, number assigned by an authority to identify the national identity number of a person..
   */
  PersonIdentifierTypeEnum_NIDN PersonIdentifierTypeEnum = iota + 1
  /**
   * Natural Person Identifier. To identify the person who is acting as private individual, not as business entity. Used for regulatory reporting.
   */
  PersonIdentifierTypeEnum_NPID PersonIdentifierTypeEnum = iota + 1
  /**
   * Privacy Law Identifier. It refers to the DMO Letter No. 17-16, http://www.cftc.gov/idc/groups/public/@lrlettergeneral/documents/letter/17-16.pdf
   */
  PersonIdentifierTypeEnum_PLID PersonIdentifierTypeEnum = iota + 1
  /**
   * Social Security Number, number assigned by an authority to identify the social security number of a person.
   */
  PersonIdentifierTypeEnum_SOSE PersonIdentifierTypeEnum = iota + 1
  /**
   * Tax Identification Number, number assigned by a tax authority to identify a person.
   */
  PersonIdentifierTypeEnum_TXID PersonIdentifierTypeEnum = iota + 1
  )    
