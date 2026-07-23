package org_isda_cdm

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */

/**
 * The enumeration values to qualify the type of account.
 */
type AccountTypeEnum int			
/**
 * The enumeration values to specify the actions associated with transactions.
 */
type ActionEnum int			
/**
 * Specifies the types of events found in the related Master Agreement that may lead to the early termination of the Master Agreement, including suspension of the affected party's rights to rehypothecate, transfer, or substitute collateral.
 */
type AdditionalTerminationEventEnum int			
/**
 * The enumerated values to specify the Additional Type of transaction that can require the collection or delivery of initial margin under a given regulatory regime for the purposes of Covered Transactions.
 */
type AdditionalTypeEnum int			
/**
 * Enumeration for the different types of affirmation status.
 */
type AffirmationStatusEnum int			
/**
 * If there is an alternative to interest amounts, how is it specified?
 */
type AlternativeToInterestAmountEnum int			
/**
 * The enumerated values to specify the effective date of the Amendment to Termination Currency when specified as a specific date (e.g. the annex date).
 */
type AmendmentEffectiveDateEnum int			
/**
 * Defines the enumerated values to specify the ancillary roles to the transaction. The product is agnostic to the actual parties involved in the transaction, with the party references abstracted away from the product definition and replaced by the AncillaryRoleEnum. The AncillaryRoleEnum can then be positioned in the product and the AncillaryParty type, which is positioned outside of the product definition, allows the AncillaryRoleEnum to be associated with an actual party reference.
 */
type AncillaryRoleEnum int			
/**
 * An arithmetic operator that can be passed to a function
 */
type ArithmeticOperationEnum int			
/**
 * Specifies the underlying asset or asset pool securing an Asset Backed debt instrument.
 */
type AssetBackedEnum int			
/**
 * The enumerated values to specify the FpML asset class categorization.
 */
type AssetClassEnum int			
/**
 * Extends product identifiers with additional identifier sources for Assets.
 */
type AssetIdTypeEnum int			
/**
 * An enumerator to differentiate the different trade types used in securities finance and modelled on an AssetPayout.
 */
type AssetPayoutTradeTypeEnum int			
/**
 * The qualification of the type of asset transfer.
 */
type AssetTransferTypeEnum int			
/**
 * Represents an enumeration list to identify the asset type.
 */
type AssetTypeEnum int			
/**
 * Enumeration to describe the type of AvailableInventory
 */
type AvailableInventoryTypeEnum int			
/**
 * Indicates the type of equity average trading volume (single) the highest amount on one exchange, or (consolidated) volumes across more than one exchange.
 */
type AverageTradingVolumeMethodologyEnum int			
/**
 * Specifies enumerations for the type of averaging calculation.
 */
type AveragingCalculationMethodEnum int			
/**
 * The enumerated values to specify the type of averaging used in an Asian option.
 */
type AveragingInOutEnum int			
/**
 * The enumerated values to specify the method of calculation to be used when averaging rates. Per ISDA 2000 Definitions, Section 6.2. Certain Definitions Relating to Floating Amounts.
 */
type AveragingWeightingMethodEnum int			
/**
 * Defines whether the bank holidays are treated as weekdays or weekends in terms of delivery profile in the context of commodity products, in particular those with peak or off-peak delivery profiles.
 */
type BankHolidayTreatmentEnum int			
/**
 * The enumerated values to specify the type of Broker Confirm that the FpML trade represents.
 */
type BrokerConfirmationTypeEnum int			
/**
 * The enumerated values to specify the business centers.
 */
type BusinessCenterEnum int			
/**
 * The enumerated values to specify the convention for adjusting any relevant date if it would otherwise fall on a day that is not a valid business day.
 */
type BusinessDayConventionEnum int			
/**
 * Specifies the types of events in the related Master Agreement that, when triggered, could temporarily or permanently suspend rights to rehypothecate, transfer or substitute collateral.
 */
type CSASpecifiedOrAccessConditionEnum int			
/**
 * What calculation type is required, averaging or compounding. This enumeration is used to represent the definitions of modular calculated rates as described in the 2021 ISDA Definitions, section 7.
 */
type CalculationMethodEnum int			
/**
 *  the specific calculation method, e.g. lookback. This enumeration is used to represent the definitions of modular calculated rates as described in the 2021 ISDA Definitions, section 7.
 */
type CalculationShiftMethodEnum int			
/**
 * Specification of which party is calculating the value of collateral to be delivered or returned.
 */
type CalculationValuationAgentPartyEnum int			
/**
 * Represents the enumeration values that indicate the intended status of message type, such as expected call, notification of a call or a margin call.
 */
type CallTypeEnum int			
/**
 * Identifies a party to the on-demand repo transaction that has a right to demand for termination of the Security Finance transaction.
 */
type CallingPartyEnum int			
/**
 * Provides enumerated values for capacity units, generally used in the context of defining quantities for commodities.
 */
type CapacityUnitEnum int			
/**
 * Details the day on which cash collateral is required to be transferred relative to the Notification Time.
 */
type CashCTSTimeEnum int			
/**
 * Defines the different cash settlement methods for a product where cash settlement is applicable.
 */
type CashSettlementMethodEnum int			
/**
 * The enumerated values to specify what led to the contract or execution closure.
 */
type ClosedStateEnum int			
type CollateralAssetDefinitionsEnum int			
/**
 * Represents an enumeration list to identify the type of entity issuing or guaranteeing the collateral asset.
 */
type CollateralEntityTypeEnum int			
/**
 * How is collateral interest to be handled?
 */
type CollateralInterestHandlingEnum int			
/**
 * The enumerated values to specify the type of margin for which a legal agreement is named.
 */
type CollateralMarginTypeEnum int			
/**
 * Represents the enumeration list to identify the settlement status of the collateral.
 */
type CollateralStatusEnum int			
/**
 * Specifies the types of collateral that are accepted by the Lender
 */
type CollateralTypeEnum int			
/**
 * Specifies the type of Collateralized Obligation.
 */
type CollateralizedObligationEnum int			
type CommodityBusinessCalendarEnum int			
/**
 * Defines a publication in which the price can be found. (e.g Gas Daily, Platts Bloomberg.
 */
type CommodityInformationPublisherEnum int			
/**
 * Defines the enumerated values to specify the nature of a location identifier.
 */
type CommodityLocationIdentifierTypeEnum int			
/**
 * The enumeration values to specify the Commodity Reference Prices specified in the Annex to the 2005 ISDA Commodity Definitions.
 */
type CommodityReferencePriceEnum int			
type CompareOp int			
/**
 * The enumerated values to specify the type of compounding, e.g. flat, straight.
 */
type CompoundingMethodEnum int			
/**
 * The enumerated values to specify how the compounding calculation is done
 */
type CompoundingTypeEnum int			
/**
 * Represents the enumerated values to identify where a concentration limit is applied in the eligible collateral schedule.
 */
type ConcentrationLimitTypeEnum int			
/**
 * Enumeration for the different types of confirmation status.
 */
type ConfirmationStatusEnum int			
/**
 * The enumerated values to specify a set of standard contract definitions relevant to the transaction.
 */
type ContractualDefinitionsEnum int			
/**
 * The enumerated values to define the supplements to a base set of ISDA Definitions that are applicable to the transaction.
 */
type ContractualSupplementTypeEnum int			
/**
 * The enumerated values to specify the origin of a corporate action transfer.
 */
type CorporateActionTypeEnum int			
/**
 * Defines the enumerated values to specify the two counterparties to the transaction.
 */
type CounterpartyRoleEnum int			
/**
 * Specifies the type of Covered Bond.
 */
type CoveredBondEnum int			
/**
 * Represents the enumerated values to specify a credit event type.
 */
type CreditEventTypeEnum int			
/**
 * The enumeration values to qualify the type of credit limits.
 */
type CreditLimitTypeEnum int			
/**
 * Identifies an agency rating as a simple scale boundary of minimum or maximum.
 */
type CreditNotationBoundaryEnum int			
/**
 * Represents an enumeration list to identify the characteristics of the rating if there are several agency issue ratings but not equivalent, reference will be made to label characteristics of the rating such as the lowest/highest available.
 */
type CreditNotationMismatchResolutionEnum int			
/**
 * Represents the enumerated values to specify the rating agencies.
 */
type CreditRatingAgencyEnum int			
/**
 * Represents the enumerated values to specify the credit watch rating.
 */
type CreditRatingCreditWatchEnum int			
/**
 * Represents the enumerated values to specify the credit rating outlook.
 */
type CreditRatingOutlookEnum int			
/**
 * Represents an enumeration list to identify tranched or untranched credit risk.
 */
type CreditRiskEnum int			
/**
 * Seniority of debt instruments comprising the index.
 */
type CreditSeniorityEnum int			
/**
 * The enumerated values to specify the type of Credit Support Agreement governing the transaction.
 */
type CreditSupportAgreementTypeEnum int			
/**
 * Specifies whether the meaning of credit support amount follows the pre-print definition or deviates from the standard (either based on a rating or some other methodology).
 */
type CreditSupportAmountEnum int			
/**
 * The enumerated values to specify the Credit Support Document Terms
 */
type CreditSupportDocumentTermsEnum int			
/**
 * The enumerated values to specify the Credit Support Provider Terms
 */
type CreditSupportProviderTermsEnum int			
/**
 * How is the Creadit Support Annex defined for this transaction as defined in the 2021 ISDA Definitions, section 18.2.1 
 */
type CsaTypeEnum int			
/**
 * Union of the enumerated values defined by the International Standards Organization (ISO) and the FpML nonISOCurrencyScheme which consists of offshore and historical currencies (https://www.fpml.org/coding-scheme/non-iso-currency), as of 28-Oct-2016.
 */
type CurrencyCodeEnum int			
/**
 * The enumerated values to specify the day count fraction.
 */
type DayCountFractionEnum int			
/**
 * Denotes the method by which the pricing days are distributed across the pricing period.
 */
type DayDistributionEnum int			
/**
 * The enumerated values to specify a day of the seven-day week.
 */
type DayOfWeekEnum int			
/**
 * Lists the enumerated values to specify the day type classification used in counting the number of days between two dates.
 */
type DayTypeEnum int			
/**
 * Represents an enumeration list that identifies the type of debt.
 */
type DebtClassEnum int			
/**
 * Represents an enumeration list that specifies the general rule for periodic interest rate payment.
 */
type DebtInterestEnum int			
/**
 * Represents an enumeration list that specifies the general rule for repayment of principal.
 */
type DebtPrincipalEnum int			
/**
 * Specifies the order of repayment in the event of a sale or bankruptcy of the issuer or a related party (eg guarantor).
 */
type DebtSeniorityEnum int			
/**
 * The enumerated values to specify the application of Interest Amount with respect to the Delivery Amount through standard language.
 */
type DeliveryAmountElectionEnum int			
/**
 * Specifies delivery methods for securities transactions. This coding-scheme defines the possible delivery methods for securities.
 */
type DeliveryMethodEnum int			
type DeliveryNearbyTypeEnum int			
/**
 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be transferred to it as a Delivery or Return Amount.
 */
type DeliveryReturnAmountEnum int			
type DepositaryReceiptTypeEnum int			
/**
 * The enumerated values to specify the method according to which an amount or a date is determined.
 */
type DeterminationMethodEnum int			
/**
 * Defines the enumerated values to specify the determination roles to the transaction. Such roles mostly address any determination that would be required when some Extraordinary Events would happen, for instance qualifying the effectiveness of such event, or when a calculation is required, etc. else any other kind of determination as need be.
 */
type DeterminationRoleEnum int			
/**
 * The enumerated values to specify the method of calculating discounted payment amounts. This enumerations combines the FpML DiscountingTypeEnum and FraDiscountingEnum enumerations.
 */
type DiscountingTypeEnum int			
/**
 * The enumerated values to specify whether the dividend is paid with respect to the Dividend Period.
 */
type DividendAmountTypeEnum int			
/**
 * The enumerated values to specify how the composition of Dividends is to be determined.
 */
type DividendCompositionEnum int			
/**
 * The enumerated values to specify the date by reference to which the dividend will be paid.
 */
type DividendDateReferenceEnum int			
/**
 * The enumerated values to specify the date on which the receiver of the equity payout is entitled to the dividend.
 */
type DividendEntitlementEnum int			
/**
 * 2002 ISDA Equity Derivatives Definitions: First Period, Second Period |
 */
type DividendPeriodEnum int			
/**
 * Identifies European Union Eligible Collateral Assets classification categories based on EMIR Uncleared Margin Rules. Eligible Collateral asset classes for both initial margin (IM) and variation margin (VM) posted and collected between specified entities. Please note: EMIR regulation will detail which eligible collateral assets classes apply to each type of entity pairing (counterparty) and which apply to posting of IM and VM.
 */
type EU_EMIR_EligibleCollateralEnum int			
/**
 * The enumerated values to specify an elective amount.
 */
type ElectiveAmountEnum int			
/**
 * The enumerated values to specify the reference entity types corresponding to a list of types defined in the ISDA First to Default documentation.
 */
type EntityTypeEnum int			
/**
 * The enumerated values to specify an Equity index.
 */
type EquityIndexEnum int			
/**
 * Represents an enumeration list to identify the type of Equity.
 */
type EquityTypeEnum int			
/**
 * The enumeration values to qualify the intent associated with a transaction event.
 */
type EventIntentEnum int			
/**
 * The enumeration values to qualify the timestamps that can be associated with a lifecycle event. The reason for such approach is that the experience of integrating the DTCC and CME data representations suggests that a wide set of timestamps are currently utilized among service providers, while there is not at present an objective set of criteria that could help suggest a defined set of timestamps as part of the CDM. Implementers are expected to evaluate the current enumeration values to determine whether those meet their requirements. If not, they are expected to engage with the CDM team to evaluate the addition of further value(s) to this enumeration, which will then participate to the development of a compendium for further evaluation at a later point in order to determine whether this modeling is appropriate.
 */
type EventTimestampQualificationEnum int			
/**
 * Values to specify the normalized exceptions applicable to an Initial Margin CSA.
 */
type ExceptionEnum int			
/**
 * The enumerated values to specify the Execution Location of a Security Agreement
 */
type ExecutionLocationEnum int			
/**
 * The enumerated values to specify how a contract has been executed, e.g. electronically, verbally, ...
 */
type ExecutionTypeEnum int			
/**
 * Defines the principal party to the trade that has the right to exercise.
 */
type ExerciseNoticeGiverEnum int			
/**
 * The time of day at which the equity option expires, for example the official closing time of the exchange.
 */
type ExpirationTimeTypeEnum int			
/**
 * Description of the relevant derivative transactions excluded from the calculation of exposure.
 */
type ExposureScopeProductEnum int			
/**
 * Specifies the fallback provisions in respect to the applicable Futures Price Valuation.
 */
type FPVFinalPriceElectionFallbackEnum int			
/**
 * The enumerated values to specify an event that has given rise to a fee.
 */
type FeeTypeEnum int			
/**
 * To be specified only for products that embed a redemption payment.
 */
type FinalPrincipalExchangeCalculationEnum int			
/**
 * Provides enumerated values for financial units, generally used in the context of defining quantities for securities.
 */
type FinancialUnitEnum int			
/**
 * 3rd level ISDA FRO category.
 */
type FloatingRateIndexCalculationMethodEnum int			
/**
 * Top level ISDA FRO category.
 */
type FloatingRateIndexCategoryEnum int			
/**
 * The enumerated values to specify the list of floating rate index.
 */
type FloatingRateIndexEnum int			
/**
 * This enumeration provides guidance on how to process a given floating rate index.  It's based on the ISDA Floating Rate Index information, but transforms it into the specific categories needed for calculation 
 */
type FloatingRateIndexProcessingTypeEnum int			
/**
 * Second level ISDA FRO category.
 */
type FloatingRateIndexStyleEnum int			
/**
 * Represents an enumeration list to identify the fund product type.
 */
type FundProductTypeEnum int			
/**
 * The enumerated values to specify whether such terms i.e. Force Majeure or Act Of State shall be treated either as defined per the documents applicable to the FX transaction.
 */
type FxForceMajeureOrActOfSStateEnum int			
/**
 * The enumerated values to specify whether such terms i.e. Illegality or Impossibility shall be treated either as defined per the documents applicable to the FX transaction.
 */
type FxIllegalityOrImpossibilityEnum int			
type FxSubstitutionProvisionTypeEnum int			
/**
 * The enumerated values to specify the law governing the contract or legal document.
 */
type GoverningLawEnum int			
/**
 * Represents the enumeration indicators to specify if an asset or group of assets valuation is based on any valuation treatment haircut.
 */
type HaircutIndicatorEnum int			
/**
 * The enumerated values to specify condition(s) required by a party from the other party to hold its posted collateral.
 */
type HoldingPostedCollateralEnum int			
/**
 * The enumerated values to specify standard currency codes according to the International Standards Organization (ISO). The set of codes in this enumerated list is sourced from ISO Standard 3166 (ISO-3166-1alpha-2)(https://www.iso.org/iso-3166-country-codes.html) as at 14-Aug-23.
 */
type ISOCountryCodeEnum int			
/**
 * The enumerated values to specify standard currency codes according to the International Standards Organization (ISO).  The set of codes in this enumerated list is sourced from ISO Standard 4217 (https://www.currency-iso.org/en/home/tables/table-a1.html), as of 29-Aug-18.
 */
type ISOCurrencyCodeEnum int			
/**
 * Specifies how multiple credit ratings are compared when determining the applicable Independent Amount, and whether that measure is highest, lowest, or a comparison of the ratings.
 */
type IndependentAmountCompareEnum int			
/**
 * The enumerated values to specify the instances where the independent amount eligible collateral is not defined as a set of eligible collateral assets.
 */
type IndependentAmountEligibilityEnum int			
/**
 * The enumerated values to specify the CDX index annex source.
 */
type IndexAnnexSourceEnum int			
/**
 * The enumerated values to specify the consequences of Index Events.
 */
type IndexEventConsequenceEnum int			
/**
 * Indicates how to use the inflation index to calculate the payment (e.g. Ratio, Return, Spread). Added for Inflation Asset Swap
 */
type InflationCalculationMethodEnum int			
/**
 * Indicates the style of how the inflation index calculates the payment (e.g. YearOnYear, ZeroCoupon).
 */
type InflationCalculationStyleEnum int			
/**
 * The enumerated values to specify the list of inflation rate indices.
 */
type InflationRateIndexEnum int			
/**
 * The enumerated values to specify the list of information providers.
 */
type InformationProviderEnum int			
/**
 * The enumeration values indicating the BusinessEvent function associated input instructions.
 */
type InstructionFunctionEnum int			
/**
 * Represents an enumeration list to indentify the type of an instrument.
 */
type InstrumentTypeEnum int			
/**
 * Values to specify the interest adjustment periodicity election through standard language.
 */
type InterestAdjustmentPeriodicityEnum int			
/**
 * The enumerated values to specify the interest shortfall cap, applicable to mortgage derivatives.
 */
type InterestShortfallCapEnum int			
/**
 * The enumerated values to specify the interpolation method, e.g. linear.
 */
type InterpolationMethodEnum int			
/**
 * Details of the consent requirements (if any) for Substitution.
 */
type LegacyConsentEnum int			
/**
 * Details of the alternative dispute resolution procedure (if any).
 */
type LegacyResolutionAlternativeEnum int			
/**
 * Details of how the Custodian is determined.
 */
type LegacyVMCustodianEnum int			
/**
 * The enumerated values to specify the legal agreement publisher.
 */
type LegalAgreementPublisherEnum int			
/**
 * The enumerated values to specify the legal agreement type.
 */
type LegalAgreementTypeEnum int			
/**
 * The enumerated values to specify the length unit in the Resource type.
 */
type LengthUnitEnum int			
/**
 * The enumeration values to specify the level at which the limit is set: customer business, proprietary business or account level. This is part of the CME specification for clearing credit limits, although not specified as a set of enumerated values as part of the clearing confirmation specification.
 */
type LimitLevelEnum int			
/**
 * Specifies the load type of the delivery.
 */
type LoadTypeEnum int			
/**
 * Values to specify the margin approach specific to Initial Margin agreements.
 */
type MarginApproachEnum int			
/**
 * Represents the enumeration values to identify the collateral action instruction.
 */
type MarginCallActionEnum int			
/**
 * Represents the enumeration values to define the response type to a margin call.
 */
type MarginCallResponseTypeEnum int			
/**
 * This indicator defines which type of assets (cash or securities) is specified to apply as margin to the repo transaction.
 */
type MarginTypeEnum int			
/**
 * The enumerated values to specify the handling of an averaging date market disruption for an equity derivative transaction.
 */
type MarketDisruptionEnum int			
type MasterAgreementClauseIdentifierEnum int			
/**
 * The enumerated values to specify the type of the master agreement governing the transaction.
 */
type MasterAgreementTypeEnum int			
type MasterAgreementVariantIdentifierEnum int			
/**
 * The enumerated values to specify the type of annex to be used with master confirmation agreement governing the transaction.
 */
type MasterConfirmationAnnexTypeEnum int			
/**
 * The enumerated values to specify the type of master confirmation agreement governing the transaction. While FpML positions the date a prefix, the CDM positions it as the suffix to handle grammar type constraints.
 */
type MasterConfirmationTypeEnum int			
/**
 * The enumerated values to specify a scheme of transaction types specified in the Equity Derivatives Settlement Matrix.
 */
type MatrixTermEnum int			
/**
 * The enumerated values to specify the identification the form of applicable matrix.
 */
type MatrixTypeEnum int			
/**
 * Represents an enumeration list to identify the Maturity.
 */
type MaturityTypeEnum int			
type MoneyMarketTypeEnum int			
/**
 * Defines the consequences of nationalization, insolvency and delisting events relating to the underlying.
 */
type NationalizationOrInsolvencyOrDelistingEventEnum int			
/**
 * The enumerated values for the natural person's role.
 */
type NaturalPersonRoleEnum int			
/**
 * The enumerated values to specify the method of calculating payment obligations when a floating rate is negative (either due to a quoted negative floating rate or by operation of a spread that is subtracted from the floating rate).
 */
type NegativeInterestRateTreatmentEnum int			
/**
 * The enumerated values to specify the treatment of Non-Cash Dividends.
 */
type NonCashDividendTreatmentEnum int			
/**
 * The enumerated values applicable to define the what conditions apply to trigger a Not Rated condition.
 */
type NotRatedByEnum int			
/**
 * The enumerated values to specify the conditions that govern the adjustment to the number of units of the return swap.
 */
type NotionalAdjustmentEnum int			
/**
 * The enumerated values to define the number of Rating Agencies that must be considered to meet the rating condition.
 */
type NumberOfRatingAgenciesEnum int			
/**
 * The enumerated values used in both the obligations and deliverable obligations of the credit default swap to represent a class or type of securities which apply.
 */
type ObligationCategoryEnum int			
/**
 * The enumerated values to specify whether rate calculations occur relative to the first or last day of a calculation period. Done in uppercase due to a bug in code generation. This enumeration is used to represent the definitions of modular calculated rates as described in the 2021 ISDA Definitions, section 7.
 */
type ObservationPeriodDatesEnum int			
/**
 * The enumerated values to specify the option exercise style. i.e., European, Bermuda or American.
 */
type OptionExerciseStyleEnum int			
/**
 * The enumerated values to specify the type or strategy of the option.
 */
type OptionTypeEnum int			
/**
 * The enumerated values to specify how a calculation agent will be determined.
 */
type PartyDeterminationEnum int			
/**
 * The enumeration values associated with party identifier sources.
 */
type PartyIdentifierTypeEnum int			
/**
 * The enumerated values for the party role. The enumerated values go beyond the FpML partyRoleScheme as they also include elements that are part of the FpML Trade, such as the Barrier Determination Agent and the Hedging Party.
 */
type PartyRoleEnum int			
/**
 * The enumerated values to specify whether payments occur relative to the calculation period start date or end date, each reset date, valuation date or the last pricing date.
 */
type PayRelativeToEnum int			
/**
 * The enumerated values to specify an interest rate stream payer or receiver party.
 */
type PayerReceiverEnum int			
/**
 * The enumerated values to specify the origin of a performance transfer
 */
type PerformanceTransferTypeEnum int			
/**
 * The enumerated values to specify the period, e.g. day, week.
 */
type PeriodEnum int			
/**
 * The enumerated values to specify a time period containing the additional value of Term.
 */
type PeriodExtendedEnum int			
/**
 * The enumeration values to specify a time period containing additional values such as Term.
 */
type PeriodTimeEnum int			
/**
 * The enumeration values associated with person identifier sources.
 */
type PersonIdentifierTypeEnum int			
type PositionEventIntentEnum int			
/**
 * Enumeration to describe the different (risk) states of a Position, whether executed, settled, matured...etc
 */
type PositionStatusEnum int			
/**
 * The enumerated values to specify the premium type for forward start options.
 */
type PremiumTypeEnum int			
/**
 * Enumerated values to specify whether the price is expressed in absolute or relative terms.
 */
type PriceExpressionEnum int			
type PriceOperandEnum int			
/**
 * Provides a list of possible sub types of the chosen price type, used in conjunction with the PriceTypeEnum.
 */
type PriceSubTypeEnum int			
type PriceTimingEnum int			
/**
 * Provides enumerated values for types of prices in the Price data type in order to explain how to interpret the amount and use it in calculations.
 */
type PriceTypeEnum int			
/**
 * Identifies the grade of physical commodity product to be delivered.
 */
type ProductGradeEnum int			
/**
 * Provides the enumerated values to specify the product identifier source.
 */
type ProductIdTypeEnum int			
/**
 * Specifies different types of property.
 */
type PropertyTypeEnum int			
/**
 * The enumerated values to specify the types of listed derivative options.
 */
type PutCallEnum int			
/**
 * Represents the enumerated values to specify a logical quantification, i.e. either All or Any.
 */
type QuantifierEnum int			
/**
 * Specifies whether a quantity change is an increase, a decrease or a replacement, whereby the quantity is always specified as a positive number.
 */
type QuantityChangeDirectionEnum int			
/**
 * The enumerated values to specify the type of quotation rate to be obtained from each cash settlement reference bank.
 */
type QuotationRateTypeEnum int			
/**
 * The enumerated values to specify the side from which perspective a value is quoted.
 */
type QuotationSideEnum int			
/**
 * The enumerated values to specify the actual quotation style (e.g. PointsUpFront, TradedSpread) used to quote a credit default swap fee leg.
 */
type QuotationStyleEnum int			
/**
 * The enumerated values to specify how an exchange rate is quoted.
 */
type QuoteBasisEnum int			
/**
 * The enumerated values to specify the methods for converting rates from one basis to another.
 */
type RateTreatmentEnum int			
/**
 * The enumerated values applicable where a Rated Party is defined within the relevant clause.
 */
type RatedPartyEnum int			
/**
 * Represents an enumeration list to identify which Collateral Criteria type should have priority over others. If set to 'Issuer', the rating in the 
 Issuer Criteria has priority or is used if there is no Asset criteria. If set to 'Asset', the rating in the Asset Criteria has priority or is used if there is no Issuer rating.
 */
type RatingPriorityResolutionEnum int			
/**
 * The relevant rating type.
 */
type RatingTypeEnum int			
/**
 * The contract specifies which price must satisfy the boundary condition.  Used for variance, volatility and correlation caps and floors.
 */
type RealisedVarianceMethodEnum int			
/**
 * Values to specify the procedure under which the market value of posted collateral will be recalculated.
 */
type RecalculationOfValueElectionEnum int			
/**
 * The enumeration of the account level for the billing summary.
 */
type RecordAmountTypeEnum int			
/**
 * Represents the enumeration values to specify the role of the party in relation to a regulatory initial margin call.
 */
type RegIMRoleEnum int			
/**
 * Represents the enumeration values to specify the margin type in relation to bilateral or regulatory obligation.
 */
type RegMarginTypeEnum int			
/**
 * Values to specify the regulatory regimes.
 */
type RegulatoryRegimeEnum int			
/**
 * A duration code for a Repo (or Securities Lending) transaction. There are many business and market rules that are derived from the duration of the transaction.
 */
type RepoDurationEnum int			
/**
 * The enumerated values to specify whether resets occur relative to the first or last day of a calculation period.
 */
type ResetRelativeToEnum int			
/**
 * The enumerated values to specify the type of a resource (e.g. document).
 */
type ResourceTypeEnum int			
/**
 * The enumerated values to specify the form of the restructuring credit event that is applicable to the credit default swap.
 */
type RestructuringEnum int			
/**
 * The enumerated values to specify the type of return associated the equity payout.
 */
type ReturnTypeEnum int			
/**
 * The enumerated values to specify the period term as part of a periodic schedule, i.e. the calculation period end date within the regular part of the calculation period. The value could be a rule, e.g. IMM Settlement Dates, which is the 3rd Wednesday of the month, or it could be a specific day of the month, such as the first day of the applicable month.
 */
type RollConventionEnum int			
/**
 * Used in conjunction with an exchange-based pricing source. Identifies a date source calendar from which the pricing dates and thus roll to the next contract will be based off (e.g. pricing is based on the NYMEX WTI First Nearby Futures Contract, if Future is chosen, the pricing will roll to the next futures contract on expiration, if ListedOption is chosen, the pricing will roll to the next futures contract on the Option expiration date which is three business days before the expiration of the NYMEX WTI futures contract.) Omitting this element will result in the default behavior expected with the pricing source described within the commodity element.
 */
type RollSourceCalendarEnum int			
/**
 * The enumerated values to specify the rounding direction and precision to be used in the rounding of a number.  Used by function cdm.base.math.RoundToPrecision.
 */
type RoundingDirectionEnum int			
/**
 * How often is rounding performed
 */
type RoundingFrequencyEnum int			
/**
 * The enumerated values to specify the rounding direction when rounding of a number to nearest.  Used by function cdm.base.math.RoundToNearest.
 */
type RoundingModeEnum int			
/**
 * The qualification of the type of transfers and cash flows associated with contracts and their lifecycle events.
 */
type ScheduledTransferEnum int			
/**
 * Specifies the type of secured debt.
 */
type SecuredTypeEnum int			
/**
 * Details the day on which securities collateral is required to be transferred relative to the Notification Time.
 */
type SecuritiesCTSTimeEnum int			
/**
 * The enumerated values applicable for Security Interest for Obligations.
 */
type SecurityInterestObligationsEnum int			
/**
 * The enumerated values applicable for Security Interest for Obligations Obligee.
 */
type SecurityInterestObligeeEnum int			
/**
 * Values to specify the methodology according to which sensitivities to (i) equity indices, funds and ETFs, and (ii) commodity indices are computed.
 */
type SensitivitiesEnum int			
/**
 * The enumerated values to specify the relevant settled entity matrix source.
 */
type SettledEntityMatrixSourceEnum int			
/**
 * Defines the settlement centre for a securities transaction.
 */
type SettlementCentreEnum int			
/**
 * The enumerated values to specify the settlement rate options as specified in the Annex A to the 1998 FX and Currency Options Definitions.
 */
type SettlementRateOptionEnum int			
/**
 * The enumeration values to specify how the option is to be settled when exercised.
 */
type SettlementTypeEnum int			
/**
 * The enumerated values to specify the consequences of extraordinary events relating to the underlying.
 */
type ShareExtraordinaryEventEnum int			
/**
 * Values to specify the SIMM normalized exception approaches.
 */
type SimmExceptionApplicableEnum int			
/**
 * The enumerated values to specify the Event of Default or Termination event for which Specified Entities terms are being defined.
 */
type SpecifiedEntityClauseEnum int			
/**
 * The enumerated values to specify the specified entity terms for the Event of Default or Termination Event specified.
 */
type SpecifiedEntityTermsEnum int			
type SpecifiedValueEnum int			
/**
 * Method by which spread is calculated. For example on an asset swap: 'ParPar' or 'Proceeds' may be the method indicated.
 */
type SpreadCalculationMethodEnum int			
/**
 * The enumerated values to specify a long or short spread value.
 */
type SpreadScheduleTypeEnum int			
/**
 * The enumerated values to specify whether a trade is settling using standard settlement instructions as well as whether it is a candidate for settlement netting.
 */
type StandardSettlementStyleEnum int			
type StandardizedScheduleAssetClassEnum int			
type StandardizedScheduleProductClassEnum int			
/**
 * The enumerated values to specify how to deal with a non standard calculation period within a swap stream.
 */
type StubPeriodTypeEnum int			
/**
 * Represents an enumeration list to identify the type of supranational entity issuing the asset.
 */
type SupraNationalIssuerTypeEnum int			
/**
 * Represents the enumerated values to specify taxonomy sources.
 */
type TaxonomySourceEnum int			
/**
 * The enumerated values to specify the type of telephone number, e.g. work vs. mobile.
 */
type TelephoneTypeEnum int			
type TerminationCurrencyConditionEnum int			
/**
 * The enumerated values to specify points in the day when option exercise and valuation can occur.
 */
type TimeTypeEnum int			
/**
 * The enumeration values to qualify the allowed units of time.
 */
type TimeUnitEnum int			
/**
 * Defines the enumerated values to specify the nature of a trade identifier.
 */
type TradeIdentifierTypeEnum int			
/**
 * The enumeration values to specify how the transfer will settle, e.g. DvP.
 */
type TransferSettlementEnum int			
/**
 * The enumeration values to specify the transfer status.
 */
type TransferStatusEnum int			
/**
 * The enumerated values to specify the time of day which would be considered for valuing the knock event.
 */
type TriggerTimeTypeEnum int			
/**
 * The enumerated values to specify whether an option will trigger or expire depending upon whether the spot rate is above or below the barrier rate.
 */
type TriggerTypeEnum int			
/**
 * Identifies United Kingdom Eligible Collateral Assets classification categories based on UK Onshored EMIR Uncleared Margin Rules. Eligible Collateral asset classes for both initial margin (IM) and variation margin (VM) posted and collected between specified entities. Please note: UK EMIR regulation will detail which eligible collateral assets classes apply to each type of entity pairing (counterparty) and which apply to posting of IM and VM.
 */
type UK_EMIR_EligibleCollateralEnum int			
/**
 * Identifies US Eligible Collateral Assets classification categories based on Uncleared Margin Rules published by the CFTC and the US Prudential Regulator. Note: While the same basic categories exist in the CFTC and US Prudential Regulators margin rules, the precise definitions or application of those rules could differ between the two rules.
 */
type US_CFTC_PR_EligibleCollateralEnum int			
type UmbrellaPartyRoleEnum int			
/**
 * Specifies how the Valuation Agent is determined.
 */
type ValuationAgentDeterminationEnum int			
/**
 * Defines whether the date of a Valuation is defined by a certain day of the week or a particular date in the month.
 */
type ValuationDateDateEnum int			
/**
 * Defines the conditions for the day for a Valuation Date.
 */
type ValuationDateDayEnum int			
/**
 * The regularity of the event for the Valuation Date.
 */
type ValuationDateFrequencyEnum int			
/**
 * The enumerated values to specify the ISDA defined methodology for determining the final price of the reference obligation for purposes of cash settlement.
 */
type ValuationMethodEnum int			
/**
 * Scope of the valuation i.e. what it applies to.
 */
type ValuationScopeEnum int			
/**
 * Source for the valuation of the transaction by the valuation party.
 */
type ValuationSourceEnum int			
/**
 * The day at which the value of the collateral and obligations to transfer or return collateral are to be calculated relative to the Valuation Date.
 */
type ValuationTimeDayEnum int			
/**
 * The type of time specified for the Valuation Time.
 */
type ValuationTimeEnum int			
/**
 * Method used for the valuation of the transaction by the valuation party.
 */
type ValuationTypeEnum int			
/**
 * Details of how cash collateral is valued when resolving disputes.
 */
type ValueCashEnum int			
/**
 * Details of how securities collateral is valued when resolving disputes.
 */
type ValueSecuritiesEnum int			
type WarehouseIdentityEnum int			
/**
 * Provides enumerated values for weather units, generally used in the context of defining quantities for commodities.
 */
type WeatherUnitEnum int			
/**
 * The enumerated values to specify the weekly roll day.
 */
type WeeklyRollConventionEnum int			
type WorkflowStatusEnum int			
/**
 * The enumerated values for defining the relevant trigger(s) for the Minimum Transfer Amount (MTA) to fall to zero.
 */
type ZeroEventEnum int			
