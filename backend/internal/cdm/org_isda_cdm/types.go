package org_isda_cdm

/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */

import (
	"time"

	. "github.com/hondyman/semlayer/backend/internal/cdm/org_isda_cdm_metafields"
)

/**
 * A class to specify an account as an account number alongside, optionally. an account name, an account type, an account beneficiary and a servicing party.
 */
type Account struct {
	/**
	 * A reference to the party to which the account refers to.
	 */
	PartyReference ReferenceWithMeta
	/**
	 * The account number.
	 */
	AccountNumber FieldWithMeta
	/**
	 * The name by which the account is known.
	 */
	AccountName FieldWithMeta
	/**
	 * The type of account, e.g. client, house.
	 */
	AccountType FieldWithMeta
	/**
	 * A reference to the party beneficiary of the account.
	 */
	AccountBeneficiary ReferenceWithMeta
	/**
	 * The reference to the legal entity that services the account, i.e. in the books of which the account is held.
	 */
	ServicingParty ReferenceWithMeta
	Meta           MetaFields
}

/**
 * The accrual rate and related terms, to adjust the price of an underlier impacted by a Corporate Action when economic impact consists in freezing the underlier price to the last fixing then applying accruals per each remaining underlier price fixing date required i.e. for each fixing date, adjustedPrice = lastFixingPrice x accrualFactor, where: accrualFactor = (1 + accrualRateValue x DCF).
 */
type AccrualFactor struct {
	/**
	 * The rate to be applied to the last fixing price, for price accrual calculation purposes. DatedValue may be used for the purpose of representing Price series if such calculation is required for multiple dates.
	 */
	Value FieldWithMeta
	/**
	 * Describes the input terms involved in the calculation of the accrual factor.
	 */
	CalculationTerms AccrualFactorCalculationTerms
}

/**
 * Describes the input terms involved in the calculation of the accrual factor. Optionnally, long and short stub interpolation rates can be specified.
 */
type AccrualFactorCalculationTerms struct {
	/**
	 * The duration between last fixing date and the payment date of accruals, calculated in accordance with the appropriate DayCountFraction.
	 */
	TenorTillMaturity float64
	/**
	 * The enumerated values to specify the day count fraction.
	 */
	DayCountFraction DayCountFractionEnum
	/**
	 * Describes the rate, tenor, period duration for the short and long stubs, when the accrualRate optionnaly results from an interpolation method.
	 */
	InterpolationTerms StubValue
}

type AcctOwnr struct {
	Id Id
}

/**
 * A type for defining the Additional Disruption Events.
 */
type AdditionalDisruptionEvents struct {
	ChangeInLaw            bool
	FailureToDeliver       bool
	InsolvencyFiling       bool
	HedgingDisruption      bool
	IncreasedCostOfHedging bool
	ForeignOwnershipEvent  bool
	LossOfStockBorrow      bool
	/**
	 * Specifies the maximum stock loan rate for Loss of Stock Borrow. A percentage of 5% is represented as 0.05.
	 */
	MaximumStockLoanRate       float64
	IncreasedCostOfStockBorrow bool
	/**
	 * Specifies the initial stock loan per ISDA Def. A percentage of 5% is represented as 0.05.
	 */
	InitialStockLoanRate float64
	/**
	 * Specifies the party which determines additional disruption events.
	 */
	DeterminingParty AncillaryRoleEnum
	/**
	 * Where parties may optionnaly describe any extra bespoke agreements, in regards of the standardized Extraordinary Events.
	 */
	AdditionalBespokeTerms []Clause
}

/**
 * A class to specify the events that will give rise to the payment additional fixed payments.
 */
type AdditionalFixedPayments struct {
	/**
	 * An additional Fixed Payment Event. Corresponds to the payment by or on behalf of the Issuer of an actual interest amount in respect to the reference obligation that is greater than the expected interest amount. ISDA 2003 Term: Interest Shortfall Reimbursement.
	 */
	InterestShortfallReimbursement bool
	/**
	 * An additional Fixed Payment Event. Corresponds to the payment by or on behalf of the Issuer of an actual principal amount in respect to the reference obligation that is greater than the expected principal amount. ISDA 2003 Term: Principal Shortfall Reimbursement.
	 */
	PrincipalShortfallReimbursement bool
	/**
	 * An Additional Fixed Payment. Corresponds to the payment by or on behalf of the issuer of an amount in respect to the reference obligation in reduction of the prior writedowns. ISDA 2003 Term: Writedown Reimbursement.
	 */
	WritedownReimbursement bool
}

/**
 * The election of party specific additional obligations applicable to the agreement.
 */
type AdditionalObligations struct {
	/**
	 * The party that the additional obligations apply to.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The party specific additional obligations applicable to the agreement.
	 */
	AdditionalObligations string
}

/**
 * A class to specify the Additional Representation.
 */
type AdditionalRepresentation struct {
	/**
	 * A qualification as to whether the Additional Representation is applicable.
	 */
	PartyElection []AdditionalRepresentationElection
	/**
	 * A supplemental custom election that might be specified by the parties for the purpose of specifying the Additional Representation.
	 */
	CustomElection string
}

/**
 * A class to specify the parties' Additional Representation(s) election.
 */
type AdditionalRepresentationElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The Additional Representation is applicable when True, and not applicable when False.
	 */
	IsApplicable bool
	/**
	 * Details of any Additional Representations given by the relevant parties in relation to the collateral arrangement.
	 */
	AdditionalLanguage string
}

/**
 * A class to specify Additional Representations that may be applicable to an agreement.
 */
type AdditionalRepresentations struct {
	/**
	 * The specification of the Additional Representation that may be applicable to the agreement.
	 */
	AdditionalRepresentation AdditionalRepresentation
	/**
	 * The qualification of whether Additional Information related to Regulatory Compliance and Concentration Limits is applicable or not.
	 */
	RegulatoryComplianceRepresentation bool
}

/**
 * A class to specify the Pledgor/Obligor/Chargor Additional Rights Event election.
 */
type AdditionalRightsEvent struct {
	/**
	 * The Pledgor Additional Rights Event election is applicable when True, and not applicable when False.
	 */
	IsApplicable bool
	/**
	 * The qualification of the Pledgor Additional Rights Event election, when specified.
	 */
	Qualification string
}

/**
 * A class to specify an optional termination event, additional to the Termination Events that will be deemed an Access Condition (Initial Margin CSA) or a Specified Condition (Variation Margin CSA).
 */
type AdditionalTerminationEvent struct {
	/**
	 * The name of the additional termination event.
	 */
	Name string
	/**
	 * Whether the additional termination event is applicable for the relevant party.
	 */
	ApplicableParty []CounterpartyRoleEnum
}

/**
 * The specification of the Additional Type of transaction that can require the collection or delivery of initial margin under a given regulatory regime for the purposes of Covered Transactions.
 */
type AdditionalType struct {
	/**
	 * The qualification of the Additional Type of transaction that can require the collection or delivery of initial margin when specified as a standard value.
	 */
	StandardValue AdditionalTypeEnum
	/**
	 * The qualification of the Additional Type of transaction that can require the collection or delivery of initial margin when specified as a custom value by the parties to the legal agreement.
	 */
	CustomValue string
}

/**
 * A class to specify a post or street address.
 */
type Address struct {
	/**
	 * The set of street and building number information that identifies a postal address within a city.
	 */
	Street []string
	/**
	 * The city component of the postal address.
	 */
	City string
	/**
	 * A country subdivision used in postal addresses in some countries. For example, US states, Canadian provinces, Swiss cantons, ...
	 */
	State string
	/**
	 * The ISO 3166 standard code for the country within which the postal address is located.
	 */
	Country FieldWithMeta
	/**
	 * The code, required for computerized mail sorting systems, that is allocated to a physical address by a national postal authority.
	 */
	PostalCode string
}

/**
 * Specifies the address for transfer as specified by the respective parties to the agreement.
 */
type AddressesForTransfer struct {
	/**
	 * The parties' contact and transfer information election.
	 */
	PartyElection []TransferInformationElection
}

type AddtlAttrbts struct {
	RskRdcgTx        string
	SctiesFincgTxInd string
}

/**
 * A class for defining a date that shall be subject to adjustment if it would otherwise fall on a day that is not a business day in the specified business centers, together with the convention for adjusting the date.
 */
type AdjustableDate struct {
	/**
	 * A date subject to adjustment. While in FpML this date is required, this cardinality constraint has been relaxed as part of the CDM in order to support the FRA representation, which effective and termination dates are specified in FpML as adjusted dates.
	 */
	UnadjustedDate time.Time
	/**
	 * The business day convention and financial business centers used for adjusting the date if it would otherwise fall on a day that is not a business date in the specified business centers.
	 */
	DateAdjustments BusinessDayAdjustments
	/**
	 * A pointer style reference to date adjustments defined elsewhere in the document.
	 */
	DateAdjustmentsReference ReferenceWithMeta
	/**
	 * The date once the adjustment has been performed. (Note that this date may change if the business center holidays change).
	 */
	AdjustedDate FieldWithMeta
	Meta         MetaFields
}

/**
 * A class for defining a series of dates that shall be subject to adjustment if they would otherwise fall on a day that is not a business day in the specified business centers, together with the convention for adjusting the dates.
 */
type AdjustableDates struct {
	/**
	 * A date subject to adjustment.
	 */
	UnadjustedDate []time.Time
	/**
	 * The business day convention and financial business centers used for adjusting the date if it would otherwise fall on a day that is not a business date in the specified business centers.
	 */
	DateAdjustments BusinessDayAdjustments
	/**
	 * The date(s) once the adjustment has been performed. (Note that this date may change if the business center holidays change).
	 */
	AdjustedDate []FieldWithMeta
	Meta         MetaFields
}

/**
 * A class for defining a date that shall be subject to adjustment if it would otherwise fall on a day that is not a business day in the specified business centers, together with the convention for adjusting the date.
 */
type AdjustableOrAdjustedDate struct {
	/**
	 * A date subject to adjustment.
	 */
	UnadjustedDate time.Time
	/**
	 * The business day convention and financial business centers used for adjusting the date if it would otherwise fall on a day that is not a business date in the specified business centers.
	 */
	DateAdjustments BusinessDayAdjustments
	/**
	 * The date once the adjustment has been performed. (Note that this date may change if the business center holidays change).
	 */
	AdjustedDate FieldWithMeta
	Meta         MetaFields
}

/**
 * This Rosetta class specifies the date as either an unadjusted, adjusted or relative date. It supplements the features of the AdjustableOrAdjustedDate to support the credit default swap option premium, which uses the relative date construct.
 */
type AdjustableOrAdjustedOrRelativeDate struct {
	/**
	 * A date subject to adjustment.
	 */
	UnadjustedDate time.Time
	/**
	 * The business day convention and financial business centers used for adjusting the date if it would otherwise fall on a day that is not a business date in the specified business centers.
	 */
	DateAdjustments BusinessDayAdjustments
	/**
	 * The date once the adjustment has been performed. (Note that this date may change if the business center holidays change).
	 */
	AdjustedDate FieldWithMeta
	/**
	 * A date specified as some offset to another date (the anchor date).
	 */
	RelativeDate RelativeDateOffset
}

/**
 * A class giving the choice between defining a date as an explicit date together with applicable adjustments or as relative to some other (anchor) date.
 */
type AdjustableOrRelativeDate struct {
	/**
	 * A date that shall be subject to adjustment if it would otherwise fall on a day that is not a business day in the specified business centers, together with the convention for adjusting the date.
	 */
	AdjustableDate AdjustableDate
	/**
	 * A date specified as some offset to another date (the anchor date).
	 */
	RelativeDate AdjustedRelativeDateOffset
	Meta         MetaFields
}

/**
 * A class giving the choice between defining a series of dates as an explicit list of dates together with applicable adjustments or as relative to some other series of (anchor) dates.
 */
type AdjustableOrRelativeDates struct {
	/**
	 * A series of dates that shall be subject to adjustment if they would otherwise fall on a day that is not a business day in the specified business centers, together with the convention for adjusting the date.
	 */
	AdjustableDates AdjustableDates
	/**
	 * A series of dates specified as some offset to another series of dates (the anchor dates).
	 */
	RelativeDates RelativeDates
	Meta          MetaFields
}

/**
 * A class giving the choice between defining a series of dates as an explicit list of dates together with applicable adjustments or as relative to some other series of (anchor) dates, or as a calculation period schedule.
 */
type AdjustableRelativeOrPeriodicDates struct {
	/**
	 * A series of dates that shall be subject to adjustment if they would otherwise fall on a day that is not a business day in the specified business centers, together with the convention for adjusting the date.
	 */
	AdjustableDates AdjustableDates
	/**
	 * A series of dates specified as some offset to another series of dates (the anchor dates).
	 */
	RelativeDates RelativeDates
	/**
	 * A calculation period schedule.
	 */
	PeriodicDates PeriodicDates
	Meta          MetaFields
}

/**
 * A type defining a date (referred to as the derived date) as a relative offset from another date (referred to as the anchor date) plus optional date adjustments.
 */
type AdjustedRelativeDateOffset struct {
	/**
	 * A time period multiplier, e.g. 1, 2 or 3 etc. A negative value can be used when specifying an offset relative to another date, e.g. -2 days.
	 */
	PeriodMultiplier int
	/**
	 * A time period, e.g. a day, week, month or year of the stream. If the periodMultiplier value is 0 (zero) then period must contain the value D (day).
	 */
	Period PeriodEnum
	Meta   MetaFields
	/**
	 * In the case of an offset specified as a number of days, this element defines whether consideration is given as to whether a day is a good business day or not. If a day type of business days is specified then non-business days are ignored when calculating the offset. The financial business centers to use for determination of business days are implied by the context in which this element is used. This element must only be included when the offset is specified as a number of days. If the offset is zero days then the dayType element should not be included.
	 */
	DayType DayTypeEnum
	/**
	 * The convention for adjusting a date if it would otherwise fall on a day that is not a business day, as specified by an ISDA convention (e.g. Following, Precedent).
	 */
	BusinessDayConvention BusinessDayConventionEnum
	BusinessCenters       BusinessCenters
	/**
	 * A pointer style reference to a set of financial business centers defined elsewhere in the document. This set of business centers is used to determine whether a particular day is a business day or not.
	 */
	BusinessCentersReference ReferenceWithMeta
	/**
	 * Specifies the anchor as an href attribute. The href attribute value is a pointer style reference to the element or component elsewhere in the document where the anchor date is defined.
	 */
	DateRelativeTo ReferenceWithMeta
	/**
	 * The date once the adjustment has been performed. (Note that this date may change if the business center holidays change).
	 */
	AdjustedDate time.Time
	/**
	 * The business day convention and financial business centers used for adjusting the relative date if it would otherwise fall on a day that is not a business date in the specified business centers.
	 */
	RelativeDateAdjustments BusinessDayAdjustments
}

/**
 * Describes the terms involved for adjusting the price of the underlier impacted by a Corporate Action.
 */
type AdjustmentFactor struct {
	/**
	 * The multipler value applied to the price of the underlier impacted by a Corporate Action.
	 */
	Value float64
	/**
	 * Describes the input terms involved in the calculation of the adjustment factor applied to the price of the underlier impacted by a Corporate Action.
	 */
	CalculationTerms AdjustmentFactorCalculationTerms
}

/**
 * Describes the input terms involved in the calculation of the adjustment factor applied to the price of the underlier impacted by a Corporate Action.
 */
type AdjustmentFactorCalculationTerms struct {
	/**
	 * Multiple value, say 'M/N' where 'M' is the number of shares after the event and 'N' is the number of shares prior to the event.
	 */
	ShareForShareRatio float64
	/**
	 * Multiple value, say 'M/N' where 'M' is the number of rights after the event, and 'N' of shares prior to the event.
	 */
	ShareForRightsRatio float64
	/**
	 * The price to pay per each right.
	 */
	RightsSubscriptionPrice Price
	/**
	 * Multipler value (e.g. 85 %, etc.) to calculate net dividend amount, given gross amount before taking into account the relevant fiscal features.
	 */
	DividendRatio float64
	/**
	 * Defines the underlier for both the Parent Company and the Child Compagny involved in Spin Off corporate action, as well as the respective price observation for each.
	 */
	SpinOff SpinOff
	/**
	 * Defines the underlier for both the Purchaser Company and the Acquired Compagny involved in Spin Off corporate action, as well as the respective price observation for each.
	 */
	Merger Merger
	/**
	 * The accrual rate and related terms, to adjust the price of an underlier impacted by a Corporate Action when economic impact consists in freezing the underlier price to last fixing and applying accruals for each remaining underlier price fixing date required i.e. for each, adjustedPrice = lastFixingPrice x (1 + accrualRate x DCF).
	 */
	AccrualFactor AccrualFactor
	/**
	 * The last fixing price of the Underlier affected by the CorporateAction, before any adjustment with regards to the CorporateAction.
	 */
	LastFixingPriceBeforeAdjustment Price
	/**
	 * To record observations of dividends.
	 */
	DividendObservation PriceSchedule
	/**
	 * To describe the formula used to calculate the Adjustment Factor.
	 */
	BespokeCalculationFormula string
}

/**
 * Represents a class to specify a credit notation.
 */
type AgencyRatingCriteria struct {
	/**
	 * Indicates the agency rating criteria specified for the asset or issuer.
	 */
	CreditNotation CreditNotation
	/**
	 * Indicator for options to be used if several agency ratings (>1) are specified and its necessary to identify specific charateristics. i.e (lowest or highest).
	 */
	MismatchResolution CreditNotationMismatchResolutionEnum
	/**
	 * identifies the dominant reference agency if there is a missmatch and several reference agencies exsist.
	 */
	ReferenceAgency CreditRatingAgencyEnum
	/**
	 * Indicates the boundary of a credit agency rating i.e minimum or maximum.
	 */
	Boundary CreditNotationBoundaryEnum
}

/**
 *  Parameters to be used to filter events that are relevant to a given portfolio in order to calculate the state of this portfolio. The attributes correspond to all the possible aggregation criteria that can be used and these criteria can be combined. All the attributes are optional.
 */
type AggregationParameters struct {
	/**
	 * To aggregate as of a particular date
	 */
	DateTime time.Time
	/**
	 * Specifies whether to calculate total position to given date, or only daily position for the given date.
	 */
	TotalPosition bool
	/**
	 * To aggregate based on position status (EXECUTED, SETTLED etc)
	 */
	PositionStatus PositionStatusEnum
	/**
	 * To aggregate based on a selection of party(ies) / legal entity(ies).
	 */
	Party []ReferenceWithMeta
	/**
	 * To aggregate based on a selection of products.
	 */
	Product []NonTransferableProduct
	/**
	 * To aggregate based on a selection of product type(s).
	 */
	ProductQualifier []string
	TradeReference   []ReferenceWithMeta
}

/**
 * Specification of the standard set of terms that define a legal agreement.
 */
type Agreement struct {
	/**
	 * Elections to specify a Credit Support Annex or Credit Support Deed for Intial or Variation Margin.
	 */
	CreditSupportAgreementElections CreditSupportAgreementElections
	/**
	 * Elections to specify a Collateral Transfer Agreement.
	 */
	CollateralTransferAgreementElections CollateralTransferAgreementElections
	/**
	 * Elections to specify a Security agreement.
	 */
	SecurityAgreementElections SecurityAgreementElections
	/**
	 * Elections to specify a Master Agreement Schedule.
	 */
	MasterAgreementSchedule MasterAgreementSchedule
	/**
	 * Any additional terms which mainly intend to specify the extraordinary events that may affect a trade and the related contractual rights and obligation of the parties when this happens
	 */
	TransactionAdditionalTerms TransactionAdditionalTerms
	/**
	 * A legal representation of the different possible master agreements
	 */
	MasterAgreementElections MasterAgreementElections
}

/**
 * Specifies the agreement name through an agreement type and optional detailed sub agreement type.
 */
type AgreementName struct {
	/**
	 * Specification of the legal agreement type.
	 */
	AgreementType LegalAgreementTypeEnum
	/**
	 * Specification of the credit support agreement type.
	 */
	CreditSupportAgreementType FieldWithMeta
	/**
	 * specifies the type of margin for which a legal agreement is named.
	 */
	CreditSupportAgreementMarginType CollateralMarginTypeEnum
	/**
	 * The definitions such as those published by ISDA that will define the terms of the trade.
	 */
	ContractualDefinitionsType []FieldWithMeta
	/**
	 * A contractual supplement (such as those published by ISDA) that will apply to the trade.
	 */
	ContractualTermsSupplement []ContractualTermsSupplement
	/**
	 * A reference to a contractual matrix of elected terms/values (such as those published by ISDA) that shall be deemed to apply to the trade. The applicable matrix is identified by reference to a name and optionally a publication date. Depending on the structure of the matrix, an additional term (specified in the matrixTerm element) may be required to further identify a subset of applicable terms/values within the matrix.
	 */
	ContractualMatrix []ContractualMatrix
	/**
	 * Specification of the master agreement type.
	 */
	MasterAgreementType FieldWithMeta
	/**
	 * The type of master confirmation executed between the parties.
	 */
	MasterConfirmationType FieldWithMeta
	/**
	 * The type of master confirmation annex executed between the parties.
	 */
	MasterConfirmationAnnexType FieldWithMeta
	/**
	 * Definition of an agreement that is not enumerated in the CDM.
	 */
	OtherAgreement         string
	BrokerConfirmationType BrokerConfirmationTypeEnum
}

/**
 * Specification of the content of a legal agreement.
 */
type AgreementTerms struct {
	/**
	 * Specification of the standard set of terms that define a legal agreement.
	 */
	Agreement Agreement
	/**
	 * Specification of whether the agreement terms have been negotiated using the clause library methodology.
	 */
	ClauseLibrary bool
	/**
	 * Specification of the roles of the counterparties to the agreement.
	 */
	Counterparty []Counterparty
}

/**
 * Used to combine two or more Collateral Criteria using AND logic.
 */
type AllCriteria struct {
	AllCriteria []CollateralCriteria
}

/**
 * Used to combine two or more Ratings and associated direction using AND logic.
 */
type AllDirectionRating struct {
	AllDirectionRating []DirectionRatingSet
}

/**
 * Used to combine two or more Criteria using AND logic.
 */
type AllEligibilityToHoldCollateralCriteria struct {
	AllCriteria []EligibilityToHoldCollateralCriteria
}

/**
 * A class to specify the effective date of the Amendment to Termination Currency. This date can be specified as either an actual date, a specific date (e.g. the annex date) or as a custom provision.
 */
type AmendmentEffectiveDate struct {
	/**
	 * The effective date of the Amendment to Termination Currency when specified as an actual date.
	 */
	Date time.Time
	/**
	 * The effective date of the Amendment to Termination Currency when specified as relative to another date (e.g. the annex date).
	 */
	SpecificDate AmendmentEffectiveDateEnum
	/**
	 * The effective date of the Amendment to Termination Currency when specified as a non normalized custom provision.
	 */
	CustomProvision string
}

/**
 * The amount of net capital.
 */
type AmountOfNetCapital struct {
	/**
	 * The amount of net capital.
	 */
	AmountOfNetCapital Money
}

/**
 * A class to specify a currency amount or a currency amount schedule.
 */
type AmountSchedule struct {
	/**
	 * The initial rate or amount, as the case may be. An initial rate of 5% would be represented as 0.05.
	 */
	Value float64
	/**
	 * The schedule of step date and value pairs. On each step date the associated step value becomes effective. A list of steps may be ordered in the document by ascending step date. An FpML document containing an unordered list of steps is still regarded as a conformant document.
	 */
	DatedValue []DatedValue
	/**
	 * The currency in which the amount schedule is denominated. The currency is specified outside of the actual schedule in order to be applied uniformly to it. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	Currency []FieldWithMeta
}

/**
 * Holds an identifier for an ancillary entity, either identified directly via its ancillary role or directly as a legal entity.
 */
type AncillaryEntity struct {
	/**
	 * Identifies a party via its ancillary role on a transaction (e.g. CCP or DCO through which the trade should be cleared.)
	 */
	AncillaryParty AncillaryRoleEnum
	LegalEntity    LegalEntity
}

/**
 * Defines an ancillary role enumerated value with an associated party reference. The product is agnostic to the actual parties involved in the transaction, with the party references abstracted away from the product definition and replaced by the AncillaryRoleEnum. The AncillaryRoleEnum can then be positioned in the product and this AncillaryParty type, which is positioned outside of the product definition, allows the AncillaryRoleEnum to be associated with an actual party reference.
 */
type AncillaryParty struct {
	/**
	 * Specifies the AncillaryRoleEnum that is associated to the party reference. An ancillary party is any involved party that is not one of the two principal parties to the transaction.
	 */
	Role AncillaryRoleEnum
	/**
	 * Specifies the party, or parties, associated to the ancillary role.
	 */
	PartyReference []ReferenceWithMeta
	/**
	 * Optionally specifies the counterparty that the ancillary party is acting on behalf of.
	 */
	OnBehalfOf CounterpartyRoleEnum
}

/**
 * Used to combine two or more Collateral Criteria using OR logic.
 */
type AnyCriteria struct {
	AnyCriteria []CollateralCriteria
}

/**
 * Used to combine two or more Ratings and associated direction using OR logic.
 */
type AnyDirectionRating struct {
	AnyDirectionRating []DirectionRatingSet
}

/**
 * Used to combine two or more Criteria using OR logic.
 */
type AnyEligibilityToHoldCollateralCriteria struct {
	AnyCriteria []EligibilityToHoldCollateralCriteria
}

/**
 * A class to specify the applicable regulatory regime(s) that parties to a legal agreement, such as the ISDA 2016 and 2018 CSA for Initial Margin, might be subject to.
 */
type ApplicableRegime struct {
	/**
	 * The applicable regulatory regime, as specified through an enumeration.
	 */
	Regime RegulatoryRegimeEnum
	/**
	 * The additional regulatory regime as specified by the parties.
	 */
	AdditionalRegime string
	/**
	 * A class that is used by the ApplicableRegime and the AdditionalRegime classes to specify the terms that are specific to each party and regime which are referred to in the Regime Table as part of certain legal agreements, such as the ISDA 2016 and 2018 CSA for Initial Margin.
	 */
	RegimeTerms    []RegimeTerms
	AdditionalType AdditionalTypeEnum
	/**
	 * The bespoke Additional Type for the purposes of Covered Transactions (IM).
	 */
	AdditionalTerms string
}

/**
 * A class to specify the Valuation of Appropriated Collateral.
 */
type AppropriatedCollateralValuation struct {
	/**
	 * The qualification of whether the Valuation of Appropriate Collateral provision is applicable (True) or not applicable (False).
	 */
	IsSpecified bool
	/**
	 * The parties' election that qualify the Valuation of Appropriate Collateral in the case where it is deemed applicable.
	 */
	Election string
}

/**
 * An Asset is defined as something that can be owned and transferred in the financial markets. As a choice data type, one and only one of the attributes must be used.
 */
type Asset struct {
	/**
	 * An Asset that consists solely of a monetary holding in a currency.
	 */
	Cash Cash
	/**
	 * An Asset comprised of raw or refined materials or agricultural products, eg gold, oil or wheat.
	 */
	Commodity Commodity
	/**
	 * An Asset that exists only in digital form, eg Bitcoin or Ethereum; excludes the digital representation of other Assets.
	 */
	DigitalAsset DigitalAsset
	/**
	 * An asset that is issued by one party to one or more others; Instrument is also a choice data type.
	 */
	Instrument Instrument
}

type AssetAgencyRating struct {
	/**
	 * Represents an agency rating based on default risk and creditors claim in event of default associated with specific instrument.
	 */
	AssetAgencyRating AgencyRatingCriteria
}

/**
 * The base data type to specify common attributes for all Assets.
 */
type AssetBase struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
}

type AssetCountryOfOrigin struct {
	/**
	 * Represents a filter on the asset country of origin based on the ISO Standard 3166.
	 */
	AssetCountryOfOrigin ISOCountryCodeEnum
}

/**
 * Contains the information relative to the delivery of the asset.
 */
type AssetDeliveryInformation struct {
	/**
	 * Defines the periods of delivery, including the delivery profile.
	 */
	Periods AssetDeliveryPeriods
	/**
	 * Defines the location of the delivery of the commodity.
	 */
	Location []LocationIdentifier
	/**
	 * The number of units included in the transaction for each delivery interval
	 */
	DeliveryCapacity Quantity
	/**
	 * Grade of product being delivered (e.g., grade of oil).
	 */
	CommodityGrade []ProductGradeEnum
}

/**
 * Defines the periods of delivery, including the delivery profile.
 */
type AssetDeliveryPeriods struct {
	/**
	 * Defines the delivery profile of the asset, including the load type and the delivery intervals.
	 */
	Profile []AssetDeliveryProfile
	/**
	 * Delivery start date
	 */
	StartDate time.Time
	/**
	 * Delivery end date
	 */
	EndDate time.Time
}

/**
 * Defines the delivery profile of the asset, including the load type and the delivery intervals.
 */
type AssetDeliveryProfile struct {
	/**
	 * Identification of the delivery profile.
	 */
	LoadType LoadTypeEnum
	/**
	 * Defines a delivery profile block, including start and end time, days of the week, duration, delivery capacity and price time interval quantity.
	 */
	Block []AssetDeliveryProfileBlock
	/**
	 * Specifies whether the dates defined include holidays or not.
	 */
	BankHolidaysTreatment BankHolidayTreatmentEnum
}

/**
 * Defines a delivery profile block, including start and end time, days of the week, duration, delivery capacity and price time interval quantity.
 */
type AssetDeliveryProfileBlock struct {
	/**
	 * The start time of the delivery interval for each block or shape.
	 */
	StartTime time.Time
	/**
	 * The end time of the delivery interval for each block or shape.
	 */
	EndTime time.Time
	/**
	 * The days of the week of the delivery.
	 */
	DayOfWeek []DayOfWeekEnum
	/**
	 * The number of units included in the transaction for each delivery interval
	 */
	DeliveryCapacity Quantity
	/**
	 * Price per quantity per delivery time interval.
	 */
	PriceTimeIntervalQuantity Price
}

/**
 * Defines the basic parameters of an asset transfer, e.g. a cashflow: what (the asset), how much (the quantity) and when (the settlement date).
 */
type AssetFlowBase struct {
	/**
	 * Represents the amount of the asset to be transferred. The cashflow amount is always a positive number, as the cashflow direction is implied by the payer/receiver attribute.
	 */
	Quantity NonNegativeQuantity
	/**
	 * Represents the object that is subject to the transfer, it could be an asset or a reference.
	 */
	Asset Asset
	/**
	 * Represents the date on which the transfer to due.
	 */
	SettlementDate AdjustableOrAdjustedOrRelativeDate
}

/**
 * The unique identifier for an Asset, specified using an Asset Identifier Type enumerator.
 */
type AssetIdentifier struct {
	/**
	 * The identifier value.
	 */
	Identifier FieldWithMeta
	/**
	 * Defines the symbology source of the Asset Identifier, eg CUSIP, ISIN, etc.
	 */
	IdentifierType AssetIdTypeEnum
}

/**
 * Defines each asset movement of an asset payout.
 */
type AssetLeg struct {
	/**
	 * Specifies the settlement date of securities.  In a repo transaction the purchase date would always be the effective date as specified under Economic Terms, the repurchase date would always be the termination date as specified under Economic Terms.
	 */
	SettlementDate AdjustableOrRelativeDate
	/**
	 * Specifies a delivery method for the security transaction.
	 */
	DeliveryMethod DeliveryMethodEnum
}

type AssetMaturity struct {
	/**
	 * Specifies whether the maturity range is the remaining or original maturity.
	 */
	MaturityType MaturityTypeEnum
	/**
	 * Represents a filter based on the underlying asset maturity.
	 */
	MaturityRange PeriodRange
}

/**
 * Security finance payout specification in case the product payout involves some form of security collateral, as in a securities financing transaction. Plus additional description for ICMA.
 */
type AssetPayout struct {
	/**
	 * Canonical representation of the payer and receiver parties applicable to each payout leg.
	 */
	PayerReceiver PayerReceiver
	/**
	 * Each payout leg must implement the quantity concept as a 'resolvable' type, which allows for different payout legs to be linked to each other (e.g. in the case of cross-curreny products).
	 */
	PriceQuantity ResolvablePriceQuantity
	/**
	 * The specification of the principal exchange. Optional as only applicable in the case of cross-currency or zero-coupon swaps with a final payment.
	 */
	PrincipalPayment PrincipalPayments
	/**
	 * Each payout leg must specifies its settlement terms, including the delivery type (i.e. cash vs physical, and their respective terms), the transfer type (DvP etc.) and settlement date, if any.
	 */
	SettlementTerms SettlementTerms
	/**
	 * Defines each asset movement as a buy/sell at different dates, typically 1 near leg and 1 far leg in a securities financing transaction.
	 */
	AssetLeg []AssetLeg
	/**
	 * Specifies the Purchased Asset, usually a Security.
	 */
	Underlier Asset
	/**
	 * A contractual minimum amount which the borrower will pay, regardless of the duration of the loan. A mechanism for making sure that a trade generates enough income.
	 */
	MinimumFee Money
	/**
	 * Specifies the terms under which dividends received by the borrower are passed through to the lender.
	 */
	DividendTerms DividendTerms
	/**
	 * The trade type, eg repurchase transaction or buy/sell-back.
	 */
	TradeType AssetPayoutTradeTypeEnum
}

/**
 * Represents a class to allow specification of the asset product type.
 */
type AssetType struct {
	/**
	 * Represents a filter based on the type of collateral asset.
	 */
	AssetType AssetTypeEnum
	/**
	 * Represents a filter based on the type of security.
	 */
	InstrumentType InstrumentTypeEnum
	/**
	 * Represents a filter based on the type of bond.
	 */
	DebtType DebtType
	/**
	 * Represents a filter based on the type of equity.
	 */
	EquityType EquityType
	/**
	 * Represents a filter based on the type of fund.
	 */
	FundType FundProductTypeEnum
	/**
	 * Specifies the eligible asset type when not enumerated.
	 */
	OtherAssetType []string
}

/**
 * A class to specify the identifier value and its associated version.
 */
type AssignedIdentifier struct {
	/**
	 * The identifier value.
	 */
	Identifier FieldWithMeta
	/**
	 * The identifier version, which is specified as an integer and is meant to be incremented each time the transaction terms (whether contract or event) change. This version is made option to support the use case where the identifier is referenced without the version. The constraint that a contract and a lifecycle event need to have an associated version is enforced through data rules.
	 */
	Version int
}

/**
 * A class to specify the Automatic Early Termination provision applicable to a Master Agreement.
 */
type AutomaticEarlyTermination struct {
	/**
	 * Specifies whether automatic termination applies in the event of certain bankruptcy events.
	 */
	FallbackAET bool
	/**
	 * Specifies whether indemnity applies in the event AET is applicable to one or both of the parties.
	 */
	Indemnity bool
	/**
	 * The party election specific to the Automatic Early Termination Clause.
	 */
	PartyElection []AutomaticEarlyTerminationElection
}

/**
 * A class to specify the party elections specific to the Automatic Early Termination Clause.
 */
type AutomaticEarlyTerminationElection struct {
	/**
	 * The party for which the Automatic Early Termination provisions are being specified.
	 */
	Party Party
	/**
	 * A boolean election to specify whether the Automatic Early Termination provisions of Section 6(a) are applicable (True) or not applicable (False).
	 */
	IsApplicable bool
}

/**
 * A type to define automatic exercise of a swaption. With automatic exercise the option is deemed to have exercised if it is in the money by more than the threshold amount on the exercise date.
 */
type AutomaticExercise struct {
	/**
	 * A threshold rate. The threshold of 0.10% would be represented as 0.001
	 */
	ThresholdRate float64
	/**
	 * Boolean that indicates if it has an automaticExercise
	 */
	IsApplicable bool
}

/**
 * A data type that can be used to describe the inventory of securities that a party holds. The securities are held in the AvailableInventoryRecord, with each item in the array being an individual security and its associated criteria. Criteria can include the quantity available, the rate at which the security is available to borrow at, as well as other details that can affect the decision as to whether a party wants to utilise the securities listed.
 */
type AvailableInventory struct {
	/**
	 * Defines the purpose of this inventory.
	 */
	AvailableInventoryType AvailableInventoryTypeEnum
	/**
	 * Allows details related to the availability messaging use case to be defined
	 */
	MessageInformation MessageInformation
	/**
	 * Defines all parties involved for the list of inventory records in this set of inventory. For example, when used to describe securities lending availability, this could hold the sender of the availability, the intended recipient, the beneficial owner(s), the lender (which may differ from the sender as the lender may have the same piece of availability going through multiple agents), an agent or a venue.
	 */
	Party []Party
	/**
	 * Defines the role(s) that party(ies) may have in relation to the inventory.
	 */
	PartyRole []PartyRole
	/**
	 * An array holding the list of inventory being described. Each element in the inventoryRecord array represents an individual piece of inventory i.e. a security.
	 */
	AvailableInventoryRecord []AvailableInventoryRecord
}

/**
 * An individual piece of available inventory. This represents a single security and its associated criteria. The criteria are used to describe any restrictions on the securities.
 */
type AvailableInventoryRecord struct {
	/**
	 * Unique identifier for this record. This can be used to uniquely identify a specific piece of inventory.
	 */
	Identifer AssignedIdentifier
	/**
	 * The security details.
	 */
	Security Security
	/**
	 * There may be a set period/time restriction associated to the security.
	 */
	ExpirationDateTime time.Time
	/**
	 * The type of collateral can often be required when determining if the piece of availability being described is suitable for a party.
	 */
	Collateral []CollateralProvisions
	/**
	 * An individual security may be held by several agents. Including the party role at this level allows us to reference the party holding this specific item.
	 */
	PartyRole []PartyRole
	/**
	 * The quantity of the security
	 */
	Quantity Quantity
	/**
	 * An optional element which can be used to hold a rate associated to this piece of availability.
	 */
	InterestRate Price
}

/**
 * Represents the average trading volume of an Equity product upon an exchange or set of exchanges.
 */
type AverageTradingVolume struct {
	/**
	 * Represents the period of the equities average trading volume on the exchange/s.
	 */
	Period Period
	/**
	 * Indicates the type of equity average trading volume being stated (single) the highest amount on one exchange, or (consolidated) volumes across multiple exchanges.
	 */
	Methodology AverageTradingVolumeMethodologyEnum
}

/**
 * Defines parameters for use in cases when a valuation or other term is based on an average of market observations.
 */
type AveragingCalculation struct {
	/**
	 * Specifies enumerations for the type of averaging calculation.
	 */
	AveragingMethod AveragingCalculationMethod
	/**
	 * Rounding applied to the average calculation.
	 */
	Precision Rounding
}

/**
 * Defines the ways in which multiple values can be aggregated into a single value.
 */
type AveragingCalculationMethod struct {
	/**
	 * Identifies whether the average values will be weighted or unweighted.
	 */
	IsWeighted bool
	/**
	 * Identifies which of the Pythagorean means is being used to compute an average value.
	 */
	CalculationMethod AveragingCalculationMethodEnum
}

/**
 * As per ISDA 2002 Definitions.
 */
type AveragingFeature struct {
	/**
	 * Defines parameters for use in cases when a valuation or other term is based on an average of market observations.
	 */
	AveragingCalculation AveragingCalculation
	AveragingInOut       AveragingInOutEnum
	/**
	 * The factor of strike.
	 */
	StrikeFactor float64
	/**
	 * The averaging in period.
	 */
	AveragingPeriodIn AveragingPeriod
	/**
	 * The averaging out period.
	 */
	AveragingPeriodOut AveragingPeriod
}

/**
 * An unordered list of weighted averaging observations.
 */
type AveragingObservationList struct {
	/**
	 * A single weighted averaging observation.
	 */
	AveragingObservation []WeightedAveragingObservation
}

/**
 * Period over which an average value is taken.
 */
type AveragingPeriod struct {
	/**
	 * A schedule for generating averaging observation dates.
	 */
	Schedule []AveragingSchedule
	/**
	 * An unweighted list of averaging observation date and times.
	 */
	AveragingDateTimes DateTimeList
	/**
	 * A weighted list of averaging observation date and times.
	 */
	AveragingObservations AveragingObservationList
	/**
	 * The market disruption event as defined by ISDA 2002 Definitions.
	 */
	MarketDisruption FieldWithMeta
}

/**
 * Class to representing a method for generating a series of dates.
 */
type AveragingSchedule struct {
	/**
	 * Date on which this period begins.
	 */
	StartDate time.Time
	/**
	 * Date on which this period ends.
	 */
	EndDate time.Time
	/**
	 * The frequency at which averaging period occurs with the regular part of the valuation schedule and their roll date convention.
	 */
	AveragingPeriodFrequency CalculationPeriodFrequency
}

/**
 * Defines the terms required to calculate the average observations associated with an averaging strike.
 */
type AveragingStrikeFeature struct {
	/**
	 * Defines parameters for use in cases when a valuation or other term is based on an average of market observations.
	 */
	AveragingCalculation AveragingCalculation
	/**
	 * Class containing terms that are associated with observing a price/benchmark/index across either single or multple observations.
	 */
	ObservationTerms ObservationTerms
}

/**
 * As per ISDA 2002 Definitions.
 */
type Barrier struct {
	/**
	 * A trigger level approached from beneath.
	 */
	BarrierCap TriggerEvent
	/**
	 * A trigger level approached from above.
	 */
	BarrierFloor TriggerEvent
}

/**
 * The base and eligible currency(ies) for the document as specified by the parties to the agreement.
 */
type BaseAndEligibleCurrency struct {
	/**
	 * The common agreed currency into which relevant amounts of all collateral arrangements between the parties are converted, or if not an actual currency, the process through which this is determined. Where hardcoded (e.g. under the 1994 ISDA Credit Support Annex (Security Interest  NY Law)), the currency that effectively performs this function.
	 */
	BaseCurrency ISOCurrencyCodeEnum
	/**
	 * A flag detailing whether the Base Currency is set to  the Termination Currency as defined in the related  Master Agreement.
	 */
	BaseCurrencyTerminationCurrency bool
	/**
	 * Utilised where the clause data structure is not able to capture a material aspect of the clause.
	 */
	BaseCurrencyOther FieldWithMeta
	/**
	 * A definition of a currency agreed by the parties, typically to indicate the currencies of eligible cash collateral.
	 */
	EligibleCurrency []ISOCurrencyCodeEnum
	/**
	 * A flag detailing whether the Base Currency is included as an Eligible Currency.
	 */
	EligibleCurrencyInclBaseCurrency bool
	/**
	 * Utilised where the clause data structure is not able to capture a material aspect of the clause.
	 */
	EligibleCurrencyOther FieldWithMeta
}

/**
 * Defines a custom basket by referencing an identifier and its constituents.
 */
type Basket struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * Identifies the constituents of the basket
	 */
	BasketConstituent []FieldWithMeta
}

/**
 * Identifies the constituents of the basket
 */
type BasketConstituent struct {
	/**
	 * The object to be observed is an Asset, ie something that can be owned and transferred in the financial markets.
	 */
	Asset Asset
	/**
	 * The object to be observed is a Basket, ie a collection of Observables with an identifier and optional weightings.
	 */
	Basket Basket
	/**
	 * The object to be observed is an Index, ie an observable computed on the prices, rates or valuations of a number of assets.
	 */
	Index Index
	/**
	 * Specifies a quantity schedule to be associated to an individual underlier that is a basket constituent. The multiple cardinality is aligned to the one of the PriceQuantity->quantity that this quantity is referencing.
	 */
	Quantity []ReferenceWithMeta
	/**
	 * Specifies an initial price schedule to be associated to an individual underlier that is a basket constituent. The multiple cardinality is aligned to the one of the PriceQuantity->price that this price is referencing.
	 */
	InitialValuationPrice []ReferenceWithMeta
	/**
	 * Specifies an interim price schedule to be associated to an individual underlier that is a basket constituent. The multiple cardinality is aligned to the one of the PriceQuantity->price that this price is referencing.
	 */
	InterimValuationPrice []ReferenceWithMeta
	/**
	 * Specifies a final price schedule to be associated to an individual underlier that is a basket constituent. The multiple cardinality is aligned to the one of the PriceQuantity->price that this price is referencing.
	 */
	FinalValuationPrice []ReferenceWithMeta
}

/**
 * CDS Basket Reference Information.
 */
type BasketReferenceInformation struct {
	/**
	 * The name of the basket expressed as a free format string. FpML does not define usage rules for this element.
	 */
	BasketName FieldWithMeta
	/**
	 * A CDS basket identifier.
	 */
	BasketId []FieldWithMeta
	/**
	 * This element contains all the reference pool items to define the reference entity and reference obligation(s) in the basket.
	 */
	ReferencePool ReferencePool
	/**
	 * N th reference obligation to default triggers payout.
	 */
	NthToDefault int
	/**
	 * M th reference obligation to default to allow representation of N th to M th defaults.
	 */
	MthToDefault int
	/**
	 * This element contains CDS tranche terms.
	 */
	Tranche Tranche
}

/**
 * A class to specify bespoke Calculation Date terms for the purposes of Initial Margin.
 */
type BespokeCalculationDate struct {
	/**
	 * Additional Calculation Date terms are applicable when True and not applicable when False.
	 */
	IsApplicable bool
	/**
	 * The Additional Calculation Date terms for the purposes of Initial Margin.
	 */
	CalculationDateImTerms string
}

/**
 * A class to specify additional Calculation Time terms for the purposes of Initial Margin.
 */
type BespokeCalculationTime struct {
	/**
	 * If set to True, the Calculation Time for Initial Margin is the time as of which the Calculation Agent (IM) computes its end of day valuations of derivatives transactions.
	 */
	AsCalculationAgent bool
	/**
	 * Additional Terms applicable to Calculation Time for Initial Margin.
	 */
	BespokeCalculationTimeTerms string
}

/**
 * A class to specify any bespoke Transfer Timing language by each party to the agreement.
 */
type BespokeTransferTiming struct {
	/**
	 * A boolean flag to specify whether bespoke transfer terms are applicable or not.
	 */
	IsApplicable bool
	/**
	 * The bespoke transfer timing terms applicable to the agreement.
	 */
	BespokeTransferTimingTerms string
}

/**
 * Specifies the instructions for creation of a Security Lending billing invoice.
 */
type BillingInstruction struct {
	/**
	 * The party issuing the invoice
	 */
	SendingParty Party
	/**
	 * The party receiving the invoice
	 */
	ReceivingParty Party
	/**
	 * The starting date of the period described by this invoice
	 */
	BillingStartDate time.Time
	/**
	 * The ending date of the period described by this invoice
	 */
	BillingEndDate time.Time
	/**
	 * Instructions for creating the billing records contained within the invoice
	 */
	BillingRecordInstruction []BillingRecordInstruction
	/**
	 * The billing summaries contained within the invoice
	 */
	BillingSummary []BillingSummaryInstruction
}

/**
 * Specifies individual records within a billing invoice.
 */
type BillingRecord struct {
	/**
	 * The trade for the individual billing record.
	 */
	TradeState ReferenceWithMeta
	/**
	 * The settlement terms for the billing record
	 */
	RecordTransfer Transfer
	/**
	 * The starting date of the period described by this record
	 */
	RecordStartDate time.Time
	/**
	 * The ending date of the period described by this record
	 */
	RecordEndDate time.Time
	/**
	 * Indicates the minimum fee amount applied to the billing record, if any.
	 */
	MinimumFee Money
}

/**
 * Specifies the instructions for creation of a billing record.
 */
type BillingRecordInstruction struct {
	/**
	 * The trade for the individual billing record.
	 */
	TradeState ReferenceWithMeta
	/**
	 * The observations used to calculate the billing amount.
	 */
	Observation []Observation
	/**
	 * The starting date of the period described by this record
	 */
	RecordStartDate time.Time
	/**
	 * The ending date of the period described by this record
	 */
	RecordEndDate time.Time
	/**
	 * The date for settlement of the transfer.
	 */
	SettlementDate time.Time
}

/**
 * Specifies individual summaries within a billing invoice.
 */
type BillingSummary struct {
	/**
	 * The settlement terms for the billing summary
	 */
	SummaryTransfer Transfer
	/**
	 * The account level for the billing summary.
	 */
	SummaryAmountType RecordAmountTypeEnum
}

/**
 * Specifies the instructions for creation of a billing summary.
 */
type BillingSummaryInstruction struct {
	/**
	 * The account level for the billing summary.
	 */
	SummaryAmountType RecordAmountTypeEnum
}

/**
 * Reference to a bond underlier to represent an asset swap or Condition Precedent Bond.
 */
type BondReference struct {
	/**
	 * Reference to a bond underlier.
	 */
	Bond Security
	/**
	 * To indicate whether the Condition Precedent Bond is applicable. The swap contract is only valid if the bond is issued and if there is any dispute over the terms of fixed stream then the bond terms would be used.
	 */
	ConditionPrecedentBond bool
	/**
	 * To indicate whether the Discrepancy Clause is applicable.
	 */
	DiscrepancyClause bool
	/**
	 * Specifies the coupon rate (expressed in percentage) of a fixed income security or convertible bond.
	 */
	CouponRate FixedRateSpecification
}

/**
 * Describes correlation bounds, which form a cap and a floor on the realized correlation.
 */
type BoundedCorrelation struct {
	/**
	 * Minimum Boundary as a percentage of the Strike Price.
	 */
	MinimumBoundaryPercent float64
	/**
	 * Maximum Boundary as a percentage of the Strike Price.
	 */
	MaximumBoundaryPercent float64
}

type BoundedVariance struct {
	/**
	 * The contract specifies which price must satisfy the boundary condition.
	 */
	RealisedVarianceMethod RealisedVarianceMethodEnum
	/**
	 * The contract specifies whether the notional should be scaled by the Number of Days in Range divided by the Expected N. The number of Days in Ranges refers to the number of returns that contribute to the realized volatility.
	 */
	DaysInRangeAdjustment bool
	/**
	 * All observations above this price level will be excluded from the variance calculation.
	 */
	UpperBarrier float64
	/**
	 * All observations below this price level will be excluded from the variance calculation.
	 */
	LowerBarrier float64
}

/**
 * Identifies the market sector in which the trade has been arranged.
 */
type BrokerConfirmation struct {
	/**
	 * The type of broker confirmation executed between the parties.
	 */
	BrokerConfirmationType FieldWithMeta
}

/**
 * A class for defining a time with respect to a business day calendar location. For example, 11:00:00 GBLO.
 */
type BusinessCenterTime struct {
	/**
	 * A time specified in hh:mm:ss format where the second component must be '00', e.g. 11am would be represented as 11:00:00.
	 */
	HourMinuteTime time.Time
	/**
	 * A code identifying a business day calendar location. A business day calendar location is drawn from the list identified by the business day calendar location enumeration.
	 */
	BusinessCenter FieldWithMeta
}

/**
 * A class for specifying the business day calendar location used in determining whether a day is a business day or not, either by specifying this business center by reference to an enumerated list that is maintained by the FpML standard, or by reference to such specification when it exists elsewhere as part of the instance document. This class corresponds to the FpML BusinessCentersOrReference.model.
 */
type BusinessCenters struct {
	/**
	 * A code identifying one or several business day calendar location(s). The set of business day calendar locations are specified by the business day calendar location enumeration which is maintained by the FpML standard.
	 */
	BusinessCenter            []FieldWithMeta
	CommodityBusinessCalendar []FieldWithMeta
	/**
	 * A reference to a financial business center location specified elsewhere in the instance document.
	 */
	BusinessCentersReference ReferenceWithMeta
	Meta                     MetaFields
}

/**
 * A class defining a range of contiguous business days by defining an unadjusted first date, an unadjusted last date and a business day convention and business centers for adjusting the first and last dates if they would otherwise fall on a non business day in the specified business centers. The days between the first and last date must also be good business days in the specified centers to be counted in the range.
 */
type BusinessDateRange struct {
	/**
	 * The first date of a date range.
	 */
	StartDate time.Time
	/**
	 * The last date of a date range.
	 */
	EndDate time.Time
	/**
	 * The convention for adjusting a date if it would otherwise fall on a day that is not a business day, as specified by an ISDA convention (e.g. Following, Precedent).
	 */
	BusinessDayConvention BusinessDayConventionEnum
	/**
	 * The business center(s), specified either explicitly or by reference to those specified somewhere else in the instance document.
	 */
	BusinessCenters BusinessCenters
}

/**
 * A class defining the business day convention and financial business centers used for adjusting any relevant date if it would otherwise fall on a day that is not a business day in the specified business center.
 */
type BusinessDayAdjustments struct {
	/**
	 * The convention for adjusting a date if it would otherwise fall on a day that is not a business day.
	 */
	BusinessDayConvention BusinessDayConventionEnum
	/**
	 * The business center(s), specified either explicitly or by reference to those specified somewhere else in the instance document.
	 */
	BusinessCenters BusinessCenters
	Meta            MetaFields
}

/**
 * This allows an offset to be specified as, for instance, N business days, with a business centers specified as included.
 */
type BusinessDayOffset struct {
	/**
	 * A time period multiplier, e.g. 1, 2 or 3 etc. A negative value can be used when specifying an offset relative to another date, e.g. -2 days.
	 */
	PeriodMultiplier int
	/**
	 * A time period, e.g. a day, week, month or year of the stream. If the periodMultiplier value is 0 (zero) then period must contain the value D (day).
	 */
	Period PeriodEnum
	Meta   MetaFields
	/**
	 * The business centers for the offset.
	 */
	BusinessCenters BusinessCenters
	/**
	 * Legal text that underlies the Fixing Offset. ISDA Fixing Offset Definition. (e.g. One day that is either a Sydney Business Day or a Melbourne Business Day following the Reset Date)
	 */
	FixingOffsetDefinition string
	/**
	 * Fixing Offset Reason
	 */
	FixingOffsetReason string
}

/**
 * A business event represents a life cycle event of a trade. The combination of the state changes results in a qualifiable life cycle event. An example of a Business Event is a PartialTermination which is a defined by a quantity change primitive event.
 */
type BusinessEvent struct {
	/**
	 * The intent attribute is meant to be specified when the event qualification cannot be programmatically inferred from the event features. As a result it is only associated with those primitives that can give way to such ambiguity, the quantityChange being one of those. An example of such is a reduction in the trade notional, which could be interpreted as either a trade correction (unless a maximum period of time post-event is specified as part of the qualification), a partial termination or a portfolio rebalancing in the case of an equity swap. On the other hand, an event such as the exercise is not expected to have an associated intent as there should not be ambiguity.
	 */
	Intent                EventIntentEnum
	CorporateActionIntent CorporateActionTypeEnum
	/**
	 * Specifies the date on which the event is taking place. This date is equal to the trade date in the case of a simple execution.  However it can be different from the trade date, for example in the case of a partial termination.
	 */
	EventDate time.Time
	/**
	 * The date on which the event contractually takes effect, when different from the event date.
	 */
	EffectiveDate time.Time
	/**
	 * Specifies the package information in case the business event represents several trades executed as a package (hence this attribute is optional). The package information is only instantiated once at the business event level to preserve referential integrity, whereas individual trades make reference to it to identify that they are part of a package.
	 */
	PackageInformation IdentifiedList
	/**
	 * Specifies the instructions to create the Business Event.
	 */
	Instruction []Instruction
	/**
	 * The CDM event qualifier, which corresponds to the outcome of the isEvent qualification logic which qualifies the lifecycle event as a function of its features (e.g. PartialTermination, ClearingSubmission, Novation, ...).
	 */
	EventQualifier string
	/**
	 * Specifies the after trade state(s) created.
	 */
	After []TradeState
	Meta  MetaFields
}

/**
 * A class to specify an organizational unit.
 */
type BusinessUnit struct {
	/**
	 * A name used to describe the organizational unit
	 */
	Name string
	/**
	 * An identifier used to uniquely identify the organizational unit
	 */
	Identifier Identifier
	/**
	 * The contact information for such business unit, when different from the contact information associated with the party.
	 */
	ContactInformation ContactInformation
	Meta               MetaFields
}

/**
 * This class corresponds to the FpML BuyerSeller.model construct.
 */
type BuyerSeller struct {
	/**
	 * Buyer party that can be resolved as one of the two principal parties to the transaction. The party that buys this instrument, i.e. pays for this instrument and receives the rights defined by it. ISDA 2002 Equity Definitions section 1.18: `Buyer` means the party specified as such in the related Confirmation. | ISDA 2006 Definitions article 12.1 (b)(i) relating to a Swaption: 'Buyer' means the party that will, on each Premium Payment Date, pay to Seller the Premium | ISDA 2006 Definitions article 12.1 (b)(ii) relating to Swap Transactions with applicable Early Termination: the party specified as such in the related Confirmation, or the Exercising Party if neither party is specified | ISDA 2006 Definitions article 12.1 (b)(iii) relating to any other Option Transaction: the party specified as such in the related Confirmation. | ISDA 2014 Credit Definition article 1.4: `Buyer` means the Fixed Rate Payer.
	 */
	Buyer CounterpartyRoleEnum
	/**
	 * Seller party that can be resolved as one of the two principal parties to the transaction. The party that sells ('writes') this instrument, i.e. that grants the rights defined by this instrument and in return receives a payment for it. ISDA 2002 Equity Definitions section 1.19: `Seller` means the party specified as such in the related Confirmation. | ISDA 2006 Definitions article 12.1 (a)(i) relating to a Swaption: 'Seller' means the party the party specified as such or as writer in the related Confirmation | ISDA 2006 Definitions article 12.1 (a)(ii) relating to Swap Transactions with applicable Early Termination: the party specified as such or as writer in the related Confirmation or, if neither party is specified as such, the Non-exercising Party | ISDA 2006 Definitions article 12.1 (a)(iii) relating to any other Option Transaction: the party specified as such in the related Confirmation. | ISDA 2014 Credit Definition article 1.4: `Seller` means the Floating Rate Payer.
	 */
	Seller CounterpartyRoleEnum
}

type Buyr struct {
	AcctOwnr AcctOwnr
}

/**
 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be transferred to it as a Delivery Amount.
 */
type CSADeliveryAmount struct {
	/**
	 * Specifies whether the meaning of Delivery Amount follows the pre-print definition or deviates from this (either based on a rating or some other methodology).
	 */
	DeliveryAmount DeliveryReturnAmountEnum
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * Defines a combination of Rating Agency, Rating Value, amount and Currency code.
 */
type CSAMinimumTransferAmountVariableSet struct {
	/**
	 * The name of the Rating Agency.
	 */
	Name CreditRatingAgencyEnum
	/**
	 * The value assigned to the rating.
	 */
	Value string
	/**
	 * The Minimum Transfer Amount (MTA) applicable to the Rating.
	 */
	Amount int
	/**
	 * A flag that defines the Minimum Transfer Amount (MTA) as Infinity.
	 */
	AmountIsInfinity bool
}

/**
 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be recalled to it as a Return Amount.
 */
type CSAReturnAmount struct {
	/**
	 * Specifies whether the meaning of Return Amount follows the pre-print definition or deviates from this (either based on a rating or some other methodology).
	 */
	ReturnAmount DeliveryReturnAmountEnum
}

/**
 * Defines a combination of Rating Agency, Rating Value, Threshold amount and Currency code.
 */
type CSAThresholdVariableSet struct {
	/**
	 * The relevant rating agency.
	 */
	Name CreditRatingAgencyEnum
	/**
	 * Rating applicable to the party from a given rating agency.
	 */
	Value string
	/**
	 * The amount value applicable to the Threshold.
	 */
	Amount int
}

/**
 * Specifies the date on which the value of collateral or exposure are calculated in accordance with the agreement.
 */
type CSAValuationDate struct {
	/**
	 * Defines whether a date is defined by a certain day of the week or a particular date in the month. If not specified here, then additional language must be used.
	 */
	Date ValuationDateDateEnum
	/**
	 * Specific day(s) or day of the week, if any, specified for Valuation Time.
	 */
	Day ValuationDateDayEnum
	/**
	 * Defines the day of the calendar if this is specified for in the ValuationDateDateEnum.
	 */
	CalendarDay float64
	/**
	 * Defines the regularity of the date specified for the Valuation Date.
	 */
	Frequency ValuationDateFrequencyEnum
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * Defines the tradeState or payout on which to create a Transfer along with all necessary resets.
 */
type CalculateTransferInstruction struct {
	TradeState    TradeState
	Payout        ReferenceWithMeta
	Resets        []Reset
	PayerReceiver PayerReceiver
	/**
	 * Specifies quantity amount returned if not the full amount from the TradeState, e.g. partial return
	 */
	Quantity Quantity
	Date     time.Time
}

/**
 * Type for reporting details of calculated rates, including the observations that went into the final reported rate.
 */
type CalculatedRateDetails struct {
	/**
	 * The observation dates and weights for each observation date.
	 */
	Observations CalculatedRateObservations
	/**
	 * The weighted value of each observation.
	 */
	WeightedRates []float64
	/**
	 * The daily growth factors, showing the weighted rates divided by the day count basis plus one, giving how much the value grows for each step in the calculation.
	 */
	GrowthFactor []float64
	/**
	 * The compounding curve, showing how the initial value grew during the calculation period.
	 */
	CompoundedGrowth []float64
	/**
	 * The total sum or product of all the individual terms that went into the calculated rate.
	 */
	AggregateValue float64
	/**
	 * The total weight of all the terms that went into the calculated rate.
	 */
	AggregateWeight float64
	/**
	 * The resulting calculated weight.
	 */
	CalculatedRate float64
}

/**
 * Type for reporting the observations dates and the corresponding weights going into a daily calculated rate
 */
type CalculatedRateObservationDatesAndWeights struct {
	/**
	 * The observation date upon which the rate is observed.
	 */
	ObservationDates []time.Time
	/**
	 * The corresponding weight for each date.
	 */
	Weights []float64
}

/**
 * Type for reporting observations that went into the final reported rate.
 */
type CalculatedRateObservations struct {
	/**
	 * The observation date upon which the rate is observed.
	 */
	ObservationDates []time.Time
	/**
	 * The corresponding weight for each date.
	 */
	Weights []float64
	/**
	 * The value observed for that date
	 */
	ObservedRates []float64
	/**
	 * The value after any processing, such as application of caps or floors.
	 */
	ProcessedRates []float64
}

/**
 * A class defining the ISDA calculation agent responsible for performing duties as defined in the applicable product definitions.
 */
type CalculationAgent struct {
	/**
	 * Specifies the party which is the ISDA Calculation Agent for the trade. If more than one party is referenced then the parties are assumed to be co-calculation agents, i.e. they have joint responsibility.
	 */
	CalculationAgentParty AncillaryRoleEnum
	/**
	 * Specifies the ISDA calculation agent responsible for performing duties as defined in the applicable product definitions. For example, the Calculation Agent may be defined as being the Non-exercising Party.
	 */
	CalculationAgentPartyEnum PartyDeterminationEnum
	/**
	 * The city in which the office through which ISDA Calculation Agent is acting for purposes of the transaction is located The short-form confirm for a trade that is executed under a Sovereign or Asia Pacific Master Confirmation Agreement ( MCA ), does not need to specify the Calculation Agent. However, the confirm does need to specify the Calculation Agent City. This is due to the fact that the MCA sets the value for Calculation Agent but does not set the value for Calculation Agent City.
	 */
	CalculationAgentBusinessCenter FieldWithMeta
}

/**
 * Details of the party calculating the value of collateral to be delivered or returned.
 */
type CalculationAgentTerms struct {
	/**
	 * The party which is specified as Calculation Agent for Initial Margin.
	 */
	Party CalculationValuationAgentPartyEnum
	/**
	 * The Calculation Agent (IM) terms when specified.
	 */
	BespokeCalculationAgentTerms string
}

/**
 * A logical container designed to hold a set of related data. In this case, designed to hold a set of clauses that are common amongst Legacy (1994/95), Variation Margin and Initial Margin Credit Support Documents in regards to Calculation, Valuation and Timing terms.
 */
type CalculationAndTimingBase struct {
	/**
	 * The time by which a demand for the Transfer of Eligible Credit Support (IM) or Posted Credit Support (IM) needs to be made in order for the transfer to take place in accordance with the Transfer Timing provisions.
	 */
	NotificationTime NotificationTime
}

/**
 * Specification of the Calculation, Valuation and Timing terms specific to the Initial Margin Credit Support Annex.
 */
type CalculationAndTimingCollateralTransferAgreement struct {
	/**
	 * The time by which a demand for the Transfer of Eligible Credit Support (IM) or Posted Credit Support (IM) needs to be made in order for the transfer to take place in accordance with the Transfer Timing provisions.
	 */
	NotificationTime NotificationTime
	/**
	 * Details of the party calculating the value of collateral to be delivered or returned.
	 */
	CalculationAgentTerms CalculationAgentTerms
	/**
	 * The specification of bespoke Calculation Date terms for the purposes of Initial or Variation Margin by the parties to the agreement.
	 */
	BespokeCalculationDate BespokeCalculationDate
	/**
	 * Bespoke terms to describe the time as of which such party (or the Calculation Agent (IM) (if applicale)) computes its end of day valuations of derivatives transactions.
	 */
	BespokeCalculationTime BespokeCalculationTime
	/**
	 * The specified location where the credit exposure will be calculated by the respective parties.
	 */
	CalculationDateLocation ValuationCalculationDateLocation
	CashSettlementDay       string
	SecuritiesSettlementDay string
}

/**
 * Specification of the Calculation, Valuation and Timing terms specific to the Initial Margin Credit Support Annex.
 */
type CalculationAndTimingInitialMargin struct {
	/**
	 * The time by which a demand for the Transfer of Eligible Credit Support (IM) or Posted Credit Support (IM) needs to be made in order for the transfer to take place in accordance with the Transfer Timing provisions.
	 */
	NotificationTime NotificationTime
	/**
	 * Details of the party calculating the value of collateral to be delivered or returned.
	 */
	CalculationAgentTerms CalculationAgentTerms
	/**
	 * The specification of bespoke Calculation Date terms for the purposes of Initial or Variation Margin by the parties to the agreement.
	 */
	BespokeCalculationDate BespokeCalculationDate
	/**
	 * Bespoke terms to describe the time as of which such party (or the Calculation Agent (IM) (if applicale)) computes its end of day valuations of derivatives transactions.
	 */
	BespokeCalculationTime BespokeCalculationTime
	/**
	 * The specified location where the credit exposure will be calculated by the respective parties.
	 */
	CalculationDateLocation ValuationCalculationDateLocation
	CashSettlementDay       string
	SecuritiesSettlementDay string
}

/**
 * Specification of the Calculation, Valuation and Timing terms specific to the Legacy (1994/95) Credit Support Annex.
 */
type CalculationAndTimingLegacy struct {
	/**
	 * The time by which a demand for the Transfer of Eligible Credit Support (IM) or Posted Credit Support (IM) needs to be made in order for the transfer to take place in accordance with the Transfer Timing provisions.
	 */
	NotificationTime NotificationTime
	/**
	 * Details of the days on which calculations are to be made to determine the Delivery Amount and/or Return Amount.
	 */
	ValuationDate CSAValuationDate
	/**
	 * Specifies the time on which the value of collateral or exposure are calculated in accordance with the agreement.
	 */
	ValuationTime ValuationTime
	/**
	 * Details of the party calculating the value of collateral to be delivered or returned.
	 */
	ValuationAgent ValuationAgent
}

/**
 * Specification of the Calculation, Valuation and Timing terms specific to the Variation Margin Credit Support Annex.
 */
type CalculationAndTimingVariationMargin struct {
	/**
	 * The time by which a demand for the Transfer of Eligible Credit Support (IM) or Posted Credit Support (IM) needs to be made in order for the transfer to take place in accordance with the Transfer Timing provisions.
	 */
	NotificationTime NotificationTime
	/**
	 * Details of the days on which calculations are to be made to determine the Delivery Amount and/or Return Amount.
	 */
	ValuationDate CSAValuationDate
	/**
	 * Specifies the time on which the value of collateral or exposure are calculated in accordance with the agreement.
	 */
	ValuationTime ValuationTime
	/**
	 * Details of the party valuating the collateral to be delivered or returned.
	 */
	ValuationAgent ValuationAgent
	/**
	 * The specified location where the credit exposure will be calculated by the respective parties.
	 */
	ValuationDateLocation ValuationCalculationDateLocation
}

/**
 * A class to specify the ISDA SIMM Calculation Currency.
 */
type CalculationCurrencyElection struct {
	/**
	 * The party which the SIMM Calculation Currency qualification applies to.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The SIMM Calculation Currency (also known as SIMM Reporting Currency) means the Base Currency when True. It means a different currency when False. In that latter case, the SIMM Calculation Currency is specified as part of the currency attribute.
	 */
	IsBaseCurrency bool
	/**
	 * The currency in which the ISDA SIMM Calculation is denominated, when different from the Base Currency. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	Currency FieldWithMeta
}

/**
 * Represents the parameters for describing how often something (such as collateral interest) is to be calculated.
 */
type CalculationFrequency struct {
	/**
	 * Specifies the time period at which calculation is performed, e.g. 1 month.
	 */
	Period Period
	/**
	 * Specifies the month of the year if used.
	 */
	MonthOfYear float64
	/**
	 * Specifies the day of the month if used.
	 */
	DayOfMonth float64
	/**
	 * Specifies the day of the week if used.
	 */
	DayOfWeek DayOfWeekEnum
	/**
	 * Specifies the week of the month if used.
	 */
	WeekOfMonth float64
	/**
	 * Specifies how many days from the trigger event should the payment occur.
	 */
	OffsetDays float64
	/**
	 * Specifies where is the time measured.
	 */
	DateLocation BusinessCenterTime
	/**
	 * Specifies the business center for adjustment of calculation period.
	 */
	BusinessCenter []BusinessCenter
}

/**
 * A data defining:  the parameters used in the calculation of a fixed or floating rate calculation period amount. This data forms:  part of cashflows representation of a swap stream.
 */
type CalculationPeriod struct {
	/**
	 * The calculation period start date, adjusted according to any relevant business day convention.
	 */
	AdjustedStartDate time.Time
	/**
	 * The calculation period end date, adjusted according to any relevant business day convention.
	 */
	AdjustedEndDate time.Time
	Meta            MetaFields
	/**
	 * The calculation start date, unadjusted.
	 */
	UnadjustedStartDate time.Time
	/**
	 * The calculation end date, unadjusted.
	 */
	UnadjustedEndDate time.Time
	/**
	 * The number of days from the adjusted effective / start date to the adjusted termination / end date calculated in accordance with the applicable day count fraction.
	 */
	CalculationPeriodNumberOfDays int
	/**
	 * The amount that a cashflow will accrue interest on.
	 */
	NotionalAmount float64
	/**
	 * The amount that a cashflow will accrue interest on. This is the calculated amount of the FX linked - i.e. the other currency notional amount multiplied by the appropriate FX spot rate.
	 */
	FxLinkedNotionalAmount FxLinkedNotionalAmount
	/**
	 * The floating rate reset information for the calculation period.
	 */
	FloatingRateDefinition FloatingRateDefinition
	/**
	 * The calculation period fixed rate. A per annum rate, expressed as a decimal. A fixed rate of 5% would be represented as 0.05.
	 */
	FixedRate float64
	/**
	 * The year fraction value of the calculation period, result of applying the ISDA rules for day count fraction defined in the ISDA Annex.
	 */
	DayCountYearFraction float64
	/**
	 * The amount representing the forecast of the accrued value of the calculation period. An intermediate value used to generate the forecastPaymentAmount in the PaymentCalculationPeriod.
	 */
	ForecastAmount Money
	/**
	 * A value representing the forecast rate used to calculate the forecast future value of the accrual period. This is a calculated rate determined based on averaging the rates in the rateObservation elements, and incorporates all of the rate treatment and averaging rules. A value of 1% should be represented as 0.01.
	 */
	ForecastRate float64
}

/**
 * The calculation period adjusted start and end dates, which are the baseline arguments needed to compute an interest accrual calculation.
 */
type CalculationPeriodBase struct {
	/**
	 * The calculation period start date, adjusted according to any relevant business day convention.
	 */
	AdjustedStartDate time.Time
	/**
	 * The calculation period end date, adjusted according to any relevant business day convention.
	 */
	AdjustedEndDate time.Time
	Meta            MetaFields
}

type CalculationPeriodData struct {
	StartDate            time.Time
	EndDate              time.Time
	DaysInPeriod         int
	DaysInLeapYearPeriod int
	IsFirstPeriod        bool
	IsLastPeriod         bool
}

/**
 * A data for:  defining the parameters used to generate the calculation period dates schedule, including the specification of any initial or final stub calculation periods. A calculation period schedule consists of an optional initial stub calculation period, one or more regular calculation periods and an optional final stub calculation period. In the absence of any initial or final stub calculation periods, the regular part of the calculation period schedule is assumed to be between the effective date and the termination date. No implicit stubs are allowed, i.e. stubs must be explicitly specified using an appropriate combination of firstPeriodStartDate, firstRegularPeriodStartDate and lastRegularPeriodEndDate.
 */
type CalculationPeriodDates struct {
	/**
	 * The first day of the terms of the trade. This day may be subject to adjustment in accordance with a business day convention.
	 */
	EffectiveDate AdjustableOrRelativeDate
	/**
	 * The last day of the terms of the trade. This date may be subject to adjustments in accordance with the business day convention. It can also be specified in relation to another scheduled date (e.g. the last payment date).
	 */
	TerminationDate AdjustableOrRelativeDate
	/**
	 * The specification of the business day convention and financial business centers used for adjusting any calculation period date if it would otherwise fall on a day that is not a business day in the specified business center.
	 */
	CalculationPeriodDatesAdjustments BusinessDayAdjustments
	/**
	 * The start date of the calculation period. FpML specifies that for interest rate swaps this date must only be specified if it is not equal to the effective date. It is always specified in the case of equity swaps and credit default swaps with periodic payments. This date may be subject to adjustment in accordance with a business day convention.
	 */
	FirstPeriodStartDate AdjustableOrRelativeDate
	/**
	 * The start date of the regular part of the calculation period schedule. It must only be specified if there is an initial stub calculation period. This day may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments.
	 */
	FirstRegularPeriodStartDate time.Time
	/**
	 * The end date of the initial compounding period when compounding is applicable. It must only be specified when the compoundingMethod element is present and not equal to a value of None. This date may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments.
	 */
	FirstCompoundingPeriodEndDate time.Time
	/**
	 * The end date of the regular part of the calculation period schedule. It must only be specified if there is a final stub calculation period. This day may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments.
	 */
	LastRegularPeriodEndDate time.Time
	/**
	 * Method to allocate any irregular period remaining after regular periods have been allocated between the effective and termination date.
	 */
	StubPeriodType StubPeriodTypeEnum
	/**
	 * The frequency at which calculation period end dates occur with the regular part of the calculation period schedule and their roll date convention.
	 */
	CalculationPeriodFrequency CalculationPeriodFrequency
	Meta                       MetaFields
}

/**
 * A class to specify the frequency at which calculation period end dates occur within the regular part of the calculation period schedule and their roll date convention.
 */
type CalculationPeriodFrequency struct {
	/**
	 * A time period multiplier, e.g. 1, 2, or 3. If the period value is T (Term) then period multiplier must contain the value 1.
	 */
	PeriodMultiplier int
	/**
	 * A time period, e.g. a day, week, month, year or term of the stream.
	 */
	Period PeriodExtendedEnum
	Meta   MetaFields
	/**
	 * The roll convention specifies the period term as part of a periodic schedule, i.e. the calculation period end date within the regular part of the calculation period. The value could be a rule, e.g. IMM Settlement Dates, which is the 3rd Wednesday of the month, or it could be a specific day of the month, such as the first day of the applicable month. It is used in conjunction with a frequency and the regular period start date of a calculation period.
	 */
	RollConvention RollConventionEnum
	/**
	 * Indicates, when true, that that the first Calculation Period should run from the Effective Date to the end of the calendar period in which the Effective Date falls, e.g. Jan 15 - Jan 31 if the calculation periods are one month long and Effective Date is Jan 15. If false, the first Calculation Period should run from the Effective Date for one whole period, e.g. Jan 15 to Feb 14 if the calculation periods are one month long and Effective Date is Jan 15. Mostly used in Commmodity Swaps.
	 */
	BalanceOfFirstPeriod bool
}

/**
 * A class that allows the full representation of a payout by defining a set of schedule periods. It supports standard schedule customization by expressing all the dates, quantities, and pricing data in a non-parametric way.
 */
type CalculationSchedule struct {
	/**
	 * Defines a period of a calculation schedule structure.
	 */
	SchedulePeriod []SchedulePeriod
}

/**
 * Period and time profile over which the delivery takes place.
 */
type CalculationScheduleDeliveryPeriods struct {
	/**
	 * Defines the delivery profile of the asset, including the load type and the delivery intervals.
	 */
	Profile []AssetDeliveryProfile
	/**
	 * Delivery start date
	 */
	StartDate time.Time
	/**
	 * Delivery end date
	 */
	EndDate time.Time
	/**
	 * The number of units included in the transaction for each delivery interval
	 */
	DeliveryCapacity Quantity
	/**
	 * Price per quantity per delivery time interval.
	 */
	PriceTimeIntervalQuantity Price
}

/**
 * A type for defining a calendar spread feature.
 */
type CalendarSpread struct {
	ExpirationDateTwo AdjustableOrRelativeDate
}

/**
 * A data defining:  the right of a party to cancel a swap transaction on the specified exercise dates. The provision is for 'walk-away' cancellation (i.e. the fair value of the swap is not paid). A fee payable on exercise can be specified. As a difference from the FpML construct, the canonical model extends the BuyerSeller class.
 */
type CancelableProvision struct {
	/**
	 * Buyer party that can be resolved as one of the two principal parties to the transaction. The party that buys this instrument, i.e. pays for this instrument and receives the rights defined by it. ISDA 2002 Equity Definitions section 1.18: `Buyer` means the party specified as such in the related Confirmation. | ISDA 2006 Definitions article 12.1 (b)(i) relating to a Swaption: 'Buyer' means the party that will, on each Premium Payment Date, pay to Seller the Premium | ISDA 2006 Definitions article 12.1 (b)(ii) relating to Swap Transactions with applicable Early Termination: the party specified as such in the related Confirmation, or the Exercising Party if neither party is specified | ISDA 2006 Definitions article 12.1 (b)(iii) relating to any other Option Transaction: the party specified as such in the related Confirmation. | ISDA 2014 Credit Definition article 1.4: `Buyer` means the Fixed Rate Payer.
	 */
	Buyer CounterpartyRoleEnum
	/**
	 * Seller party that can be resolved as one of the two principal parties to the transaction. The party that sells ('writes') this instrument, i.e. that grants the rights defined by this instrument and in return receives a payment for it. ISDA 2002 Equity Definitions section 1.19: `Seller` means the party specified as such in the related Confirmation. | ISDA 2006 Definitions article 12.1 (a)(i) relating to a Swaption: 'Seller' means the party the party specified as such or as writer in the related Confirmation | ISDA 2006 Definitions article 12.1 (a)(ii) relating to Swap Transactions with applicable Early Termination: the party specified as such or as writer in the related Confirmation or, if neither party is specified as such, the Non-exercising Party | ISDA 2006 Definitions article 12.1 (a)(iii) relating to any other Option Transaction: the party specified as such in the related Confirmation. | ISDA 2014 Credit Definition article 1.4: `Seller` means the Floating Rate Payer.
	 */
	Seller CounterpartyRoleEnum
	/**
	 * Definition of the party to whom notice of exercise should be given.
	 */
	ExerciseNotice ExerciseNotice
	/**
	 * A flag to indicate whether follow-up confirmation of exercise (written or electronic) is required following telephonic notice by the buyer to the seller or seller's agent.
	 */
	FollowUpConfirmation bool
	/**
	 * The adjusted dates associated with a cancelable provision. These dates have been adjusted for any applicable business day convention.
	 */
	CancelableProvisionAdjustedDates CancelableProvisionAdjustedDates
	/**
	 * Business date convention adjustment to final payment period per leg (swapStream) upon exercise event. The adjustments can be made in-line with leg level BDC's or they can be specified separately.
	 */
	FinalCalculationPeriodDateAdjustment []FinalCalculationPeriodDateAdjustment
	/**
	 * An initial fee for the cancelable option.
	 */
	InitialFee Transfer
	/**
	 * The party with right to exercise a cancellation. Allows for buyer, seller or either.
	 */
	CallingParty CallingPartyEnum
	/**
	 * The first day when cancelation is permitted to take effect. A party may give notice prior to this date and taken together with the effective period would be necessary to cancel on this date.
	 */
	EarliestDate AdjustableOrRelativeDate
	/**
	 * The last day within the term of the contract that cancelation is allowed.
	 */
	ExpirationDate AdjustableOrRelativeDate
	/**
	 * The effective date if cancelation is invoked otherwise the cancellation period defines the cancellation date.
	 */
	EffectiveDate AdjustableOrRelativeDates
	/**
	 * Effective period for cancelation when notice is given. This is the period after notice is given that cancellation becomes effecticve.
	 */
	EffectivePeriod Period
	/**
	 * The earliest time in a business day that notice of cancelation can be given.
	 */
	EarliestCancellationTime BusinessCenterTime
	/**
	 * The latest time at which notice of cancelation can be given.
	 */
	LatestCancelationTime BusinessCenterTime
	/**
	 * The exercise terms associated with the cancelable provision, including details such as exercise style, exercise fees, and any other relevant conditions or terms governing the cancellation of the swap transaction.
	 */
	ExerciseTerms ExerciseTerms
}

/**
 * A data to:  define the adjusted dates for a cancelable provision on a swap transaction.
 */
type CancelableProvisionAdjustedDates struct {
	/**
	 * The adjusted dates for an individual cancellation date.
	 */
	CancellationEvent []CancellationEvent
}

/**
 * The adjusted dates for a specific cancellation date, including the adjusted exercise date and adjusted termination date.
 */
type CancellationEvent struct {
	/**
	 * The date on which option exercise takes place. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedExerciseDate time.Time
	/**
	 * The early termination date that is applicable if an early termination provision is exercised. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedEarlyTerminationDate time.Time
	Meta                         MetaFields
}

/**
 * An Asset that consists solely of a monetary holding in a currency. The currency of the Cash asset is held in the string Identifier (from AssetBase) and the AssetIdTypeEnum must be set to define that a CurrencyCode is set.  The function SetCashCurrency can be used to create (or update) a Cash object and the function GetCashCurrency can be used to retrieve the currency of a Cash object.
 */
type Cash struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
}

/**
 * This type is a generic structure that can represent the parameters of several mid-market valuation and replacement value methods described in the 2021 ISDA Definitions.
 */
type CashCollateralValuationMethod struct {
	/**
	 * This may be used to specify what type of CSA (credit support annex/agreement) is to be used for cash settlement purposes.
	 */
	ApplicableCsa CsaTypeEnum
	/**
	 * This may be used to indicate the currency of cash collateral for cash settlement purposes.
	 */
	CashCollateralCurrency string
	/**
	 * This may be used to indicate the interest rate to be used for cash collateral for cash settlement purposes.
	 */
	CashCollateralInterestRate FieldWithMeta
	/**
	 * This may be used to indicate the discount rate to be used for cash collateral for cash settlement purposes.
	 */
	AgreedDiscountRate FieldWithMeta
	/**
	 * This may be used to specify which party is protected (e.g. under Replacement Value cash settlement methods).
	 */
	ProtectedParty []PartyDeterminationEnum
	/**
	 * This may be used to indicate that 'prescribed documentation adjustment' is applicable.
	 */
	PrescribedDocumentationAdjustment bool
}

/**
 * Specifies the nature of a cash price either as a fee type, cash price type, or premium expression.
 */
type CashPrice struct {
	/**
	 * Specifies the type of Cash Price.
	 */
	PriceSubType PriceSubTypeEnum
	/**
	 * Specifies a premium when expressed in a way other than an amount, and any required forward starting price definition.
	 */
	PremiumExpression PremiumExpression
	/**
	 * Specifies the event type associated with a fee.
	 */
	FeeType FeeTypeEnum
}

/**
 * Defines the terms required to compute and settle a cash settlement amount according to a fixing value, including the fixing source, fixing method and fixing date. In FpML, PhysicalSettlementTerms and CashSettlementTerms extend SettlementTerms. In the CDM, this extension paradigm has not been used because SettlementTerms class has been used for purposes related to securities transactions, while it is not used as such in the FpML standard (i.e. only as an abstract construct.
 */
type CashSettlementTerms struct {
	/**
	 * Specifies the type of cash settlement method: cash price, yield curve etc.
	 */
	CashSettlementMethod CashSettlementMethodEnum
	/**
	 * Specifies the parameters required to obtain a valuation, including the source, quotation method (bid, mid etc.) and any applicable quotation amount.
	 */
	ValuationMethod ValuationMethod
	/**
	 * Defines the different methods to specify a valuation date, as used for cash settlement. The Single / Multiple ValuationDate is used for the determination of recovery in a credit event, the RelativeDateOffset is used for cash-settled option, and FxFixingDate is used for cross-currency settlement.
	 */
	ValuationDate ValuationDate
	/**
	 * The time of the cash settlement valuation date when the cash settlement amount will be determined according to the cash settlement method, if the parties have not otherwise been able to agree the cash settlement amount. When using quations, this is the time of day in the specified business center when the calculation agent seeks quotations for an amount of the reference obligation for purposes of cash settlement. ISDA 2003 Term: Valuation Time.
	 */
	ValuationTime BusinessCenterTime
	/**
	 * The amount paid by the seller to the buyer for cash settlement on the cash settlement date. If not otherwise specified, would typically be calculated as 100 (or the Reference Price) minus the price of the Reference Obligation (all expressed as a percentage) times Floating Rate Payer Calculation Amount. ISDA 2003 Term: Cash Settlement Amount.
	 */
	CashSettlementAmount Money
	/**
	 * Used for fixed recovery, specifies the recovery level, determined at contract formation, to be applied on a default. Used to calculate the amount paid by the seller to the buyer for cash settlement on the cash settlement date. Amount calculation is (1 minus the Recovery Factor) multiplied by the Floating Rate Payer Calculation Amount. The currency will be derived from the Floating Rate Payer Calculation Amount.
	 */
	RecoveryFactor float64
	/**
	 * Used for Recovery Lock, to indicate whether fixed Settlement is Applicable or Not Applicable. If Buyer fails to deliver an effective Notice of Physical Settlement on or before the Buyer NOPS Cut-off Date, and if Seller fails to deliver an effective Seller NOPS on or before the Seller NOPS Cut-off Date, then either: (a) if Fixed Settlement is specified in the related Confirmation as not applicable, then the Seller NOPS Cut-off Date shall be the Termination Date; or (b) if Fixed Settlement is specified in the related Confirmation as applicable, then: (i) if the Fixed Settlement Amount is a positive number, Seller shall, subject to Section 3.1 (except for the requirement of satisfaction of the Notice of Physical Settlement Condition to Settlement), pay the Fixed Settlement Amount to Buyer on the Fixed Settlement Payment Date; and (ii) if the Fixed Settlement Amount is a negative number, Buyer shall, subject to Section 3.1 (except for the requirement of satisfaction of the Notice of Physical Settlement Condition to Settlement), pay the absolute value of the Fixed Settlement Amount to Seller on the Fixed Settlement Payment Date.
	 */
	FixedSettlement bool
	/**
	 * Indicates whether accrued interest is included (true) or not (false). For cash settlement this specifies whether quotations should be obtained inclusive or not of accrued interest. For physical settlement this specifies whether the buyer should deliver the obligation with an outstanding principal balance that includes or excludes accrued interest. ISDA 2003 Term: Include/Exclude Accrued Interest.
	 */
	AccruedInterest bool
	Meta            MetaFields
}

/**
 * Class to specify a cashflow, i.e. the outcome of either of computation (e.g. interest accrual) or an assessment of some sort (e.g. a fee). The cashflow can then be turned into a cash transfer, artefact to be used as the input to a payment system or the outcome of it. The associated globalKey denotes the ability to associate a hash value to the Cashflow instantiations for the purpose of model cross-referencing, in support of functionality such as the event effect and the lineage.
 */
type Cashflow struct {
	/**
	 * Represents the amount of the asset to be transferred. The cashflow amount is always a positive number, as the cashflow direction is implied by the payer/receiver attribute.
	 */
	Quantity NonNegativeQuantity
	/**
	 * Represents the object that is subject to the transfer, it could be an asset or a reference.
	 */
	Asset Asset
	/**
	 * Represents the date on which the transfer to due.
	 */
	SettlementDate AdjustableOrAdjustedOrRelativeDate
	/**
	 * Specifies who pays / receives the cashflow, though a normalised Party1 / Party2 enumerator.
	 */
	PayerReceiver PayerReceiver
	/**
	 * The qualification of the type of cashflow, e.g. brokerage fee, premium, upfront fee etc. Particularly relevant when it cannot be inferred directly through lineage.
	 */
	CashflowType CashflowType
	/**
	 * FpML specifies the FpML PaymentDiscounting.model group for representing the discounting elements that can be associated with a payment.
	 */
	PaymentDiscounting PaymentDiscounting
}

/**
 * A data defining:  the cashflow representation of a swap trade.
 */
type CashflowRepresentation struct {
	/**
	 * A true/false flag to indicate whether the cashflows match the parametric definition of the stream, i.e. whether the cashflows could be regenerated from the parameters without loss of information.
	 */
	CashflowsMatchParameters bool
	/**
	 * The adjusted payment date and associated calculation period parameters required to calculate the actual or projected payment amount. A list of payment calculation period elements may be ordered in the document by ascending adjusted payment date. An FpML document containing an unordered list of payment calculation periods is still regarded as a conformant document.
	 */
	PaymentCalculationPeriod []PaymentCalculationPeriod
}

/**
 * Characterises the type of cashflow, which can result from either a scheduled or a non-scheduled lifecycle event.
 */
type CashflowType struct {
	/**
	 * Type of cashflow corresponding to a scheduled event.
	 */
	CashflowType ScheduledTransferEnum
	/**
	 * Type of cashflow corresponding to a non-scheduled event, where a price must be agreed between the parties.
	 */
	CashPrice       CashPrice
	PriceExpression PriceExpressionEnum
}

/**
 * Result for the CheckEligibilityByDetails and CheckEligibilityForProduct functions
 */
type CheckEligibilityResult struct {
	/**
	 * a simple boolean which is set to true if the asset described in the EligibilityQuery input is eligible
	 */
	IsEligible bool
	/**
	 * if there was a match, this will be the one or more criteria that were supplied in the EligbilityCollateralSpecification which matched with the query input
	 */
	MatchingEligibleCriteria []EligibleCollateralCriteria
	/**
	 * a copy of the input query that was checked against the eligible collateral specification
	 */
	EligibilityQuery EligibilityQuery
	/**
	 * a copy of the input EligbilityCollateralSpecification that was checked against the query
	 */
	Specification EligibleCollateralSpecification
}

/**
 * A type for documenting additional clause that cannot yet be represented with the model and yet needed for a digital representation of the agreement.
 */
type Clause struct {
	/**
	 * The  name or identifier associated to this clause.
	 */
	Identifier string
	/**
	 * Content of this bespoke clause.
	 */
	Terms string
	/**
	 * Additional hierarchical components of the clause if relevant.
	 */
	Subcomponents []Clause
}

/**
 * All information required to perform the clear life cycle event; the clearing party (CCP), the two parties facing each other on the alpha contract, and optionally the parties acting as clearing members.
 */
type ClearingInstruction struct {
	/**
	 * The contract that will be submitted to the clearing house for clearing. The contract should indicate that it should be cleared by assigning a clearing organisation as a party role.
	 */
	AlphaContract TradeState
	/**
	 * The Central Counter party (CCP) that the contract will be submitted to for clearing.
	 */
	ClearingParty Party
	/**
	 * First party facing the CCP if it is clearing for its own account.
	 */
	Party1 Party
	/**
	 * Second party facing the CCP if it is clearing for its own account.
	 */
	Party2 Party
	/**
	 * Optional party facing the CCP, acting as clearing member for party1.
	 */
	ClearerParty1 Party
	/**
	 * Optional party facing the CCP, acting as clearing member for party2.
	 */
	ClearerParty2 Party
	/**
	 * Open Offer
	 */
	IsOpenOffer bool
}

/**
 *  A class to qualify the closed state of an execution or a contract through the combination or a state (e.g. terminated, novated) and a set of dates: activity date, effective date and, when relevant, last payment date.
 */
type ClosedState struct {
	/**
	 * The qualification of what gave way to the contract or execution closure, e.g. allocation, termination, ...
	 */
	State ClosedStateEnum
	/**
	 * The activity date on which the closing state took place, i.e. either the event date of the closing event (e.g. option exercise, contract early termination) or the contractual termination date.
	 */
	ActivityDate time.Time
	/**
	 * The date on which the closing event contractually takes effect, when different from the activity date. When an explicit event effective date attribute is associated with the closing event, it will be that date. In the case of a cancellation event, it will be the date on which the cancelled event took place.
	 */
	EffectiveDate time.Time
	/**
	 * The date associated with the last payment in relation to the artefact (e.g. contract) to which this closed state applies. As an example, in the case of an early termination event, it would be the settlement date of the associated fee, if applicable.
	 */
	LastPaymentDate time.Time
}

/**
 * A type hold a list of codes for business data validation
 */
type CodeList struct {
	/**
	 * Identification information about the code list
	 */
	Identification CodeListIdentification
	/**
	 * Code values
	 */
	Codes []CodeValue
}

/**
 * A type to identify the version of a codelist
 */
type CodeListIdentification struct {
	/**
	 * A description  of the code list, e.g. A list of codes identifying business centers
	 */
	Description string
	/**
	 * The most recent date the code list was published, e.g. 2021-10-05
	 */
	PublicationDate string
	/**
	 * A short name (without whitespace) for the code list, e.g. businessCenterScheme
	 */
	ShortName string
	/**
	 * The the most recent version of the codelist, eg. 9-1
	 */
	Version string
	/**
	 * The name of the code list, e.g. business-center
	 */
	CodeShortId string
	/**
	 * Canonical URI which uniquely identifies all versions (collectively) of the referenced code list. Must be an absolute URI. Must not be used as a de facto location URI.
	 */
	CanonicalUri string
	/**
	 * Canonical URI which uniquely identifies a specific version of the referenced code list. Must be an absolute URI. Must not be used as a de facto location URI.
	 */
	CanonicalVersionUri string
	/**
	 * Suggested retrieval location for this version, in genericode format. An application must signal an error to the user if a document retrieved using a LocationUri is not in genericode format.
	 */
	LocationUri string
}

/**
 * A type to hold the value of the code in a codelist
 */
type CodeValue struct {
	/**
	 * The value of the code, e.g. USNY
	 */
	Value string
	/**
	 * A description of the meaning of the code, e.g. New York City, USA
	 */
	Description string
	/**
	 * Where the code was defined,e.g. ISDA, FpML, Bloomberg
	 */
	Source string
	/**
	 * When the code became/becomes effective
	 */
	EffectiveDate time.Time
	/**
	 * When the code ceased to be valid for new uses
	 */
	DeprecatedDate time.Time
}

/**
 * A type for defining the obligations of the counterparty subject to credit support requirements.
 */
type Collateral struct {
	/**
	 * Independent Amount is an amount that usually less creditworthy counterparties are asked to provide. It can either be a fixed amount or a percentage of the Transaction's value. The Independent Amount can be: (i) transferred before any trading between the parties occurs (as a deposit at a third party's account or with the counterparty) or (ii) callable after trading has occurred (typically because a downgrade has occurred). In situation (i), the Independent Amount is not included in the calculation of Exposure, but in situation (ii), it is included in the calculation of Exposure. Thus, for situation (ii), the Independent Amount may be transferred along with any collateral call. Independent Amount is a defined term in the ISDA Credit Support Annex. ('with respect to a party, the amount specified as such for that party in Paragraph 13; if no amount is specified, zero').
	 */
	IndependentAmount IndependentAmount
	/**
	 * A list of identifiers pointing to the collateral portfolios which contain the collateral which covers a trade.
	 */
	PortfolioIdentifier []Identifier
	/**
	 * The collateral portfolios which contain the collateral which covers a trade. (NB: this can be provided by reference to a global key for each CollateralPortfolio object)
	 */
	CollateralPortfolio []ReferenceWithMeta
	/**
	 * specifies the collateral provisions of the product.
	 */
	CollateralProvisions CollateralProvisions
	Meta                 MetaFields
}

/**
 * A class to specify Collateral Access Breach language.
 */
type CollateralAccessBreach struct {
	/**
	 * Collateral Access Breach terms are applicable when True and not applicable when False.
	 */
	IsApplicable bool
	/**
	 * Determination of whether the Collateral Access Breach end date is a number of days (True) or specified (False).
	 */
	CabEndDateElection bool
	/**
	 * The business days following the related Collateral Access Breach when the additional terms end.
	 */
	CabEndDate float64
	/**
	 * Specific terms for when Collateral Access Breach terms end.
	 */
	CabEndDateTerms string
}

/**
 * Represents the parameters needed to calculate the floating rate paid on collateral holdings.
 */
type CollateralAgreementFloatingRate struct {
	RateOption ReferenceWithMeta
	/**
	 * The ISDA Spread or a Spread schedule expressed as explicit spreads and dates. In the case of a schedule, the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The spread is a per annum rate, expressed as a decimal. For purposes of determining a calculation period amount, if positive the spread will be added to the floating rate and if negative the spread will be subtracted from the floating rate. A positive 10 basis point (0.1%) spread would be represented as 0.001.
	 */
	SpreadSchedule SpreadSchedule
	/**
	 * The cap rate or cap rate schedule, if any, which applies to the floating rate. The cap rate (strike) is only required where the floating rate on a swap stream is capped at a certain level. A cap rate schedule is expressed as explicit cap rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The cap rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A cap rate of 5% would be represented as 0.05.
	 */
	CapRateSchedule StrikeSchedule
	/**
	 * The floor rate or floor rate schedule, if any, which applies to the floating rate. The floor rate (strike) is only required where the floating rate on a swap stream is floored at a certain strike level. A floor rate schedule is expressed as explicit floor rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The floor rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A floor rate of 5% would be represented as 0.05.
	 */
	FloorRateSchedule StrikeSchedule
	Meta              MetaFields
	/**
	 * Specifies how negative rates should be applied.  If rates go negative, should the payment be reversed (true) or zeroed out (false)?
	 */
	NegativeInterest bool
	/**
	 * Specifies how spreads should be applied in a low/negative rate environment.  If true, spread is applied only if rate is positive.
	 */
	CompressibleSpread bool
}

/**
 * Represents common attributes to define a collateral balance recorded by the principal as held or posted.
 */
type CollateralBalance struct {
	/**
	 * Defines the collateral balance breakdown of settlement status.
	 */
	CollateralBalanceStatus CollateralStatusEnum
	/**
	 * Indicates if the collateral balance amount is based on pre or post haircut, if a haircut is associated with the collateral asset
	 */
	HaircutIndicator HaircutIndicatorEnum
	/**
	 * Specifies the collateral balance amount in base currency determined within a collateral legal agreement, or defined for reporting purposes.
	 */
	AmountBaseCurrency Money
	/**
	 * Specifies each of the parties in the collateral balance and its perspective with regards to the direction of the collateral balance, posted or received.
	 */
	PayerReceiver PartyReferencePayerReceiver
}

/**
 * The possible different terms that can be combined, using AND, OR and NOT logic, to define the issuers and/or assets that meet a given criteria for collateral.
 */
type CollateralCriteria struct {
	/**
	 * Enables two or more Collateral Criteria to be combined using AND logic.
	 */
	AllCriteria AllCriteria
	/**
	 * Enables two or more Collateral Criteria to be combined using OR logic.
	 */
	AnyCriteria AnyCriteria
	/**
	 * Enables a single Collateral Criteria to be excluded using NOT logic.
	 */
	NegativeCriteria *NegativeCriteria
	/**
	 * Criteria is the type of entity issuing the asset.
	 */
	CollateralIssuerType CollateralIssuerType
	/**
	 * Criteria is the type of entity guaranteeing the asset.
	 */
	CollateralGuarantorType CollateralGuarantorType
	/**
	 * Criteria is the asset type of the collateral.
	 */
	AssetType AssetType
	/**
	 * Criteria is the issuing entity country of origin.
	 */
	IssuerCountryOfOrigin IssuerCountryOfOrigin
	/**
	 * Criteria is the collateral asset country of origin.
	 */
	AssetCountryOfOrigin AssetCountryOfOrigin
	/**
	 * Criteria is the denominated currency of the collateral.
	 */
	CurrencyCodeEnum CurrencyCodeEnum
	/**
	 * Criteria is a specific named issuer entity.
	 */
	IssuerName IssuerName
	/**
	 * Criteria is the agency rating(s) of the issuer.
	 */
	IssuerAgencyRating IssuerAgencyRating
	/**
	 * Criteria is the agency rating(s) of the country of the issuer.
	 */
	SovereignAgencyRating SovereignAgencyRating
	/**
	 * Criteria is the agency rating(s) of the collateral asset.
	 */
	AssetAgencyRating AssetAgencyRating
	/**
	 * Criteria is the maturity characteristics of the collateral asset.
	 */
	AssetMaturity AssetMaturity
	/**
	 * Criteria is a specifically identified asset
	 */
	SpecificAsset SpecificAsset
	/**
	 * Criteria is the taxonomy characteristics of an collateral.
	 */
	CollateralTaxonomy CollateralTaxonomy
	/**
	 * Criteria is that the collateral is listed on a specific exchange.
	 */
	ListingExchange ListingExchange
	/**
	 * Criteria is the industry sector of the collateral asset.
	 */
	ListingSector ListingSector
	/**
	 * Criteria is that the collateral is a constituent of a specific index.
	 */
	Index Index
	/**
	 * Criteria includes collateral issued by the counterparty.
	 */
	CounterpartyOwnIssuePermitted CounterpartyOwnIssuePermitted
	/**
	 * Criteria is that collateral must be denominated in the domestic currency of the issuer.
	 */
	DomesticCurrencyIssued DomesticCurrencyIssued
}

/**
 * Represents a set of criteria used to specify and describe collateral.
 */
type CollateralCriteriaBase struct {
	/**
	 * The specific criteria that applies. It can be created using AND, OR and NOT logic, and both asset and issuer characteristics.
	 */
	CollateralCriteria CollateralCriteria
	/**
	 * Specifies which of the two counterparties the criteria applies to (either one or both counterparties). This attribute is optional, in case the applicable party is already specified elsewhere within a party election.
	 */
	AppliesTo []CounterpartyRoleEnum
	/**
	 * Restrict the criteria to only apply to a specific type of margin, ie IM or VM.
	 */
	RestrictTo CollateralMarginTypeEnum
	/**
	 * Denotes which Criteria has priority if more than one agency rating applies.
	 */
	RatingPriorityResolution RatingPriorityResolutionEnum
}

/**
 * Specifies the origin of entity guaranteeing the collateral.
 */
type CollateralGuarantorType struct {
	/**
	 * Specifies the origin of entity guaranteeing the collateral.
	 */
	GuarantorType CollateralEntityTypeEnum
}

/**
 * Represents parameters for calculating the amount the floating interest calculation, e.g.  for a single currency or defaults for all currencies.
 */
type CollateralInterestCalculationParameters struct {
	/**
	 * Specifies the applicable fixed rate  if used.
	 */
	FixedRate float64
	/**
	 * Specifies the floating interest rate to be used.
	 */
	FloatingRate CollateralAgreementFloatingRate
	/**
	 * If True, specifies that the interest transfers should be converted to base currency equivalent, or if False specifies that the transfer should be in the currency of the collateral.
	 */
	InBaseCurrency bool
	/**
	 * Specifies the type of compounding to be applied (None, Business, Calendar).
	 */
	CompoundingType CompoundingTypeEnum
	/**
	 * Specifies the applicable business centers for compounding.
	 */
	CompoundingBusinessCenter []BusinessCenter
	/**
	 * Specifies the day count fraction to use for that currency.
	 */
	DayCountFraction DayCountFractionEnum
	/**
	 * Specifies the rounding rules for settling in that currency.
	 */
	Rounding Rounding
	/**
	 * Specifies when/how often is rounding applied?
	 */
	RoundingFrequency RoundingFrequencyEnum
	/**
	 * Specifies the withholding tax rate if a withholding tax is applicable.
	 */
	WithholdingTaxRate float64
}

/**
 * Represents parameters that describe how calculated interest amounts are handled, i.e. are they transferred/distributed, or is the collateral balance adjusted, is netting done, and any other special handling.
 */
type CollateralInterestHandlingParameters struct {
	/**
	 * Specifies how the collateral interest is to be handled.
	 */
	InterestPaymentHandling CollateralInterestHandlingEnum
	/**
	 * Specifies applicable business centers for payments.
	 */
	PaymentBusinessCenter []BusinessCenter
	/**
	 * Indicates whether to net Held and Posted Interest Payments (i.e. whether interest payable for a period can be netted with interest receivable).
	 */
	NetPostedAndHeldInterest bool
	/**
	 * Indicates whether the interest amount may be offset against any margin call deliver or return amounts?   (aka 'net payments' indicator).
	 */
	NetInterestWithMarginCalls bool
	/**
	 * Indicates whether or not to include the open interest accrual in the margin calculation.
	 */
	IncludeAccrualInMarginCalc bool
	/**
	 * Indicates whether interest accruing on unsettled interest amount is included (continues to be accrued) in the following period.
	 */
	AccrueInterestOnUnsettledInterest bool
	/**
	 * Indicates the option that accrued interest should be calculated and distributed when a full return of collateral occurs.
	 */
	OnFullReturn bool
	/**
	 * Indicates the option that accrued interest should be calculated and distributed when a partial return collateral occurs.
	 */
	OnPartialReturn bool
	/**
	 * The application of Interest Amount with respect to the Delivery Amount and the Return Amount.
	 */
	InterestAmountApplication InterestAmountApplication
	/**
	 * Specifies the level below which the interest will be rolled over.
	 */
	InterestRolloverLimit NumberBound
	/**
	 * Specifies the level below which the interest will be written off; if omitted write-off is not applicable.
	 */
	WriteoffLimit NumberBound
	/**
	 * Specifies the alternative to interest amounts.
	 */
	AlternativeToInterestAmount AlternativeToInterestAmountEnum
	/**
	 * Specifies an alternative to interest amount, when the alternative provision clause is specified.
	 */
	AlternativeProvision string
	/**
	 * Specifies the time of day that interest needs to be confirmed by.
	 */
	CutoffTime time.Time
	/**
	 * Specifies the terms describing notification requirements.
	 */
	Notification CollateralInterestNotification
}

/**
 * Represents the parameters describing when notifications should be made for required collateral interest transfers.
 */
type CollateralInterestNotification struct {
	/**
	 * Specifies what triggers notification (should be enum) Interest Statement Frequency, Period End Date.
	 */
	Trigger string
	/**
	 * Specifies the number of days before (negative) or after (positive) the trigger event.
	 */
	Offset float64
	/**
	 * Specifies the time of day that the notification should occur.
	 */
	NotificationTime time.Time
	/**
	 * The type of days on which notification should occur.
	 */
	NotificationDayType DayTypeEnum
}

/**
 * Represents the floating interest calculation and distribution parameters for a single currency.
 */
type CollateralInterestParameters struct {
	/**
	 * Represents the party to which these parameters apply (the applicable party).  In other words, if the parameters are different depending on which party is posting/holding the collateral, for which party to the Collateral Agreement (Party 1 or Party 2) that is posting the collateral do these parameters apply?
	 */
	PostingParty CounterpartyRoleEnum
	/**
	 * Specifies the type of margin for which interest is being calculated, if the parameters are different depending on type of margin (initial or variation).
	 */
	MarginType CollateralMarginTypeEnum
	/**
	 * Specifies the currency for which the parameters are captured.
	 */
	Currency string
	/**
	 * Represents the basic interest calculation parameters.
	 */
	InterestCalculationParameters CollateralInterestCalculationParameters
	/**
	 * Represents how often and when interest is calculated.
	 */
	InterestCalculationFrequency CalculationFrequency
	/**
	 * Represents the parameters describing how and when interest transfer occurs.
	 */
	InterestHandlingParameters CollateralInterestHandlingParameters
}

/**
 * Represents a class to allow specification of the type of entity issuing the collateral.
 */
type CollateralIssuerType struct {
	/**
	 * Specifies the origin of entity issuing the collateral.
	 */
	IssuerType CollateralEntityTypeEnum
	/**
	 * Specifies debt issued by international organisations and multilateral banks.
	 */
	SupraNationalType SupraNationalIssuerTypeEnum
	/**
	 * Specifies debt issues by institutions or bodies, typically constituted by statute, with a function mandated by the government and subject to government supervision inclusive of profit- and non-profit making bodies. Includes the US Agencies and GSEs and the EU concept of public sector entities. Excluding any entities which are also Regional Government.
	 */
	QuasiGovernmentType QuasiGovernmentIssuerType
	/**
	 * Specifies Regional government, local authority or municipal.
	 */
	RegionalGovernmentType RegionalGovernmentIssuerType
	/**
	 * Specifies a subsidiary company that is formed to undertake a specific business purpose of acquisition and financing of specific assets on a potentially limited recourse basis dependent of how it is designed. E.g. asset backed securities, including securitisations.
	 */
	SpecialPurposeVehicleType SpecialPurposeVehicleIssuerType
}

/**
 * A class to specify the Collateral Management Agreement election.
 */
type CollateralManagementAgreement struct {
	/**
	 * The parties' Collateral Management Agreement election.
	 */
	PartyElection []CollateralManagementAgreementElection
}

/**
 * A class to specify the Collateral Management Agreement election.
 */
type CollateralManagementAgreementElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The designated Collateral Management Agreement with respect to the elective party as the Obligee.
	 */
	CollateralManagementAgreement string
}

/**
 * Represents common attributes to define the details of collateral assets, to be used in margin call messaging and contribute to collateral balances e.g securities in a collateral account.
 */
type CollateralPortfolio struct {
	/**
	 * Specifies a unique identifier for a set of collateral positions in a portfolio.
	 */
	PortfolioIdentifier Identifier
	/**
	 * Specifies the individual components of the collateral positions in the collateral portfolio.
	 */
	CollateralPosition []CollateralPosition
	/**
	 * Represents the populated or calculated collateral aggregate balance amount for the collateral portfolio.
	 */
	CollateralBalance []CollateralBalance
	/**
	 *  The specification of a legal agreement between two parties governing the collateral relationship such as Credit Support Agreement or Collateral Transfer Agreement etc. (NB: this can be provided by reference to a global key for each LegalAgreement object).
	 */
	LegalAgreement ReferenceWithMeta
	Meta           MetaFields
}

/**
 * Specifies the individual components of collateral positions.
 */
type CollateralPosition struct {
	/**
	 * Position with many price quantities.
	 */
	PriceQuantity []PriceQuantity
	/**
	 * The product underlying the position.
	 */
	Product Product
	/**
	 * The asset that underlies the position.
	 */
	Asset Asset
	/**
	 * The aggregate cost of proceeds.
	 */
	CashBalance Money
	/**
	 * Reference to the Contract, in case product is contractual and the contract has been formed.
	 */
	TradeReference ReferenceWithMeta
	/**
	 * Specifies if there is any treatment to be applied to collateral, such as percentage discount which will impact collateral value.
	 */
	Treatment CollateralTreatment
	/**
	 * Indicates the collateral positions settlement status.
	 */
	CollateralPositionStatus CollateralStatusEnum
}

/**
 * Contains collateral attributes which can also inherit information from a GMRA
 */
type CollateralProvisions struct {
	/**
	 * Enumerates the collateral types which are accepted by the Seller.
	 */
	CollateralType CollateralTypeEnum
	/**
	 * The eligible collateral as specified in relation to the transaction.
	 */
	EligibleCollateral []EligibleCollateralCriteria
	/**
	 * The provisions for collateral substitutions such as how many and when they are allowed.
	 */
	SubstitutionProvisions SubstitutionProvisions
}

/**
 * A class to specify the rounding methodology applicable to the Delivery Amount and the Return Amount.
 */
type CollateralRounding struct {
	/**
	 * The nearest integral multiple of Base Currency units to which the Delivery Amount will be rounded in accordance with the specified rounding direction.
	 */
	DeliveryAmount float64
	/**
	 * The rounding rule applicable to the Delivery Amount (which can be (i) up to nearest; (ii) down to nearest).
	 */
	DeliveryDirection RoundingModeEnum
	/**
	 * The nearest integral multiple of Base Currency units to which the Return Amount will be rounded in accordance with the specified rounding direction.
	 */
	ReturnAmount float64
	/**
	 * The rounding rule applicable to the Return Amount (which can be (i) up to nearest; (ii) down to nearest).
	 */
	ReturnDirection RoundingModeEnum
	/**
	 * The currency in which the Delivery Amount and Return Amount rounding amounts are specified.
	 */
	Currency ISOCurrencyCodeEnum
	/**
	 * Utilised where the clause data structure is not able to capture a material aspect of the clause.
	 */
	Other string
}

/**
 * Specifies the collateral taxonomy, which is composed of a taxonomy value and a taxonomy source.
 */
type CollateralTaxonomy struct {
	/**
	 * Specifies the taxonomy value.
	 */
	TaxonomyValue CollateralTaxonomyValue
	/**
	 * Specifies the taxonomy source.
	 */
	TaxonomySource TaxonomySourceEnum
}

/**
 * Specifies the collateral taxonomy value, either as a specified enumeration or as a string.
 */
type CollateralTaxonomyValue struct {
	/**
	 * Identifies European Union Eligible Collateral Assets classification categories based on EMIR Uncleared Margin Rules. Eligible Collateral asset classes for both initial margin (IM) and variation margin (VM) posted and collected between specified entities. Please note: EMIR regulation will detail which eligible collateral assets classes apply to each type of entity pairing (counterparty) and which apply to posting of IM and VM
	 */
	Eu_EMIR_EligibleCollateral []EU_EMIR_EligibleCollateralEnum
	/**
	 * Identifies United Kingdom Eligible Collateral Assets classification categories based on UK Onshored EMIR Uncleared Margin Rules Eligible Collateral asset classes for both initial margin (IM) and variation margin (VM) posted and collected between specified entities. Please note: UK EMIR regulation will detail which eligible collateral assets classes apply to each type of entity pairing (counterparty) and which apply to posting of IM and VM.
	 */
	Uk_EMIR_EligibleCollateral []UK_EMIR_EligibleCollateralEnum
	/**
	 * Identifies US Eligible Collateral Assets classification categories based on Uncleared Margin Rules published by the CFTC and the US Prudential Regulator. Note: While the same basic categories exist in the CFTC and US Prudential Regulators’ margin rules, the precise definitions or application of those rules could differ between the two rules.
	 */
	Us_CFTC_PR_EligibleCollateral []US_CFTC_PR_EligibleCollateralEnum
	/**
	 * Identifies the taxonomy value when not specified as an enumeration.
	 */
	NonEnumeratedTaxonomyValue []FieldWithMeta
}

/**
 * The set of elections which specify a Collateral Transfer Agreement.
 */
type CollateralTransferAgreementElections struct {
	/**
	 * The Regime Table provision , which determines the regulatory regime(s) applicable to each of the parties to the agreement.
	 */
	Regime Regime
	/**
	 * The determination of whether the One Way Provisions are applicable (true) or not applicable (false).
	 */
	OneWayProvisions OneWayProvisions
	/**
	 * The specification of the ISDA SIMM Method for all Covered Transactions with respect to all Regimes.
	 */
	GeneralSimmElections GeneralSimmElections
	/**
	 * The qualification of whether cross-currency swaps need to be identified in the Confirmation so that the obligations to exchange principal be disregarded for the purpose of determining the Delivery Amount or Return Amount.
	 */
	IdentifiedCrossCurrencySwap bool
	/**
	 * The specification of methodologies to compute sensitivities specific to the agreement.
	 */
	SensitivityMethodologies SensitivityMethodologies
	/**
	 * The reference currency for the purpose of specifying the FX Haircut relating to a posting obligation, as being either the Termination Currency or an FX Designated Currency.
	 */
	FxHaircutCurrency FxHaircutCurrency
	/**
	 * The security providers posting obligations.
	 */
	PostingObligations PostingObligations
	/**
	 * The specification of Additional regimes for purposes of determining whether a Regulatory Event has occurred.
	 */
	SubstitutedRegime []SubstitutedRegime
	/**
	 * The base and eligible currency(ies) for the document as specified by the parties to the agreement.
	 */
	BaseAndEligibleCurrency BaseAndEligibleCurrency
	/**
	 * The Credit Support Obligations applicable to the agreement.
	 */
	CreditSupportObligations CreditSupportObligationsCollateralTransferAgreement
	/**
	 * The set of elections for determining Valuation and Timing terms specific to the agreement.
	 */
	CalculationAndTiming CalculationAndTimingCollateralTransferAgreement
	/**
	 * The set of elections that may overwrite the default Condition Precedent provision, and the set of provisions that are deemed Access Condition.
	 */
	ConditionsPrecedent ConditionsPrecedent
	/**
	 * The conditions under which the Security Provider can substitute posted collateral.
	 */
	Substitution Substitution
	/**
	 * The election terms under which a party disputes (i) the Calculation Agents calculation of a Delivery Amount or a Return Amount, or (ii) the Value of any Transfer of Eligible Credit Support or Posted Credit Support.
	 */
	DisputeResolution DisputeResolution
	/**
	 * The bespoke provisions that might be specified by the parties to the agreement to specify the rights of Security Taker and/or Security Provider when an Early Termination or Access Condition event has occurred.
	 */
	RightsEvents RightsEvents
	/**
	 * The Custodian and Segregated Account details in respect of each party to the agreement.
	 */
	CustodyArrangements CustodyArrangements
	/**
	 * The specification Additional Representations that may be applicable to the agreement.
	 */
	AdditionalRepresentations AdditionalRepresentations
	/**
	 * The optional specification of address where the demands, specifications and notices will be communicated to for each of the parties to the agreement.
	 */
	DemandsAndNotices DemandsAndNotices
	/**
	 * The optional specification of address for transfer as specified by the respective parties to the agreement.
	 */
	AddressesForTransfer AddressesForTransfer
	/**
	 * The bespoke definition of other agreement terms as specified by the parties to the agreement.
	 */
	OtherAgreements OtherAgreements
	/**
	 * The bespoke provision that might be specified by the parties to the agreement applicable to Termination Currency.  Unless specified the definition of Termination Currency has the meaning specified in the Schedule to the ISDA Master Agreement.
	 */
	TerminationCurrencyAmendment TerminationCurrencyAmendment
	/**
	 * The bespoke provision that might be specified by the parties to the agreement applicable to Minimum Transfer Amount.  Unless specified the definition of Minimum Transfer Amount in any Other Regulatory CSA has the meaning specified in such Other Regulatory CSA.
	 */
	MinimumTransferAmountAmendment MinimumTransferAmountAmendment
	/**
	 * The bespoke provision that might be specified by the parties to the agreement applicable to Interpretations.
	 */
	InterpretationTerms string
	/**
	 * The Process Agent that might be appointed by the parties to the agreement.
	 */
	ProcessAgent ProcessAgent
	/**
	 * The jurisdiction specific terms relevant to the agreement.
	 */
	JurisdictionRelatedTerms JurisdictionRelatedTerms
	/**
	 * Any additional amendments that might be specified by the parties to the agreement.
	 */
	AdditionalAmendments string
	/**
	 * Any additional terms that might be specified applicable.
	 */
	AdditionalBespokeTerms string
	/**
	 * The terms of the Rider for the ISDA Euroclear 2019 Collateral Transfer Agreement with respect to the use of a Pledgee Representative attached to this Agreement.
	 */
	PledgeeRepresentativeRider PledgeeRepresentativeRider
	/**
	 * A clause providing that final return collateral is to be provided without consideration of any Minimum Transfer Amount and collateral haircut.
	 */
	FinalReturns FinalReturns
}

/**
 * Specification of transfer / settlement timing for cash and securities collateral assuming that the notice is provided before the Notification Time (and with a 1LBD lag introduced if the notice is delivered after the Notification Time), and that securities allow for a further Local Business Day for transfer / settlement.
 */
type CollateralTransferTiming struct {
	/**
	 * Specification of transfer / settlement timing for cash and securities collateral assuming that the notice is provided before the Notification Time (and with a 1LBD lag introduced if the notice is delivered after the Notification Time), and that securities allow for a further Local Business Day for transfer / settlement.
	 */
	TransferSettlementTiming TransferSettlementTiming
	/**
	 * Bespoke language removing the pre-print requirement to transfer legal title of securities collateral through written instructions to the relevant depository institution or other securities intermediaries solely for the purposes of the 1994 New York Law Credit Support Annex.
	 */
	CollateralTransferTimingDefinition CollateralTransferTimingDefinition
}

/**
 * Bespoke language removing the pre-print requirement to transfer legal title of securities collateral through written instructions to the relevant depository institution or other securities intermediaries solely for the purposes of the 1994 New York Law Credit Support Annex.
 */
type CollateralTransferTimingDefinition struct {
	/**
	 * A boolean flag to represent bespoke amendment to the pre-print definition of Transfer.
	 */
	IsApplicable bool
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * Specifies the treatment terms for the eligible collateral criteria specified.
 */
type CollateralTreatment struct {
	/**
	 * Specification of the valuation treatment for the specified collateral.
	 */
	ValuationTreatment CollateralValuationTreatment
	/**
	 * Specification of concentration limits applicable to the collateral criteria.
	 */
	ConcentrationLimit []ConcentrationLimit
	/**
	 * A boolean attribute to specify whether collateral critieria are inclusion (True) or exclusion (False) criteria.
	 */
	IsIncluded bool
}

/**
 * Specification of the valuation treatment for the specified collateral.
 */
type CollateralValuationTreatment struct {
	/**
	 * Specifies a haircut percentage to be applied to the value of asset and used as a discount factor to the value of the collateral asset,expressed as a percentage in decimal terms. As an example a 0.5% haircut would be represented as a decimal number 0.005.
	 */
	HaircutPercentage float64
	/**
	 * Specifies a percentage value of transaction needing to be posted as collateral expressed as a valuation. As an example a 104% requirement would be represented as a decimal number 1.04.
	 */
	MarginPercentage float64
	/**
	 * Specifies an FX haircut applied to a specific asset which is agreed between the parties (for example, if pledgor eligible collateral is not denominated in the termination currency or under other specified cases in collateral support documents both for initial margin and variation margin).The percentage value is expressed as the discount haircut to the value of the collateral- as an example an 8% FX haircut would be expressed as 0.08.
	 */
	FxHaircutPercentage float64
	/**
	 * Specifies a percentage value of any additional haircut to be applied to a collateral asset,the percentage value is expressed as the discount haircut to the value of the collateral- as an example a 5% haircut would be expressed as 0.05.
	 */
	AdditionalHaircutPercentage float64
}

/**
 * Details how particular types of collateral are valued (other than in respect of a default).
 */
type CollateralValueMethod struct {
	/**
	 * Details of how cash collateral is valued.
	 */
	CashCollateral ValueCashEnum
	/**
	 * Details of how securities collateral is valued.
	 */
	SecuritiesCollateral ValueSecuritiesEnum
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * Identifies a specific commodity by referencing a product identifier or by a product definition.
 */
type Commodity struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * Specifies the commodity underlier in the event that no ISDA Commodity Reference Benchmark exists.
	 */
	CommodityProductDefinition CommodityProductDefinition
	/**
	 * Describes the required quote type of the underlying price that will be observed. Example values include 'Bid, 'Ask', 'Settlement' (for a futures contract) and 'WeightedAverage' (for some published prices and indices).
	 */
	PriceQuoteType QuotationSideEnum
	/**
	 * Specifies the parameters for identifying the relevant contract date when the commodity reference price is a futures contract.
	 */
	DeliveryDateReference DeliveryDateParameters
	/**
	 * Provides additional information about the commodity underlier.
	 */
	Description string
}

/**
 * Payout based on the averaged price of a referenced underlier. (e.g. Commodities). Can represent both average (average of many) & bullet (average of 1) pricing
 */
type CommodityPayout struct {
	/**
	 * Canonical representation of the payer and receiver parties applicable to each payout leg.
	 */
	PayerReceiver PayerReceiver
	/**
	 * Each payout leg must implement the quantity concept as a 'resolvable' type, which allows for different payout legs to be linked to each other (e.g. in the case of cross-curreny products).
	 */
	PriceQuantity ResolvablePriceQuantity
	/**
	 * The specification of the principal exchange. Optional as only applicable in the case of cross-currency or zero-coupon swaps with a final payment.
	 */
	PrincipalPayment PrincipalPayments
	/**
	 * Each payout leg must specifies its settlement terms, including the delivery type (i.e. cash vs physical, and their respective terms), the transfer type (DvP etc.) and settlement date, if any.
	 */
	SettlementTerms SettlementTerms
	/**
	 * Indicates if the averaging calculation, when applicable, is weighted or unweighted.
	 */
	AveragingFeature AveragingCalculation
	/**
	 * Defines parameters in which the commodity price is assessed.
	 */
	CommodityPriceReturnTerms CommodityPriceReturnTerms
	/**
	 * Specifies specific dates or parametric rules for the dates on which the price will be determined.
	 */
	PricingDates PricingDates
	/**
	 * Allows the full representation of a payout by defining a set of schedule periods. It supports standard schedule customization by expressing all the dates, quantities, and pricing data in a non-parametric way.
	 */
	Schedule CalculationSchedule
	/**
	 * Defines the calculation period dates schedule.
	 */
	CalculationPeriodDates CalculationPeriodDates
	/**
	 * Defines the payment date schedule, as defined by the parameters that are needed to specify it, either in a parametric way or by reference to another schedule of dates (e.g. the valuation dates).
	 */
	PaymentDates PaymentDates
	/**
	 * Identifies the underlying product that is referenced for pricing of the applicable leg in a swap. Referenced in the '2018 ISDA CDM Equity Confirmation for Security Equity Swap' as Security.
	 */
	Underlier Underlier
	/**
	 * Defines quanto or composite FX features that are included in the swap leg.
	 */
	FxFeature FxFeature
	/**
	 * Contains the information relative to the delivery of the asset.
	 */
	Delivery AssetDeliveryInformation
}

/**
 * Defines parameters in which the commodity price is assessed.
 */
type CommodityPriceReturnTerms struct {
	/**
	 * Defines rounding rules and precision to be used in the rounding of a number.
	 */
	Rounding Rounding
	/**
	 * Defines a spread value for one or more defined dates.
	 */
	Spread SpreadSchedule
	/**
	 * Used in conjunction with an exchange-based pricing source. Identifies a way in which the futures contracts referenced will roll between periods.
	 */
	RollFeature RollFeature
	/**
	 * Defines the conversion applied if the quantity unit on contract is different from unit on referenced underlier.
	 */
	ConversionFactor float64
}

/**
 * Specifies the commodity underlier in the event that no ISDA Commodity Reference Price exists.
 */
type CommodityProductDefinition struct {
	/**
	 * Specifies the type of commodity.
	 */
	ReferenceFramework CommodityReferenceFramework
	/**
	 * Specifies a publication that provides the commodity price, including, where applicable the details of where in the publication the price is published.  Applicable when the commodity reference price is not a futures contract
	 */
	PriceSource PriceSource
	/**
	 * Specifies the publication where the commodity prices can be found.
	 */
	CommodityInfoPublisher CommodityInformationPublisherEnum
	/**
	 *  Identifies the exchange from which the reference price should be sourced, using the scheme at the following url: http://www.fpml.org/coding-scheme/external/exchange-id-MIC-1-0
	 */
	ExchangeId FieldWithMeta
}

/**
 * Specifies the type of commodity.
 */
type CommodityReferenceFramework struct {
	/**
	 * Identifies the commodity more specifically. Where possible, this should follow the naming convention used in the 2005 ISDA Commodity Definitions SubAnnex A, including the subCommodity and additional qualifiers, but should be limited to 256 characters or less.
	 */
	CommodityName string
	/**
	 * Provides an enumerated value for a capacity unit, generally used in the context of defining quantities for commodities.
	 */
	CapacityUnit CapacityUnitEnum
	/**
	 * Provides an enumerated values for a weather unit, generally used in the context of defining quantities for commodities.
	 */
	WeatherUnit WeatherUnitEnum
	/**
	 * Defines the currency in which the commodity is priced.
	 */
	Currency FieldWithMeta
}

/**
 * Specifies the conditions to be applied for converting into a reference currency when the actual currency rate is not determined upfront.
 */
type Composite struct {
	/**
	 * Specifies the method according to which an amount or a date is determined.
	 */
	DeterminationMethod DeterminationMethodEnum
	/**
	 * A date specified as some offset to another date (the anchor date).
	 */
	RelativeDate RelativeDateOffset
	/**
	 * Specifies the methodology (reference source and, optionally, fixing time) to be used for determining a currency conversion rate.
	 */
	FxSpotRateSource FxSpotRateSource
	/**
	 * The time at which the spot currency exchange rate will be observed. It is specified as a time in a business day calendar location, e.g. 11:00am London time.
	 */
	FixingTime BusinessCenterTime
}

/**
 * A class to specify the outcome of a computed amount, for testing purposes.
 */
type ComputedAmount struct {
	CallFunction string
	Amount       float64
	/**
	 * The currency in which the computed amount is denominated. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	Currency FieldWithMeta
}

/**
 * Represents a class to describe concentration limits that may be applicable to eligible collateral criteria.
 */
type ConcentrationLimit struct {
	/**
	 * Specifies a set of criteria to describe the assets that the concentration limits apply to.
	 */
	ConcentrationLimitCriteria ConcentrationLimitCriteria
	/**
	 * Specifies the value of collateral limit represented as a range.
	 */
	ValueLimit MoneyRange
	/**
	 * Specifies the perecentage of collateral limit represented as a decimal number - example 25% is 0.25.
	 */
	PercentageLimit NumberRange
}

/**
 * Respresents a class to describe a set of criteria to describe specific assets that the concentration limits apply to.
 */
type ConcentrationLimitCriteria struct {
	/**
	 * The specific criteria that applies. It can be created using AND, OR and NOT logic, and both asset and issuer characteristics.
	 */
	CollateralCriteria CollateralCriteria
	/**
	 * Specifies which of the two counterparties the criteria applies to (either one or both counterparties). This attribute is optional, in case the applicable party is already specified elsewhere within a party election.
	 */
	AppliesTo []CounterpartyRoleEnum
	/**
	 * Restrict the criteria to only apply to a specific type of margin, ie IM or VM.
	 */
	RestrictTo CollateralMarginTypeEnum
	/**
	 * Denotes which Criteria has priority if more than one agency rating applies.
	 */
	RatingPriorityResolution RatingPriorityResolutionEnum
	/**
	 * Specifies the type of concentration limit to be applied.
	 */
	ConcentrationLimitType ConcentrationLimitTypeEnum
	/**
	 * Specifies an average trading volume on an exchange in relation to Equity products.
	 */
	AverageTradingVolume AverageTradingVolume
}

/**
 * A class to specify the two set of elections that may overwrite the default Condition Precedent provision.
 */
type ConditionsPrecedent struct {
	/**
	 * The election to specify whether the standard Conditions Precedent apply.
	 */
	ConditionsPrecedentElection ExceptionEnum
	/**
	 * The custom provisions that might be specified by the parties to the agreement for the purpose of overwriting the default Condition Precedent provision specified in the CSA.
	 */
	CustomProvision string
	/**
	 * Specifies the events elected by the parties that are deemed an Access Condition (for Initial Margin CSA) or a Specified Condition (for Variation Margin and Legacy CSA), which, if triggered, may temporarily or permanently suspend rights to rehypothecate, transfer, or substitute collateral.
	 */
	SpecifiedConditionOrAccessCondition SpecifiedConditionOrAccessCondition
}

/**
 * A class describing the weight of each of the underlier constituent within the basket, either in absolute or relative terms.
 */
type ConstituentWeight struct {
	/**
	 * The number of units (index or securities) that constitute the underlier of the swap. In the case of a basket swap, this element is used to reference both the number of basket units, and the number of each asset components of the basket when these are expressed in absolute terms.
	 */
	OpenUnits float64
	/**
	 * The relative weight of each respective basket constituent, expressed in percentage. A basket percentage of 5% would be represented as 0.05.
	 */
	BasketPercentage float64
}

/**
 * A class to specify contact information associated with a party: telephone, postal/street address, email and web page.
 */
type ContactInformation struct {
	/**
	 * The telephone number.
	 */
	Telephone []TelephoneNumber
	/**
	 * The street/postal address.
	 */
	Address []Address
	/**
	 * The email address.
	 */
	Email []string
	/**
	 * The web page. This attribute is not specified as part of the FpML ContactInformation complex type.
	 */
	WebPage []string
}

/**
 * Specifies the party reference and the associated contact information.
 */
type ContactInformationElection struct {
	/**
	 * The party to which the election and details apply.
	 */
	PartyReference CounterpartyRoleEnum
	/**
	 * The main notice/address information of the party.
	 */
	PrimaryContactInformation ContactInformation
}

/**
 * Encapsulates data features common to trade and position.
 */
type ContractBase struct {
	/**
	 * Represents information specific to trades or positions involving contractual products.
	 */
	ContractDetails ReferenceWithMeta
	/**
	 * Defines specific attributes that relate to trade or position executions.
	 */
	ExecutionDetails ReferenceWithMeta
	/**
	 * Represents the collateral obligations of a party.
	 */
	Collateral ReferenceWithMeta
}

/**
 * Defines specific attributes that relate to contractual details of trades.
 */
type ContractDetails struct {
	/**
	 * Represents the legal document(s) that governs a trade and associated contractual product terms, either as a reference to such documents when specified as part of the CDM, or through identification of some of the key terms of those documents, such as the type of document, the document identifier, the publisher, the document vintage and the agreement date.
	 */
	Documentation []LegalAgreement
	/**
	 * Represents the law governing the trade and associated contractual product terms.
	 */
	GoverningLaw FieldWithMeta
	Meta         MetaFields
}

/**
 * Specifies instructions to create a fully formed contract, with optional legal agreements.
 */
type ContractFormationInstruction struct {
	/**
	 * Optional legal agreements associated to the contract being formed, for instance a master agreement.
	 */
	LegalAgreement []LegalAgreement
}

type ContractualDefinition struct {
	/**
	 * Corresponds to the unique identifier of the Contractual Definition in which the code is published
	 */
	Identifier Identifier
	/**
	 * Contractual Definition Identifier in which the code is published. Includes Document Type and Document Version
	 */
	ContractualDefinitionIdentifier ContractualDefinitionIdentifier
	/**
	 * 2021-06-11
	 */
	PublicationDate time.Time
}

type ContractualDefinitionIdentifier struct {
	/**
	 * e.g. ISDA2021Definitions
	 */
	ContractualDefinitionType ContractualDefinitionsEnum
	/**
	 * e.g. V1
	 */
	ContractualDefinitionVersion string
}

type ContractualMatrix struct {
	/**
	 * Identifies the form of applicable matrix.
	 */
	MatrixType FieldWithMeta
	/**
	 * Defines any applicable key into the relevant matrix. For example, the Transaction Type would be the single term required for the Credit Derivatives Physical Settlement Matrix. This element should be omitted in the case of the 2000 ISDA Definitions Settlement Matrix for Early Termination and Swaptions.
	 */
	MatrixTerm FieldWithMeta
}

/**
 * A contractual supplement (such as those published by ISDA) and its publication date that will apply to the trade.
 */
type ContractualTermsSupplement struct {
	/**
	 * Identifies the form of applicable contractual supplement.
	 */
	ContractualTermsSupplementType FieldWithMeta
	/**
	 * Specifies the publication date of the applicable version of the contractual supplement.
	 */
	PublicationDate time.Time
}

/**
 * A class to specify the relationship between the Control Agreement and the Credit Support Agreement.
 */
type ControlAgreement struct {
	/**
	 * The party specific elections.
	 */
	PartyElection []ControlAgreementElections
}

/**
 * A class to specify the Control Agreement election sby each party to the agreement.
 */
type ControlAgreementElections struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The identification of whether the Control Agreement is a Credit Support Document with respect to each party.
	 */
	ControlAgreementAsCsd bool
	/**
	 * Unless specified as inapplicable in the event of any inconsistency between this Deed and the Control Agreement, this Deed will prevail over the Control Agreement.
	 */
	ConsistencyWithControlAgreement bool
	/**
	 * Unless specified as inapplicable the parties recognise that the Control Agreement is a means by which the parties can perform their obligations.
	 */
	RelationshipWithControlAgreement bool
}

/**
 * A class to specify Control Agreement language related to delivery of a Notice of Exclusive Control.
 */
type ControlAgreementNecEvent struct {
	/**
	 * Indicates party specific Control Agreement language related to delivery of a Notice of Exclusive Control.
	 */
	ControlAgreementNecEventElection []ControlAgreementNecEventElection
}

/**
 * A class to specify party specific Control Agreement language related to delivery of a Notice of Exclusive Control.
 */
type ControlAgreementNecEventElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * Indicates Control Agreement language related to delivery of a Notice of Exclusive Control.
	 */
	NecEvent bool
}

/**
 * Specifies the relevant data regarding a corporate action
 */
type CorporateAction struct {
	/**
	 * The type of corporate action taking place.
	 */
	CorporateActionType CorporateActionTypeEnum
	/**
	 * The date on which the corporate action is known to have taken place.
	 */
	ExDate time.Time
	/**
	 * The date on which resulting from the corporate action are delivered.
	 */
	PayDate time.Time
	/**
	 * The date on which the account phyical balance and related underlier ownership is recorded by the custodian.
	 */
	RecordDate time.Time
	/**
	 * The date on which the corporate action is announced by the issuer.
	 */
	AnnouncementDate time.Time
	/**
	 * The underlier impacted by the corporate action.
	 */
	Underlier Underlier
	/**
	 * Specifies any additional details e.g. further descriptions depending on the particular type of Corporate Action, adjustmentFactor, calculations
	 */
	AdjustmentFactor AdjustmentFactor
	/**
	 * A class defining the source of data used in an Observation for any purposes. The data is either sourced from a Provider, or from a Party.
	 */
	InformationSource InformationSource
	/**
	 * To record observations of dividends.
	 */
	DividendObservation PriceSchedule
	/**
	 * Corporate action triggered by the occurrence of an event which description is seperatly agreed between the parties and/or described in any referenced agreement (for instance bespoke term described in any string type objects present in most root attributes of Agreement type).
	 */
	BespokeEventDescription string
}

type CorrelationReturnTerms struct {
	/**
	 * Contains all non-date valuation information.
	 */
	ValuationTerms ValuationTerms
	/**
	 * This specifies the numerator of an annualization factor. Frequently this number is equal to the number of observations of prices in a year e.g. 252.
	 */
	AnnualizationFactor int
	/**
	 * The parameters which define whether dividends are applicable
	 */
	DividendApplicability DividendApplicability
	/**
	 * Contains Equity Underlyer provisions regarding jurisdiction and fallbacks.
	 */
	EquityUnderlierProvisions EquityUnderlierProvisions
	/**
	 * Indicates whether the price of shares is adjusted for dividends or not.
	 */
	SharePriceDividendAdjustment bool
	/**
	 * Expected number of trading days.
	 */
	ExpectedN int
	/**
	 * Contract will strike off this initial level. Providing just the initialLevel without initialLevelSource, infers that this is AgreedInitialPrice - a specified Initial Index Level.
	 */
	InitialLevel float64
	/**
	 * In this context, this is AgreedInitialPrice - a specified Initial Index Level.
	 */
	InitialLevelSource DeterminationMethodEnum
	/**
	 * Specifies whether Mean Adjustment is applicable or not in the calculation of the Realized Volatility, Variance or Correlation
	 */
	MeanAdjustment bool
	/**
	 * Performance calculation, in accordance with Part 1 Section 12 of the 2018 ISDA CDM Equity Confirmation for Security Equity Swap, Para 75. 'Equity Performance'. Cumulative performance is used as a notional multiplier factor on both legs of an Equity Swap.
	 */
	Performance string
	/**
	 * Correlation Strike Price in accordance with the ISDA 2011 Equity Derivatives Definitions.
	 */
	CorrelationStrikePrice Price
	/**
	 * Describes correlation bounds, which form a cap and a floor on the realized correlation.
	 */
	BoundedCorrelation NumberRange
	/**
	 * Number of data series, normal market practice is that correlation data sets are drawn from geographic market areas, such as America, Europe and Asia Pacific, each of these geographic areas will have its own data series to avoid contagion.
	 */
	NumberOfDataSeries int
}

/**
 * Defines a counterparty enumerated value, e.g. Party1 or Party2, with an associated party reference. The product is agnostic to the actual parties to the transaction, with the party references abstracted away from the product definition and replaced by the CounterpartyEnum (e.g. values Party1 or Party2). The CounterpartyEnum can then be positioned in the product (e.g. to specify which counterparty is the payer, receiver etc) and this Counterparty type, which is positioned outside of the product definition, allows the CounterpartyEnum to be associated with an actual party reference.
 */
type Counterparty struct {
	/**
	 * Specifies the CounterpartyEnum, e.g. either Party1 or Party2, that is associated to the partyReference.
	 */
	Role CounterpartyRoleEnum
	/**
	 * Specifies the party that is associated to the counterparty.
	 */
	PartyReference ReferenceWithMeta
}

type CounterpartyOwnIssuePermitted struct {
	/**
	 * Represents a filter based on whether it is permitted for the underlying asset to be issued by the posting entity or part of their corporate family.
	 */
	CounterpartyOwnIssuePermitted bool
}

/**
 * A Position describes the accumulated effect of a set of securities or financial transactions.
 */
type CounterpartyPosition struct {
	/**
	 * Represents information specific to trades or positions involving contractual products.
	 */
	ContractDetails ReferenceWithMeta
	/**
	 * Defines specific attributes that relate to trade or position executions.
	 */
	ExecutionDetails ReferenceWithMeta
	/**
	 * Represents the collateral obligations of a party.
	 */
	Collateral ReferenceWithMeta
	/**
	 * Represents the identifier(s) that uniquely identify a position for an identity issuer. A position can include multiple identifiers, for example an internal position identifer and a UTI (Unique Trade Identifier).
	 */
	PositionIdentifier []PositionIdentifier
	/**
	 * The date and time when the position was opened.
	 */
	OpenDateTime time.Time
	/**
	 * Reference to all the trades that constitute the position.
	 */
	TradeReference []ReferenceWithMeta
	/**
	 * The parties involved in the position, including the Clearing Organization.
	 */
	Party []Party
	/**
	 * Represents the role each specified party takes in the position.
	 */
	PartyRole []PartyRole
	/**
	 * Encapsulates the core constituents that characterize a position.
	 */
	PositionBase TradableProduct
}

/**
 * A business event represents a life cycle event of a position. The combination of the state changes results in a qualifiable life cycle event.
 */
type CounterpartyPositionBusinessEvent struct {
	/**
	 * The intent attribute is meant to be specified when the event qualification cannot be programmatically inferred from the event features. As a result it is only associated with those primitives that can give way to such ambiguity, the quantityChange being one of those.
	 */
	Intent PositionEventIntentEnum
	/**
	 * The intent of a corporate action on the position.
	 */
	CorporateActionIntent CorporateActionTypeEnum
	/**
	 * Specifies the date on which the event is taking place. This date is equal to the trade date in the case of a simple execution.  However it can be different from the trade date, for example in the case of a partial termination.
	 */
	EventDate time.Time
	/**
	 * The date on which the event contractually takes effect, when different from the event date.
	 */
	EffectiveDate time.Time
	/**
	 * Specifies the package information in case the business event represents several trades executed as a package (hence this attribute is optional). The package information is only instantiated once at the business event level to preserve referential integrity, whereas individual trades make reference to it to identify that they are part of a package.
	 */
	PackageInformation IdentifiedList
	/**
	 * Specifies the after position state(s) created.
	 */
	After []CounterpartyPositionState
}

/**
 * Defines the fundamental financial information that can be changed by a Primitive Event and by extension any business or life-cycle event. Each PositionState specifies where a Position is in its life-cycle. PositionState is a root type and as such, can be created independently to any other CDM data type, but can also be used as part of the CDM Event Model.
 */
type CounterpartyPositionState struct {
	/**
	 * Represents the Position that has been effected by a business or life-cycle event.
	 */
	CounterpartyPosition CounterpartyPosition
	/**
	 * Represents the State of the Position through its life-cycle.
	 */
	State State
	/**
	 * Represents the observed events related to a particular product or process, such as credit events or corporate actions.
	 */
	ObservationHistory []ObservationEvent
	ValuationHistory   []Valuation
	Meta               MetaFields
}

/**
 * Specification of Transactions covered by the legal agreement.
 */
type CoveredTransactions struct {
	/**
	 * Includes any Transaction specified below that is entered into on or after the specified date.
	 */
	InclusionDate time.Time
	/**
	 * Covered Transactions when expressed using the ISDA taxonomy.
	 */
	CoveredTransactions []ProductTaxonomy
	/**
	 * Covered Transactions when not expressed using the ISDA taxonomy.
	 */
	BespokeCoveredTransactions []string
	/**
	 * The party specific additional obligations applicable to the document.
	 */
	AdditionalObligations []AdditionalObligations
	/**
	 * Specification of amendments to the calculation of Exposure in terms of the Transactions covered.
	 */
	Exposure ExposureScope
}

/**
 *  The credit default payout specification provides the details necessary for determining when a credit payout will be triggered as well as the parameters for calculating the payout and the settlement terms. The associated globalKey denotes the ability to associate a hash value to the CreditDefaultPayout instantiations for the purpose of model cross-referencing, in support of functionality such as the event effect and the lineage.
 */
type CreditDefaultPayout struct {
	/**
	 * Canonical representation of the payer and receiver parties applicable to each payout leg.
	 */
	PayerReceiver PayerReceiver
	/**
	 * Each payout leg must implement the quantity concept as a 'resolvable' type, which allows for different payout legs to be linked to each other (e.g. in the case of cross-curreny products).
	 */
	PriceQuantity ResolvablePriceQuantity
	/**
	 * The specification of the principal exchange. Optional as only applicable in the case of cross-currency or zero-coupon swaps with a final payment.
	 */
	PrincipalPayment PrincipalPayments
	/**
	 * Each payout leg must specifies its settlement terms, including the delivery type (i.e. cash vs physical, and their respective terms), the transfer type (DvP etc.) and settlement date, if any.
	 */
	SettlementTerms SettlementTerms
	/**
	 * The specification of the non-monetary terms for the Credit Derivative Transaction, including the buyer and seller and selected items from the ISDA 2014 Credit Definition article II, such as the reference obligation and related terms.
	 */
	GeneralTerms GeneralTerms
	/**
	 * Specifies the terms for calculating a payout to protect the buyer of the swap in the case of a qualified credit event. These terms include the applicable credit events, the reference obligation, and in the case of a CDS on mortgage-backed securities, the floatingAmountEvents.
	 */
	ProtectionTerms []ProtectionTerms
	/**
	 * The qualification of the price at which the contract has been transacted, in terms of market fixed rate, initial points, market price and/or quotation style. In FpML, those attributes are positioned as part of the fee leg.
	 */
	TransactedPrice TransactedPrice
}

/**
 * Specifies the relevant data regarding a credit event.
 */
type CreditEvent struct {
	/**
	 * The type of credit event taking place.
	 */
	CreditEventType CreditEventTypeEnum
	/**
	 * The date in which the credit event is determined by the Credit Derivatives Determinations Comitee.
	 */
	EventDeterminationDate time.Time
	/**
	 * The date on which the auction is scheduled to occur.
	 */
	AuctionDate time.Time
	/**
	 * The final price resulting from the auction.
	 */
	FinalPrice Price
	/**
	 * The percentage of the original value of the asset affected by the credit event that can be recovered.
	 */
	RecoveryPercent float64
	/**
	 * A public information source, e.g. a particular newspaper or electronic news service, that may publish relevant information used in the determination of whether or not a credit event has occurred.
	 */
	PubliclyAvailableInformation []Resource
	/**
	 * The reference entity, part of a credit basket, impacted by the credit event.
	 */
	ReferenceInformation ReferenceInformation
}

type CreditEventNotice struct {
	/**
	 * The notifying party is the party that notifies the other party when a credit event has occurred by means of a credit event notice. If more than one party is referenced as being the notifying party then either party may notify the other of a credit event occurring. ISDA 2003 Term: Notifying Party.
	 */
	NotifyingParty []CounterpartyRoleEnum
	/**
	 * Inclusion of this business center element implies that Greenwich Mean Time in Section 3.3 of the 2003 ISDA Credit Derivatives Definitions is replaced by the local time of the city indicated by the businessCenter element value.
	 */
	BusinessCenter BusinessCenter
	/**
	 * A specified condition to settlement. Publicly available information means information that reasonably confirms any of the facts relevant to determining that a credit event or potential repudiation/moratorium, as applicable, has occurred. The ISDA defined list (2003) is the market standard and is considered comprehensive, and a minimum of two differing public sources must have published the relevant information, to declare a Credit Event. ISDA 2003 Term: Notice of Publicly Available Information Applicable.
	 */
	PubliclyAvailableInformation PubliclyAvailableInformation
}

/**
 * A class to specify the applicable Credit Events that would trigger a settlement, as specified in the related Confirmation and defined in the ISDA 2014 Credit Definition article IV section 4.1.
 */
type CreditEvents struct {
	/**
	 * A credit event. The reference entity has been dissolved or has become insolvent. It also covers events that may be a precursor to insolvency such as instigation of bankruptcy or insolvency proceedings. Sovereign trades are not subject to Bankruptcy as 'technically' a Sovereign cannot become bankrupt. ISDA 2003 Term: Bankruptcy.
	 */
	Bankruptcy bool
	/**
	 * A credit event. This credit event triggers, after the expiration of any applicable grace period, if the reference entity fails to make due payments in an aggregate amount of not less than the payment requirement on one or more obligations (e.g. a missed coupon payment). ISDA 2003 Term: Failure to Pay.
	 */
	FailureToPay FailureToPay
	/**
	 * A credit event. Corresponds to the failure by the Reference Entity to pay an expected principal amount or the payment of an actual principal amount that is less than the expected principal amount. ISDA 2003 Term: Failure to Pay Principal.
	 */
	FailureToPayPrincipal bool
	/**
	 * A credit event. Corresponds to the failure by the Reference Entity to pay an expected interest amount or the payment of an actual interest amount that is less than the expected interest amount. ISDA 2003 Term: Failure to Pay Interest.
	 */
	FailureToPayInterest bool
	/**
	 * A credit event. One or more of the obligations have become capable of being declared due and payable before they would otherwise have been due and payable as a result of, or on the basis of, the occurrence of a default, event of default or other similar condition or event other than failure to pay. ISDA 2003 Term: Obligation Default.
	 */
	ObligationDefault bool
	/**
	 * A credit event. One or more of the obligations have been declared due and payable before they would otherwise have been due and payable as a result of, or on the basis of, the occurrence of a default, event of default or other similar condition or event other than failure to pay (preferred by the market over Obligation Default, because more definitive and encompasses the definition of Obligation Default - this is more favorable to the Seller). Subject to the default requirement amount. ISDA 2003 Term: Obligation Acceleration.
	 */
	ObligationAcceleration bool
	/**
	 * A credit event. The reference entity, or a governmental authority, either refuses to recognise or challenges the validity of one or more obligations of the reference entity, or imposes a moratorium thereby postponing payments on one or more of the obligations of the reference entity. Subject to the default requirement amount. ISDA 2003 Term: Repudiation/Moratorium.
	 */
	RepudiationMoratorium bool
	/**
	 * A credit event. A restructuring is an event that materially impacts the reference entity's obligations, such as an interest rate reduction, principal reduction, deferral of interest or principal, change in priority ranking, or change in currency or composition of payment. ISDA 2003 Term: Restructuring.
	 */
	Restructuring Restructuring
	/**
	 * A credit event. A governmental intervention is an event resulting from an action by a governmental authority that materially impacts the reference entity's obligations, such as an interest rate reduction, principal reduction, deferral of interest or principal, change in priority ranking, or change in currency or composition of payment. ISDA 2014 Term: Governmental Intervention.
	 */
	GovernmentalIntervention bool
	/**
	 * A credit event. Results from the fact that the rating of the reference obligation is down-graded to a distressed rating level. From a usage standpoint, this credit event is typically not applicable in case of RMBS trades.
	 */
	DistressedRatingsDowngrade bool
	/**
	 * A credit event. Results from the fact that the underlier fails to make principal payments as expected.
	 */
	MaturityExtension bool
	/**
	 * A credit event. Results from the fact that the underlier writes down its outstanding principal amount.
	 */
	Writedown bool
	/**
	 * A credit event. Results from the fact that losses occur to the underlying instruments that do not result in reductions of the outstanding principal of the reference obligation.
	 */
	ImpliedWritedown bool
	/**
	 * In relation to certain credit events, serves as a threshold for Obligation Acceleration, Obligation Default, Repudiation/Moratorium and Restructuring. Market standard is USD 10,000,000 (JPY 1,000,000,000 for all Japanese Yen trades). This is applied on an aggregate or total basis across all Obligations of the Reference Entity. Used to prevent technical/operational errors from triggering credit events. ISDA 2003 Term: Default Requirement.
	 */
	DefaultRequirement Money
	/**
	 * A specified condition to settlement. An irrevocable written or verbal notice that describes a credit event that has occurred. The notice is sent from the notifying party (either the buyer or the seller) to the counterparty. It provides information relevant to determining that a credit event has occurred. This is typically accompanied by Publicly Available Information. ISDA 2003 Term: Credit Event Notice.
	 */
	CreditEventNotice CreditEventNotice
	Meta              MetaFields
}

/**
 * Specification of an index based on credit risk, typically composed using corporate debt instruments in a region or industry sector, e.g. the iTraxx indices.
 */
type CreditIndex struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * A description of the Index.
	 */
	Name FieldWithMeta
	/**
	 * The organisation that creates or maintains the Index.
	 */
	Provider LegalEntity
	/**
	 * The Asset Class of the Index.
	 */
	AssetClass AssetClassEnum
	/**
	 * A CDS index series identifier, e.g. 1, 2, 3 etc.
	 */
	IndexSeries int
	/**
	 * A CDS index series version identifier, e.g. 1, 2, 3 etc.
	 */
	IndexAnnexVersion int
	/**
	 * A CDS index series annex date.
	 */
	IndexAnnexDate time.Time
	/**
	 * A CDS index series annex source.
	 */
	IndexAnnexSource FieldWithMeta
	/**
	 * Excluded reference entity.
	 */
	ExcludedReferenceEntity []ReferenceInformation
	/**
	 * This element contains CDS tranche terms.
	 */
	Tranche Tranche
	/**
	 * Used to specify the Relevant Settled Entity Matrix when there are settled entities at the time of the trade.
	 */
	SettledEntityMatrix SettledEntityMatrix
	/**
	 * Index Factor is the index version factor or percent, expressed as an absolute decimal value between 0 and 1, that multiplied by the original notional amount yields the notional amount covered by the seller of protection.
	 */
	IndexFactor float64
	/**
	 * Seniority of debt instruments comprising the index.
	 */
	Seniority CreditSeniorityEnum
	Meta      MetaFields
}

/**
 * A class to represent the credit limit utilisation information.
 */
type CreditLimitInformation struct {
	LimitApplicable []LimitApplicableExtended
}

/**
 * Credit limit utilisation breakdown by executed trades and pending orders.
 */
type CreditLimitUtilisation struct {
	/**
	 * Credit limit utilisation attributable to executed trades.
	 */
	Executed CreditLimitUtilisationPosition
	/**
	 * Credit limit utilisation attributable to pending unexecuted orders.
	 */
	Pending CreditLimitUtilisationPosition
}

type CreditLimitUtilisationPosition struct {
	/**
	 * Credit limit utilisation attributable to short positions.
	 */
	ShortPosition float64
	/**
	 * Credit limit utilisation attributable to long positions.
	 */
	LongPosition float64
	/**
	 * Global credit limit utilisation amount, agnostic of long/short position direction.
	 */
	Global float64
}

/**
 * Represents a class to specify the credit notation as the combination of agency, notation, scale and debt type qualifications.
 */
type CreditNotation struct {
	/**
	 * Specifies The credit agency to which the other variables (notation, scale, debt type) refer to.
	 */
	Agency CreditRatingAgencyEnum
	/**
	 * Specifies The credit rating notation. As it varies among credit rating agencies, FpML doesn't specify a default scheme.
	 */
	Notation FieldWithMeta
	/**
	 * Specifies the credit rating scale, with a typical distinction between short term, long term. FpML doesn't specify a default scheme, which is hence not specified as an enumeration as part of the CDM.
	 */
	Scale FieldWithMeta
	/**
	 * Specifies the credit rating debt type (e.g. long term, high yield, deposits, ...) associated with the credit rating notation and scale.
	 */
	Debt CreditRatingDebt
	/**
	 * Assesses the potential direction of a long-term credit rating over the intermediate term, which is generally up to two years for investment grade and generally up to one year for speculative grade.
	 */
	Outlook CreditRatingOutlookEnum
	/**
	 * Indicates the potential direction of a short-term or long-term rating. It focuses on identifiable events and short-term trends that cause ratings to be placed under special surveillance.
	 */
	CreditWatch CreditRatingCreditWatchEnum
}

/**
 * Represents the credit rating notation higher level construct, which provides the ability to specify multiple rating notations.
 */
type CreditNotations struct {
	/**
	 * Specifies only one credit notation is determined.
	 */
	CreditNotation CreditNotation
	/**
	 * Specifies if several credit notations exist, alongside an 'any' or 'all' or all condition.
	 */
	CreditNotations MultipleCreditNotations
}

/**
 * Specifies the credit rating debt type(s) associated with the credit rating notation and scale. When several debt types are specified, they must be qualified through an 'any' or 'all'.
 */
type CreditRatingDebt struct {
	/**
	 * Specifies when there is only one debt type. FpML doesn't specify values in relation to the associated scheme, which is hence not specified as an enumeration as part of the CDM.
	 */
	DebtType FieldWithMeta
	/**
	 * Specifies if there are several debt types, alongside an 'any' or 'all' or all condition. As an example, Baa1 rating is required for any long term debt and deposit.
	 */
	DebtTypes MultipleDebtTypes
}

/**
 * The set of elections which specify a Credit Support Annex or Deed.
 */
type CreditSupportAgreementElections struct {
	/**
	 * The set of elections which specify an Initial Margin Credit Support Annex or Deed.
	 */
	CreditSupportAgreementInitialMarginElections CreditSupportAgreementInitialMarginElections
	/**
	 * The set of elections which specify a Variation Margin Credit Support Annex or Deed.
	 */
	CreditSupportAgreementVariationMarginElections CreditSupportAgreementVariationMarginElections
	/**
	 * The set of elections which specify a Legacy (1994 or 1995) Credit Support Annex or Deed.
	 */
	CreditSupportAgreementLegacyElections CreditSupportAgreementLegacyElections
}

/**
 * A logical container designed to hold a set of related data. In this case, designed to hold a set of clauses that are common amongst Legacy (1994/95), Variation Margin and Initial Margin Credit Support Documents.
 */
type CreditSupportAgreementElectionsBase struct {
	/**
	 * The base and eligible currency(ies) for the document as specified by the parties to the agreement.
	 */
	BaseAndEligibleCurrency BaseAndEligibleCurrency
	/**
	 * The set of elections that may overwrite the default Condition Precedent provision, and the set of provisions that are deemed Access Condition.
	 */
	ConditionsPrecedent ConditionsPrecedent
	/**
	 * The conditions under which the Security Provider can substitute posted collateral.
	 */
	Substitution Substitution
	/**
	 * The election terms under which a party disputes (i) the Calculation Agents calculation of a Delivery Amount or a Return Amount, or (ii) the Value of any Transfer of Eligible Credit Support or Posted Credit Support.
	 */
	DisputeResolution DisputeResolution
	/**
	 * The elections for the holding and using of posted collateral by the respective parties to the Credit Support Annex for Variation Margin.
	 */
	HoldingAndUsingPostedCollateral HoldingAndUsingPostedCollateral
	/**
	 * The Distributions and Interest Payment terms specified as part of the agreement.
	 */
	DistributionAndInterestPayment DistributionAndInterestPayment
	/**
	 * The Other Eligible Support elections associated with margin agreements.
	 */
	OtherEligibleAndPostedSupport OtherEligibleAndPostedSupport
	/**
	 * The optional specification of address where the demands, specifications and notices will be communicated to for each of the parties to the agreement.
	 */
	DemandsAndNotices DemandsAndNotices
	/**
	 * The specification Additional Representations that may be applicable to the agreement.
	 */
	AdditionalRepresentations AdditionalRepresentations
	/**
	 * Defines the dates agreed by the parties as the date of the Credit Support Annex or Credit Support Deed.
	 */
	MasterAgreementDatedAsOfDate MasterAgreementDatedAsOfDate
	/**
	 * A clause providing that final return collateral is to be provided without consideration of any Minimum Transfer Amount and collateral haircut.
	 */
	FinalReturns FinalReturns
}

/**
 * The set of elections which specify an Initial Margin Credit Support Annex or Deed.
 */
type CreditSupportAgreementInitialMarginElections struct {
	/**
	 * The base and eligible currency(ies) for the document as specified by the parties to the agreement.
	 */
	BaseAndEligibleCurrency BaseAndEligibleCurrency
	/**
	 * The set of elections that may overwrite the default Condition Precedent provision, and the set of provisions that are deemed Access Condition.
	 */
	ConditionsPrecedent ConditionsPrecedent
	/**
	 * The conditions under which the Security Provider can substitute posted collateral.
	 */
	Substitution Substitution
	/**
	 * The election terms under which a party disputes (i) the Calculation Agents calculation of a Delivery Amount or a Return Amount, or (ii) the Value of any Transfer of Eligible Credit Support or Posted Credit Support.
	 */
	DisputeResolution DisputeResolution
	/**
	 * The elections for the holding and using of posted collateral by the respective parties to the Credit Support Annex for Variation Margin.
	 */
	HoldingAndUsingPostedCollateral HoldingAndUsingPostedCollateral
	/**
	 * The Distributions and Interest Payment terms specified as part of the agreement.
	 */
	DistributionAndInterestPayment DistributionAndInterestPayment
	/**
	 * The Other Eligible Support elections associated with margin agreements.
	 */
	OtherEligibleAndPostedSupport OtherEligibleAndPostedSupport
	/**
	 * The optional specification of address where the demands, specifications and notices will be communicated to for each of the parties to the agreement.
	 */
	DemandsAndNotices DemandsAndNotices
	/**
	 * The specification Additional Representations that may be applicable to the agreement.
	 */
	AdditionalRepresentations AdditionalRepresentations
	/**
	 * Defines the dates agreed by the parties as the date of the Credit Support Annex or Credit Support Deed.
	 */
	MasterAgreementDatedAsOfDate MasterAgreementDatedAsOfDate
	/**
	 * A clause providing that final return collateral is to be provided without consideration of any Minimum Transfer Amount and collateral haircut.
	 */
	FinalReturns FinalReturns
	/**
	 * The Regime Table provision , which determines the regulatory regime(s) applicable to each of the parties to the agreement.
	 */
	Regime Regime
	/**
	 * The determination of whether the One Way Provisions are applicable (true) or not applicable (false).
	 */
	OneWayProvisions OneWayProvisions
	/**
	 * The specification of the ISDA SIMM Method for all Covered Transactions with respect to all Regimes.
	 */
	GeneralSimmElections GeneralSimmElections
	/**
	 * The qualification of whether cross-currency swaps need to be identified in the Confirmation so that the obligations to exchange principal be disregarded for the purpose of determining the Delivery Amount or Return Amount.
	 */
	IdentifiedCrossCurrencySwap bool
	/**
	 * The specification of methodologies to compute sensitivities specific to the agreement.
	 */
	SensitivityMethodologies SensitivityMethodologies
	/**
	 * The reference currency for the purpose of specifying the FX Haircut relating to a posting obligation, as being either the Termination Currency or an FX Designated Currency.
	 */
	FxHaircutCurrency FxHaircutCurrency
	/**
	 * The security providers posting obligations.
	 */
	PostingObligations PostingObligations
	/**
	 * The specification of Additional regimes for purposes of determining whether a Regulatory Event has occurred.
	 */
	SubstitutedRegime []SubstitutedRegime
	/**
	 * The specification of transactions covered by the terms of the agreement.
	 */
	CoveredTransactions CoveredTransactions
	/**
	 * The bespoke provisions that might be specified by the parties to the agreement to specify the rights of Security Taker and/or Security Provider when an Early Termination or Access Condition event has occurred.
	 */
	RightsEvents RightsEvents
	/**
	 * The Custodian and Segregated Account details in respect of each party to the agreement.
	 */
	CustodyArrangements CustodyArrangements
	/**
	 * The bespoke definition of other agreement terms as specified by the parties to the agreement.
	 */
	OtherAgreements OtherAgreements
	/**
	 * The bespoke provision that might be specified by the parties to the agreement applicable to Termination Currency.  Unless specified the definition of Termination Currency has the meaning specified in the Schedule to the ISDA Master Agreement.
	 */
	TerminationCurrencyAmendment TerminationCurrencyAmendment
	/**
	 * The bespoke provision that might be specified by the parties to the agreement applicable to Minimum Transfer Amount.  Unless specified the definition of Minimum Transfer Amount in any Other Regulatory CSA has the meaning specified in such Other Regulatory CSA.
	 */
	MinimumTransferAmountAmendment MinimumTransferAmountAmendment
	/**
	 * The bespoke provision that might be specified by the parties to the agreement applicable to Interpretations.
	 */
	InterpretationTerms string
	/**
	 * The Process Agent that might be appointed by the parties to the agreement.
	 */
	ProcessAgent ProcessAgent
	/**
	 * The election for the Valuation of Appropriate Collateral.
	 */
	AppropriatedCollateralValuation AppropriatedCollateralValuation
	/**
	 * The jurisdiction specific terms relevant to the agreement.
	 */
	JurisdictionRelatedTerms JurisdictionRelatedTerms
	/**
	 * Any additional amendments that might be specified by the parties to the agreement.
	 */
	AdditionalAmendments string
	/**
	 * Any additional terms that might be specified applicable.
	 */
	AdditionalBespokeTerms string
	/**
	 * The qualification of whether Trust Scheme Addendum is applicable (True) or not applicable (False).
	 */
	TrustSchemeAddendum bool
	/**
	 * Details of language relating to Client Asset and Money Rules (CASS).
	 */
	ClientAssetSourcebookAdditionalLanguage string
	/**
	 * The Credit Support Obligations applicable to the agreement.
	 */
	CreditSupportObligations CreditSupportObligationsInitialMargin
	/**
	 * The set of elections for determining Valuation and Timing terms specific to the agreement.
	 */
	CalculationAndTiming CalculationAndTimingInitialMargin
}

/**
 * The set of elections which specify a Legacy (1994 or 1995) Credit Support Annex or Deed.
 */
type CreditSupportAgreementLegacyElections struct {
	/**
	 * The base and eligible currency(ies) for the document as specified by the parties to the agreement.
	 */
	BaseAndEligibleCurrency BaseAndEligibleCurrency
	/**
	 * The set of elections that may overwrite the default Condition Precedent provision, and the set of provisions that are deemed Access Condition.
	 */
	ConditionsPrecedent ConditionsPrecedent
	/**
	 * The conditions under which the Security Provider can substitute posted collateral.
	 */
	Substitution Substitution
	/**
	 * The election terms under which a party disputes (i) the Calculation Agents calculation of a Delivery Amount or a Return Amount, or (ii) the Value of any Transfer of Eligible Credit Support or Posted Credit Support.
	 */
	DisputeResolution DisputeResolution
	/**
	 * The elections for the holding and using of posted collateral by the respective parties to the Credit Support Annex for Variation Margin.
	 */
	HoldingAndUsingPostedCollateral HoldingAndUsingPostedCollateral
	/**
	 * The Distributions and Interest Payment terms specified as part of the agreement.
	 */
	DistributionAndInterestPayment DistributionAndInterestPayment
	/**
	 * The Other Eligible Support elections associated with margin agreements.
	 */
	OtherEligibleAndPostedSupport OtherEligibleAndPostedSupport
	/**
	 * The optional specification of address where the demands, specifications and notices will be communicated to for each of the parties to the agreement.
	 */
	DemandsAndNotices DemandsAndNotices
	/**
	 * The specification Additional Representations that may be applicable to the agreement.
	 */
	AdditionalRepresentations AdditionalRepresentations
	/**
	 * Defines the dates agreed by the parties as the date of the Credit Support Annex or Credit Support Deed.
	 */
	MasterAgreementDatedAsOfDate MasterAgreementDatedAsOfDate
	/**
	 * A clause providing that final return collateral is to be provided without consideration of any Minimum Transfer Amount and collateral haircut.
	 */
	FinalReturns FinalReturns
	/**
	 * The bespoke exchange date terms that might be specified by the parties to the agreement.
	 */
	ExchangeDate string
	/**
	 * The optional specification of address for transfer as specified by the respective parties to the agreement.
	 */
	AddressesForTransfer AddressesForTransfer
	/**
	 * The party to whom obligations under the Master Agreement are owed and secured by the credit support arrangements. Applicable to 1994 NY CSA, 1995 English Law CSD and 2016 New York Law VM CSA.
	 */
	SecurityInterestForObligations SecurityInterestForObligations
	/**
	 * Where only a single  party is explicitly designated to post collateral, the party which is specified as the sole poster of collateral.
	 */
	SinglePostingParty SinglePostingParty
	/**
	 * The Credit Support Obligations applicable to the agreement.
	 */
	CreditSupportObligations CreditSupportObligationsLegacy
	/**
	 * The set of elections for determining Valuation and Timing terms specific to the agreement.
	 */
	CalculationAndTiming CalculationAndTimingLegacy
}

/**
 * The set of elections which specify a Variation Margin Credit Support Annex or Deed.
 */
type CreditSupportAgreementVariationMarginElections struct {
	/**
	 * The base and eligible currency(ies) for the document as specified by the parties to the agreement.
	 */
	BaseAndEligibleCurrency BaseAndEligibleCurrency
	/**
	 * The set of elections that may overwrite the default Condition Precedent provision, and the set of provisions that are deemed Access Condition.
	 */
	ConditionsPrecedent ConditionsPrecedent
	/**
	 * The conditions under which the Security Provider can substitute posted collateral.
	 */
	Substitution Substitution
	/**
	 * The election terms under which a party disputes (i) the Calculation Agents calculation of a Delivery Amount or a Return Amount, or (ii) the Value of any Transfer of Eligible Credit Support or Posted Credit Support.
	 */
	DisputeResolution DisputeResolution
	/**
	 * The elections for the holding and using of posted collateral by the respective parties to the Credit Support Annex for Variation Margin.
	 */
	HoldingAndUsingPostedCollateral HoldingAndUsingPostedCollateral
	/**
	 * The Distributions and Interest Payment terms specified as part of the agreement.
	 */
	DistributionAndInterestPayment DistributionAndInterestPayment
	/**
	 * The Other Eligible Support elections associated with margin agreements.
	 */
	OtherEligibleAndPostedSupport OtherEligibleAndPostedSupport
	/**
	 * The optional specification of address where the demands, specifications and notices will be communicated to for each of the parties to the agreement.
	 */
	DemandsAndNotices DemandsAndNotices
	/**
	 * The specification Additional Representations that may be applicable to the agreement.
	 */
	AdditionalRepresentations AdditionalRepresentations
	/**
	 * Defines the dates agreed by the parties as the date of the Credit Support Annex or Credit Support Deed.
	 */
	MasterAgreementDatedAsOfDate MasterAgreementDatedAsOfDate
	/**
	 * A clause providing that final return collateral is to be provided without consideration of any Minimum Transfer Amount and collateral haircut.
	 */
	FinalReturns FinalReturns
	/**
	 * The reference currency for the purpose of specifying the FX Haircut relating to a posting obligation, as being either the Termination Currency or an FX Designated Currency.
	 */
	FxHaircutCurrency FxHaircutCurrency
	/**
	 * The bespoke exchange date terms that might be specified by the parties to the agreement.
	 */
	ExchangeDate string
	/**
	 * The specification of transactions covered by the terms of the agreement.
	 */
	CoveredTransactions CoveredTransactions
	/**
	 * The specification of whether the standard Credit Support Offset provisions are applicable (true) or not applicable (false).
	 */
	CreditSupportOffsets bool
	/**
	 * The Custodian and Segregated Account details in respect of each party to the agreement.
	 */
	CustodyArrangements CustodyArrangements
	/**
	 * The optional specification of address for transfer as specified by the respective parties to the agreement.
	 */
	AddressesForTransfer AddressesForTransfer
	/**
	 * The bespoke definition of other agreement terms as specified by the parties to the agreement.
	 */
	OtherAgreements OtherAgreements
	/**
	 * The party to whom obligations under the Master Agreement are owed and secured by the credit support arrangements. Applicable to 1994 NY CSA, 1995 English Law CSD and 2016 New York Law VM CSA.
	 */
	SecurityInterestForObligations SecurityInterestForObligations
	/**
	 * The Credit Support Obligations applicable to the agreement.
	 */
	CreditSupportObligations CreditSupportObligationsVariationMargin
	/**
	 * The set of elections for determining Valuation and Timing terms specific to the agreement.
	 */
	CalculationAndTiming CalculationAndTimingVariationMargin
}

/**
 * The total amount one counterparty must deliver to the other at any time: the combination of the Exposure to that party and the net Independent Amounts it must post, minus any agreed Threshold.
 */
type CreditSupportAmount struct {
	/**
	 * Specifies whether the meaning of credit support amount follows the pre-print definition or deviates from the standard (either based on a rating or some other methodology).
	 */
	CreditSupportAmount CreditSupportAmountEnum
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * Identification of party specific Credit Support Documents applicable to the document.
 */
type CreditSupportDocument struct {
	/**
	 * The party election of Credit Support Document(s), if any.
	 */
	CreditSupportDocumentElection []CreditSupportDocumentElection
}

/**
 * The party election of Credit Support Provider(s), if any.
 */
type CreditSupportDocumentElection struct {
	/**
	 * The elective party.
	 */
	Party Party
	/**
	 * Specification of the Credit Support Document terms.
	 */
	CreditSupportDocumentTerms CreditSupportDocumentTermsEnum
	/**
	 * If a Credit Support Document is specified but not yet in CDM representation, it can still be captured. This argument should be used if the creditSupportDocument argument cannot be used due to not being able to represent the document fully in CDM format.
	 */
	CreditSupportDocumentTypes []LegalAgreementIdentification
	/**
	 * The specified Credit Support Document(s), if any.
	 */
	CreditSupportDocument []LegalAgreement
	/**
	 * Specification of a document when not captured under RelatedAgreement.
	 */
	BespokeCreditSuppportDocument string
}

/**
 * A logical container designed to hold a set of related data. In this case, designed to hold a set of clauses that are common amongst Legacy (1994/95), Variation Margin and Initial Margin Credit Support Documents in regards to Credit Support Obligations.
 */
type CreditSupportObligationsBase struct {
	/**
	 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be transferred to it as a Delivery Amount.
	 */
	DeliveryAmount CSADeliveryAmount
	/**
	 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be recalled to it as a Return Amount.
	 */
	ReturnAmount CSAReturnAmount
	/**
	 * Represents the threshold specified in the agreement below which collateral need not be delivered, and represents the smallest incremental value of exposure that must be reached before the parties are obliged to transfer collateral.
	 */
	MinimumTransferAmount MinimumTransferAmount
	/**
	 * Specification of transfer / settlement timing for cash and securities collateral assuming that the notice is provided before the Notification Time (and with a 1LBD lag introduced if the notice is delivered after the Notification Time), and that securities allow for a further Local Business Day for transfer / settlement.
	 */
	CollateralTransferTiming CollateralTransferTiming
	/**
	 * The time by which the transfer of collateral must take place when different from the Regular Settlement Day as a result of parties' election.
	 */
	BespokeTransferTiming BespokeTransferTiming
}

/**
 * Specification of the Credit Support Obligations applicable to the Initial Margin Credit Support Annex.
 */
type CreditSupportObligationsCollateralTransferAgreement struct {
	/**
	 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be transferred to it as a Delivery Amount.
	 */
	DeliveryAmount CSADeliveryAmount
	/**
	 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be recalled to it as a Return Amount.
	 */
	ReturnAmount CSAReturnAmount
	/**
	 * Represents the threshold specified in the agreement below which collateral need not be delivered, and represents the smallest incremental value of exposure that must be reached before the parties are obliged to transfer collateral.
	 */
	MinimumTransferAmount MinimumTransferAmount
	/**
	 * Specification of transfer / settlement timing for cash and securities collateral assuming that the notice is provided before the Notification Time (and with a 1LBD lag introduced if the notice is delivered after the Notification Time), and that securities allow for a further Local Business Day for transfer / settlement.
	 */
	CollateralTransferTiming CollateralTransferTiming
	/**
	 * The time by which the transfer of collateral must take place when different from the Regular Settlement Day as a result of parties' election.
	 */
	BespokeTransferTiming BespokeTransferTiming
	/**
	 * The selection of Margin Approach applicable to the agreement.
	 */
	MarginApproach MarginApproach
	/**
	 * The Other Eligible Support election. If not specified, this election is deemed as not applicable.
	 */
	OtherEligibleSupportIM string
	/**
	 * Represents the amount of unsecured risk that a party is willing to tolerate without holding any posted collateral provided by the other party.
	 */
	Threshold Threshold
	/**
	 * The total amount one counterparty must deliver to the other at any time: the combination of the Exposure to that party and the net Independent Amounts it must post, minus any agreed Threshold.
	 */
	CreditSupportAmount CreditSupportAmount
	/**
	 * The rounding methodology applicable to the Delivery Amount and the Return Amount in terms of nearest integral multiple of Base Currency units.
	 */
	Rounding CollateralRounding
}

/**
 * Specification of the Credit Support Obligations applicable to the Initial Margin Credit Support Annex.
 */
type CreditSupportObligationsInitialMargin struct {
	/**
	 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be transferred to it as a Delivery Amount.
	 */
	DeliveryAmount CSADeliveryAmount
	/**
	 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be recalled to it as a Return Amount.
	 */
	ReturnAmount CSAReturnAmount
	/**
	 * Represents the threshold specified in the agreement below which collateral need not be delivered, and represents the smallest incremental value of exposure that must be reached before the parties are obliged to transfer collateral.
	 */
	MinimumTransferAmount MinimumTransferAmount
	/**
	 * Specification of transfer / settlement timing for cash and securities collateral assuming that the notice is provided before the Notification Time (and with a 1LBD lag introduced if the notice is delivered after the Notification Time), and that securities allow for a further Local Business Day for transfer / settlement.
	 */
	CollateralTransferTiming CollateralTransferTiming
	/**
	 * The time by which the transfer of collateral must take place when different from the Regular Settlement Day as a result of parties' election.
	 */
	BespokeTransferTiming BespokeTransferTiming
	/**
	 * The selection of Margin Approach applicable to the agreement.
	 */
	MarginApproach MarginApproach
	/**
	 * The Other Eligible Support election. If not specified, this election is deemed as not applicable.
	 */
	OtherEligibleSupportIM string
	/**
	 * The rounding methodology applicable to the Delivery Amount and the Return Amount in terms of nearest integral multiple of Base Currency units.
	 */
	Rounding CollateralRounding
	/**
	 * Represents the amount of unsecured risk that a party is willing to tolerate without holding any posted collateral provided by the other party.
	 */
	Threshold Threshold
	/**
	 * The total amount one counterparty must deliver to the other at any time: the combination of the Exposure to that party and the net Independent Amounts it must post, minus any agreed Threshold.
	 */
	CreditSupportAmount CreditSupportAmount
}

/**
 * Specification of the Credit Support Obligations applicable to the Legacy (1994/95) Credit Support Annex.
 */
type CreditSupportObligationsLegacy struct {
	/**
	 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be transferred to it as a Delivery Amount.
	 */
	DeliveryAmount CSADeliveryAmount
	/**
	 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be recalled to it as a Return Amount.
	 */
	ReturnAmount CSAReturnAmount
	/**
	 * Represents the threshold specified in the agreement below which collateral need not be delivered, and represents the smallest incremental value of exposure that must be reached before the parties are obliged to transfer collateral.
	 */
	MinimumTransferAmount MinimumTransferAmount
	/**
	 * Specification of transfer / settlement timing for cash and securities collateral assuming that the notice is provided before the Notification Time (and with a 1LBD lag introduced if the notice is delivered after the Notification Time), and that securities allow for a further Local Business Day for transfer / settlement.
	 */
	CollateralTransferTiming CollateralTransferTiming
	/**
	 * The time by which the transfer of collateral must take place when different from the Regular Settlement Day as a result of parties' election.
	 */
	BespokeTransferTiming BespokeTransferTiming
	/**
	 * Represents the amount of unsecured risk that a party is willing to tolerate without holding any posted collateral provided by the other party.
	 */
	Threshold Threshold
	/**
	 * The rounding methodology applicable to the Delivery Amount and the Return Amount in terms of nearest integral multiple of Base Currency units.
	 */
	Rounding CollateralRounding
	/**
	 * Specifies the additional amount of collateral that is required to be delivered by a party irrespective of the mark-to-market exposure under the related Transactions. It may take the form of a fixed amount or may be determined using ratings.
	 */
	IndependentAmount LegacyIndependentAmount
	/**
	 * The total amount one counterparty must deliver to the other at any time: the combination of the Exposure to that party and the net Independent Amounts it must post, minus any agreed Threshold.
	 */
	CreditSupportAmount CreditSupportAmount
	/**
	 * Specification of the Eligible Collateral or Credit Support applicable to the Agreement.
	 */
	EligibleCreditSupport EligibleCreditSupport
}

/**
 * Specification of the Credit Support Obligations applicable to the Variation Margin Credit Support Annex.
 */
type CreditSupportObligationsVariationMargin struct {
	/**
	 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be transferred to it as a Delivery Amount.
	 */
	DeliveryAmount CSADeliveryAmount
	/**
	 * In respect of a Valuation Date, the amount of collateral which a party is entitled to demand to be recalled to it as a Return Amount.
	 */
	ReturnAmount CSAReturnAmount
	/**
	 * Represents the threshold specified in the agreement below which collateral need not be delivered, and represents the smallest incremental value of exposure that must be reached before the parties are obliged to transfer collateral.
	 */
	MinimumTransferAmount MinimumTransferAmount
	/**
	 * Specification of transfer / settlement timing for cash and securities collateral assuming that the notice is provided before the Notification Time (and with a 1LBD lag introduced if the notice is delivered after the Notification Time), and that securities allow for a further Local Business Day for transfer / settlement.
	 */
	CollateralTransferTiming CollateralTransferTiming
	/**
	 * The time by which the transfer of collateral must take place when different from the Regular Settlement Day as a result of parties' election.
	 */
	BespokeTransferTiming BespokeTransferTiming
	/**
	 * The parties to which the provisions of the Credit Support Annex for Variation Margin will apply to.
	 */
	IneligibleCreditSupport IneligibleCreditSupport
	/**
	 * The additional currencies that are specified as Major Currency for the purpose of applying the FX Haircut Percentage.
	 */
	MajorCurrency []FieldWithMeta
	/**
	 * The alternative definition for FX haircut percentage that applies to each party and item of Eligible Collateral unless this item is denominated in a Major Currency or in the Base Currency.
	 */
	FxHaircut string
	/**
	 * The rounding methodology applicable to the Delivery Amount and the Return Amount in terms of nearest integral multiple of Base Currency units.
	 */
	Rounding CollateralRounding
	/**
	 * Specification of the Eligible Collateral or Credit Support applicable to the Agreement.
	 */
	EligibleCreditSupport EligibleCreditSupport
}

/**
 * Identification of party specific Credit Support Providers applicable to the document.
 */
type CreditSupportProvider struct {
	/**
	 * The party election of Credit Support Provider(s), if any.
	 */
	CreditSupportProviderElection []CreditSupportProviderElection
}

/**
 * The party election of Credit Support Provider(s), if any.
 */
type CreditSupportProviderElection struct {
	/**
	 * The elective party.
	 */
	Party Party
	/**
	 * Specification of the Credit Support Provider terms.
	 */
	CreditSupportProviderTerms CreditSupportProviderTermsEnum
	/**
	 * The specified Credit Support Provider(s), if any.
	 */
	CreditSupportProvider []LegalEntity
	/**
	 * ...
	 */
	BespokeCreditSuppportProvider string
}

type Curve struct {
	InterestRateCurve InterestRateCurve
	CommodityCurve    FieldWithMeta
}

/**
 * A class to specify the custodian and custody account details for each party to the agreement.
 */
type Custodian struct {
	/**
	 * The party specific elections.
	 */
	PartyElection []CustodianElection
}

/**
 * A class to specify the custodian and custody account details for each party to the agreement.
 */
type CustodianElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The custody agent. While CreateiQ only specifies the custodian's name, specifying the legal entity as part of the CDM is deemed more appropriate, while this will still provide the ability to accommodate situations where only the entity name is available, as the entityId attribute is optional as part of the LegalEntity class.
	 */
	Custodian LegalEntity
	/**
	 * The identification of the segregated cash account for the purpose of holding cash collateral.
	 */
	SegregatedCashAccount Account
	/**
	 * The identification of the segregated security account for the purpose of holding security collateral.
	 */
	SegregatedSecurityAccount Account
	/**
	 * Details how the Custodian is determined.
	 */
	LegacyVMCustodian LegacyVMCustodianEnum
	/**
	 * The identity of the relevant custodian where Named Entity is elected.
	 */
	LegacyNamedEntity LegalEntity
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * A class to specify the Custodian Event.
 */
type CustodianEvent struct {
	/**
	 * The qualification as to whether the Custodian Event (English Law & New York Law ISDA CSA) or the Collateral Manager Event (Japanese Law ISDA CSA) is applicable.
	 */
	IsApplicable bool
	/**
	 * The qualification of the Custodian Event (English Law & New York Law ISDA CSA) or Collateral Manager Event (Japanese Law ISDA CSA) End Date.
	 */
	EndDate CustodianEventEndDate
}

/**
 * A class to specify the Custodian Event.
 */
type CustodianEventEndDate struct {
	/**
	 * Election to specify the number of days after the occurrence of the Custodian Event/Collateral Management Event for the purpose of qualifying the CE/CME End Date.
	 */
	DaysAfterCustodianEvent CustomisableOffset
	/**
	 * Election to specify the number of days prior to the termination of the Control Agreement /Collateral Management Event for the purpose of qualifying the CE/CME End Date, in the case where advance notice is given.
	 */
	ReleaseDate CustomisableOffset
	/**
	 * The parties' election to specify the number of days prior to the end of the safekeeping period (Clearstream CTA) purpose of qualifying the CE/CME End Date, in the case where advance notice is given.
	 */
	SafekeepingPeriodExpiry CustomisableOffset
	/**
	 * The parties' election to specify the number of days one party has effectively provided the Timely Statement to the other party.
	 */
	DateOfTimelyStatement CustomisableOffset
}

/**
 * A class to specify the Custodian Risk elections specific to a Credit Support Agreement.
 */
type CustodianRisk struct {
	/**
	 * The party specific elections.
	 */
	PartyElection []CustodianRiskElection
}

/**
 * A class to specify the Custodian Risk.
 */
type CustodianRiskElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The qualification as to whether the risk is deemed as Specified.
	 */
	IsSpecified bool
	/**
	 * The Custodian Risk or Collateral Manager Risk qualification.
	 */
	Qualification string
}

/**
 * A class to specify the requirements applicable to the custodian with respect to the holding of posted collateral.
 */
type CustodianTerms struct {
	/**
	 * The minimal level of assets requirement with respect to the custody agent.
	 */
	MinimumAssets Money
	/**
	 * The minimal rating requirement with respect to the custody agent.
	 */
	MinimumRating CreditNotation
	/**
	 * Election to specify the initial custodian.
	 */
	InitialDesignation LegalEntity
}

/**
 * Describes how posted collateral is held, maintained, and safeguarded by a custodian or third-party agent, in accordance with the terms of the governing agreement.
 */
type CustodyArrangements struct {
	/**
	 * Unless specified as True, the Control Agreement is not a Credit Support Document under the agreement with respect to a party.
	 */
	IsCreditSupportDocument bool
	/**
	 * Control Agreement language is specified when True.
	 */
	HasControlAgreementLanguage bool
	/**
	 * Other Custom Provisions of Custody Arrangements.
	 */
	OtherProvisions string
	/**
	 * The elections specific to Collateral Access Breach language.
	 */
	CollateralAccessBreach CollateralAccessBreach
	/**
	 * The qualification of the Custodian Risk.
	 */
	CustodianRisk CustodianRisk
	/**
	 * The party-specific election with respect to the control agreement.
	 */
	ControlAgreement ControlAgreement
	/**
	 * Specifies if and until when a Custodian Event clause applies.
	 */
	CustodianEvent CustodianEvent
	/**
	 * The custodian and segregated account details for each party to the agreement.
	 */
	Custodian                     Custodian
	CollateralManagementAgreement CollateralManagementAgreement
}

/**
 * A class to specify an offset either as a normalized [multiplier, period, dayType] or as a custom provision of type string.
 */
type CustomisableOffset struct {
	Offset          Offset
	CustomProvision string
}

/**
 * In its initial iteration, this class is meant to support the DTCC TIW workflow information.
 */
type CustomisedWorkflow struct {
	/**
	 * In this initial iteration, this corresponds to the DTCC TIW element name.
	 */
	ItemName string
	/**
	 * In this initial iteration, this corresponds to the DTCC value.
	 */
	ItemValue string
}

/**
 * List of dates.
 */
type DateList struct {
	Date []time.Time
}

/**
 * A class defining a contiguous series of calendar dates. The date range is defined as all the dates between and including the start and the end date. The start date must fall on or before the end date.
 */
type DateRange struct {
	/**
	 * The first date of a date range.
	 */
	StartDate time.Time
	/**
	 * The last date of a date range.
	 */
	EndDate time.Time
}

/**
 * A data to:  provide the ability to point to multiple payment nodes in the document through the unbounded paymentDatesReference.
 */
type DateRelativeToCalculationPeriodDates struct {
	/**
	 * A set of href pointers to calculation period dates defined somewhere else in the document.
	 */
	CalculationPeriodDatesReference []ReferenceWithMeta
}

/**
 * A data to:  provide the ability to point to multiple payment nodes in the document through the unbounded paymentDatesReference.
 */
type DateRelativeToPaymentDates struct {
	/**
	 * A set of href pointers to payment dates defined somewhere else in the document.
	 */
	PaymentDatesReference []ReferenceWithMeta
}

/**
 * A data to:  provide the ability to point to multiple payment nodes in the document through the unbounded paymentDatesReference.
 */
type DateRelativeToValuationDates struct {
	/**
	 * A set of href pointers to valuation period dates defined somewhere else in the document.
	 */
	ValuationDatesReference []ReferenceWithMeta
}

/**
 * List of dateTimes.
 */
type DateTimeList struct {
	/**
	 * The CDM specifies that the zoned date time is to be expressed in accordance with ISO 8601, either as UTC as an offset to UTC.
	 */
	DateTime []time.Time
}

/**
 * Defines a date and value pair. This definition is used for varying rate or amount schedules, e.g. a notional amortisation or a step-up coupon schedule.
 */
type DatedValue struct {
	/**
	 * The date on which the associated step value becomes effective. This day may be subject to adjustment in accordance with a business day convention.
	 */
	Date time.Time
	/**
	 * The rate of amount which becomes effective on the associated step date. A rate of 5% would be represented as 0.05.
	 */
	Value float64
	Meta  MetaFields
}

/**
 * Specifies selected economics of a debt instrument.
 */
type DebtEconomics struct {
	/**
	 * Specifies the order of repayment in the event of a sale or bankruptcy of the issuer or a related party (eg guarantor).
	 */
	Seniority DebtSeniorityEnum
	/**
	 * Specifies the general rule for periodic interest rate payment.
	 */
	Interest DebtInterestEnum
	/**
	 * Specifies the general rule for repayment of principal.
	 */
	Principal DebtPrincipalEnum
	/**
	 * Specifies the type of secured debt.
	 */
	Secured SecuredDebt
}

/**
 * Specifies the type of debt instrument.
 */
type DebtType struct {
	/**
	 * Specifies the characteristics of a debt instrument.
	 */
	DebtClass DebtClassEnum
	/**
	 * Specifies selected financial terms of a debt instrument.
	 */
	DebtEconomics []DebtEconomics
}

/**
 * A class to specify all the ISDA terms relevant to defining the deliverable obligations.
 */
type DeliverableObligations struct {
	/**
	 * Indicates whether accrued interest is included (true) or not (false). For cash settlement this specifies whether quotations should be obtained inclusive or not of accrued interest. For physical settlement this specifies whether the buyer should deliver the obligation with an outstanding principal balance that includes or excludes accrued interest. ISDA 2003 Term: Include/Exclude Accrued Interest.
	 */
	AccruedInterest bool
	/**
	 * Used in both obligations and deliverable obligations to represent a class or type of securities which apply. ISDA 2003 Term: Obligation Category/Deliverable Obligation Category.
	 */
	Category ObligationCategoryEnum
	/**
	 * An obligation and deliverable obligation characteristic. An obligation that ranks at least equal with the most senior Reference Obligation in priority of payment or, if no Reference Obligation is specified in the related Confirmation, the obligations of the Reference Entity that are senior. ISDA 2003 Term: Not Subordinated.
	 */
	NotSubordinated bool
	/**
	 * An obligation and deliverable obligation characteristic. The currency or currencies in which an obligation or deliverable obligation must be payable. ISDA 2003 Term: Specified Currency.
	 */
	SpecifiedCurrency SpecifiedCurrency
	/**
	 * An obligation and deliverable obligation characteristic. Any obligation that is not primarily (majority) owed to a Sovereign or Supranational Organisation. ISDA 2003 Term: Not Sovereign Lender.
	 */
	NotSovereignLender bool
	/**
	 * An obligation and deliverable obligation characteristic. Any obligation that is payable in any currency other than the domestic currency. Domestic currency is either the currency so specified or, if no currency is specified, the currency of (a) the reference entity, if the reference entity is a sovereign, or (b) the jurisdiction in which the relevant reference entity is organised, if the reference entity is not a sovereign. ISDA 2003 Term: Not Domestic Currency.
	 */
	NotDomesticCurrency NotDomesticCurrency
	/**
	 * An obligation and deliverable obligation characteristic. If the reference entity is a Sovereign, this means any obligation that is not subject to the laws of the reference entity. If the reference entity is not a sovereign, this means any obligation that is not subject to the laws of the jurisdiction of the reference entity. ISDA 2003 Term: Not Domestic Law.
	 */
	NotDomesticLaw bool
	/**
	 * An obligation and deliverable obligation characteristic. Indicates whether or not the obligation is quoted, listed or ordinarily purchased and sold on an exchange. ISDA 2003 Term: Listed.
	 */
	Listed bool
	/**
	 * A deliverable obligation characteristic. In essence Not Contingent means the repayment of principal cannot be dependant on a formula/index, i.e. to prevent the risk of being delivered an instrument that may never pay any element of principal, and to ensure that the obligation is interest bearing (on a regular schedule). ISDA 2003 Term: Not Contingent.
	 */
	NotContingent bool
	/**
	 * An obligation and deliverable obligation characteristic. Any obligation other than an obligation that was intended to be offered for sale primarily in the domestic market of the relevant Reference Entity. This specifies that the obligation must be an internationally recognised bond. ISDA 2003 Term: Not Domestic Issuance.
	 */
	NotDomesticIssuance bool
	/**
	 * A deliverable obligation characteristic. A loan that is freely assignable to a bank or financial institution without the consent of the Reference Entity or the guarantor, if any, of the loan (or the consent of the applicable borrower if a Reference Entity is guaranteeing the loan) or any agent. ISDA 2003 Term: Assignable Loan.
	 */
	AssignableLoan PCDeliverableObligationCharac
	/**
	 * A deliverable obligation characteristic. A loan that is capable of being assigned with the consent of the Reference Entity or the guarantor, if any, of the loan or any agent. ISDA 2003 Term: Consent Required Loan.
	 */
	ConsentRequiredLoan PCDeliverableObligationCharac
	/**
	 * A deliverable obligation characteristic. A loan with a participation agreement whereby the buyer is capable of creating, or procuring the creation of, a contractual right in favour of the seller that provides the seller with recourse to the participation seller for a specified share in any payments due under the relevant loan which are received by the participation seller. ISDA 2003 Term: Direct Loan Participation.
	 */
	DirectLoanParticipation LoanParticipation
	/**
	 * A deliverable obligation characteristic. An obligation that is transferable to institutional investors without any contractual, statutory or regulatory restrictions. ISDA 2003 Term: Transferable.
	 */
	Transferable bool
	/**
	 * A deliverable obligation characteristic. An obligation that has a remaining maturity from the Physical Settlement Date of not greater than the period specified. ISDA 2003 Term: Maximum Maturity.
	 */
	MaximumMaturity Period
	/**
	 * A deliverable obligation characteristic. An obligation at time of default is due to mature and due to be repaid, or as a result of downgrade/bankruptcy is due to be repaid as a result of an acceleration clause. ISDA 2003 Term: Accelerated or Matured.
	 */
	AcceleratedOrMatured bool
	/**
	 * A deliverable obligation characteristic. Any obligation that is not a bearer instrument. This applies to Bonds only and is meant to avoid tax, fraud and security/delivery provisions that can potentially be associated with Bearer Bonds. ISDA 2003 Term: Not Bearer.
	 */
	NotBearer bool
	/**
	 * An obligation and deliverable obligation characteristic. Defined in the ISDA published additional provisions for U.S. Municipal as Reference Entity. ISDA 2003 Term: Full Faith and Credit Obligation Liability.
	 */
	FullFaithAndCreditObLiability bool
	/**
	 * An obligation and deliverable obligation characteristic. Defined in the ISDA published additional provisions for U.S. Municipal as Reference Entity. ISDA 2003 Term: General Fund Obligation Liability.
	 */
	GeneralFundObligationLiability bool
	/**
	 * An obligation and deliverable obligation characteristic. Defined in the ISDA published additional provisions for U.S. Municipal as Reference Entity. ISDA 2003 Term: Revenue Obligation Liability.
	 */
	RevenueObligationLiability bool
	/**
	 * ISDA 1999 Term: Indirect Loan Participation. NOTE: Only applicable as a deliverable obligation under ISDA Credit 1999.
	 */
	IndirectLoanParticipation LoanParticipation
	/**
	 * A free format string to specify any excluded obligations or deliverable obligations, as the case may be, of the reference entity or excluded types of obligations or deliverable obligations. ISDA 2003 Term: Excluded Obligations/Excluded Deliverable Obligations.
	 */
	Excluded string
	/**
	 * This element is used to specify any other obligations of a reference entity in both obligations and deliverable obligations. The obligations can be specified free-form. ISDA 2003 Term: Other Obligations of a Reference Entity.
	 */
	OthReferenceEntityObligations string
}

/**
 * A class to specify the application of Interest Amount with respect the Delivery Amount.
 */
type DeliveryAmount struct {
	/**
	 * The standard election as specified by an enumeration.
	 */
	StandardElection DeliveryAmountElectionEnum
	/**
	 * The custom election that might be specified by the parties to the agreement.
	 */
	CustomElection string
}

/**
 * Specifies a specific date or the parameters for identifying the relevant contract date when the commodity reference price is a futures contract.
 */
type DeliveryDateParameters struct {
	/**
	 * Provides a container for the parametric representation that specifies which nearby contract date would be used as a refrence for a price.
	 */
	DeliveryNearby Offset
	/**
	 * Specifies the specific contract date for the contract that should be referenced for a price.
	 */
	DeliveryDate AdjustableDate
	/**
	 * Specifies, for a Commodity Transaction that references a listed future, the day on which the specified future will roll to the next nearby month prior to the expiration of the referenced future. If the future will not roll at all - i.e. the price will be taken from the expiring contract, 0 days should be specified here. If the future will roll to the next nearby on the last trading day - i.e. the price will be taken from the next nearby on the last trading day, then 1 business day should be specified and so on.
	 */
	DeliveryDateRollConvention Offset
	/**
	 * Specifies, for a Commodity Transaction that references a listed future, the day on which the specified future will expire ahead of the actual expiration of the referenced future. For example: Z21 Contract expires on 19Nov21, with an adjust of 2D the 'expire' will be 16Nov21. DeliveryDateRollConvention takes precedence. Example: Pricing on the Z21 Contract with NearbyContractDay and a deliveryDateRoll of 10D, Sampling of the F22 Contract will occur on 8Nov21 through the last Date of the Z21 Contract. With an ExpConvention of 5D, the last sampling date on the F22 contract will be 12Nov21.
	 */
	DeliveryDateExpirationConvention Offset
}

/**
 * Specifies the address where the demands, specifications and notices will be communicated to for each of the parties to the agreement.
 */
type DemandsAndNotices struct {
	/**
	 * The parties' contact information election.
	 */
	PartyElection []NoticeInformationElection
	/**
	 * Specifies if notice shall be deemed to be effective at the time such notice is actually received unless such notice is received on a day which is not a Local Business Day or after the Notification Time on any Local Business Day in which event such notice shall be deemed to be effective on the next succeeding Local Business Day. If deemed effective,'True'. If not deemed effective, or not stated,'False'.
	 */
	DeemedEffectiveNextLBD bool
}

type DerivInstrmAttrbts struct {
	XpryDt        string
	PricMltplr    string
	UndrlygInstrm UndrlygInstrm
	DlvryTp       string
}

/**
 * Specifies the method according to which an amount or a date is determined.
 */
type DeterminationMethodology struct {
	/**
	 * Represents a more granular dimention of observation. Typically relevent for resolving a unique equity price, which can be expressed as trade-weighted or volume-weighted averages.
	 */
	DeterminationMethod DeterminationMethodEnum
	/**
	 * Specifies enumerations for the type of averaging calculation.
	 */
	AveragingMethod AveragingCalculationMethodEnum
}

/**
 * Defines the roles and related terms which document the agreement of parties about any determination requirements ; mostly about Extraordinary Events, without being necessarily restricted to such scope, as further specified in the particular product at stake e.g. for instance when Calculation Agent is mentioned as the Price Determination Method enumarated value, etc.
 */
type DeterminationRolesAndTerms struct {
	/**
	 * Defines the enumerated values to specify the determination roles to the transaction. Such roles mostly address any determination that would be required when some Extraordinary Events would happen, for instance qualifying the effectiveness of such event, or when a calculation is required, etc. else any other kind of determination as need be. Yet for clarity, other kinds of determination may be covered by such roles, which are not necessarily restricted to the scope of Extarordinary Events, depending on particular product specifications e.g. for instance when Calculation Agent is mentioned as the Price Determination Method enumarated value, etc.
	 */
	DeterminationRole DeterminationRoleEnum
	/**
	 * Designates which Counterparty to the transaction is granted with the particular role ; cardinality of this object is 2, in case parties are both having the same role e.g. jointly CalculationAgent, jointly Determining Party, etc.
	 */
	WhoToDetermine []CounterpartyRoleEnum
	/**
	 * Where the party who is not granted with the determination role at least has a right to dispute the determination given by the counterparty with such role. As an example, a given PartyA is the unique Counterparty with the Role of CalculationAgent, yet Party B could be Disputing Party in regard of such Role.
	 */
	DisputingParty CounterpartyRoleEnum
	/**
	 * Where parties may optionnaly describe any extra bespoke agreements about fallback procedure attached to a particular determination role. This fallback bespoke paragraph would typically address topics such as : delay for the Disputing Party to notify a disagreement when receiving a determination from the Calculation Agent ; Independent Third Party Dealers election terms e.g. number of Dealers to be jointly elected by the counterparties ; delay for joint election of Dealers by the counterparties ; further fallback election procedure in case of disagreement between parties to jointly elect all the Dealers e.g. typically each party designates 1 Dealer, each a Party Representant, which thus result in 2 Dealers in charge of joint election of 3 other Dealers who will be the final ones to provide the response, each a Final Dealer ; delay for joint election of the Final Dealers by the Party Representants ; fees repartition e.g. typically cost and fees incurred by the election of Dealers are equally shared by the parties ; etc.
	 */
	FallbackLanguageBespokeTerms Clause
}

/**
 * An Asset that exists only in digital form, eg Bitcoin or Ethereum, that is not backed by other Assets; excludes the digital representation of other Assets, eg coins or Tokenised assets.
 */
type DigitalAsset struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
}

/**
 * A logical container to hold a defined set of related data. In this case the collation of Ratings Agency, Rating value and an associated direction.
 */
type DirectionRating struct {
	/**
	 * Specifies The credit agency to which the other variables (notation, scale, debt type) refer to.
	 */
	Agency CreditRatingAgencyEnum
	/**
	 * Specifies The credit rating notation. As it varies among credit rating agencies, FpML doesn't specify a default scheme.
	 */
	Notation FieldWithMeta
	/**
	 * Specifies the credit rating scale, with a typical distinction between short term, long term. FpML doesn't specify a default scheme, which is hence not specified as an enumeration as part of the CDM.
	 */
	Scale FieldWithMeta
	/**
	 * Specifies the credit rating debt type (e.g. long term, high yield, deposits, ...) associated with the credit rating notation and scale.
	 */
	Debt CreditRatingDebt
	/**
	 * Assesses the potential direction of a long-term credit rating over the intermediate term, which is generally up to two years for investment grade and generally up to one year for speculative grade.
	 */
	Outlook CreditRatingOutlookEnum
	/**
	 * Indicates the potential direction of a short-term or long-term rating. It focuses on identifiable events and short-term trends that cause ratings to be placed under special surveillance.
	 */
	CreditWatch CreditRatingCreditWatchEnum
	/**
	 * Specifies the direction of measurement for the rated party.
	 */
	Direction CompareOp
}

/**
 * A logical container to hold a defined set of related data. In this case the collation of Ratings Information and associated exposure multiplier.
 */
type DirectionRatingMultiplier struct {
	/**
	 * A logical container to hold a defined set of related data. In this case the collation of Ratings Agency, Rating value and an associated direction.
	 */
	VariableSet []DirectionRatingSet
	/**
	 * Specifies the multiplier applied to exposure based on the credit rating of the rated party.
	 */
	ExposureMultiplier float64
}

/**
 * A logical container to hold a defined set of related data. In this case the collation of Ratings Agency, Rating value and an associated direction that are grouped by either AND or OR logic.
 */
type DirectionRatingSet struct {
	/**
	 * Enables two or more Direction and Ratings to be combined using AND logic.
	 */
	AllDirectionRating AllDirectionRating
	/**
	 * Enables two or more Direction and Ratings to be combined using OR logic.
	 */
	AnyDirectionRating AnyDirectionRating
	/**
	 * A logical container to hold a defined set of related data. In this case the collation of Ratings Agency, Rating value and an associated direction.
	 */
	DirectionRating DirectionRating
}

/**
 * A data defining:  discounting information. The 2000 ISDA definitions, section 8.4. discounting (related to the calculation of a discounted fixed amount or floating amount) apply. This type must only be included if discounting applies.
 */
type DiscountingMethod struct {
	/**
	 * The discounting method that is applicable.
	 */
	DiscountingType DiscountingTypeEnum
	/**
	 * A discount rate, expressed as a decimal, to be used in the calculation of a discounted amount. A discount amount of 5% would be represented as 0.05.
	 */
	DiscountRate float64
	/**
	 * A discount day count fraction to be used in the calculation of a discounted amount.
	 */
	DiscountRateDayCountFraction FieldWithMeta
}

/**
 * A class to specify the election terms under which a party disputes the Value of any Transfer of Eligible Credit Support or Posted Credit Support.
 */
type DisputeResolution struct {
	/**
	 * The time by which the dispute needs to be resolved, failure of which would trigger a recalculation alongside a process that is specified as part of the agreement.
	 */
	ResolutionTime BusinessCenterTime
	/**
	 * The custom Resolution Time election that might be specified by the parties.
	 */
	OtherTerms string
	/**
	 * The alternative dispute resolution procedure if specified.
	 */
	AlternativeTerms string
	/**
	 * The elections to specify terms for recalculation of the market value of posted collateral.
	 */
	RecalculationOfValue RecalculationOfValue
	/**
	 * Alternative procedures to those in the pre-print for resolving collateral related disputes.
	 */
	LegacyAlternative LegacyResolutionAlternative
	/**
	 * Methodology for resolving disputes in respect of the determination of the value of collateral.
	 */
	Value ResolutionValue
}

/**
 * A class to specify the Distributions and Interest Payment provisions applicable to the collateral agreement.
 */
type DistributionAndInterestPayment struct {
	/**
	 * Represents the interest parameters for the various currencies, margin types, posting parties.
	 */
	InterestParameters []CollateralInterestParameters
}

/**
 * The parameters which define whether dividends are applicable
 */
type DividendApplicability struct {
	/**
	 * If present and true, then options exchange dividends are applicable.
	 */
	OptionsExchangeDividends bool
	/**
	 * If present and true, then additional dividends are applicable.
	 */
	AdditionalDividends bool
	/**
	 * Represents the European Master Confirmation value of 'All Dividends' which, when applicable, signifies that, for a given Ex-Date, the daily observed Share Price for that day is adjusted (reduced) by the cash dividend and/or the cash value of any non cash dividend per Share (including Extraordinary Dividends) declared by the Issuer. All Dividends in accordance with the ISDA 2002 Equity Derivatives Definitions.
	 */
	AllDividends bool
}

/**
 * A class to specify the currency in which the dividends will be denominated, i.e. either in the dividend currency or in a currency specified as part of the contract.
 */
type DividendCurrency struct {
	/**
	 * The currency in which the dividend is denominated. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	Currency FieldWithMeta
	/**
	 * Specifies the method according to which the dividend is determined, e.g. the dividend currency.
	 */
	DeterminationMethod DeterminationMethodEnum
	/**
	 * Reference to a currency specified elsewhere in the document
	 */
	CurrencyReference ReferenceWithMeta
}

/**
 * A class to specify the dividend date by reference to another date, with the ability to apply and offset. This class doesn't exist in FpML and is meant to simplify the choice constraint associated with the DividendPaymentDate class.
 */
type DividendDateReference struct {
	/**
	 * Specification of the dividend date using an enumeration, with values such as the pay date, the ex-date or the record date.
	 */
	DateReference DividendDateReferenceEnum
	/**
	 * Only to be used when SharePayment has been specified in the dividendDateReference element. The number of Currency Business Days following the day on which the Issuer of the Shares pays the relevant dividend to holders of record of the Shares.
	 */
	PaymentDateOffset Offset
}

/**
 * A class describing the date on which the dividend will be paid/received. This class is also used to specify the date on which the FX rate will be determined, when applicable.
 */
type DividendPaymentDate struct {
	DividendDateReference DividendDateReference
	DividendDate          ReferenceWithMeta
}

/**
 * A class describing the dividend payout ratio associated with an equity underlier. In certain cases the actual ratio is not known on trade inception, and only general conditions are then specified.
 */
type DividendPayoutRatio struct {
	/**
	 * Specifies the total actual dividend payout ratio associated with the equity underlier. A ratio of 90% should be expressed at 0.90.
	 */
	TotalRatio float64
	/**
	 * Specifies the cash actual dividend payout ratio associated with the equity underlier. A ratio of 90% should be expressed at 0.90.
	 */
	CashRatio float64
	/**
	 * Specifies the non cash actual dividend payout ratio associated with the equity underlier. A ratio of 90% should be expressed at 0.90.
	 */
	NonCashRatio float64
	/**
	 * In the case of a basket underlier, specifies to which component of the basket this particular set of dividend payout ratios correspond.
	 */
	BasketConstituent ReferenceWithMeta
}

/**
 * Time bounded dividend payment periods, each with a dividend payment date per period.
 */
type DividendPeriod struct {
	/**
	 * Dividend period start date.
	 */
	StartDate DividendPaymentDate
	/**
	 * Dividend period end date.
	 */
	EndDate DividendPaymentDate
	/**
	 * Date adjustments for all unadjusted dates in this dividend period.
	 */
	DateAdjustments BusinessDayAdjustments
	/**
	 * For basket underliers, reference to the basket component which is paying dividends in the specified period.
	 */
	BasketConstituent ReferenceWithMeta
	/**
	 * Specifies when the dividend will be paid to the receiver of the equity return. Has the meaning as defined in the ISDA 2002 Equity Derivatives Definitions. Is not applicable in the case of a dividend reinvestment election.
	 */
	DividendPaymentDate DividendPaymentDate
	/**
	 * Specifies the dividend valuation dates of the swap.
	 */
	DividendValuationDate AdjustableOrRelativeDate
}

/**
 * A class describing the conditions governing the payment of dividends to the receiver of the equity return, with the exception of the dividend payout ratio, which is defined for each of the underlying components.
 */
type DividendReturnTerms struct {
	/**
	 * Specifies the dividend payout ratio associated with each underlier. In FpML 5.10 the payout is positioned at the underlier level, although there is an intent to reconsider this approach and position it at the leg level. This is approach adopted by the CDM.
	 */
	DividendPayoutRatio []DividendPayoutRatio
	/**
	 * Boolean element that defines whether the dividend will be reinvested or not.
	 */
	DividendReinvestment bool
	/**
	 * Defines the date on which the receiver of the equity return is entitled to the dividend.
	 */
	DividendEntitlement DividendEntitlementEnum
	/**
	 * Specifies whether the dividend is paid with respect to the Dividend Period.
	 */
	DividendAmountType DividendAmountTypeEnum
	/**
	 * Performance calculation, in accordance with Part 1 Section 12 of the 2018 ISDA CDM Equity Confirmation for Security Equity Swap, Para 75. 'Equity Performance'. Cumulative performance is used as a notional multiplier factor on both legs of an Equity Swap.
	 */
	Performance string
	/**
	 * 2002 ISDA Equity Derivatives Definitions: Dividend Period as either the First Period or the Second Period. |
	 */
	FirstOrSecondPeriod DividendPeriodEnum
	/**
	 * Specifies the party which determines if dividends are extraordinary in relation to normal levels.
	 */
	ExtraordinaryDividendsParty AncillaryRoleEnum
	/**
	 * Determination of Gross Cash Dividend per Share.
	 */
	ExcessDividendAmount DividendAmountTypeEnum
	/**
	 * Specifies the currency in which the dividend will be denominated, e.g. the dividend currency, or a specified currency. This class is not specified as such in FpML, which makes use of the CurrencyAndDeterminationMethod.model to specify such terms.
	 */
	DividendCurrency DividendCurrency
	/**
	 * Specifies the treatment of Non-Cash Dividends.
	 */
	NonCashDividendTreatment NonCashDividendTreatmentEnum
	/**
	 * Specifies how the composition of Dividends is to be determined.
	 */
	DividendComposition DividendCompositionEnum
	/**
	 * Specifies the method according to which special dividends are determined.
	 */
	SpecialDividends bool
	/**
	 * If present and true, then material non cash dividends are applicable.
	 */
	MaterialDividend bool
	/**
	 * One to many time bounded dividend payment periods, each with a dividend payment date per period.
	 */
	DividendPeriod []DividendPeriod
}

/**
 * Information related to dividends and payments.
 */
type DividendTerms struct {
	/**
	 * Specifies the proportion of the value of the dividend on the borrowed shares that the borrower is legally obligated to return to the lender.
	 */
	ManufacturedIncomeRequirement DividendPayoutRatio
	/**
	 * Defines the date on which the receiver of the equity return is entitled to the dividend.
	 */
	DividendEntitlement DividendEntitlementEnum
	/**
	 * daily fee increments accrue until a threshold is crossed, at which point payment becomes due)
	 */
	MinimumBillingAmount Money
}

type Document struct {
	FinInstrmRptgTxRpt FinInstrmRptgTxRpt
}

type DomesticCurrencyIssued struct {
	/**
	 * Identifies that the Security must be denominated in the domestic currency of the issuer.
	 */
	DomesticCurrencyIssued bool
}

/**
 * A data to:  define the adjusted dates associated with an early termination provision.
 */
type EarlyTerminationEvent struct {
	/**
	 * The date on which option exercise takes place. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedExerciseDate time.Time
	/**
	 * The early termination date that is applicable if an early termination provision is exercised. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedEarlyTerminationDate time.Time
	/**
	 * The date by which the cash settlement amount must be agreed. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedCashSettlementValuationDate time.Time
	/**
	 * The date on which the cash settlement amount is paid. This date should already be adjusted for any applicable business date convention.
	 */
	AdjustedCashSettlementPaymentDate time.Time
	/**
	 * The date on which the exercise fee amount is paid. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedExerciseFeePaymentDate time.Time
	Meta                           MetaFields
}

/**
 * A data defining:  an early termination provision for a swap. This early termination is at fair value, i.e. on termination the fair value of the product must be settled between the parties.
 */
type EarlyTerminationProvision struct {
	/**
	 * A mandatory early termination provision to terminate the swap at fair value.
	 */
	MandatoryEarlyTermination MandatoryEarlyTermination
	/**
	 * Period after trade date of the mandatory early termination date.
	 */
	MandatoryEarlyTerminationDateTenor Period
	/**
	 * An option for either or both parties to terminate the swap at fair value.
	 */
	OptionalEarlyTermination OptionalEarlyTermination
	/**
	 * Definition of the first early termination date and the frequency of the termination dates subsequent to that. American exercise is defined by having a frequency of one day.
	 */
	OptionalEarlyTerminationParameters ExercisePeriod
	Meta                               MetaFields
}

/**
 *  This class represents the full set of price-forming features associated with a contractual product: the payout component, the notional/quantity, the effective and termination date and the date adjustment provisions when applying uniformily across the payout components. This class also includes the legal provisions which have valuation implications: cancelable provision, extendible provision, early termination provision and extraordinary events specification.
 */
type EconomicTerms struct {
	/**
	 * The first day of the terms of the trade. This day may be subject to adjustment in accordance with a business day convention.
	 */
	EffectiveDate AdjustableOrRelativeDate
	/**
	 * The last day of the terms of the trade. This date may be subject to adjustments in accordance with the business day convention. It can also be specified in relation to another scheduled date (e.g. the last payment date).
	 */
	TerminationDate AdjustableOrRelativeDate
	/**
	 * The business day adjustment convention when it applies across all the payout components. This specification of the business day convention and financial business centers is used for adjusting any calculation period date if it would otherwise fall on a day that is not a business day in the specified business center.
	 */
	DateAdjustments BusinessDayAdjustments
	/**
	 * The payout specifies the future cashflow computation methodology which characterizes a financial product.
	 */
	Payout []Payout
	/**
	 * Contains optional provisions pertaining to the termination characteristics of a contract.
	 */
	TerminationProvision TerminationProvision
	/**
	 * The ISDA calculation agent responsible for performing duties as defined in the applicable product definitions.
	 */
	CalculationAgent CalculationAgent
	/**
	 * Specifies, when boolean value is True, that additional economic terms exist that have not been included in the product representation.
	 */
	NonStandardisedTerms bool
	/**
	 * Represents the collateral obligations of a party.
	 */
	Collateral Collateral
}

/**
 * A class to specify the party elective amounts which can be used for the purpose of specifying elections such as the ISDA CSA Threshold and Minimum Transfer Amount.
 */
type ElectiveAmountElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * Specifies an enumerated election to express the elective amount.
	 */
	ElectiveAmount ElectiveAmountEnum
	/**
	 * The elective amount when expressed as a currency amount. The associated PartyElectiveAmount_amount data rule enforces that the currency amount is actually greater than 0.
	 */
	Amount Money
	/**
	 * The elective amount when expressed as a custom election by the party.
	 */
	CustomElection string
}

/**
 * Query to check against an EligibleCollateralSpecification
 */
type EligibilityQuery struct {
	/**
	 * Maturity in years
	 */
	Maturity float64
	/**
	 * The asset product type.
	 */
	CollateralAssetType AssetType
	/**
	 * The asset country of origin.
	 */
	AssetCountryOfOrigin ISOCountryCodeEnum
	/**
	 * The underlying asset denominated currency.
	 */
	DenominatedCurrency CurrencyCodeEnum
	/**
	 * The agency rating based on default risk and creditors claim in event of default associated with specific instrument.
	 */
	AgencyRating AgencyRatingCriteria
	/**
	 * Represents a filter based on the type of entity issuing the asset.
	 */
	IssuerType CollateralIssuerType
	/**
	 * Specifies the issuing entity name or LEI.
	 */
	IssuerName LegalEntity
}

/**
 * Specifies the conditions under which a party and its custodian(s) are entitled to hold collateral.
 */
type EligibilityToHoldCollateral struct {
	/**
	 * Specifies whether the party's custodian is eligible to hold the other party's posted collateral. If 'True' then the custodian is eligible to hold the Posted Collateral given that the specified criteria are met.
	 */
	CustodianEligibility bool
	/**
	 * Election to specify the initial custodian.
	 */
	InitialCustodian LegalEntity
	/**
	 * A logical container to hold a defined set of related data. In this case the collation of criteria required by the party and its custodian to Hold the other party's Posted Collateral.
	 */
	Criteria EligibilityToHoldCollateralCriteria
	/**
	 * The restrictions that might be required by a party and its custodian in terms of country(ies) where the other party's collateral can be held.
	 */
	EligibleCountry []FieldWithMeta
	/**
	 * Specifies any additional conditions relevant to the Custodian of the respective parties as set out in the Holding and Using Posted Collateral clause.
	 */
	CustodianCondition string
}

/**
 * Defines a set of Criteria required by the party and its custodian to hold the other party's Posted Collateral.
 */
type EligibilityToHoldCollateralCriteria struct {
	/**
	 * Enables two or more Criteria to be combined using AND logic.
	 */
	AllEligibilityToHoldCollateralCriteria AllEligibilityToHoldCollateralCriteria
	/**
	 * Enables two or more Criteria to be combined using OR logic.
	 */
	AnyEligibilityToHoldCollateralCriteria AnyEligibilityToHoldCollateralCriteria
	/**
	 * A logical container designed to define a combination of Ratings based attributes. In this case, the minimum Rating Value, associated Rating Agency and Rating Type for the Party/Custodian to be entitled to hold Posted Collateral.
	 */
	RatingAgencyValue RatingAgencyValue
	/**
	 * The minimum amount of net capital to be entitled to hold Posted Collateral.
	 */
	AmountOfNetCapital AmountOfNetCapital
}

/**
 * Represents a set of criteria used to specify eligible collateral.
 */
type EligibleCollateralCriteria struct {
	/**
	 * The specific criteria that applies. It can be created using AND, OR and NOT logic, and both asset and issuer characteristics.
	 */
	CollateralCriteria CollateralCriteria
	/**
	 * Specifies which of the two counterparties the criteria applies to (either one or both counterparties). This attribute is optional, in case the applicable party is already specified elsewhere within a party election.
	 */
	AppliesTo []CounterpartyRoleEnum
	/**
	 * Restrict the criteria to only apply to a specific type of margin, ie IM or VM.
	 */
	RestrictTo CollateralMarginTypeEnum
	/**
	 * Denotes which Criteria has priority if more than one agency rating applies.
	 */
	RatingPriorityResolution RatingPriorityResolutionEnum
	/**
	 * The specific criteria in an Eligible Collateral Schedule. This is a mandatory attribute of an ECS so an override is required to enforce the cardinality.
	 */
	/**
	 * Identifies the treatment of specified collateral, e.g., haircuts,holding limits or exclusions.
	 */
	Treatment CollateralTreatment
}

/**
 * Specifies the collateral posting obligations for the security provider party(ies).
 */
type EligibleCollateralElection struct {
	/**
	 * Specifies the party providing the security.
	 */
	Party CounterpartyRoleEnum
	/**
	 * If set to True, the Control Agreement is a Credit Support Document with respect to the party(ies).
	 */
	AsPermitted bool
	/**
	 * The eligible collateral as specified in relation to the pledgor/chargor/obligor(s) posting obligation.
	 */
	EligibleCollateral []EligibleCollateralCriteria
	/**
	 * The excluded collateral as specified in relation to the pledgor/chargor/obligor(s) posting obligation.
	 */
	ExcludedCollateral string
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
	/**
	 * Specifies the Other Eligible Support conditions. If not specified, this election is deemed as not applicable.
	 */
	OtherEligibleSupport string
}

/**
 * Represents a set of criteria used to specify eligible collateral.
 */
type EligibleCollateralSpecification struct {
	/**
	 * Specifies the identifier(s) to uniquely identify eligible collateral or a set of eligible collateral, such as a schedule or equivalant for an identity issuer.
	 */
	Identifier []Identifier
	/**
	 * The parties associated with the specification.
	 */
	Party []Party
	/**
	 * Specification of the roles of the counterparties to the specification.
	 */
	Counterparty []Counterparty
	/**
	 * Represents a set of criteria used to specify eligible collateral.
	 */
	Criteria []EligibleCollateralCriteria
	/**
	 * Specifies the role(s) that each of the party(s) is playing in the context of the specification, eg Payor or Receiver.
	 */
	PartyRole []PartyRole
	Meta      MetaFields
}

/**
 * Specifies the collateral posting obligations of the security provider or security providers as specified in the corresponding agreement
 */
type EligibleCreditSupport struct {
	/**
	 * The specification of the collateral posting obligations for the security provider party(ies)
	 */
	PartyElection []EligibleCollateralElection
}

/**
 * A class to specify Enforcement Events specific to Security Agreements.
 */
type EnforcementEvent struct {
	/**
	 * The early termination election.
	 */
	EarlyTerminationDate bool
	/**
	 * The failure to pay election.
	 */
	FailureToPay bool
}

/**
 * Transaction AdditionalTerms that apply to Equity asset class.
 */
type EquityAdditionalTerms struct {
	ExtraordinaryEvents   ExtraordinaryEvents
	DeterminationTerms    []DeterminationRolesAndTerms
	SubstitutionProvision UnderlierSubstitutionProvision
}

/**
 * A class for defining the merger events and their treatment.
 */
type EquityCorporateEvents struct {
	/**
	 * Shall occur if a Merger Event occurs and the consideration for the relevant Security consists solely of Combined Consideration.
	 */
	ShareForShare ShareExtraordinaryEventEnum
	/**
	 * Shall occur if a Merger Event occurs and the consideration for the relevant Security consists solely of Other Consideration.
	 */
	ShareForOther ShareExtraordinaryEventEnum
	/**
	 * Shall occur if a Merger Event occurs and the consideration for the relevant Security consists solely of Combined Consideration.
	 */
	ShareForCombined ShareExtraordinaryEventEnum
}

/**
 * Specification of an index based on equity securities, e.g. the S&P 500..
 */
type EquityIndex struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * A description of the Index.
	 */
	Name FieldWithMeta
	/**
	 * The organisation that creates or maintains the Index.
	 */
	Provider LegalEntity
	/**
	 * The Asset Class of the Index.
	 */
	AssetClass AssetClassEnum
	/**
	 * The equity index value speciied from the EquityIndexEnum.
	 */
	EquityIndex EquityIndexEnum
}

/**
 * Specification for General Terms and Elections of an Equity Master Confirmation that is applicable across multiple Equity confirmations and is referenced by each of these confirmations, an example of which being the 2018 ISDA CDM Equity Confirmation for Security Equity Swap.
 */
type EquityMasterConfirmation struct {
}

/**
 * Specification for the General Terms and Relationship Supplement Elections as provided in the 2018 ISDA CDM Equity Confirmation for Security Equity Swap.
 */
type EquitySwapMasterConfirmation2018 struct {
	/**
	 * Per Part 1 Section 4, 'Dividend Obligations', of the 2018 ISDA CDM Equity Confirmation, Para 4.2 'Dividend Returns'.
	 */
	TypeOfSwapElection ReturnTypeEnum
	/**
	 * Per Part 1 Section 5, 'Pricing', of the 2018 ISDA CDM Equity Confirmation, Para 5.1.
	 */
	PricingMethodElection PriceReturnTerms
	/**
	 * Per Part 1 Section 3, 'Floating Obligations', of the 2018 ISDA CDM Equity Confirmation. Para 3.3.
	 */
	LinearInterpolationElection InterpolationMethodEnum
	/**
	 * Per Part 1 Section 8, 'Settlement', of the 2018 ISDA CDM Equity Confirmation for Security Equity Swap.
	 */
	SettlementTerms SettlementTerms
	/**
	 * The parameters used to generate the 'Equity Valuation Dates' schedule, including the Effective Date and Termination Date for the Swap.
	 */
	ValuationDates ValuationDates
	/**
	 * The parameters used to generate the payment date schedule, relative to the equityCalculationPeriod. Per Part 1 Section 12, 'Definitions', of the 2018 ISDA CDM Equity Confirmation. Para 73.
	 */
	EquityCashSettlementDates PaymentDates
}

type EquityType struct {
	EquityType        EquityTypeEnum
	DepositaryReceipt DepositaryReceiptTypeEnum
}

type EquityUnderlierProvisions struct {
	/**
	 * For an index option or swap transaction, a flag to indicate whether a relevant Multiple Exchange Index Annex is applicable to the transaction. This annex defines additional provisions which are applicable where an index is comprised of component securities that are traded on multiple exchanges.
	 */
	MultipleExchangeIndexAnnexFallback bool
	/**
	 * For an index option or swap transaction, a flag to indicate whether a relevant Component Security Index Annex is applicable to the transaction.
	 */
	ComponentSecurityIndexAnnexFallback bool
	/**
	 * The ISO 3166 standard code for the country within which the postal address is located.
	 */
	LocalJurisdiction FieldWithMeta
	/**
	 * The ISO 3166 standard code for the country within which the postal address is located.
	 */
	RelevantJurisdiction FieldWithMeta
}

/**
 * Refers to the segregation deposit procedure parties may want to commmit into during the time a Disruption Event is effective
 */
type EscrowArrangement struct {
	EscrowArrangementIsApplicable bool
	MaximumDaysOfDisruption       ValuationPostponement
	BespokeEscrowArrangements     Clause
}

/**
 * Defines the currency referenced as such in any terms related to FX Disruption Events
 */
type EventCurrency struct {
	/**
	 * Defines the currency referenced as such in any terms related to FX Disruption Events
	 */
	EventCurrency FieldWithMeta
	/**
	 * Defines the Event Currency Amount to be exchanged on the Settlement Date.
	 */
	EventCurrentAmount Quantity
	/**
	 * Unless otherwised specified, the Buyer is the party to which the Event Currency Amount is owed on the Settlement Date and the Seller is the party which owes the Event Currency Amount on the Settlement Date.
	 */
	EventCurrencyBuyerSeller []BuyerSeller
}

/**
 * Specifies instructions to create a BusinessEvent.
 */
type EventInstruction struct {
	/**
	 * The intent attribute is meant to be specified when the event qualification cannot be programmatically inferred from the event features. As a result it is only associated with those primitives that can give way to such ambiguity, the quantityChange being one of those. An example of such is a reduction in the trade notional, which could be interpreted as either a trade correction (unless a maximum period of time post-event is specified as part of the qualification), a partial termination or a portfolio rebalancing in the case of an equity swap. On the other hand, an event such as the exercise is not expected to have an associated intent as there should not be ambiguity.
	 */
	Intent                EventIntentEnum
	CorporateActionIntent CorporateActionTypeEnum
	/**
	 * Specifies the date on which the event is taking place. This date is equal to the trade date in the case of a simple execution.  However it can be different from the trade date, for example in the case of a partial termination.
	 */
	EventDate time.Time
	/**
	 * The date on which the event contractually takes effect, when different from the event date.
	 */
	EffectiveDate time.Time
	/**
	 * Specifies the package information in case the business event represents several trades executed as a package (hence this attribute is optional). The package information is only instantiated once at the business event level to preserve referential integrity, whereas individual trades make reference to it to identify that they are part of a package.
	 */
	PackageInformation IdentifiedList
	/**
	 * Specifies the instructions to create the Business Event.
	 */
	Instruction []Instruction
}

/**
 * A class to represent the various set of timestamps that can be associated with lifecycle events, as a collection of [dateTime, qualifier].
 */
type EventTimestamp struct {
	/**
	 * The CDM specifies that the zoned date time is to be expressed in accordance with ISO 8601, either as UTC as an offset to UTC.
	 */
	DateTime time.Time
	/**
	 * The timestamp qualifier is specified through an enumeration because the experience of integrating the DTCC and CME data representations suggests that a wide set of timestamps are currently utilized among service providers, while there is not at present an objective set of criteria that could help suggest a defined set of timestamps as part of the CDM. At some future point, one possible baseline could be developed from the review of the set of timestamps specified across regulatory regimes and regulations (incl. regulations such as high frequency trading). Also, the integration with a further set of implementations and the specification of business workflows such as clearing as part of the CDM development should help confirm the implementation approach in this respect.
	 */
	Qualification EventTimestampQualificationEnum
}

/**
 * Specifies a transaction which automatically extends for a specified timeframe until the exercise of an embedded option.
 */
type EvergreenProvision struct {
	/**
	 * If evergreen termination is not available to both parties then this component specifies the buyer and seller of the option.
	 */
	SinglePartyOption PartyRole
	/**
	 * The length of each evergreen extension period relative to the effective date of the preceding contract.
	 */
	NoticePeriod RelativeDateOffset
	/**
	 * Defines the minimum period before an evergreen is scheduled to terminate that notice can be given that it will terminate beyond the scheduled termination date.
	 */
	NoticeDeadlinePeriod RelativeDateOffset
	/**
	 * A specific date and time for the notice deadline
	 */
	NoticeDeadlineDateTime time.Time
	/**
	 * The frequency with which the evergreen contract will be extended if notice is not given.
	 */
	ExtensionFrequency AdjustableRelativeOrPeriodicDates
	/**
	 * An optional adjustment to the rate for the last period of the evergreen i.e. the period from when notice is given to stop rolling the contract through to the termination date.
	 */
	FinalPeriodFeeAdjustment Price
}

/**
 * Description of the relevant derivative transactions excluded from the calculation of exposure.
 */
type ExcludedProducts struct {
	/**
	 * Description of the relevant derivative transactions excluded from the calculation of exposure.
	 */
	Product []ExposureScopeProductEnum
	/**
	 * Description of the specific branch entered into by the relevant party.
	 */
	Branch []BusinessCenter
	/**
	 * A flag to indicate whether transactions entered into by a specific branch with respect to the relevant party are excluded; set to True if excluded, false if included.
	 */
	Excluded bool
}

type ExctgPrsn struct {
	Prsn Prsn
}

/**
 * Defines specific attributes that relate to trade executions.
 */
type ExecutionDetails struct {
	/**
	 * Identifies the type of execution, e.g. via voice, electronically...
	 */
	ExecutionType ExecutionTypeEnum
	/**
	 * Represents the venue on which a trade was executed.
	 */
	ExecutionVenue LegalEntity
	/**
	 * A reference to the package linking the trade with other trades, in case the trade was executed as part of a package (hence this attribute is optional).
	 */
	PackageReference IdentifiedList
	Meta             MetaFields
}

/**
 * Specifies instructions for execution of a transaction, consisting of a product, price, quantity, parties, trade identifier, execution details, and settlement terms.
 */
type ExecutionInstruction struct {
	/**
	 * Defines the financial product to be executed and contract formed.
	 */
	Product NonTransferableProduct
	/**
	 * Defines the prices (e.g. spread, equity price, FX rate), quantities (e.g. currency amount, no. shares) and settlement terms (e.g. initial fee, broker fee, up-front cds payment or option premium settlement) associated with the constituents of the transacted product.
	 */
	PriceQuantity []PriceQuantity
	/**
	 * Maps two defined parties to counterparty enums for the transacted product.
	 */
	Counterparty []Counterparty
	/**
	 * Maps any ancillary parties, e.g. parties involved in the transaction that are not one of the two principal parties.
	 */
	AncillaryParty []AncillaryParty
	/**
	 * Defines all parties to that execution, including agents and brokers.
	 */
	Parties []Party
	/**
	 * Defines the role(s) that party(ies) may have in relation to the execution.
	 */
	PartyRoles []PartyRole
	/**
	 * Specifies the type and venue of execution, e.g. via voice, or electronically.
	 */
	ExecutionDetails ExecutionDetails
	/**
	 * Denotes the trade/execution date.
	 */
	TradeDate FieldWithMeta
	/**
	 * Denotes the trade time and timezone as agreed by the parties to the trade.
	 */
	TradeTime FieldWithMeta
	/**
	 * Denotes one or more identifiers associated with the transaction.
	 */
	TradeIdentifier []TradeIdentifier
	/**
	 * Detail the collateral requirement anticipated with the transaction.
	 */
	Collateral Collateral
	/**
	 * Lot Identifier associated with the transaction.
	 */
	LotIdentifier Identifier
}

/**
 * A class to specify execution language terms of a Security Agreement.
 */
type ExecutionLanguage struct {
	/**
	 * A boolean attribute to determine if standard language is applicable or not.
	 */
	StandardLanguage bool
	/**
	 * The number of original documents.
	 */
	NumberOfOriginals string
	/**
	 * Bespoke execution language to be included when specified.
	 */
	OtherLanguage string
}

/**
 * A class to specify execution location terms of a Security Agreement.
 */
type ExecutionLocation struct {
	/**
	 * The execution location of the agreement.
	 */
	ExecutionLocation ExecutionLocationEnum
	/**
	 * Bespoke execution location language to be included when specified.
	 */
	OtherLanguage string
	/**
	 * The payer of documentary duty.
	 */
	DutyPayer string
	/**
	 * Bespoke terms specific to the payment of documentary duty.
	 */
	DutyPayerLanguage string
	/**
	 * The date that documentary duty will be paid.
	 */
	DutyPaymentDate time.Time
	/**
	 * Bespoke terms specific to the date that documentary duty will be paid.
	 */
	DutyPaymentLanguage string
}

/**
 * A class to specify execution location and language of execution to determine duty to be paid.
 */
type ExecutionTerms struct {
	/**
	 * The bespoke execution language election.
	 */
	ExecutionLanguage ExecutionLanguage
	/**
	 * The bespoke execution location election.
	 */
	ExecutionLocation ExecutionLocation
}

/**
 * A data defining:  the adjusted dates associated with a particular exercise event.
 */
type ExerciseEvent struct {
	/**
	 * The date on which the option exercise takes place. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedExerciseDate time.Time
	/**
	 * The effective date of the underlying swap associated with a given exercise date. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedRelevantSwapEffectiveDate time.Time
	/**
	 * The date by which the cash settlement amount must be agreed. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedCashSettlementValuationDate time.Time
	/**
	 * The date on which the cash settlement amount is paid. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedCashSettlementPaymentDate time.Time
	/**
	 * The date on which the exercise fee amount is paid. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedExerciseFeePaymentDate time.Time
	Meta                           MetaFields
}

/**
 * A class defining the fee payable on exercise of an option. This fee may be defined as an amount or a percentage of the notional exercised. As a difference with FpML, it extends the BuyerSeller class.
 */
type ExerciseFee struct {
	/**
	 * Specifies the counterparty responsible for making the payments defined by this structure.  The party is one of the two principal parties to the transaction.
	 */
	Payer CounterpartyRoleEnum
	/**
	 * Specifies the party that receives the payments corresponding to this structure.  The party is one of the two counterparties to the transaction.
	 */
	Receiver CounterpartyRoleEnum
	/**
	 * A pointer style reference to the associated notional schedule defined elsewhere in the document.
	 */
	NotionalReference ReferenceWithMeta
	/**
	 * The amount of fee to be paid on exercise. The fee currency is that of the referenced notional.
	 */
	FeeAmount float64
	/**
	 * A fee represented as a percentage of some referenced notional. A percentage of 5% would be represented as 0.05.
	 */
	FeeRate float64
	/**
	 * The date on which exercise fee(s) will be paid. It is specified as a relative date.
	 */
	FeePaymentDate RelativeDateOffset
}

/**
 * A class to define a fee or schedule of fees to be payable on the exercise of an option. This fee may be defined as an amount or a percentage of the notional exercised. As a difference with FpML, it extends the BuyerSeller class.
 */
type ExerciseFeeSchedule struct {
	/**
	 * Specifies the counterparty responsible for making the payments defined by this structure.  The party is one of the two principal parties to the transaction.
	 */
	Payer CounterpartyRoleEnum
	/**
	 * Specifies the party that receives the payments corresponding to this structure.  The party is one of the two counterparties to the transaction.
	 */
	Receiver CounterpartyRoleEnum
	/**
	 * A pointer style reference to the associated notional schedule defined elsewhere in the document.
	 */
	NotionalReference ReferenceWithMeta
	/**
	 * The exercise fee amount schedule. The fees are expressed as currency amounts. The currency of the fee is assumed to be that of the notional schedule referenced.
	 */
	FeeAmountSchedule AmountSchedule
	/**
	 * The exercise free rate schedule. The fees are expressed as percentage rates of the notional being exercised. The currency of the fee is assumed to be that of the notional schedule referenced.
	 */
	FeeRateSchedule Schedule
	/**
	 * The date on which exercise fee(s) will be paid. It is specified as a relative date.
	 */
	FeePaymentDate RelativeDateOffset
}

/**
 * Specifies the information required to communicate the choices made by the exercising party, in a financial product endowing the party with at least one option.
 */
type ExerciseInstruction struct {
	/**
	 * Contains instructions for exercising the option including a quantity change, and optionally a transfer.
	 */
	ExerciseQuantity *PrimitiveInstruction
	/**
	 * Specifies the Option Payout being exercised on the trade.
	 */
	ExerciseOption ReferenceWithMeta
	/**
	 * Specifies the date on which an option contained within the financial product would be exercised. The date may be omitted if the contractual product allows for only a single date of exercise (European exercise).
	 */
	ExerciseDate AdjustableOrAdjustedDate
	/**
	 * Specifies the time at which an option contained within the financial product woulld be exercised. The time may be omitted if the contractual product allows for only a single time of exercise (European exercise).
	 */
	ExerciseTime BusinessCenterTime
	/**
	 * Specifies the trade identifier to apply to the replacement trade for physical exercise.
	 */
	ReplacementTradeIdentifier []TradeIdentifier
}

/**
 * Defines to whom and where notice of execution should be given. The exerciseNoticeGiver refers to one or both of the principal parties of the trade. If present the exerciseNoticeReceiver refers to a party, other than the principal party, to whom notice should be given.
 */
type ExerciseNotice struct {
	/**
	 * Specifies the principal party of the trade that has the right to exercise.
	 */
	ExerciseNoticeGiver ExerciseNoticeGiverEnum
	/**
	 * Specifies the party to which notice of exercise should be given, e.g. by the buyer of the option. Although in many cases it is the buyer of the option who sends the exercise notice to the seller of the option, this component is reused, e.g. in case of OptionEarlyTermination, either or both parties have the right to exercise.
	 */
	ExerciseNoticeReceiver AncillaryRoleEnum
	/**
	 * Specifies the location where the exercise must be reported, e.g. where the exercise notice receiver is based.
	 */
	BusinessCenter FieldWithMeta
}

/**
 * This defines the time interval to the start of the exercise period, i.e. the earliest exercise date, and the frequency of subsequent exercise dates (if any).
 */
type ExercisePeriod struct {
	/**
	 * The time interval to the first (and possibly only) exercise date in the exercise period.
	 */
	EarliestExerciseDateTenor Period
	/**
	 * The frequency of subsequent exercise dates in the exercise period following the earliest exercise date. An interval of 1 day should be used to indicate an American style exercise period.
	 */
	ExerciseFrequency Period
	Meta              MetaFields
}

/**
 * A class describing how notice of exercise should be given. This can be either manual or automatic.
 */
type ExerciseProcedure struct {
	/**
	 * Specifies that the notice of exercise must be given by the buyer to the seller or seller's agent.
	 */
	ManualExercise ManualExercise
	/**
	 * If automatic is specified, then the notional amount of the underlying swap not previously exercised under the swaption will be automatically exercised at the expiration time on the expiration date if at such time the buyer is in-the-money, provided that the difference between the settlement rate and the fixed rate under the relevant underlying swap is not less than the specified threshold rate. The term in-the-money is assumed to have the meaning defining in the 2000 ISDA Definitions, Section 17.4 In-the-money.
	 */
	AutomaticExercise AutomaticExercise
	/**
	 * A flag to indicate whether follow-up confirmation of exercise (written or electronic) is required following telephonic notice by the buyer to the seller or seller's agent.
	 */
	FollowUpConfirmation bool
	/**
	 * Has the meaning defined as part of the 1997 ISDA Government Bond Option Definitions, section 4.5 Limited Right to Confirm Exercise. If present, (i) the Seller may request the Buyer to confirm its intent if not done on or before the expiration time on the Expiration date (ii) specific rules will apply in relation to the settlement mode.
	 */
	LimitedRightToConfirm bool
	/**
	 * Typically applicable to the physical settlement of bond and convertible bond options. If present, means that the party required to deliver the bonds will divide those to be delivered as notifying party desires to facilitate delivery obligations.
	 */
	SplitTicket bool
}

/**
 * A class defining the exercise period for an option together with any rules governing the notional amount of the underlying which can be exercised on any given exercise date and any associated exercise fees.
 */
type ExerciseTerms struct {
	/**
	 * Whether the option has a single exercise (european), multiple exercise dates (bermuda), or a continuous range of exercise (american).
	 */
	Style OptionExerciseStyleEnum
	/**
	 * The first day of the exercise period for an American style option.
	 */
	CommencementDate AdjustableOrRelativeDate
	/**
	 * The dates that define the Bermuda option exercise dates and the expiration date. The last specified date is assumed to be the expiration date. The dates can either be specified as a series of explicit dates and associated adjustments or as a series of dates defined relative to another schedule of dates, for example, the calculation period start dates. Where a relative series of dates are defined the first and last possible exercise dates can be separately specified.
	 */
	ExerciseDates AdjustableOrRelativeDates
	/**
	 * The last day within an exercise period for an American style option. For a European style option it is the only day within the exercise period.
	 */
	ExpirationDate []AdjustableOrRelativeDate
	/**
	 * The effective date on the underlying product if the option is exercised.  For example, for a swaption it is the swap effective date, for an option on an FX spot or forward it is the value date for settlement, and in an extendible/cancelable provision it is the swap termination date, which is the date on which the termination is effective.'
	 */
	RelevantUnderlyingDate AdjustableOrRelativeDates
	/**
	 * The earliest time at which notice of exercise can be given by the buyer to the seller (or seller's agent) to, and including, the expiration date.
	 */
	EarliestExerciseTime BusinessCenterTime
	/**
	 * For a Bermuda or American style option, the latest time on an exercise business day (excluding the expiration date) within the exercise period that notice can be given by the buyer to the seller or seller's agent. Notice of exercise given after this time will be deemed to have been given on the next exercise business day.
	 */
	LatestExerciseTime BusinessCenterTime
	/**
	 * The latest time for exercise on expirationDate. It is made mandatory given that for all option styles, this field is required.
	 */
	ExpirationTime BusinessCenterTime
	/**
	 * The time of day at which the equity option expires, for example the official closing time of the exchange.
	 */
	ExpirationTimeType ExpirationTimeTypeEnum
	/**
	 * As defined in the 2000 ISDA Definitions, Section 12.4. Multiple Exercise, the buyer of the option has the right to exercise all or less than all the unexercised notional amount of the underlying swap on one or more days in the exercise period, but on any such day may not exercise less than the minimum notional amount or more that the maximum notional amount, and if an integral multiple amount is specified, the notional amount exercised must be equal to, or be an integral multiple of, the integral multiple amount.
	 */
	MultipleExercise MultipleExercise
	/**
	 * The fees associated with an exercise date. The fees are conditional on the exercise occurring. The fees can be specified as actual currency amounts or as percentages of the notional amount being exercised.
	 */
	ExerciseFeeSchedule ExerciseFeeSchedule
	/**
	 * The set of parameters defining the procedure associated with the exercise, e.g. manual exercise.
	 */
	ExerciseProcedure ExerciseProcedure
	/**
	 * A fee to be paid on exercise. This could be represented as an amount or a rate and notional reference on which to apply the rate.
	 */
	ExerciseFee ExerciseFee
	/**
	 * As defined in the 2000 ISDA Definitions, Section 12.3. Partial Exercise, the buyer of the option has the right to exercise all or less than all the notional amount of the underlying swap on the expiration date, but may not exercise less than the minimum notional amount, and if an integral multiple amount is specified, the notional amount exercised must be equal to, or be an integral multiple of, the integral multiple amount.
	 */
	PartialExercise PartialExercise
	Meta            MetaFields
}

/**
 * Represents the current mark to market value or IM calculation value of the trade portfolio as recorded by the principle (in base currency).
 */
type Exposure struct {
	/**
	 * Represents a Portfolio that describes all the positions held at a given time, in various states which can be either traded, settled, etc., with lineage information to the previous state.
	 */
	TradePortfolio ReferenceWithMeta
	/**
	 * Represents the aggregate value of the portfolio in base currency.
	 */
	AggregateValue Money
	/**
	 * Indicates the date when the exposure is calculated if different from valuation date.
	 */
	CalculationDateTime time.Time
	/**
	 * Indicates the valuation date of the exposure underlying the calculation.
	 */
	ValuationDateTime time.Time
}

/**
 * Optionally, for each party, specifies the amendments to the included coverage in the calculation of Exposure.
 */
type ExposureScope struct {
	/**
	 * Defines the electice scope for a specific party.
	 */
	PartyElection []LegacyExposureScopeElection
}

/**
 * A data defining:  an option to extend an existing swap transaction on the specified exercise dates for a term ending on the specified new termination date. As a difference from FpML, it extends the BuyerSeller class, which represents the BuyerSeller.model.
 */
type ExtendibleProvision struct {
	/**
	 * Buyer party that can be resolved as one of the two principal parties to the transaction. The party that buys this instrument, i.e. pays for this instrument and receives the rights defined by it. ISDA 2002 Equity Definitions section 1.18: `Buyer` means the party specified as such in the related Confirmation. | ISDA 2006 Definitions article 12.1 (b)(i) relating to a Swaption: 'Buyer' means the party that will, on each Premium Payment Date, pay to Seller the Premium | ISDA 2006 Definitions article 12.1 (b)(ii) relating to Swap Transactions with applicable Early Termination: the party specified as such in the related Confirmation, or the Exercising Party if neither party is specified | ISDA 2006 Definitions article 12.1 (b)(iii) relating to any other Option Transaction: the party specified as such in the related Confirmation. | ISDA 2014 Credit Definition article 1.4: `Buyer` means the Fixed Rate Payer.
	 */
	Buyer CounterpartyRoleEnum
	/**
	 * Seller party that can be resolved as one of the two principal parties to the transaction. The party that sells ('writes') this instrument, i.e. that grants the rights defined by this instrument and in return receives a payment for it. ISDA 2002 Equity Definitions section 1.19: `Seller` means the party specified as such in the related Confirmation. | ISDA 2006 Definitions article 12.1 (a)(i) relating to a Swaption: 'Seller' means the party the party specified as such or as writer in the related Confirmation | ISDA 2006 Definitions article 12.1 (a)(ii) relating to Swap Transactions with applicable Early Termination: the party specified as such or as writer in the related Confirmation or, if neither party is specified as such, the Non-exercising Party | ISDA 2006 Definitions article 12.1 (a)(iii) relating to any other Option Transaction: the party specified as such in the related Confirmation. | ISDA 2014 Credit Definition article 1.4: `Seller` means the Floating Rate Payer.
	 */
	Seller CounterpartyRoleEnum
	/**
	 * Definition of the party to whom notice of exercise should be given.
	 */
	ExerciseNotice ExerciseNotice
	/**
	 * A flag to indicate whether follow-up confirmation of exercise (written or electronic) is required following telephonic notice by the buyer to the seller or seller's agent.
	 */
	FollowUpConfirmation bool
	/**
	 * The adjusted dates associated with an extendible provision. These dates have been adjusted for any applicable business day convention.
	 */
	ExtendibleProvisionAdjustedDates ExtendibleProvisionAdjustedDates
	CallingParty                     CallingPartyEnum
	/**
	 * If the ability to extend the contract is not available to both parties then this component specifies the buyer and seller of the option.
	 */
	SinglePartyOption PartyRole
	/**
	 * Defines the minimum period before a contract is scheduled to terminate that notice can be given that it will terminate beyond the scheduled termination date.
	 */
	NoticeDeadlinePeriod RelativeDateOffset
	/**
	 * A specific date and time for the notice deadline
	 */
	NoticeDeadlineDateTime time.Time
	/**
	 * The length of each extension period relative to the effective date of the preceding contract.
	 */
	ExtensionTerm RelativeDateOffset
	/**
	 * The period within which notice can be given that the contract will be extended.
	 */
	ExtensionPeriod AdjustableRelativeOrPeriodicDates
	/**
	 * The exercise terms associated with the extendible provision, including details such as exercise style, exercise fees, and any other relevant conditions or terms governing the extension of the swap transaction.
	 */
	ExerciseTerms ExerciseTerms
}

/**
 * A data defining:  the adjusted dates associated with a provision to extend a swap.
 */
type ExtendibleProvisionAdjustedDates struct {
	/**
	 * The adjusted dates associated with a single extendible exercise date.
	 */
	ExtensionEvent []ExtensionEvent
}

/**
 * A data to:  define the adjusted dates associated with an individual extension event.
 */
type ExtensionEvent struct {
	/**
	 * The date on which option exercise takes place. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedExerciseDate time.Time
	/**
	 * The termination date if an extendible provision is exercised. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedExtendedTerminationDate time.Time
	Meta                            MetaFields
}

/**
 * Where the underlying is shares, defines market events affecting the issuer of those shares that may require the terms of the transaction to be adjusted.
 */
type ExtraordinaryEvents struct {
	/**
	 * Where parties may optionnaly describe any extra bespoke agreements, in regards of the standardized Extraordinary Events.
	 */
	AdditionalBespokeTerms             []Clause
	MergerEvents                       EquityCorporateEvents
	TenderOfferEvents                  EquityCorporateEvents
	CompositionOfCombinedConsideration bool
	IndexAdjustmentEvents              IndexAdjustmentEvents
	AdditionalDisruptionEvents         AdditionalDisruptionEvents
	/**
	 * If true, failure to deliver is applicable.
	 */
	FailureToDeliver            bool
	Representations             Representations
	NationalizationOrInsolvency NationalizationOrInsolvencyOrDelistingEventEnum
	Delisting                   NationalizationOrInsolvencyOrDelistingEventEnum
}

type FailureToPay struct {
	/**
	 * Indicates whether the failure to pay provision is applicable.
	 */
	Applicable bool
	/**
	 * If this element is specified, indicates whether or not a grace period extension is applicable. ISDA 2003 Term: Grace Period Extension Applicable.
	 */
	GracePeriodExtension GracePeriodExtension
	/**
	 * Specifies a threshold for the failure to pay credit event. Market standard is USD 1,000,000 (JPY 100,000,000 for Japanese Yen trades) or its equivalent in the relevant obligation currency. This is applied on an aggregate basis across all Obligations of the Reference Entity. Intended to prevent technical/operational errors from triggering credit events. ISDA 2003 Term: Payment Requirement
	 */
	PaymentRequirement Money
}

/**
 * Defines the structure needed to represent fallback rate parameters. This type is used to represent modular computed rates in interestRatePayouts.
 */
type FallbackRateParameters struct {
	/**
	 * The floating rate index that is used as the basis of the fallback rate.
	 */
	FloatingRateIndex FloatingRateIndexEnum
	/**
	 * The date the fallback rate takes effect.
	 */
	EffectiveDate time.Time
	/**
	 * Support for modular calculated rates, such such as lockout compound calculations.
	 */
	CalculationParameters FloatingRateCalculationParameters
	/**
	 * The economic spread applied to the underlying fallback rate to replicate the original risky rate.
	 */
	SpreadAdjustment float64
}

/**
 * The method, prioritised by the order it is listed in this element, to get a replacement rate for the disrupted settlement rate option.
 */
type FallbackReferencePrice struct {
	/**
	 * Specifies how long to wait to get a quote from a settlement rate option upon a price source disruption.
	 */
	ValuationPostponement ValuationPostponement
	/**
	 * This settlement rate option will be used in its place.
	 */
	FallBackSettlementRateOption []FieldWithMeta
	/**
	 * Request rate quotes from the market. This element is set as type Empty in FpML. When present, the FpML synonym is mapped to a value True in the CDM.
	 */
	FallbackSurveyValuationPostponement bool
	/**
	 * The calculation agent will decide the rate.
	 */
	CalculationAgentDetermination CalculationAgent
}

/**
 * Payment made following trigger occurrence.
 */
type FeaturePayment struct {
	/**
	 * This attribute doesn't exist as part of the FpML construct, which makes use of the PayerReceiver.model group.
	 */
	PayerReceiver PartyReferencePayerReceiver
	/**
	 * The trigger level percentage.
	 */
	LevelPercentage float64
	/**
	 * The monetary quantity in currency units.
	 */
	Amount float64
	/**
	 * The feature payment time.
	 */
	Time TimeTypeEnum
	/**
	 * The currency in which an amount is denominated.
	 */
	Currency FieldWithMeta
	/**
	 * The feature payment date.
	 */
	PaymentDate AdjustableOrRelativeDate
	Meta        MetaFields
}

type FinInstrm struct {
	Othr Othr
}

type FinInstrmGnlAttrbts struct {
	FullNm     string
	ClssfctnTp string
	NtnlCcy    string
}

type FinInstrmRptgTxRpt struct {
	Tx Tx
}

/**
 * A data to:  define business date convention adjustment to final payment period per leg.
 */
type FinalCalculationPeriodDateAdjustment struct {
	/**
	 * Reference to the unadjusted cancellation effective dates.
	 */
	RelevantUnderlyingDateReference ReferenceWithMeta
	/**
	 * Reference to the leg, where date adjustments may apply.
	 */
	SwapStreamReference ReferenceWithMeta
	/**
	 * Override business date convention. This takes precedence over leg level information.
	 */
	BusinessDayConvention BusinessDayConventionEnum
}

/**
 * A clause providing that final return collateral is to be provided without consideration of any Minimum Transfer Amount and collateral haircut.
 */
type FinalReturns struct {
	/**
	 * Defines if the clause is applicable to this agreement or any other exceptions.
	 */
	Applicable bool
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * Type for reporting the detailed results of calculating a cash flow for a calculation period.  This is enhanced relative to the FpML-based cashflows structure to allow more information to be returned about daily compounded rates.
 */
type FixedAmountCalculationDetails struct {
	/**
	 * The calculation period for which the floating calculation was performed.
	 */
	CalculationPeriod CalculationPeriodBase
	/**
	 * The notional in effect during the calculation period.
	 */
	CalculationPeriodNotionalAmount Money
	/**
	 * The value of the fixed rate that was used.
	 */
	FixedRate float64
	/**
	 * The fraction of a year that this calculation represents, according to the day count fraction method.
	 */
	YearFraction float64
	/**
	 * The amount of the cash flow that was computed, including any spreads and other processing.
	 */
	CalculatedAmount float64
}

/**
 * A predefined price accorded by the counterparties.
 */
type FixedPrice struct {
	/**
	 * Fixed price step schedule, including an initial price specified as an absolute number.
	 */
	Price ReferenceWithMeta
}

/**
 * Represents a fixed price payout. There is no underlier associated with this payout type and is based on fixed pricing per a given unit (e.g. in commodities price per barrel)
 */
type FixedPricePayout struct {
	/**
	 * Canonical representation of the payer and receiver parties applicable to each payout leg.
	 */
	PayerReceiver PayerReceiver
	/**
	 * Each payout leg must implement the quantity concept as a 'resolvable' type, which allows for different payout legs to be linked to each other (e.g. in the case of cross-curreny products).
	 */
	PriceQuantity ResolvablePriceQuantity
	/**
	 * The specification of the principal exchange. Optional as only applicable in the case of cross-currency or zero-coupon swaps with a final payment.
	 */
	PrincipalPayment PrincipalPayments
	/**
	 * Each payout leg must specifies its settlement terms, including the delivery type (i.e. cash vs physical, and their respective terms), the transfer type (DvP etc.) and settlement date, if any.
	 */
	SettlementTerms SettlementTerms
	/**
	 * Specifies the parameters to generate the payment date schedule, either through a parametric representation or by reference to specified dates.
	 */
	PaymentDates PaymentDates
	/**
	 * Specifies the fixed price on which fixed forward payments are based.
	 */
	FixedPrice FixedPrice
	/**
	 * Allows the full representation of a payout by defining a set of schedule periods. It supports standard schedule customization by expressing all the dates, quantities, and pricing data in a non-parametric way.
	 */
	Schedule CalculationSchedule
}

/**
 * Type defining the specification for a fixed rate.
 */
type FixedRateSpecification struct {
	/**
	 * The fixed rate or fixed rate schedule expressed as explicit fixed rates and dates. In the case of a schedule, the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments.
	 */
	RateSchedule RateSchedule
	Meta         MetaFields
}

/**
 * Type for reporting the detailed results of calculating a cash flow for a calculation period.  This is enhanced relative to the FpML-based cashflows structure to allow more information to be returned about daily compounded rates.
 */
type FloatingAmountCalculationDetails struct {
	/**
	 * The calculation period for which the floating calculation was performed.
	 */
	CalculationPeriod CalculationPeriodBase
	/**
	 * The notional in effect during the calculation period.
	 */
	CalculationPeriodNotionalAmount Money
	/**
	 * The details of the floating rate setting.  (If it is a calculated rate, details of that calculation will be inside that.
	 */
	FloatingRate FloatingRateSettingDetails
	/**
	 * Details fo the floating rate treatment after the rate is observed or calculated.  This will include details of things like multipliers, spreads, caps and floors, and the raw and treated rates.
	 */
	ProcessingDetails FloatingRateProcessingDetails
	/**
	 * The rate that was actually applied, after all calculations and treatments.
	 */
	AppliedRate float64
	/**
	 * The fraction of a year that this calculation represents, according to the day count fraction method.
	 */
	YearFraction float64
	/**
	 * The amount of the cash flow that was computed, including any spreads and other processing.
	 */
	CalculatedAmount float64
	/**
	 * The amount of the cash flow excluding any spread, for subsequent processing.
	 */
	SpreadExclusiveCalculatedAMount float64
}

/**
 * A class to specify the ISDA terms relating to the floating rate payment events and the implied additional fixed payments, applicable to the credit derivatives transactions on mortgage-backed securities with pay-as-you-go or physical settlement.
 */
type FloatingAmountEvents struct {
	/**
	 * A floating rate payment event. Corresponds to the failure by the Reference Entity to pay an expected principal amount or the payment of an actual principal amount that is less than the expected principal amount. ISDA 2003 Term: Failure to Pay Principal.
	 */
	FailureToPayPrincipal bool
	/**
	 * A floating rate payment event. With respect to any Reference Obligation Payment Date, either (a) the non-payment of an Expected Interest Amount or (b) the payment of an Actual Interest Amount that is less than the Expected Interest Amount. ISDA 2003 Term: Interest Shortfall.
	 */
	InterestShortfall InterestShortFall
	/**
	 * A floating rate payment event. Results from the fact that the underlier writes down its outstanding principal amount. ISDA 2003 Term: Writedown.
	 */
	Writedown bool
	/**
	 * A floating rate payment event. Results from the fact that losses occur to the underlying instruments that do not result in reductions of the outstanding principal of the reference obligation.
	 */
	ImpliedWritedown bool
	/**
	 * Specifies the floating amount provisions associated with the floatingAmountEvents.
	 */
	FloatingAmountProvisions FloatingAmountProvisions
	/**
	 * Specifies the events that will give rise to the payment additional fixed payments.
	 */
	AdditionalFixedPayments AdditionalFixedPayments
}

type FloatingAmountProvisions struct {
	/**
	 * As specified by the ISDA Supplement for use with trades on mortgage-backed securities, 'WAC Cap' means a weighted average coupon or weighted average rate cap provision (however defined in the Underlying Instruments) of the Underlying Instruments that limits, increases or decreases the interest rate or interest entitlement, as set out in the Underlying Instruments on the Effective Date without regard to any subsequent amendment The presence of the element with value set to 'true' signifies that the provision is applicable. From a usage standpoint, this provision is typically applicable in the case of CMBS and not applicable in case of RMBS trades.
	 */
	WacCapInterestProvision bool
	/**
	 * As specified by the ISDA Standard Terms Supplement for use with trades on mortgage-backed securities. The presence of the element with value set to 'true' signifies that the provision is applicable. If applicable, the applicable step-up terms are specified as part of that ISDA Standard Terms Supplement. From a usage standpoint, this provision is typically applicable in the case of RMBS and not applicable in case of CMBS trades.
	 */
	StepUpProvision bool
}

type FloatingRate struct {
	RateOption ReferenceWithMeta
	/**
	 * The ISDA Spread or a Spread schedule expressed as explicit spreads and dates. In the case of a schedule, the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The spread is a per annum rate, expressed as a decimal. For purposes of determining a calculation period amount, if positive the spread will be added to the floating rate and if negative the spread will be subtracted from the floating rate. A positive 10 basis point (0.1%) spread would be represented as 0.001.
	 */
	SpreadSchedule SpreadSchedule
	/**
	 * The cap rate or cap rate schedule, if any, which applies to the floating rate. The cap rate (strike) is only required where the floating rate on a swap stream is capped at a certain level. A cap rate schedule is expressed as explicit cap rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The cap rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A cap rate of 5% would be represented as 0.05.
	 */
	CapRateSchedule StrikeSchedule
	/**
	 * The floor rate or floor rate schedule, if any, which applies to the floating rate. The floor rate (strike) is only required where the floating rate on a swap stream is floored at a certain strike level. A floor rate schedule is expressed as explicit floor rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The floor rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A floor rate of 5% would be represented as 0.05.
	 */
	FloorRateSchedule StrikeSchedule
	Meta              MetaFields
	/**
	 * A rate multiplier or multiplier schedule to apply to the floating rate. A multiplier schedule is expressed as explicit multipliers and dates. In the case of a schedule, the step dates may be subject to adjustment in accordance with any adjustments specified in the calculationPeriodDatesAdjustments. The multiplier can be a positive or negative decimal. This element should only be included if the multiplier is not equal to 1 (one) for the term of the stream.
	 */
	FloatingRateMultiplierSchedule RateSchedule
	/**
	 * The specification of any rate conversion which needs to be applied to the observed rate before being used in any calculations. The two common conversions are for securities quoted on a bank discount basis which will need to be converted to either a Money Market Yield or Bond Equivalent Yield. See the Annex to the 2000 ISDA Definitions, Section 7.3. Certain General Definitions Relating to Floating Rate Options, paragraphs (g) and (h) for definitions of these terms.
	 */
	RateTreatment RateTreatmentEnum
	/**
	 * Support for modular calculated rates, such such as lockout compound calculations.
	 */
	CalculationParameters FloatingRateCalculationParameters
	/**
	 * Definition of any fallback rate that may be applicable.
	 */
	FallbackRate FallbackRateParameters
}

/**
 * A class defining a floating interest rate through the specification of the floating rate index, the tenor, the multiplier schedule, the spread, the qualification of whether a specific rate treatment and/or a cap or floor apply.
 */
type FloatingRateBase struct {
	RateOption ReferenceWithMeta
	/**
	 * The ISDA Spread or a Spread schedule expressed as explicit spreads and dates. In the case of a schedule, the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The spread is a per annum rate, expressed as a decimal. For purposes of determining a calculation period amount, if positive the spread will be added to the floating rate and if negative the spread will be subtracted from the floating rate. A positive 10 basis point (0.1%) spread would be represented as 0.001.
	 */
	SpreadSchedule SpreadSchedule
	/**
	 * The cap rate or cap rate schedule, if any, which applies to the floating rate. The cap rate (strike) is only required where the floating rate on a swap stream is capped at a certain level. A cap rate schedule is expressed as explicit cap rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The cap rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A cap rate of 5% would be represented as 0.05.
	 */
	CapRateSchedule StrikeSchedule
	/**
	 * The floor rate or floor rate schedule, if any, which applies to the floating rate. The floor rate (strike) is only required where the floating rate on a swap stream is floored at a certain strike level. A floor rate schedule is expressed as explicit floor rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The floor rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A floor rate of 5% would be represented as 0.05.
	 */
	FloorRateSchedule StrikeSchedule
	Meta              MetaFields
}

/**
 * Defines the structures needed to represent the calculation parameters for daily averaged and compounded modular rates as defined in the 2021 ISDA Definitions in Section 7. This type is used to represent modular computed rates in interestRatePayouts.
 */
type FloatingRateCalculationParameters struct {
	/**
	 * calculation type (averaging or compounding).
	 */
	CalculationMethod CalculationMethodEnum
	/**
	 * any obervation shift parameters if applicable.
	 */
	ObservationShiftCalculation ObservationShiftCalculation
	/**
	 * any lookback  parameters if applicable.
	 */
	LookbackCalculation OffsetCalculation
	/**
	 * any lockout  parameters if applicable.
	 */
	LockoutCalculation OffsetCalculation
	/**
	 * the business days that are applicable for the calculation.
	 */
	ApplicableBusinessDays BusinessCenters
	/**
	 *  any applicable observation parameters, such as daily caps or floors.
	 */
	ObservationParameters ObservationParameters
}

/**
 * A data defining:  parameters associated with a floating rate reset. This data forms:  part of the cashflows representation of a stream.
 */
type FloatingRateDefinition struct {
	/**
	 * The final calculated rate for a calculation period after any required averaging of rates A calculated rate of 5% would be represented as 0.05.
	 */
	CalculatedRate float64
	/**
	 * The details of a particular rate observation, including the fixing date and observed rate. A list of rate observation elements may be ordered in the document by ascending adjusted fixing date. An FpML document containing an unordered list of rate observations is still regarded as a conformant document.
	 */
	RateObservation []RateObservation
	/**
	 * A rate multiplier to apply to the floating rate. The multiplier can be a positive or negative decimal. This element should only be included if the multiplier is not equal to 1 (one).
	 */
	FloatingRateMultiplier float64
	/**
	 * The ISDA Spread, if any, which applies for the calculation period. The spread is a per annum rate, expressed as a decimal. For purposes of determining a calculation period amount, if positive the spread will be added to the floating rate and if negative the spread will be subtracted from the floating rate. A positive 10 basis point (0.1%) spread would be represented as 0.001.
	 */
	Spread float64
	/**
	 * The cap rate, if any, which applies to the floating rate for the calculation period. The cap rate (strike) is only required where the floating rate on a swap stream is capped at a certain strike level. The cap rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A cap rate of 5% would be represented as 0.05.
	 */
	CapRate []Strike
	/**
	 * The floor rate, if any, which applies to the floating rate for the calculation period. The floor rate (strike) is only required where the floating rate on a swap stream is floored at a certain strike level. The floor rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. The floor rate of 5% would be represented as 0.05.
	 */
	FloorRate []Strike
}

/**
 * Specification of an interest rate index which can change over time, e.g. the SONIA (Sterling Overnight Index Average) in the UK.
 */
type FloatingRateIndex struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * A description of the Index.
	 */
	Name FieldWithMeta
	/**
	 * The organisation that creates or maintains the Index.
	 */
	Provider LegalEntity
	/**
	 * The Asset Class of the Index.
	 */
	AssetClass AssetClassEnum
	/**
	 * The reference index that is used to specify the floating interest rate.
	 */
	FloatingRateIndex FieldWithMeta
	/**
	 * The ISDA Designated Maturity, i.e. the floating rate tenor.
	 */
	IndexTenor Period
}

/**
 * This holds the rate calculation defaults applicable for a floating rate index.
 */
type FloatingRateIndexCalculationDefaults struct {
	/**
	 * The ISDA FRO category (e.g. screen rate or calculated rate).
	 */
	Category FloatingRateIndexCategoryEnum
	/**
	 * The ISDA FRO style (e.g. term rate, swap rate, etc).
	 */
	IndexStyle FloatingRateIndexStyleEnum
	/**
	 * The ISDA FRO calculation method (e.g. OIS Compounding).
	 */
	Method FloatingRateIndexCalculationMethodEnum
	/**
	 * The default fixing details.
	 */
	Fixing []FloatingRateIndexFixingDetails
	/**
	 * The default day count fraction.
	 */
	DayCountFraction DayCountFractionEnum
	/**
	 * The default applicable business days.
	 */
	ApplicableBusinessDays BusinessCenters
	/**
	 * Publication Calendar (e.g. EUR-ICESWAP)
	 */
	PublicationCalendar BusinessCenter
}

type FloatingRateIndexDefinition struct {
	/**
	 * The underlying FRO name and designated maturity.
	 */
	Fro FloatingRateIndexIdentification
	/**
	 * Any calculation default values.
	 */
	CalculationDefaults FloatingRateIndexCalculationDefaults
	/**
	 * The definition version or versions supported by the FRO.
	 */
	SupportedDefinition []ContractualDefinition
	/**
	 * The source of an FRO, particularly if not a Contractual Definition (e.g. the broker rates matrix).
	 */
	DefinitionalSource           string
	DesignatedMaturityApplicable bool
	/**
	 * Any mappings to other FRos.
	 */
	Mappings FloatingRateIndexMappings
	/**
	 * Any mappings to other codes for this index.
	 */
	ExternalMappings FloatingRateIndexExternalMappings
	/**
	 * YES / NO to flag FROs identified by the FpML Syndicated Loan WG as having underlying benchmark that may also be referenced in syndicated loans.
	 */
	InLoan bool
	/**
	 * FRO History
	 */
	History FroHistory
	/**
	 * Deprecation and Code Descriptions
	 */
	DeprecationReason string
	/**
	 * FpML Description
	 */
	FpmlDescription string
}

/**
 * A map for a single FRO to or from an equivalent or similar codes in a different standard such as ISO.
 */
type FloatingRateIndexExternalMap struct {
	/**
	 *  The FRO name that is being mapped to/from.
	 */
	ExternalId string
	/**
	 * The standard/version to which the map applies.
	 */
	ExternalStandard string
}

/**
 * Represents the mappings of FRO codes to other.
 */
type FloatingRateIndexExternalMappings struct {
	IsoCode FloatingRateIndexExternalMap
}

/**
 * This type holds parameters defining the fixingt time and offset for a floating rate index.
 */
type FloatingRateIndexFixingDetails struct {
	/**
	 * Parameters defining the normal fixing time (can vary by index tenor / designated maturity).
	 */
	FixingTime FloatingRateIndexFixingTime
	/**
	 * Parameters defining the normal fixing offset (can vary by index tenor / designated maturity).
	 */
	FixingOffset BusinessDayOffset
}

/**
 * This type holds parameters defining the normal fixing offset for a floating rate index.
 */
type FloatingRateIndexFixingOffset struct {
	/**
	 * A time period multiplier, e.g. 1, 2 or 3 etc. A negative value can be used when specifying an offset relative to another date, e.g. -2 days.
	 */
	PeriodMultiplier int
	/**
	 * A time period, e.g. a day, week, month or year of the stream. If the periodMultiplier value is 0 (zero) then period must contain the value D (day).
	 */
	Period PeriodEnum
	Meta   MetaFields
	/**
	 * The business centers for the offset.
	 */
	BusinessCenters BusinessCenters
	/**
	 * Legal text that underlies the Fixing Offset. ISDA Fixing Offset Definition. (e.g. One day that is either a Sydney Business Day or a Melbourne Business Day following the Reset Date)
	 */
	FixingOffsetDefinition string
	/**
	 * Fixing Offset Reason
	 */
	FixingOffsetReason string
	/**
	 * Allows a reason to be specified for using the alternative fixing offset.
	 */
	DesignatedMaturity string
}

/**
 * This type holds parameters defining the normal fixing time for a floating rate index.
 */
type FloatingRateIndexFixingTime struct {
	/**
	 * A time specified in hh:mm:ss format where the second component must be '00', e.g. 11am would be represented as 11:00:00.
	 */
	HourMinuteTime time.Time
	/**
	 * A code identifying a business day calendar location. A business day calendar location is drawn from the list identified by the business day calendar location enumeration.
	 */
	BusinessCenter FieldWithMeta
	/**
	 * Allows a designed maturity to be specified for the fixing time.
	 */
	DesignatedMaturity string
	/**
	 * Legal text that underlies the Fixing Time. ISDA Fixing Time Definition. (e.g. 09:30, Sydney time).
	 */
	FixingTimeDefinition string
	/**
	 * Fixing Reason
	 */
	FixingReason string
}

type FloatingRateIndexIdentification struct {
	/**
	 * The reference index that is used to specify the floating interest rate. The FpML standard maintains the list of such indices, which are positioned as enumeration values as part of the CDM.
	 */
	FloatingRateIndex FieldWithMeta
	/**
	 * FRO currency - 3 character ISO currrency code
	 */
	Currency ISOCurrencyCodeEnum
	/**
	 * FRO type (e.g. OIS)
	 */
	FroType string
}

/**
 * A map for a single FRO to or from an equivalent or similar FRO in a different contractual definitions version.
 */
type FloatingRateIndexMap struct {
	/**
	 *  The FRO name that is being mapped to/from.
	 */
	Index []FloatingRateIndexEnum
	/**
	 * Contractual Definition to which the map applies. Includes Document Type and Document Version
	 */
	ContractualDefinitionIdentifier ContractualDefinitionIdentifier
	/**
	 * Corresponds to the unique identifier of the Contractual Definition to which the map applies
	 */
	Identifier Identifier
}

/**
 * This type defines mappings between FROs in different definitional versions.
 */
type FloatingRateIndexMappings struct {
	/**
	 * The successor FRO that this index maps to.
	 */
	MapsTo FloatingRateIndexMap
	/**
	 * The predecessor FRO(s) that this index maps to.
	 */
	MapsFrom []FloatingRateIndexMap
}

/**
 * Type for reporting the details of the rate treatment.  This could potentially be replaced by the existing FloatingRateDefinition type , but this is slightly more detailed.
 */
type FloatingRateProcessingDetails struct {
	/**
	 * The raw or untreated rate, prior to any of the rate treatments.
	 */
	RawRate              float64
	ProcessingParameters FloatingRateProcessingParameters
	/**
	 * The value of the rate after processing.
	 */
	ProcessedRate float64
	/**
	 * The value of the processed rate without the spread applied, for subsequent compounding, etc.
	 */
	SpreadExclusiveRate float64
}

/**
 * Type to hold the processing parameters that should be or were used to calculate a floating amount.  These parameters can vary over a schedule so this type holds the acutal values applicable to this calculation.
 */
type FloatingRateProcessingParameters struct {
	/**
	 * The rate to be applied for the initial period.
	 */
	InitialRate Price
	/**
	 * floating rate multiplier.
	 */
	Multiplier float64
	/**
	 * spread to be added to the floating rate.
	 */
	Spread float64
	/**
	 * US rate treatment (Bond Equivalent Yield or Money Market Yield, if applicable.
	 */
	Treatment RateTreatmentEnum
	/**
	 * capt to be applied to the floating rate.
	 */
	CapRate float64
	/**
	 * floor to be applied to the floating rate.
	 */
	FloorRate float64
	/**
	 * THe final rate rounding to be applied.
	 */
	Rounding Rounding
	/**
	 * How to handle negative interest rates.
	 */
	NegativeTreatment NegativeInterestRateTreatmentEnum
}

/**
 * Type for reporting the raw (untreated) observed or calculated rate for a calculation period.  If this is a calculated rate, it allows details of the observations and the resulting rate to be returned.
 */
type FloatingRateSettingDetails struct {
	/**
	 * Calculated rate details (observation dates, values, and weights).
	 */
	CalculationDetails CalculatedRateDetails
	/**
	 * The day upon which the rate was observed (for term rates).
	 */
	ObservationDate time.Time
	/**
	 * The day for which the rate is needed (e.g. period beginning or end date).
	 */
	ResetDate time.Time
	/**
	 * The resulting rate that was observed or calculated.
	 */
	FloatingRate float64
}

/**
 * A class to specify the floating interest rate by extending the floating rate definition with a set of attributes that specify such rate: the initial value specified as part of the trade, the rounding convention, the averaging method and the negative interest rate treatment.
 */
type FloatingRateSpecification struct {
	RateOption ReferenceWithMeta
	/**
	 * The ISDA Spread or a Spread schedule expressed as explicit spreads and dates. In the case of a schedule, the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The spread is a per annum rate, expressed as a decimal. For purposes of determining a calculation period amount, if positive the spread will be added to the floating rate and if negative the spread will be subtracted from the floating rate. A positive 10 basis point (0.1%) spread would be represented as 0.001.
	 */
	SpreadSchedule SpreadSchedule
	/**
	 * The cap rate or cap rate schedule, if any, which applies to the floating rate. The cap rate (strike) is only required where the floating rate on a swap stream is capped at a certain level. A cap rate schedule is expressed as explicit cap rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The cap rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A cap rate of 5% would be represented as 0.05.
	 */
	CapRateSchedule StrikeSchedule
	/**
	 * The floor rate or floor rate schedule, if any, which applies to the floating rate. The floor rate (strike) is only required where the floating rate on a swap stream is floored at a certain strike level. A floor rate schedule is expressed as explicit floor rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The floor rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A floor rate of 5% would be represented as 0.05.
	 */
	FloorRateSchedule StrikeSchedule
	Meta              MetaFields
	/**
	 * A rate multiplier or multiplier schedule to apply to the floating rate. A multiplier schedule is expressed as explicit multipliers and dates. In the case of a schedule, the step dates may be subject to adjustment in accordance with any adjustments specified in the calculationPeriodDatesAdjustments. The multiplier can be a positive or negative decimal. This element should only be included if the multiplier is not equal to 1 (one) for the term of the stream.
	 */
	FloatingRateMultiplierSchedule RateSchedule
	/**
	 * The specification of any rate conversion which needs to be applied to the observed rate before being used in any calculations. The two common conversions are for securities quoted on a bank discount basis which will need to be converted to either a Money Market Yield or Bond Equivalent Yield. See the Annex to the 2000 ISDA Definitions, Section 7.3. Certain General Definitions Relating to Floating Rate Options, paragraphs (g) and (h) for definitions of these terms.
	 */
	RateTreatment RateTreatmentEnum
	/**
	 * Support for modular calculated rates, such such as lockout compound calculations.
	 */
	CalculationParameters FloatingRateCalculationParameters
	/**
	 * Definition of any fallback rate that may be applicable.
	 */
	FallbackRate FallbackRateParameters
	/**
	 * The initial floating rate reset agreed between the principal parties involved in the trade. This is assumed to be the first required reset rate for the first regular calculation period. It should only be included when the rate is not equal to the rate published on the source implied by the floating rate index. An initial rate of 5% would be represented as 0.05.
	 */
	InitialRate Price
	/**
	 * The rounding convention to apply to the final rate used in determination of a calculation period amount.
	 */
	FinalRateRounding Rounding
	/**
	 * If averaging is applicable, this component specifies whether a weighted or unweighted average method of calculation is to be used. The component must only be included when averaging applies.
	 */
	AveragingMethod AveragingWeightingMethodEnum
	/**
	 * The specification of any provisions for calculating payment obligations when a floating rate is negative (either due to a quoted negative floating rate or by operation of a spread that is subtracted from the floating rate).
	 */
	NegativeInterestRateTreatment NegativeInterestRateTreatmentEnum
}

/**
 * From FpML: A type defining either a spot or forward FX transactions.
 */
type ForeignExchange struct {
	/**
	 * This is the first of the two currency flows that define a single leg of a standard foreign exchange transaction.
	 */
	ExchangedCurrency1 Cashflow
	/**
	 * This is the second of the two currency flows that define a single leg of a standard foreign exchange transaction.
	 */
	ExchangedCurrency2 Cashflow
	/**
	 * A tenor expressed as a period type and multiplier (e.g. 1D, 1Y, etc.)
	 */
	TenorPeriod Period
}

/**
 * Specification of a rate based on the exchange of a pair of cash assets in specific currencies, e.g. USD versus GBP.
 */
type ForeignExchangeRateIndex struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * A description of the Index.
	 */
	Name FieldWithMeta
	/**
	 * The organisation that creates or maintains the Index.
	 */
	Provider LegalEntity
	/**
	 * The Asset Class of the Index.
	 */
	AssetClass AssetClassEnum
	/**
	 * Describes the composition of a rate that has been quoted or is to be quoted.
	 */
	QuotedCurrencyPair FieldWithMeta
	/**
	 * Specifies the primary source from which a rate should be observed.
	 */
	PrimaryFxSpotRateSource InformationSource
	/**
	 * Specifies an alternative, or secondary, source from which a rate should be observed.
	 */
	SecondaryFxSpotRateSource InformationSource
}

/**
 * A class to specify party specific elections when a Collateral Transfer Agreement is governed by French Law.
 */
type FrenchLawAddendum struct {
	/**
	 * The qualification of whether the French Law Addendum is deemed applicable by the parties (True) or not (False).
	 */
	IsApplicable bool
	/**
	 * The parties French Law Addendum Elections.
	 */
	PartyElection []FrenchLawAddendumElection
}

/**
 * A class to specify party specific French Law Addendum language.
 */
type FrenchLawAddendumElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The qualification of whether the party elects specific language.
	 */
	IsApplicable bool
	/**
	 * The party specific language to be included in the agreement.
	 */
	AddendumLanguage string
}

/**
 * A class for defining a date frequency, e.g. one day, three months, through the combination of an integer value and a standardized period value that is specified as part of an enumeration.
 */
type Frequency struct {
	/**
	 * A time period multiplier, e.g. 1, 2, or 3. If the period value is T (Term) then period multiplier must contain the value 1.
	 */
	PeriodMultiplier int
	/**
	 * A time period, e.g. a day, week, month, year or term of the stream.
	 */
	Period PeriodExtendedEnum
	Meta   MetaFields
}

/**
 * FRO History
 */
type FroHistory struct {
	/**
	 * The date the Floating Rate Option was added to the 2006 Definitions or 2021 Floating Rate Matrix. (e.g. 2017/04/06)
	 */
	StartDate time.Time
	/**
	 * The supplement or version the FRO was first added to the 2006 Definitions or 2021 Floating Rate Matrix. (e.g. S52)
	 */
	FirstDefinedIn ContractualDefinition
	/**
	 * The date the Floating Rate Option was last updated in the 2006 Definitions or 2021 Floating Rate Matrix. (e.g. 2021/06/11)
	 */
	UpdateDate time.Time
	/**
	 * The supplement or version the FRO was last updated in the 2006 Definitions or 2021 Floating Rate Matrix. (e.g. FRO-M-V1)
	 */
	LastUpdatedIn ContractualDefinition
	/**
	 * The date the Floating Rate Option was removed from the 2006 Definitions or 2021 Floating Rate Matrix. (e.g. 2014/01/01)
	 */
	EndDate time.Time
}

/**
 * A class defining a currency and a future value date.
 */
type FutureValueAmount struct {
	Quantity ReferenceWithMeta
	/**
	 * The currency in which the an amount is denominated. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	Currency FieldWithMeta
	/**
	 * The number of days from the adjusted calculation period start date to the adjusted value date, calculated in accordance with the applicable day count fraction.
	 */
	CalculationPeriodNumberOfDays int
	/**
	 * Adjusted value date of the future value amount.
	 */
	ValueDate time.Time
}

/**
 * TransactionAdditionalTerms which apply to the CurrencyPair asset class.
 */
type FxAdditionalTerms struct {
	DisruptionEvents   FxDisruptionEvents
	DeterminationTerms DeterminationRolesAndTerms
	NoFaultTermination bool
}

type FxBenchmarkObligationDefault struct {
	BenchmarkObligationDefaultIsApplicable bool
	LocalSubstituteProvisionType           FxSubstitutionProvisionTypeEnum
	SpecifiedValue                         SpecifiedValueEnum
	EscrowArrangement                      EscrowArrangement
}

/**
 * Additional Provisions for a Confirmation of an FX or Currency Option Transaction
 */
type FxDisruptionEvents struct {
	IllegalityOrImpossibility FxIllegalityOrImpossibilityEnum
	ForceMajeureOrActOfSState FxForceMajeureOrActOfSStateEnum
	/**
	 * Additional Provisions for a Confirmation of an FX or Currency Option Transaction.
	 */
	EventCurrency                        EventCurrency
	PriceSourceDisruption                FxPriceSourceDisruption
	BenchmarkObligationDefault           FxBenchmarkObligationDefault
	DualExchangeRate                     FxDualExchangeRate
	InconvertibilityOrNonTransferability FxInconvertibilityOrNonTransferability
	GovernmentalAuthorityDefault         GovernmentalAuthorityDefault
	Illiquidity                          Illiquidity
	MaterialChangeInCircumstance         MaterialChangeInCircumstance
	Nationalization                      Nationalization
	PriceMateriality                     PriceMateriality
	/**
	 * Where parties may optionaly describe any extra bespoke agreements about fallback procedure attached to a particular determination role.
	 */
	FallbackLanguageBespokeTerms Clause
}

type FxDualExchangeRate struct {
	DualExchangeRateIsApplicable bool
	/**
	 * The method, prioritised by the order it is listed in this element, to get a replacement rate for the disrupted settlement rate option.
	 */
	FallbackReferencePrice FallbackReferencePrice
	EscrowArrangement      EscrowArrangement
}

/**
 * A type for defining FX Features.
 */
type FxFeature struct {
	/**
	 * Specifies the reference currency of the trade.
	 */
	ReferenceCurrency FieldWithMeta
	/**
	 * If 'Composite' is specified as the Settlement Type in the relevant Transaction Supplement, an amount in the Settlement Currency, determined by the Calculation Agent as being equal to the number of Options exercised or deemed exercised, multiplied by: (Settlement Price – Strike Price) / (Strike Price – Settlement Price) x Multiplier provided that if the above is equal to a negative amount the Option Cash Settlement Amount shall be deemed to be zero.
	 */
	Composite Composite
	/**
	 * If 'Quanto' is specified as the Settlement Type in the relevant Transaction Supplement, an amount, as determined by the Calculation Agent in accordance with the Section 8.2 of the Equity Definitions.
	 */
	Quanto Quanto
	/**
	 * If 'Cross-Currency' is specified as the Settlement Type in the relevant Transaction Supplement, an amount in the Settlement Currency, determined by the Calculation Agent as being equal to the number of Options exercised or deemed exercised, multiplied by: (Settlement Price – Strike Price) / (Strike Price – Settlement Price) x Multiplier x one unit of the Reference Currency converted into an amount in the Settlement Currency using the rate of exchange of the Settlement Currency as quoted on the Reference Price Source on the Valuation Date, provided that if the above is equal to a negative amount the Option Cash Settlement Amount shall be deemed to be zero.
	 */
	CrossCurrency Composite
}

/**
 * Extends the Offset structure to specify an FX fixing date as an offset to dates specified somewhere else in the document.
 */
type FxFixingDate struct {
	/**
	 * A time period multiplier, e.g. 1, 2 or 3 etc. A negative value can be used when specifying an offset relative to another date, e.g. -2 days.
	 */
	PeriodMultiplier int
	/**
	 * A time period, e.g. a day, week, month or year of the stream. If the periodMultiplier value is 0 (zero) then period must contain the value D (day).
	 */
	Period PeriodEnum
	Meta   MetaFields
	/**
	 * In the case of an offset specified as a number of days, this element defines whether consideration is given as to whether a day is a good business day or not. If a day type of business days is specified then non-business days are ignored when calculating the offset. The financial business centers to use for determination of business days are implied by the context in which this element is used. This element must only be included when the offset is specified as a number of days. If the offset is zero days then the dayType element should not be included.
	 */
	DayType DayTypeEnum
	/**
	 * The convention for adjusting a date if it would otherwise fall on a day that is not a business day, as specified by an ISDA convention (e.g. Following, Precedent).
	 */
	BusinessDayConvention BusinessDayConventionEnum
	BusinessCenters       BusinessCenters
	/**
	 * A reference to a set of financial business centers defined elsewhere in the document. This set of business centers is used to determine whether a particular day is a business day or not.
	 */
	BusinessCentersReference ReferenceWithMeta
	/**
	 * The payment date references on which settlements in non-deliverable currency are due and will then have to be converted according to the terms specified through the other parts of the nonDeliverableSettlement structure.
	 */
	DateRelativeToPaymentDates DateRelativeToPaymentDates
	/**
	 * The calculation period references on which settlements in non-deliverable currency are due and will then have to be converted according to the terms specified through the other parts of the nonDeliverableSettlement structure. Implemented for Brazilian-CDI swaps where it will refer to the termination date of the appropriate leg.
	 */
	DateRelativeToCalculationPeriodDates DateRelativeToCalculationPeriodDates
	/**
	 * The calculation period references on which settlements in non-deliverable currency are due and will then have to be converted according to the terms specified through the other parts of the nonDeliverableSettlement structure. Implemented for Brazilian-CDI swaps where it will refer to the termination date of the appropriate leg.
	 */
	DateRelativeToValuationDates DateRelativeToValuationDates
	/**
	 * Describes the specific date when a non-deliverable forward or cash-settled option will 'fix' against a particular rate, which will be used to compute the ultimate cash settlement. This element should be omitted where a single, discrete fixing date cannot be identified e.g. on an american option, where fixing may occur at any date on a continuous range.  This attribute was formerly part of 'fxSettlementTerms', which is now being harmonised into a common 'CashSettlementTerms' that includes a 'ValuationDate'.
	 */
	FxFixingDate AdjustableOrRelativeDate
}

/**
 * A class to specify the reference currency for the purpose of specifying the FX Haircut relating to a posting obligation, as being either the Termination Currency or an FX Designated Currency.
 */
type FxHaircutCurrency struct {
	/**
	 * The reference currency for the purpose of specifying the FX Haircut relating to a posting obligation is the Termination Currency when the Boolean value is set to True.
	 */
	IsTerminationCurrency bool
	/**
	 * When specified, the reference currency for the purpose of specifying the FX Haircut relating to a posting obligation. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	FxDesignatedCurrency FieldWithMeta
}

type FxInconvertibilityOrNonTransferability struct {
	GeneralInconvertibility    GeneralInconvertibility
	GeneralNonTransferability  GeneralNonTransferability
	SpecificInconvertibility   SpecificInconvertibility
	SpecificNonTransferability SpecificNonTransferability
}

/**
 * Information source specific to Foreign Exchange products.
 */
type FxInformationSource struct {
	/**
	 * An information source for obtaining a market data point. For example Bloomberg, Reuters, Telerate, etc.
	 */
	SourceProvider FieldWithMeta
	/**
	 * A specific page for the source for obtaining a market data point. In FpML, this is specified as a scheme, rateSourcePageScheme, for which no coding Scheme or URI is specified.
	 */
	SourcePage FieldWithMeta
	/**
	 * The heading for the source on a given source page.
	 */
	SourcePageHeading string
	/**
	 * The time that the fixing will be taken along with a business center to define the time zone.
	 */
	FixingTime BusinessCenterTime
}

/**
 * A data to:  describe the cashflow representation for FX linked notionals.
 */
type FxLinkedNotionalAmount struct {
	/**
	 * The reset date.
	 */
	ResetDate time.Time
	/**
	 * The date on which the FX spot rate is observed. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedFxSpotFixingDate time.Time
	/**
	 * The actual observed FX spot rate.
	 */
	ObservedFxSpotRate float64
	/**
	 * The calculation period notional amount.
	 */
	NotionalAmount float64
}

/**
 * A data to:  describe a notional schedule where each notional that applies to a calculation period is calculated with reference to a notional amount or notional amount schedule in a different currency by means of a spot currency exchange rate which is normally observed at the beginning of each period.
 */
type FxLinkedNotionalSchedule struct {
	/**
	 * The currency of the varying notional amount, i.e. the notional amount being determined periodically based on observation of a spot currency exchange rate. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	VaryingNotionalCurrency FieldWithMeta
	/**
	 * The dates on which spot currency exchange rates are observed for purposes of determining the varying notional currency amount that will apply to a calculation period.
	 */
	VaryingNotionalFixingDates RelativeDateOffset
	/**
	 * The information source and time at which the spot currency exchange rate will be observed.
	 */
	FxSpotRateSource FxSpotRateSource
	/**
	 * The time at which the spot currency exchange rate will be observed. It is specified as a time in a business day calendar location, e.g. 11:00am London time.
	 */
	FixingTime BusinessCenterTime
	/**
	 * The dates on which interim exchanges of notional are paid. Interim exchanges will arise as a result of changes in the spot currency exchange amount or changes in the constant notional schedule (e.g. amortisation).
	 */
	VaryingNotionalInterimExchangePaymentDates RelativeDateOffset
}

type FxPriceSourceDisruption struct {
	PriceSourceDisruptionIsApplicable bool
	/**
	 * The method, prioritised by the order it is listed in this element, to get a replacement rate for the disrupted settlement rate option.
	 */
	FallbackReferencePrice SettlementRateOption
	EscrowArrangement      EscrowArrangement
}

/**
 * A class describing the rate of a currency conversion: pair of currency, quotation mode and exchange rate.
 */
type FxRate struct {
	/**
	 * Defines the two currencies for an FX trade and the quotation relationship between the two currencies.
	 */
	QuotedCurrencyPair QuotedCurrencyPair
	/**
	 * The rate of exchange between the two currencies of the leg of a deal. Must be specified with a quote basis.
	 */
	Rate float64
}

/**
 * Describes a rate source to be fixed and the date the fixing occurs
 */
type FxRateSourceFixing struct {
	SettlementRateSource FxSettlementRateSource
	/**
	 * The date on which the fixing is scheduled to occur.
	 */
	FixingDate AdjustableDate
}

/**
 * The source of the Foreign Exchange settlement rate.
 */
type FxSettlementRateSource struct {
	/**
	 * Indicates that an officially defined rate settlement rate option will be the used for the fixing.
	 */
	SettlementRateOption FieldWithMeta
	/**
	 * Indicates that a non-standard rate source will be used for the fixing.
	 */
	NonstandardSettlementRate FxInformationSource
}

/**
 * A class defining the rate source and fixing time for an FX rate.
 */
type FxSpotRateSource struct {
	/**
	 * The primary source for where the rate observation will occur. Will typically be either a page or a reference bank published rate.
	 */
	PrimarySource InformationSource
	/**
	 * An alternative, or secondary, source for where the rate observation will occur. Will typically be either a page or a reference bank published rate.
	 */
	SecondarySource InformationSource
}

type GeneralInconvertibility struct {
	GeneralInconvertibilityIsApplicable bool
	LocalSubstituteProvisionType        FxSubstitutionProvisionTypeEnum
	/**
	 * Specifies how long to wait to get a quote from a settlement rate option upon a price source disruption.
	 */
	MaximumDaysOfDisruption  ValuationPostponement
	NonDeliverableSubstitute NonDeliverableSubstitute
	/**
	 * Refers to the segregation deposit procedure parties may want to commmit into during the time a Disruption Event is effective
	 */
	EscrowArrangement EscrowArrangement
}

type GeneralNonTransferability struct {
	GeneralNonTransferabilityIsApplicable bool
	LocalSubstituteProvisionType          FxSubstitutionProvisionTypeEnum
	/**
	 * Specifies how long to wait to get a quote from a settlement rate option upon a price source disruption.
	 */
	MaximumDaysOfDisruption  ValuationPostponement
	NonDeliverableSubstitute NonDeliverableSubstitute
	/**
	 * Refers to the segregation deposit procedure parties may want to commmit into during the time a Disruption Event is effective
	 */
	EscrowArrangement EscrowArrangement
}

/**
 * A class to specify the ISDA SIMM as the Method for all Covered Transactions with respect to all Regimes.
 */
type GeneralSimmElections struct {
	/**
	 * The qualification of the ISDA SIMM version that is specified for all Covered Transactions.
	 */
	SimmVersion SimmVersion
	/**
	 * The SIMM Calculation Currency, as specified for each of the parties to the CSA Initial Margin.
	 */
	SimmCalculationCurrency SimmCalculationCurrency
}

/**
 *  A class specifying a set of non-monetary terms for the Credit Derivative Transaction, including the buyer and seller and selected items from the ISDA 2014 Credit Definition article II, such as the reference obligation and related terms. The CDM GeneralTerms class corresponds to the FpML GeneralTerms complex type, except that the effectiveDate and scheduledTerminationDate have been positioned as part of the InterestRatePayout class in the CDM instead of in GeneralTerms.
 */
type GeneralTerms struct {
	/**
	 * This attribute contains all the terms relevant to defining the reference entity and reference obligation(s).
	 */
	ReferenceInformation ReferenceInformation
	/**
	 * This attribute contains all the terms relevant to the underlying Index.
	 */
	IndexReferenceInformation CreditIndex
	/**
	 * This attribute contains all the terms relevant to defining the Credit Default Swap Basket.
	 */
	BasketReferenceInformation BasketReferenceInformation
	/**
	 * This attribute is used for representing information contained in the Additional Terms field of the 2003 Master Credit Derivatives confirm.
	 */
	AdditionalTerm []FieldWithMeta
	/**
	 * Value of this attribute set to 'true' indicates that substitution is applicable.
	 */
	Substitution bool
	/**
	 * Value of this attribute set to 'true' indicates that modified equity delivery is applicable.
	 */
	ModifiedEquityDelivery bool
}

/**
 * The set of elections that define a GMRA
 */
type GlobalMasterRepoAgreement struct {
}

/**
 * The set of elections that define a GMSLA.
 */
type GlobalMasterSecuritiesLendingAgreement struct {
}

type GovernmentalAuthorityDefault struct {
	GovernmentalAuthorityDefaultIsApplicable bool
	EscrowArrangement                        EscrowArrangement
	LocalSubstituteProvisionType             FxSubstitutionProvisionTypeEnum
	SpecifiedValue                           SpecifiedValueEnum
	MaximumDaysOfDisruption                  ValuationPostponement
}

type GracePeriodExtension struct {
	/**
	 * Indicates whether the grace period extension provision is applicable.
	 */
	Applicable bool
	/**
	 * The number of calendar or business days after any due date that the reference entity has to fulfil its obligations before a failure to pay credit event is deemed to have occurred. ISDA 2003 Term: Grace Period.
	 */
	GracePeriod Offset
}

/**
 * Specifies the conditions that need to be satisfied before the Secured Party or Custodian acting on its behalf can hold and use posted collateral.
 */
type HoldingAndUsingPostedCollateral struct {
	/**
	 * The parties' elections for the holding and using of posted collateral.
	 */
	PartyElection []HoldingAndUsingPostedCollateralElection
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * Specifies the parties' elections related to the holding and using of posted collateral.
 */
type HoldingAndUsingPostedCollateralElection struct {
	/**
	 * The party to which the elections apply.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The specification of the conditions under which a party and its custodian(s) are entitled to hold posted collateral.
	 */
	EligibilityToHoldCollateral EligibilityToHoldCollateral
	/**
	 * Indicates whether the party has the right to reuse, rehypothecate, or otherwise make use of the posted collateral. 'True' if the party has the right,'False' if the party does not have the right.
	 */
	UseOfPostedCollateral bool
}

type Id struct {
	Lei string
}

/**
 * Attaches an identifier to a collection of objects, when those objects themselves can each be represented by an identifier. One use case is the representation of package transactions, where each component is a separate trade with its own identifier, and those trades are linked together as a package with its own identifier. The data type has been named generically rather than referring to 'packages' as it may have a number of other uses.
 */
type IdentifiedList struct {
	/**
	 * The identifier for the list. In the case of a package transaction, this would be the package identifier. This attribute is mandatory to allow the list itself to be identified.
	 */
	ListId Identifier
	/**
	 * Identifiers for each component of the list. Since the data type is used to link multiple identified objects together, at least 2 components are required in the list. Creating an identified list with only 1 identified component has been deemed unnecessary, because it would just create a redundant identifier.
	 */
	ComponentId []Identifier
	/**
	 * The price of the package.
	 */
	Price Price
	Meta  MetaFields
}

/**
 * A class to specify a generic identifier, applicable to CDM artefacts such as executions, contracts, lifecycle events and legal documents. An issuer can be associated with the actual identifier value as a way to properly qualify it.
 */
type Identifier struct {
	/**
	 * The identifier issuer, when specified by reference to a party specified as part of the transaction.
	 */
	IssuerReference ReferenceWithMeta
	/**
	 * The identifier issuer, when specified explicitly alongside the identifier value (instead of being specified by reference to a party).
	 */
	Issuer FieldWithMeta
	/**
	 * The identifier value. This level of indirection between the issuer and the identifier and its version provides the ability to associate multiple identifiers to one issuer, consistently with the FpML PartyTradeIdentifier.
	 */
	AssignedIdentifier []AssignedIdentifier
	Meta               MetaFields
}

type Illiquidity struct {
	IlliquidityIsApplicable bool
	EscrowArrangement       EscrowArrangement
	MinimumAmount           Quantity
	/**
	 * The method, prioritised by the order it is listed in this element, to get a replacement rate for the disrupted settlement rate option.
	 */
	FallbackReferencePrice   SettlementRateOption
	IlliquidityValuationDate AdjustedRelativeDateOffset
}

/**
 * A class specifying the Independent Amount as the combination of a payer/receiver, a payment amount, a payment date and an associated payment calculation rule.
 */
type IndependentAmount struct {
	/**
	 * The party responsible for making the payments defined by this structure.
	 */
	PayerPartyReference ReferenceWithMeta
	/**
	 * A reference to the account responsible for making the payments defined by this structure.
	 */
	PayerAccountReference ReferenceWithMeta
	/**
	 * The party that receives the payments corresponding to this structure.
	 */
	ReceiverPartyReference ReferenceWithMeta
	/**
	 * A reference to the account that receives the payments corresponding to this structure.
	 */
	ReceiverAccountReference ReferenceWithMeta
	/**
	 * An attribute that specifies a payment as the combination of a payment amount, a payment date and an associated payment calculation rule.
	 */
	PaymentDetail []PaymentDetail
}

/**
 * A logical container designed to hold a set of related data. In this case, designed to hold a set of information required when Independent Amount is determined using credit ratings.
 */
type IndependentAmountRatings struct {
	/**
	 * Specifies how multiple credit ratings are compared when determining the applicable Independent Amount, and whether that measure is highest, lowest, or a comparison of either.
	 */
	Compare IndependentAmountCompareEnum
	/**
	 * Specifies the party to whom the rating agency rating applies to for the purposes of determining the Independent Amount.
	 */
	RatedParty RatedPartyEnum
	/**
	 * Details the Named Affiliate where the Rated Party is Named Affiliate.
	 */
	NamedAffiliate LegalEntity
	/**
	 * Details the Named Entity where the Rated Party is Named Entity.
	 */
	NamedEntity LegalEntity
	/**
	 * The relevant rating type.
	 */
	RatingType RatingTypeEnum
}

/**
 * An Index is an Observable which is computed based on the prices, rates or valuations of a number of assets that are tracked in a standardized way.  Examples include equity market indices as well as indices on interest rates, inflation and credit instruments.
 */
type Index struct {
	/**
	 * An index based on credit risk, typically composed using corporate debt instruments in a region or industry sector, e.g. the iTraxx indices.
	 */
	CreditIndex CreditIndex
	/**
	 * An index based on equity securities, e.g. the S&P 500.
	 */
	EquityIndex EquityIndex
	/**
	 * An index based in interest rates or inflation rates in a certain market.
	 */
	InterestRateIndex FieldWithMeta
	/**
	 * A rate based on the exchange of a pair of cash assets in specific currencies, e.g. USD versus GBP.
	 */
	ForeignExchangeRateIndex ForeignExchangeRateIndex
	/**
	 * An index created by a market participant which doesn't align with the other index types.
	 */
	OtherIndex OtherIndex
}

/**
 * Defines the specification of the consequences of Index Events
 */
type IndexAdjustmentEvents struct {
	/**
	 * Consequence of index modification.
	 */
	IndexModification IndexEventConsequenceEnum
	/**
	 * Consequence of index cancellation.
	 */
	IndexCancellation IndexEventConsequenceEnum
	/**
	 * Consequence of index disruption.
	 */
	IndexDisruption IndexEventConsequenceEnum
}

/**
 * Identifies an index by referencing an identifier.
 */
type IndexBase struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * A description of the Index.
	 */
	Name FieldWithMeta
	/**
	 * The organisation that creates or maintains the Index.
	 */
	Provider LegalEntity
	/**
	 * The Asset Class of the Index.
	 */
	AssetClass AssetClassEnum
}

/**
 * Defines the information needed to create a Index Transition Business Event.
 */
type IndexTransitionInstruction struct {
	/**
	 * Specifies both new floating rate index and spread adjustment for each leg to be updated.  The spread adjustment accounts for the difference between the old floating rate index relative to the new one. This spread amount is added to the existing spread to determine the new spread, which is applied from the specified effective date forward. In the case of the IBOR Fallback Rate Adjustments, the adjustment spread (also known as the Fallback Adjustment) accounts for two distinctions: i) the fact that the replacement Risk-Free Rate is an overnight rate while IBORs have term structures (e.g., 1, 3, 6-month LIBOR); and (ii) the historical spread differential between IBORs and their term equivalent Overnight Risk-Free Rate compounded rates.
	 */
	PriceQuantity []PriceQuantity
	/**
	 * Specifies the effective date of the index transition event. This is first date on which the floating rate calculation will use the new floating rate index and adjusted spread in the floating rate calculation.
	 */
	EffectiveDate time.Time
	/**
	 * Specifies the cash transfer that can optionally be tied to an index transition event.
	 */
	CashTransfer Transfer
}

type Indx struct {
	Nm Nm
}

/**
 * Identifies collateral that has been designated as not meeting the eligibility criteria set out in the agreement.
 */
type IneligibleCreditSupport struct {
	/**
	 * Specifies the date from which all credit support provided by a party is treated as ineligible under the agreement.
	 */
	TotalIneligibilityDate string
	/**
	 * Specifies the date from which the transfer of credit support by a party is treated as ineligible under the agreement.
	 */
	TransferIneligibilityDate string
	/**
	 * Identifies a party designated in the agreement for the purpose of applying particular rights, obligations, or elections.
	 */
	SpecifiedParty []CounterpartyRoleEnum
}

/**
 * Specification of an index that measures inflation in a specific market, e.g. the US Consumer Price Index.
 */
type InflationIndex struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * A description of the Index.
	 */
	Name FieldWithMeta
	/**
	 * The organisation that creates or maintains the Index.
	 */
	Provider LegalEntity
	/**
	 * The Asset Class of the Index.
	 */
	AssetClass AssetClassEnum
	/**
	 * The reference index that is used to specify the inflation interest rate.
	 */
	InflationRateIndex FieldWithMeta
	/**
	 * The ISDA Designated Maturity, i.e. the floating rate tenor.
	 */
	IndexTenor Period
}

/**
 * A data to:  specify the inflation rate.
 */
type InflationRateSpecification struct {
	RateOption ReferenceWithMeta
	/**
	 * The ISDA Spread or a Spread schedule expressed as explicit spreads and dates. In the case of a schedule, the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The spread is a per annum rate, expressed as a decimal. For purposes of determining a calculation period amount, if positive the spread will be added to the floating rate and if negative the spread will be subtracted from the floating rate. A positive 10 basis point (0.1%) spread would be represented as 0.001.
	 */
	SpreadSchedule SpreadSchedule
	/**
	 * The cap rate or cap rate schedule, if any, which applies to the floating rate. The cap rate (strike) is only required where the floating rate on a swap stream is capped at a certain level. A cap rate schedule is expressed as explicit cap rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The cap rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A cap rate of 5% would be represented as 0.05.
	 */
	CapRateSchedule StrikeSchedule
	/**
	 * The floor rate or floor rate schedule, if any, which applies to the floating rate. The floor rate (strike) is only required where the floating rate on a swap stream is floored at a certain strike level. A floor rate schedule is expressed as explicit floor rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The floor rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A floor rate of 5% would be represented as 0.05.
	 */
	FloorRateSchedule StrikeSchedule
	Meta              MetaFields
	/**
	 * A rate multiplier or multiplier schedule to apply to the floating rate. A multiplier schedule is expressed as explicit multipliers and dates. In the case of a schedule, the step dates may be subject to adjustment in accordance with any adjustments specified in the calculationPeriodDatesAdjustments. The multiplier can be a positive or negative decimal. This element should only be included if the multiplier is not equal to 1 (one) for the term of the stream.
	 */
	FloatingRateMultiplierSchedule RateSchedule
	/**
	 * The specification of any rate conversion which needs to be applied to the observed rate before being used in any calculations. The two common conversions are for securities quoted on a bank discount basis which will need to be converted to either a Money Market Yield or Bond Equivalent Yield. See the Annex to the 2000 ISDA Definitions, Section 7.3. Certain General Definitions Relating to Floating Rate Options, paragraphs (g) and (h) for definitions of these terms.
	 */
	RateTreatment RateTreatmentEnum
	/**
	 * Support for modular calculated rates, such such as lockout compound calculations.
	 */
	CalculationParameters FloatingRateCalculationParameters
	/**
	 * Definition of any fallback rate that may be applicable.
	 */
	FallbackRate FallbackRateParameters
	/**
	 * The initial floating rate reset agreed between the principal parties involved in the trade. This is assumed to be the first required reset rate for the first regular calculation period. It should only be included when the rate is not equal to the rate published on the source implied by the floating rate index. An initial rate of 5% would be represented as 0.05.
	 */
	InitialRate Price
	/**
	 * The rounding convention to apply to the final rate used in determination of a calculation period amount.
	 */
	FinalRateRounding Rounding
	/**
	 * If averaging is applicable, this component specifies whether a weighted or unweighted average method of calculation is to be used. The component must only be included when averaging applies.
	 */
	AveragingMethod AveragingWeightingMethodEnum
	/**
	 * The specification of any provisions for calculating payment obligations when a floating rate is negative (either due to a quoted negative floating rate or by operation of a spread that is subtracted from the floating rate).
	 */
	NegativeInterestRateTreatment NegativeInterestRateTreatmentEnum
	/**
	 * An off-setting period from the payment date which determines the reference period for which the inflation index is observed.
	 */
	InflationLag Offset
	/**
	 * The reference source such as Reuters or Bloomberg. FpML specifies indexSource to be of type rateSourcePageScheme, but without specifying actual values.
	 */
	IndexSource FieldWithMeta
	/**
	 * The current main publication source such as relevant web site or a government body. FpML specifies mainPublication to be of type mainPublicationSource, but without specifying actual values.
	 */
	MainPublication FieldWithMeta
	/**
	 * The method used when calculating the Inflation Index Level from multiple points. The most common is Linear.
	 */
	InterpolationMethod FieldWithMeta
	/**
	 * Initial known index level for the first calculation period.
	 */
	InitialIndexLevel float64
	/**
	 * The applicability of a fallback bond as defined in the 2006 ISDA Inflation Derivatives Definitions, sections 1.3 and 1.8.
	 */
	FallbackBondApplicable bool
	/**
	 * Indicates how to use the inflation index to calculate the payment (e.g. Ratio, Return, Spread). Added for Inflation Asset Swap
	 */
	CalculationMethod InflationCalculationMethodEnum
	/**
	 * Indicates the style of how the inflation index calculates the payment (e.g. YearOnYear, ZeroCoupon).
	 */
	CalculationStyle InflationCalculationStyleEnum
	/**
	 * To be specified only for products that embed a redemption payment.
	 */
	FinalPrincipalExchangeCalculation FinalPrincipalExchangeCalculationEnum
}

/**
 * A class defining the source for a piece of information (e.g. a rate fix or an FX fixing). The attribute names have been adjusted from FpML to address the fact that the information is not limited to rates.
 */
type InformationSource struct {
	/**
	 * An information source for obtaining a market data point. For example Bloomberg, Reuters, Telerate, etc.
	 */
	SourceProvider FieldWithMeta
	/**
	 * A specific page for the source for obtaining a market data point. In FpML, this is specified as a scheme, rateSourcePageScheme, for which no coding Scheme or URI is specified.
	 */
	SourcePage FieldWithMeta
	/**
	 * The heading for the source on a given source page.
	 */
	SourcePageHeading string
}

/**
 * A CDM class which purpose is to specify the initial fixing date either alongside the FpML interest rate specification as an offset of another date, or alongside the credit derivative specification as an unadjusted date.
 */
type InitialFixingDate struct {
	RelativeDateOffset RelativeDateOffset
	InitialFixingDate  time.Time
}

/**
 * Instruction to a function that will be used to perform a business event
 */
type Instruction struct {
	/**
	 * Specifies the primitive instructions that will be used to call primitive event functions.
	 */
	PrimitiveInstruction PrimitiveInstruction
	/**
	 * Specifies the trade state that will be acted on by the primitive event functions.
	 */
	Before ReferenceWithMeta
}

/**
 * A type of Asset that is issued by one party to one or more others.
 */
type Instrument struct {
	/**
	 * A securitized derivative on another asset that is created by an exchange.
	 */
	ListedDerivative ListedDerivative
	/**
	 * An Asset that represents a loan or borrow obligation.
	 */
	Loan Loan
	/**
	 * An Asset that is issued by a party to be held by or transferred to others.
	 */
	Security Security
}

/**
 * Defines the common attributes for all Instrument data types.
 */
type InstrumentBase struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * Identifies the type of an instrument using an enumerated list.
	 */
	InstrumentType InstrumentTypeEnum
}

/**
 * A class to specify whether the Interest Adjustment is applicable and what its periodicity is.
 */
type InterestAdjustment struct {
	/**
	 * The Interest Adjustment is applicable when True and not applicable when False.
	 */
	IsApplicable bool
	/**
	 * The qualification of the Interest Adjustment periodicity.
	 */
	Periodicity InterestAdjustmentPeriodicity
}

/**
 * A class to specify the Interest Adjustment periodicity.
 */
type InterestAdjustmentPeriodicity struct {
	/**
	 * The Interest Adjustment periodicity when specified through a standardized election.
	 */
	StandardElection InterestAdjustmentPeriodicityEnum
	/**
	 * The Interest Adjustment periodicity when specified through a custom election.
	 */
	CustomElection string
}

/**
 * A class to specify the application of Interest Amount with respect to the Delivery Amount and the Return Amount.
 */
type InterestAmount struct {
	/**
	 * The application of Interest Amount with respect the Return Amount.
	 */
	ReturnAmount ReturnAmount
	/**
	 * The application of Interest Amount with respect the Delivery Amount.
	 */
	DeliveryAmount DeliveryAmount
}

/**
 * A class to specify the application of Interest Amount with respect to the Delivery Amount and the Return Amount.
 */
type InterestAmountApplication struct {
	/**
	 * The application of Interest Amount with respect the Return Amount.
	 */
	ReturnAmount ReturnAmount
	/**
	 * The application of Interest Amount with respect the Delivery Amount.
	 */
	DeliveryAmount DeliveryAmount
}

type InterestRateCurve struct {
	FloatingRateIndex FieldWithMeta
	Tenor             Period
}

/**
 * An index based in interest rates or inflation rates in a certain market.
 */
type InterestRateIndex struct {
	/**
	 * An interest rate index which can change over time, e.g. the SONIA (Sterling Overnight Index Average) in the UK.
	 */
	FloatingRateIndex FloatingRateIndex
	/**
	 * An index that measures inflation in a specific market, e.g. the US Consumer Price Index.
	 */
	InflationIndex InflationIndex
}

/**
 *  A class to specify all of the terms necessary to define and calculate a cash flow based on a fixed, a floating or an inflation index rate. The interest rate payout can be applied to interest rate swaps and FRA (which both have two associated interest rate payouts), credit default swaps (to represent the fee leg when subject to periodic payments) and equity swaps (to represent the funding leg). The associated globalKey denotes the ability to associate a hash value to the InterestRatePayout instantiations for the purpose of model cross-referencing, in support of functionality such as the event effect and the lineage.
 */
type InterestRatePayout struct {
	/**
	 * Canonical representation of the payer and receiver parties applicable to each payout leg.
	 */
	PayerReceiver PayerReceiver
	/**
	 * Each payout leg must implement the quantity concept as a 'resolvable' type, which allows for different payout legs to be linked to each other (e.g. in the case of cross-curreny products).
	 */
	PriceQuantity ResolvablePriceQuantity
	/**
	 * The specification of the principal exchange. Optional as only applicable in the case of cross-currency or zero-coupon swaps with a final payment.
	 */
	PrincipalPayment PrincipalPayments
	/**
	 * Each payout leg must specifies its settlement terms, including the delivery type (i.e. cash vs physical, and their respective terms), the transfer type (DvP etc.) and settlement date, if any.
	 */
	SettlementTerms SettlementTerms
	/**
	 * The specification of the rate value(s) applicable to the contract using either a floating rate calculation, a single fixed rate, a fixed rate schedule, or an inflation rate calculation.
	 */
	RateSpecification RateSpecification
	/**
	 * The day count fraction. The cardinality has been relaxed when compared with the FpML interest rate swap for the purpose of accommodating standardized credit default swaps which DCF is not explicitly stated as part of the economic terms. The data rule InterestRatePayout_dayCountFraction requires that the DCF be stated for interest rate products.
	 */
	DayCountFraction FieldWithMeta
	/**
	 * The parameters used to generate the calculation period dates schedule, including the specification of any initial or final stub calculation periods.
	 */
	CalculationPeriodDates CalculationPeriodDates
	/**
	 * The payment date schedule, as defined by the parameters that are needed to specify it, either in a parametric way or by reference to another schedule of dates (e.g. the reset dates).
	 */
	PaymentDates PaymentDates
	/**
	 * The payment date, where only one date is specified, as for the FRA product.
	 */
	PaymentDate AdjustableDate
	/**
	 * Applicable to CDS on MBS to specify whether payment delays are applicable to the fixed Amount. RMBS typically have a payment delay of 5 days between the coupon date of the reference obligation and the payment date of the synthetic swap. CMBS do not, on the other hand, with both payment dates being on the 25th of each month.
	 */
	PaymentDelay bool
	/**
	 * The reset dates schedule, i.e. the dates on which the new observed index value is applied for each period and the interest rate hence begins to accrue.
	 */
	ResetDates ResetDates
	/**
	 * The parameters specifying any discounting conventions that may apply. This element must only be included if discounting applies.
	 */
	DiscountingMethod DiscountingMethod
	/**
	 * If one or more calculation period contributes to a single payment amount this element specifies whether compounding is applicable and, if so, what compounding method is to be used. This element must only be included when more than one calculation period contributes to a single payment amount.
	 */
	CompoundingMethod CompoundingMethodEnum
	/**
	 * The cashflow representation of the swap stream.
	 */
	CashflowRepresentation CashflowRepresentation
	/**
	 * The stub calculation period amount parameters. This element must only be included if there is an initial or final stub calculation period. Even then, it must only be included if either the stub references a different floating rate tenor to the regular calculation periods, or if the stub is calculated as a linear interpolation of two different floating rate tenors, or if a specific stub rate or stub amount has been negotiated.
	 */
	StubPeriod StubPeriod
	/**
	 * Reference to a bond underlier to represent an asset swap or Condition Precedent Bond.
	 */
	BondReference BondReference
	/**
	 * Fixed Amount Calculation
	 */
	FixedAmount string
	/**
	 * Floating Amount Calculation
	 */
	FloatingAmount string
	/**
	 * Method by which spread is calculated. For example on an asset swap: 'ParPar' or 'Proceeds' may be the method indicated.
	 */
	SpreadCalculationMethod SpreadCalculationMethodEnum
}

/**
 * A class to specify the interest shortfall floating rate payment event.
 */
type InterestShortFall struct {
	/**
	 * Specifies the nature of the interest Shortfall cap (i.e. Fixed Cap or Variable Cap) in the case where it is applicable. ISDA 2003 Term: Interest Shortfall Cap.
	 */
	InterestShortfallCap InterestShortfallCapEnum
	Compounding          bool
	/**
	 * The rate source in the case of a variable cap.
	 */
	RateSource FieldWithMeta
}

/**
 * A data type that can be used to describe an inventory of securities.
 */
type Inventory struct {
	/**
	 * An array holding the list of inventory being described. Each element in the inventoryRecord array represents an individual piece of inventory i.e. a security.
	 */
	InventoryRecord []InventoryRecord
}

/**
 * An individual piece of inventory. This represents a single security.
 */
type InventoryRecord struct {
	/**
	 * Unique identifier for this record. This can be used to uniquely identify a specific piece of inventory.
	 */
	Identifer AssignedIdentifier
	/**
	 * The security details.
	 */
	Security Security
}

type InvstmtDcsnPrsn struct {
	Prsn Prsn
}

type IssuerAgencyRating struct {
	/**
	 * Represents an agency rating based on default risk and creditors claim in event of default associated with asset issuer.
	 */
	IssuerAgencyRating AgencyRatingCriteria
}

type IssuerCountryOfOrigin struct {
	/**
	 * Represents a filter on the issuing entity country of origin based on the ISO Standard 3166, which is the same as filtering by eligible Sovereigns.
	 */
	IssuerCountryOfOrigin ISOCountryCodeEnum
}

type IssuerName struct {
	/**
	 * Specifies the issuing entity name or LEI.
	 */
	IssuerName LegalEntity
}

/**
 * A class to specify Japanese Securities Provision elections.
 */
type JapaneseSecuritiesProvisions struct {
	/**
	 * Japanese Securities Provisions are applicable when True and Not Applicable when False.
	 */
	IsApplicable bool
	/**
	 * Recommended Japanese Securities Provisions are applicable when True, additional Provisions are specified when False.
	 */
	RelevantProvisionsElection bool
	/**
	 * Specific terms applicable to Recommended Japanese Securities Provisions.
	 */
	RelevantProvisionsTerms string
	/**
	 * Additional Amendments to Japanese Securities Provisions are specified when True, and not specified when False.
	 */
	AmendmentsToJapaneseProvisions bool
	/**
	 * Specific terms applicable to Additional Amendments to Japanese Securities Provisions.
	 */
	AmendmentsToJapaneseProvisionsTerms string
	/**
	 * Specification of whether Clearstream Event amendment language is included (true) or excluded (false).
	 */
	ClearstreamAmendmentToJapaneseProvisions bool
}

/**
 * A class to specify terms jurisdiction related terms.
 */
type JurisdictionRelatedTerms struct {
	/**
	 * The Japanese Securities Provisions election.
	 */
	JapaneseSecuritiesProvisions JapaneseSecuritiesProvisions
	/**
	 * Classification of optional exclusive jurisdiction terms.
	 */
	ExclusiveJurisdiction bool
	/**
	 * The Jury Waiver conditions specific to the agreement.
	 */
	JuryWaiver bool
	/**
	 * The French Law Addendum Provisions specific to the agreement.
	 */
	FrenchLawAddendum FrenchLawAddendum
	/**
	 * The qualification of whether the Belgian Law Security Agreement Addendum is deemed applicable by the parties (True) or not (False).
	 */
	BelgianLawSecurityAgreement bool
}

/**
 * Knock In means option to exercise comes into existence. Knock Out means option to exercise goes out of existence.
 */
type Knock struct {
	/**
	 * The knock in.
	 */
	KnockIn TriggerEvent
	/**
	 * The knock out.
	 */
	KnockOut TriggerEvent
}

/**
 * The pricing period per calculation period if the pricing days do not wholly fall within the respective calculation period.
 */
type Lag struct {
	/**
	 * Defines the offset of the series of pricing dates relative to the calculation period.
	 */
	LagDuration Offset
	/**
	 * Defines the offset of the series of pricing dates relative to the calculation period.
	 */
	FirstObservationDateOffset Offset
}

/**
 * Specification of amendments to the calculation of Exposure in terms of the Transactions and Products covered.
 */
type LegacyExposureScopeElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * A boolean flag to represent whether any transactions are excluded (ie True) or not (ie False).
	 */
	ExcludedTransactions bool
	/**
	 * Description of the relevant derivative transactions excluded from the calculation of exposure.
	 */
	ExcludedProducts []ExcludedProducts
	/**
	 * The bespoke/additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * Sets out the additional amount of collateral that is required to be delivered by that party irrespective of the mark-to-market exposure under the related Transactions. It may take the form of a fixed amount, or be determined using credit ratings.
 */
type LegacyIndependentAmount struct {
	/**
	 * The party specific elections for the Independent Amount clause.
	 */
	PartyElection []LegacyIndependentAmountParty
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * The party specific elections for the Independent Amount clause.
 */
type LegacyIndependentAmountParty struct {
	/**
	 * Identifies the Party making the election.
	 */
	Party CounterpartyRoleEnum
	/**
	 * Specifies whether Independent Amount is applicable to the elected party. If applicable, 'True', and if not applicable then 'False'.
	 */
	IsApplicable bool
	/**
	 * Indicates the Independent Amount is specified as a fixed amount.
	 */
	FixedAmount Money
	/**
	 * Indicates the Independent Amount is determined using a credit rating.
	 */
	RatingsBased LegacyIndependentAmountRatingsBased
	/**
	 * Indicates the Independent Amount is determined using both credit rating and exposure metrics.
	 */
	RatingsXExposure LegacyIndependentAmountRatingsXExposure
}

/**
 * Specifies that the Independent Amount is determined using a credit rating.
 */
type LegacyIndependentAmountRatingsBased struct {
	/**
	 * Specifies how multiple credit ratings are compared when determining the applicable Independent Amount, and whether that measure is highest, lowest, or a comparison of either.
	 */
	Compare IndependentAmountCompareEnum
	/**
	 * Specifies the party to whom the rating agency rating applies to for the purposes of determining the Independent Amount.
	 */
	RatedParty RatedPartyEnum
	/**
	 * Details the Named Affiliate where the Rated Party is Named Affiliate.
	 */
	NamedAffiliate LegalEntity
	/**
	 * Details the Named Entity where the Rated Party is Named Entity.
	 */
	NamedEntity LegalEntity
	/**
	 * The relevant rating type.
	 */
	RatingType RatingTypeEnum
	/**
	 * A logical container designed to hold a defined set of related data. In this case the collation of Rating Information and an associated Independent Amount.
	 */
	VariableSet []RatingAgencyAmount
	/**
	 * Specifies the currency applicable to the relevant Independent Amount.
	 */
	Currency FieldWithMeta
}

/**
 * Specifies that the Independent Amount is determined using both credit rating and exposure metrics.
 */
type LegacyIndependentAmountRatingsXExposure struct {
	/**
	 * Specifies how multiple credit ratings are compared when determining the applicable Independent Amount, and whether that measure is highest, lowest, or a comparison of either.
	 */
	Compare IndependentAmountCompareEnum
	/**
	 * Specifies the party to whom the rating agency rating applies to for the purposes of determining the Independent Amount.
	 */
	RatedParty RatedPartyEnum
	/**
	 * Details the Named Affiliate where the Rated Party is Named Affiliate.
	 */
	NamedAffiliate LegalEntity
	/**
	 * Details the Named Entity where the Rated Party is Named Entity.
	 */
	NamedEntity LegalEntity
	/**
	 * The relevant rating type.
	 */
	RatingType RatingTypeEnum
	/**
	 * A logical container to hold a defined set of related data. In this case the collation of Ratings Information and associated exposure multiplier where highest or lowest of the associated credit ratings are used when determining the applicable Independent Amount.
	 */
	HighestLowestVariableSet []RatingMultiplier
	/**
	 * A logical container to hold a defined set of related data. In this case the collation of Ratings Information and associated exposure multiplier where multiple credit ratings are compared when determining the applicable Independent Amount.
	 */
	CompareVariableSet []DirectionRatingMultiplier
}

/**
 * Alternative procedures to those in the pre-print for resolving collateral related disputes.
 */
type LegacyResolutionAlternative struct {
	/**
	 * Details of the alternative dispute resolution procedure (if any).
	 */
	ResolutionAlternative LegacyResolutionAlternativeEnum
	/**
	 * The maximum amount to which the difference between the calculations by the valuation agent and disputing party can be split, where Split the Difference is elected.
	 */
	Amount Money
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * The specification of a legal agreement between two parties, being negotiated or having been executed. This includes the baseline information and the optional specialised elections
 */
type LegalAgreement struct {
	/**
	 * The date on which the legal agreement has been agreed between the parties. This corresponds to the Date of Deed in an English Law document.
	 */
	AgreementDate time.Time
	/**
	 * The date on which, or as of which, the agreement is effective, if different from the agreement date. It is expected that it will most often correspond to the agreement date, although there could be situations where the parties will explicitly agree on a distinct effective date.
	 */
	EffectiveDate time.Time
	/**
	 * The legal agreement identifier. Several identifiers can be specified.
	 */
	Identifier []Identifier
	/**
	 * The type of legal agreement, identified via a set of composable attributes: agreementName, publisher, governing law and version, e.g. ISDA 2013 Standard Credit Support Annex English Law.
	 */
	LegalAgreementIdentification LegalAgreementIdentification
	/**
	 * The two contractual parties to the legal agreement, which reference information is positioned as part of the partyInformation attribute.
	 */
	ContractualParty []ReferenceWithMeta
	/**
	 * The role(s) that other party(ies) may have in relation to the legal agreement, further to the contractual parties.
	 */
	OtherParty []PartyRole
	/**
	 * A human readable document, for example a confirmation.
	 */
	Attachment []Resource
	/**
	 * Specification of the content of the legal agreement.
	 */
	AgreementTerms AgreementTerms
	/**
	 * Specifies the agreement(s) that govern the agreement, either as a reference to such agreements when specified as part of the CDM, or through identification of some of the key terms of those agreements, such as the type of agreement, the publisher, the vintage, the agreement identifier and the agreement date.
	 */
	RelatedAgreements []LegalAgreement
	/**
	 * Specifies a set of legal entities which are part of a legal agreement beyond the two contracting parties to that agreement.
	 */
	UmbrellaAgreement UmbrellaAgreement
	Meta              MetaFields
}

/**
 * Specifies the legal agreement baseline information, being negotiated or having been executed. It excludes specialized elections
 */
type LegalAgreementBase struct {
	/**
	 * The date on which the legal agreement has been agreed between the parties. This corresponds to the Date of Deed in an English Law document.
	 */
	AgreementDate time.Time
	/**
	 * The date on which, or as of which, the agreement is effective, if different from the agreement date. It is expected that it will most often correspond to the agreement date, although there could be situations where the parties will explicitly agree on a distinct effective date.
	 */
	EffectiveDate time.Time
	/**
	 * The legal agreement identifier. Several identifiers can be specified.
	 */
	Identifier []Identifier
	/**
	 * The type of legal agreement, identified via a set of composable attributes: agreementName, publisher, governing law and version, e.g. ISDA 2013 Standard Credit Support Annex English Law.
	 */
	LegalAgreementIdentification LegalAgreementIdentification
	/**
	 * The two contractual parties to the legal agreement, which reference information is positioned as part of the partyInformation attribute.
	 */
	ContractualParty []ReferenceWithMeta
	/**
	 * The role(s) that other party(ies) may have in relation to the legal agreement, further to the contractual parties.
	 */
	OtherParty []PartyRole
	/**
	 * A human readable document, for example a confirmation.
	 */
	Attachment []Resource
}

/**
 * Specifies the type of legal agreement, identified via a set of composable attributes: agreementName, publisher, governing law and version, e.g. ISDA 2013 Standard Credit Support Annex English Law.
 */
type LegalAgreementIdentification struct {
	/**
	 * The law governing the legal agreement, e.g. English Law, New York Law or Japanese Law.
	 */
	GoverningLaw GoverningLawEnum
	/**
	 * The legal agreement name, e.g. Credit Support Annex for Variation Margin.
	 */
	AgreementName AgreementName
	/**
	 * The legal agreement publisher, e.g. ISDA.
	 */
	Publisher LegalAgreementPublisherEnum
	/**
	 * In the case where successive definitions of the legal agreement have been developed, the vintage identification. This is typically (but not necessarily) done by referencing the year, e.g. 2013 in the case of the ISDA 2013 Standard Credit Support Annex.
	 */
	Vintage int
}

/**
 * A class to specify a legal entity, with a required name and an optional entity identifier (such as the LEI).
 */
type LegalEntity struct {
	/**
	 * A legal entity identifier (e.g. RED entity code).
	 */
	EntityId []FieldWithMeta
	/**
	 * The legal entity name.
	 */
	Name FieldWithMeta
	Meta MetaFields
}

type LimitApplicable struct {
	/**
	 * Standard code to indicate which type of credit line is being referred to - i.e. IM, DV01, PV01, CS01, Notional, Clip Size, Notional, maximumOrderQuantity.
	 */
	LimitType FieldWithMeta
	/**
	 * This element is required in FpML, optional in CDM for the purpose of accommodating the CME data representation while making reference to the FpML one.
	 */
	ClipSize int
	/**
	 * The limit utilised by all the cleared trades for the limit level and limit type. While the attribute is of type integer in FpML and the CME schema, it has been specified to be of type number in the CDM to take into consideration java size limits as well as for consistency purposes with the way most monetary amounts are expressed.
	 */
	AmountUtilized float64
	Utilization    CreditLimitUtilisation
	/**
	 * The limit remaining for the limit level and limit type. This does not take into account any pending trades. While the attribute is of type integer in FpML and the CME schema, it has been specified to be of type number in the CDM to take into consideration java size limits as well as for consistency purposes with the way most monetary amounts are expressed.
	 */
	AmountRemaining float64
	/**
	 * The currency in which the applicable limit is denominated. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	Currency FieldWithMeta
	Velocity Velocity
}

/**
 * A class to represent the CDM attributes that are not part of the FpML standard. Once broader usage is confirmed, it is expected that those two classes can be collapsed.
 */
type LimitApplicableExtended struct {
	/**
	 * Standard code to indicate which type of credit line is being referred to - i.e. IM, DV01, PV01, CS01, Notional, Clip Size, Notional, maximumOrderQuantity.
	 */
	LimitType FieldWithMeta
	/**
	 * This element is required in FpML, optional in CDM for the purpose of accommodating the CME data representation while making reference to the FpML one.
	 */
	ClipSize int
	/**
	 * The limit utilised by all the cleared trades for the limit level and limit type. While the attribute is of type integer in FpML and the CME schema, it has been specified to be of type number in the CDM to take into consideration java size limits as well as for consistency purposes with the way most monetary amounts are expressed.
	 */
	AmountUtilized float64
	Utilization    CreditLimitUtilisation
	/**
	 * The limit remaining for the limit level and limit type. This does not take into account any pending trades. While the attribute is of type integer in FpML and the CME schema, it has been specified to be of type number in the CDM to take into consideration java size limits as well as for consistency purposes with the way most monetary amounts are expressed.
	 */
	AmountRemaining float64
	/**
	 * The currency in which the applicable limit is denominated. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	Currency FieldWithMeta
	Velocity Velocity
	/**
	 * The level at which the limit is set: customer business, proprietary business or account level. This attribute is specified as a string as part of the CME clearing confirmation specification.
	 */
	LimitLevel FieldWithMeta
	/**
	 * The total limit available for the limit level and limit type. While the attribute is of type integer in the CME schema, it has been specified to be of type number in the CDM to take into consideration java size limits as well as for consistency purposes with the way most monetary amounts are expressed.
	 */
	LimitAmount float64
	/**
	 * The limit utilized by this specific trade. While the attribute is of type integer in the CME schema, it has been specified to be of type number in the CDM to take into consideration java size limits as well as for consistency purposes with the way most monetary amounts are expressed.
	 */
	LimitImpactDueToTrade float64
}

/**
 * A class to provide lineage information across lifecycle events through a pointer or set of pointers into the event(s), contract(s) and, possibly, payout components that the event is dependent on or relates to. As an example, if an contractFormation event is corrected, the correction event will have a lineage into the initial event, which takes the form of a globalKey into that initial contract formation event. Two referencing mechanisms are provided as part of the CDM: either the globalKey, which corresponds to the hash value of the CDM class which is referred to, or a reference qualifier which is meant to provide support for the ingestion of xml documents with id/href mechanisms. The CDM recommends the use of the globalKey and provides a default implementation which is accessible in the generated code through org.isda.cdm.globalKey.GlobalKeyHashCalculator. If implementers want to use an alternative hashing mechanism, the API in which they need to plug it is com.rosetta.model.lib.HashFunction.
 */
type Lineage struct {
	TradeReference []ReferenceWithMeta
	/**
	 * The reference to the instantiation of an Event object, either through a globalKey or an xml-derived id/href mechanism. The definition associated to the Lineage class provides more details with respect to those referencing approaches, their expected usage and available implementation.
	 */
	EventReference []ReferenceWithMeta
	/**
	 * The reference to the previous state of a Portfolio, in a chain of Events leading up to a build of that Portfolio as the holding of Product(s) in specific Quantity(ies). As part of the PortfolioState object, a pointer to the previous PortfolioState is provided through a Lineage object, together with pointer(s) to the Event or set of Events leading up to the current (new) state.
	 */
	PortfolioStateReference []ReferenceWithMeta
}

/**
 * A securitized derivative on another asset.
 */
type ListedDerivative struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * Identifies the type of an instrument using an enumerated list.
	 */
	InstrumentType InstrumentTypeEnum
	/**
	 * Also called contract month or delivery month. However, it's not always a month. It is usually expressed using a code, e.g. Z23 would be the Dec 2023 contract, (Z = December). For crude oil, the corresponding contract might be called CLZ23. Optional as this can be uniquely identified in the identifier.
	 */
	DeliveryTerm string
	/**
	 * The type of option, ie Put or Call. Left empty if it is a Future.
	 */
	OptionType PutCallEnum
	/**
	 * Specifies the strike of the option.
	 */
	Strike float64
}

/**
 * Specifies a filter based on a stock exchange.
 */
type ListingExchange struct {
	/**
	 * Represents a filter based on the Primary Stock Exchange facilitating the listing of companies, exchange of Stocks, Exchange traded Derivatives, Bonds, and other Securities expressed in ISO standard 10383.
	 */
	Exchange []FieldWithMeta
}

/**
 * Specifies a filter based on an industry sector.
 */
type ListingSector struct {
	/**
	 * Represents a filter based on an industry sector defined under a system for classifying industry types such as Global Industry Classification Standard (GICS) and North American Industry Classification System (NAICS)
	 */
	Sector []FieldWithMeta
}

/**
 * Identifies a loan by referencing an asset identifier and through an optional set of attributes.
 */
type Loan struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * Identifies the type of an instrument using an enumerated list.
	 */
	InstrumentType InstrumentTypeEnum
	/**
	 * Specifies the borrower. There can be more than one borrower. It is meant to be used in the event that there is no Bloomberg Id or the Secured List isn't applicable.
	 */
	Borrower []LegalEntity
	/**
	 * Specifies the seniority level of the lien.
	 */
	Lien FieldWithMeta
	/**
	 * Specifies the type of loan facility (letter of credit, revolving, ...).
	 */
	FacilityType FieldWithMeta
	/**
	 * Specifies the credit agreement date is the closing date (the date where the agreement has been signed) for the loans in the credit agreement. Funding of the facilities occurs on (or sometimes a little after) the Credit Agreement date. This underlier attribute is used to help identify which of the company's outstanding loans are being referenced by knowing to which credit agreement it belongs. ISDA Standards Terms Supplement term: Date of Original Credit Agreement.
	 */
	CreditAgreementDate time.Time
	/**
	 * Denotes the loan tranche that is subject to the derivative transaction. It will typically be referenced as the Bloomberg tranche number. ISDA Standards Terms Supplement term: Bloomberg Tranche Number.
	 */
	Tranche FieldWithMeta
}

/**
 * A class to specify loan with a participation agreement whereby the buyer is capable of creating, or procuring the creation of, a contractual right in favour of the seller that provides the seller with recourse to the participation seller for a specified share in any payments due under the relevant loan which are received by the participation seller. ISDA 2003 Term: Direct Loan Participation.
 */
type LoanParticipation struct {
	/**
	 * Indicates whether the provision is applicable.
	 */
	Applicable bool
	/**
	 * Specifies whether either 'Partial Cash Settlement of Assignable Loans', 'Partial Cash Settlement of Consent Required Loans' or 'Partial Cash Settlement of Participations' is applicable. If this element is specified and Assignable Loan is a Deliverable Obligation Characteristic, any Assignable Loan that is deliverable, but where a non-receipt of Consent by the Physical Settlement Date has occurred, the Loan can be cash settled rather than physically delivered. If this element is specified and Consent Required Loan is a Deliverable Obligation Characteristic, any Consent Required Loan that is deliverable, but where a non-receipt of Consent by the Physical Settlement Date has occurred, the Loan can be cash settled rather than physically delivered. If this element is specified and Direct Loan Participation is a Deliverable Obligation Characteristic, any Participation that is deliverable, but where this participation has not been effected (has not come into effect) by the Physical Settlement Date, the participation can be cash settled rather than physically delivered.
	 */
	PartialCashSettlement bool
	/**
	 * If Direct Loan Participation is specified as a deliverable obligation characteristic, this specifies any requirements for the Qualifying Participation Seller. The requirements may be listed free-form. ISDA 2003 Term: Qualifying Participation Seller.
	 */
	QualifyingParticipationSeller string
}

/**
 * Specifies a location identifier. An issuer and an identifier type can be associated with the actual identifier value as a way to properly qualify it.
 */
type LocationIdentifier struct {
	/**
	 * The identifier issuer, when specified by reference to a party specified as part of the transaction.
	 */
	IssuerReference ReferenceWithMeta
	/**
	 * The identifier issuer, when specified explicitly alongside the identifier value (instead of being specified by reference to a party).
	 */
	Issuer FieldWithMeta
	/**
	 * The identifier value. This level of indirection between the issuer and the identifier and its version provides the ability to associate multiple identifiers to one issuer, consistently with the FpML PartyTradeIdentifier.
	 */
	AssignedIdentifier []AssignedIdentifier
	Meta               MetaFields
	/**
	 * Specifies the nature of a location identifier.
	 */
	LocationIdentifierType CommodityLocationIdentifierTypeEnum
}

/**
 * A class to specify the amount to be paid by the buyer of the option if the option is exercised prior to the Early Call Date (typically applicable to the convertible bond options).
 */
type MakeWholeAmount struct {
	FloatingRateIndex FloatingRateIndexEnum
	/**
	 * The ISDA Designated Maturity, i.e. the tenor of the floating rate.
	 */
	IndexTenor Period
	/**
	 * Spread in basis points over the floating rate index.
	 */
	Spread float64
	/**
	 * The side (bid/mid/ask) of the measure.
	 */
	Side QuotationSideEnum
	/**
	 * The type of interpolation method that the calculation agent reserves the right to use.
	 */
	InterpolationMethod InterpolationMethodEnum
	/**
	 * Date prior to which the option buyer will have to pay a Make Whole Amount to the option seller if he/she exercises the option.
	 */
	EarlyCallDate FieldWithMeta
}

/**
 * A data to:  define an early termination provision for which exercise is mandatory.
 */
type MandatoryEarlyTermination struct {
	/**
	 * The early termination date associated with a mandatory early termination of a swap.
	 */
	MandatoryEarlyTerminationDate AdjustableDate
	/**
	 * The ISDA Calculation Agent responsible for performing duties associated with an optional early termination.
	 */
	CalculationAgent CalculationAgent
	/**
	 * If specified, this means that cash settlement is applicable to the transaction and defines the parameters associated with the cash settlement procedure. If not specified, then physical settlement is applicable.
	 */
	CashSettlement SettlementTerms
	/**
	 * The adjusted dates associated with a mandatory early termination provision. These dates have been adjusted for any applicable business day convention.
	 */
	MandatoryEarlyTerminationAdjustedDates MandatoryEarlyTerminationAdjustedDates
	Meta                                   MetaFields
}

/**
 * A data defining:  the adjusted dates associated with a mandatory early termination provision.
 */
type MandatoryEarlyTerminationAdjustedDates struct {
	/**
	 * The early termination date that is applicable if an early termination provision is exercised. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedEarlyTerminationDate time.Time
	/**
	 * The date by which the cash settlement amount must be agreed. This date should already be adjusted for any applicable business day convention.
	 */
	AdjustedCashSettlementValuationDate time.Time
	/**
	 * The date on which the cash settlement amount is paid. This date should already be adjusted for any applicable business date convention.
	 */
	AdjustedCashSettlementPaymentDate time.Time
}

/**
 * A class defining manual exercise, i.e. that the option buyer counterparty must give notice to the option seller of exercise.
 */
type ManualExercise struct {
	/**
	 * Definition of the party to whom notice of exercise should be given.
	 */
	ExerciseNotice ExerciseNotice
	/**
	 * If fallback exercise is specified then the notional amount of the underlying swap, not previously exercised under the swaption, will be automatically exercised at the expiration time on the expiration date if at such time the buyer is in-the-money, provided that the difference between the settlement rate and the fixed rate under the relevant underlying swap is not less than one tenth of a percentage point (0.10% or 0.001). The term in-the-money is assumed to have the meaning defined in the 2000 ISDA Definitions, Section 17.4. In-the-money.
	 */
	FallbackExercise bool
}

/**
 * A class for selection of Margin Approach.
 */
type MarginApproach struct {
	/**
	 * Specifies the margin approach specific to Initial Margin agreements.
	 */
	MarginApproach MarginApproachEnum
}

/**
 * Represents common attributes required for Issuance and Response to a Margin Call action as a result of a demand for delivery or return of collateral determined under a legal agreement such as a credit support document or equivalent.
 */
type MarginCallBase struct {
	/**
	 * Identifies the enumeration values to specify the call notification type, direction, specific action type.
	 */
	InstructionType MarginCallInstructionType
	/**
	 * Represents the parties to the margin call. The cardinality is optional to address the case where both parties of the event are specified and a third party if applicable.
	 */
	Party []Party
	/**
	 * Represents the role each specified party takes in the margin call. further to the principal roles, payer and receiver.
	 */
	PartyRole []PartyRole
	/**
	 * Indicates the name of the Clearing Broker FCM/DCM.
	 */
	ClearingBroker Party
	/**
	 * Represents a unique Identifier for a margin call message, that can be referenced throughout all points of the process.
	 */
	CallIdentifier Identifier
	/**
	 * Specifies the legal agreement type the margin call is generated from and governed by.
	 */
	CallAgreementType AgreementName
	/**
	 * Specifies the collateral legal agreement minimum transfer amount in base currency.
	 */
	AgreementMinimumTransferAmount Money
	/**
	 * Specifies the collateral legal agreement threshold amount in base currency.
	 */
	AgreementThreshold Money
	/**
	 * Specifies the collateral legal agreement rounding in base currency.
	 */
	AgreementRounding Money
	/**
	 * Identifies margin type and if related regulatory mandate
	 */
	RegMarginType RegMarginTypeEnum
	/**
	 * Indicates the role of the party in an regulatory initial margin call instruction (i.e Pledgor party or Secured party).
	 */
	RegIMRole RegIMRoleEnum
	/**
	 * Represents the current mark to market value or IM calculation value of the trade portfolio as recorded by the principle (in base currency), to be referenced in a margin call.
	 */
	BaseCurrencyExposure MarginCallExposure
	/**
	 * Represents attributes to define the details of collateral assets within a collateral portfolio to be used in margin call messaging and contribute to collateral balances e.g securities in a collateral account recorded by the principal as held or posted.
	 */
	CollateralPortfolio ReferenceWithMeta
	/**
	 * Represents additional credit support amount over and above mark to market value.
	 */
	IndependentAmountBalance CollateralBalance
}

/**
 * Represents attributes required for mark to market value or IM calculation value of the trade portfolio as recorded by the principle (in base currency).
 */
type MarginCallExposure struct {
	/**
	 * Identifies the enumeration values to specify the call notification type, direction, specific action type.
	 */
	InstructionType MarginCallInstructionType
	/**
	 * Represents the parties to the margin call. The cardinality is optional to address the case where both parties of the event are specified and a third party if applicable.
	 */
	Party []Party
	/**
	 * Represents the role each specified party takes in the margin call. further to the principal roles, payer and receiver.
	 */
	PartyRole []PartyRole
	/**
	 * Indicates the name of the Clearing Broker FCM/DCM.
	 */
	ClearingBroker Party
	/**
	 * Represents a unique Identifier for a margin call message, that can be referenced throughout all points of the process.
	 */
	CallIdentifier Identifier
	/**
	 * Specifies the legal agreement type the margin call is generated from and governed by.
	 */
	CallAgreementType AgreementName
	/**
	 * Specifies the collateral legal agreement minimum transfer amount in base currency.
	 */
	AgreementMinimumTransferAmount Money
	/**
	 * Specifies the collateral legal agreement threshold amount in base currency.
	 */
	AgreementThreshold Money
	/**
	 * Specifies the collateral legal agreement rounding in base currency.
	 */
	AgreementRounding Money
	/**
	 * Identifies margin type and if related regulatory mandate
	 */
	RegMarginType RegMarginTypeEnum
	/**
	 * Indicates the role of the party in an regulatory initial margin call instruction (i.e Pledgor party or Secured party).
	 */
	RegIMRole RegIMRoleEnum
	/**
	 * Represents the current mark to market value or IM calculation value of the trade portfolio as recorded by the principle (in base currency), to be referenced in a margin call.
	 */
	BaseCurrencyExposure *MarginCallExposure
	/**
	 * Represents attributes to define the details of collateral assets within a collateral portfolio to be used in margin call messaging and contribute to collateral balances e.g securities in a collateral account recorded by the principal as held or posted.
	 */
	CollateralPortfolio ReferenceWithMeta
	/**
	 * Represents additional credit support amount over and above mark to market value.
	 */
	IndependentAmountBalance CollateralBalance
	/**
	 * Represents the whole overall mark to market value or IM calculation value of the trade portfolio as recorded by the principle (in base currency).
	 */
	OverallExposure Exposure
	/**
	 * Represents Initial Margin (IM) exposure derived from ISDA SIMM calculation.
	 */
	SimmIMExposure Exposure
	/**
	 * Represents Initial Margin (IM) exposure derived from schedule or Grid calculation.
	 */
	ScheduleGridIMExposure Exposure
}

/**
 * Represents enumeration values to specify the call notification type, direction, specific action type.
 */
type MarginCallInstructionType struct {
	/**
	 * Indicates the status of the call message type, such as expected call, notification of a call or an actionable margin call.
	 */
	CallType CallTypeEnum
	/**
	 * Indicates the choice if the call instruction is visible or not to the other party.
	 */
	VisibilityIndicator bool
}

/**
 * Represents common attributes required for a Margin Call Issuance under a legal agreement such as a credit support document or equivalent.
 */
type MarginCallIssuance struct {
	/**
	 * Identifies the enumeration values to specify the call notification type, direction, specific action type.
	 */
	InstructionType MarginCallInstructionType
	/**
	 * Represents the parties to the margin call. The cardinality is optional to address the case where both parties of the event are specified and a third party if applicable.
	 */
	Party []Party
	/**
	 * Represents the role each specified party takes in the margin call. further to the principal roles, payer and receiver.
	 */
	PartyRole []PartyRole
	/**
	 * Indicates the name of the Clearing Broker FCM/DCM.
	 */
	ClearingBroker Party
	/**
	 * Represents a unique Identifier for a margin call message, that can be referenced throughout all points of the process.
	 */
	CallIdentifier Identifier
	/**
	 * Specifies the legal agreement type the margin call is generated from and governed by.
	 */
	CallAgreementType AgreementName
	/**
	 * Specifies the collateral legal agreement minimum transfer amount in base currency.
	 */
	AgreementMinimumTransferAmount Money
	/**
	 * Specifies the collateral legal agreement threshold amount in base currency.
	 */
	AgreementThreshold Money
	/**
	 * Specifies the collateral legal agreement rounding in base currency.
	 */
	AgreementRounding Money
	/**
	 * Identifies margin type and if related regulatory mandate
	 */
	RegMarginType RegMarginTypeEnum
	/**
	 * Indicates the role of the party in an regulatory initial margin call instruction (i.e Pledgor party or Secured party).
	 */
	RegIMRole RegIMRoleEnum
	/**
	 * Represents the current mark to market value or IM calculation value of the trade portfolio as recorded by the principle (in base currency), to be referenced in a margin call.
	 */
	BaseCurrencyExposure MarginCallExposure
	/**
	 * Represents attributes to define the details of collateral assets within a collateral portfolio to be used in margin call messaging and contribute to collateral balances e.g securities in a collateral account recorded by the principal as held or posted.
	 */
	CollateralPortfolio ReferenceWithMeta
	/**
	 * Represents additional credit support amount over and above mark to market value.
	 */
	IndependentAmountBalance CollateralBalance
	/**
	 * Specifies the amount of margin being called for which accounts for margin calculation inclusive of exposure, independent amount,threshold,collateral balance, MTA, rounding increments (in base currency detailed in supporting collateral agreement).
	 */
	CallAmountInBaseCurrency Money
	/**
	 * Specifies the details to describe or identify non-cash collateral eligible assets for recall purposes.
	 */
	RecallNonCashCollateralDescription []EligibleCollateralCriteria
}

/**
 * Represents common attributes required for a Margin Call Response under a legal agreement such as a credit support document or equivalent.
 */
type MarginCallResponse struct {
	/**
	 * Identifies the enumeration values to specify the call notification type, direction, specific action type.
	 */
	InstructionType MarginCallInstructionType
	/**
	 * Represents the parties to the margin call. The cardinality is optional to address the case where both parties of the event are specified and a third party if applicable.
	 */
	Party []Party
	/**
	 * Represents the role each specified party takes in the margin call. further to the principal roles, payer and receiver.
	 */
	PartyRole []PartyRole
	/**
	 * Indicates the name of the Clearing Broker FCM/DCM.
	 */
	ClearingBroker Party
	/**
	 * Represents a unique Identifier for a margin call message, that can be referenced throughout all points of the process.
	 */
	CallIdentifier Identifier
	/**
	 * Specifies the legal agreement type the margin call is generated from and governed by.
	 */
	CallAgreementType AgreementName
	/**
	 * Specifies the collateral legal agreement minimum transfer amount in base currency.
	 */
	AgreementMinimumTransferAmount Money
	/**
	 * Specifies the collateral legal agreement threshold amount in base currency.
	 */
	AgreementThreshold Money
	/**
	 * Specifies the collateral legal agreement rounding in base currency.
	 */
	AgreementRounding Money
	/**
	 * Identifies margin type and if related regulatory mandate
	 */
	RegMarginType RegMarginTypeEnum
	/**
	 * Indicates the role of the party in an regulatory initial margin call instruction (i.e Pledgor party or Secured party).
	 */
	RegIMRole RegIMRoleEnum
	/**
	 * Represents the current mark to market value or IM calculation value of the trade portfolio as recorded by the principle (in base currency), to be referenced in a margin call.
	 */
	BaseCurrencyExposure MarginCallExposure
	/**
	 * Represents attributes to define the details of collateral assets within a collateral portfolio to be used in margin call messaging and contribute to collateral balances e.g securities in a collateral account recorded by the principal as held or posted.
	 */
	CollateralPortfolio ReferenceWithMeta
	/**
	 * Represents additional credit support amount over and above mark to market value.
	 */
	IndependentAmountBalance CollateralBalance
	/**
	 * Specifies the margin call action details, including collateral to be moved and direction.
	 */
	MarginCallResponseAction []MarginCallResponseAction
	/**
	 * Indicates the response type, such as, is the margin call response a 'full' 'part' agreement or 'dispute'.
	 */
	MarginResponseType MarginCallResponseTypeEnum
	/**
	 * Indicates the amount that posting entity agrees to remit in response to margin call (in base currency).
	 */
	AgreedAmountBaseCurrency Money
}

/**
 * Specifies the margin call action details, including collateral to be moved and its direction.
 */
type MarginCallResponseAction struct {
	/**
	 * Specifies the collateral to be moved and its direction.
	 */
	CollateralPositionComponent []CollateralPosition
	/**
	 * Specifies the margin call action details, specified as either Delivery or Return.
	 */
	MarginCallAction MarginCallActionEnum
}

/**
 * The set of elections that define an ISDA Master Agreement.
 */
type MasterAgreement struct {
	/**
	 * The specification of whether there is an automatic occurrence of an Early Termination Date in respect of Transactions upon the occurrence of certain bankruptcy / insolvency related events.
	 */
	AutomaticEarlyTermination AutomaticEarlyTermination
	/**
	 * Specification of the currency in which the termination payment is made (including the process by which such currency is determined).
	 */
	TerminationCurrency TerminationCurrency
	/**
	 * Specification of the address and other details for notices.
	 */
	AddressForNotices DemandsAndNotices
	/**
	 * Specification of whether the Governing Law clause extends to Non-Contractual Obligations (True) or does not extend to Non-Contractual Obligations (False).
	 */
	NonContractualObligations bool
	/**
	 * Identification of party specific Credit Support Documents applicable to the document.
	 */
	CreditSupportDocument CreditSupportDocument
	/**
	 * Identification of party specific Credit Support Providers applicable to the document.
	 */
	CreditSupportProvider CreditSupportProvider
	/**
	 * A provision that allows each party to specify its Specified Entities for certain Events of Default and Termination Events.
	 */
	SpecifiedEntities []SpecifiedEntities
}

/**
 * A set of elections that can be shared across master agreement types - this should be built upon for specific contracts.
 */
type MasterAgreementBase struct {
}

/**
 * Defines clauses that make up a Master Agreement.
 */
type MasterAgreementClause struct {
	/**
	 * Unique identifier for the clause.
	 */
	Identifer MasterAgreementClauseIdentifierEnum
	/**
	 * Optional textual description of the clause.
	 */
	Name string
	/**
	 * Optional counterparty role. This can be used where a clause needs to be assigned to a specific party on the agreement based upon their role i.e. Party A or Party B.
	 */
	Counterparty []CounterpartyRoleEnum
	/**
	 * Optional party. This can be required for umbrella agreements where a clause may need to be assigned to a specific party who may or may not be on the agreement.
	 */
	OtherParty []PartyRoleEnum
	/**
	 * Allows multiple variants to be defined for a clause. This needs to be an array as some clauses can specify different variants for different parties. At least one variant must be specified for a clause.
	 */
	Variant []MasterAgreementClauseVariant
}

/**
 * Sets the details for a specific variant associated to a clause in a Master Agreement.
 */
type MasterAgreementClauseVariant struct {
	/**
	 * Unique identifier for this variant.
	 */
	Identifier MasterAgreementVariantIdentifierEnum
	/**
	 * Optional textual description of the variant.
	 */
	Name string
	/**
	 * Optional counterparty role. This can be used where a clause needs to assign a different variant to the different parties on the agreement based upon their role i.e. Party A or Party B.
	 */
	Counterparty []CounterpartyRoleEnum
	/**
	 * Optional party. This can be used where a clause needs to assign different variants to different parties who may or may not be on the agreement.
	 */
	OtherParty []PartyRoleEnum
	/**
	 * For some variants of some clauses additional details are required to work out what has been elected. This array can be used to define the name and value of these variables. Please refer to the agreement documentation for more details of the variables that are available for any clause.
	 */
	VariableSet []MasterAgreementVariableSet
}

/**
 * Defines the dates agreed by the parties as the date of the Credit Support Annex or Credit Support Deed.
 */
type MasterAgreementDatedAsOfDate struct {
	/**
	 * The date agreed by the parties as the Dated as of Date for the Master Agreement.
	 */
	MasterAgreementDatedAsOfDate time.Time
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * A representation of a master agreement extended from a base agreement of shared elections.
 */
type MasterAgreementElections struct {
	/**
	 * The set of elections that define a GMSLA.
	 */
	IslaGmsla GlobalMasterSecuritiesLendingAgreement
	/**
	 * The set of elections that define a GMRA.
	 */
	IcmaGmra GlobalMasterRepoAgreement
	/**
	 * The set of elections that define an ISDA Master Agreement.
	 */
	IsdaMaster MasterAgreement
}

/**
 * The set of elections which specify a Master Agreement.
 */
type MasterAgreementSchedule struct {
	/**
	 * Clauses that have had elections made against them in this Master Agreement. There must be at least one clause defined in the agreement.
	 */
	Clause []MasterAgreementClause
}

/**
 * Defines a type where additional variables associated to clauses and their variants can be described.
 */
type MasterAgreementVariableSet struct {
	/**
	 * For some variants a table of variables is required. To support this use case we need to be able to specify variables within variables. Including a variable set here gives us infinite nesting opportunities - realistically we are only ever expecting that a table would need to be defined for any particular clause, so we would expect two levels of nesting as a maximum i.e. variableSet->variableSet->name/value.
	 */
	VariableSet []MasterAgreementVariableSet
	/**
	 * The name of the variable.
	 */
	Name string
	/**
	 * The value for this variable.
	 */
	Value string
}

/**
 * Legal agreement specification for General Terms and Elections that are applicable across multiple confirmations and are referenced by these confirmations.
 */
type MasterConfirmationBase struct {
}

type MaterialChangeInCircumstance struct {
	MaterialChangeInCircumstanceIsApplicable bool
	EscrowArrangement                        EscrowArrangement
	MaximumDaysOfDisruption                  ValuationPostponement
}

/**
 * Defines a concrete measure as a number associated to a unit. It extends MeasureBase by requiring the value attribute to be present. A measure may be unit-less so the unit attribute is still optional.
 */
type Measure struct {
	/**
	 * Specifies the value of the measure as a number. Optional because in a measure vector or schedule, this single value may be omitted.
	 */
	Value float64
	/**
	 * Qualifies the unit by which the amount is measured. Optional because a measure may be unit-less (e.g. when representing a ratio between amounts in the same unit).
	 */
	Unit UnitType
}

/**
 * Provides an abstract type to define a measure as a number associated to a unit. This type is abstract because all its attributes are optional. The types that extend it can specify further existence constraints.
 */
type MeasureBase struct {
	/**
	 * Specifies the value of the measure as a number. Optional because in a measure vector or schedule, this single value may be omitted.
	 */
	Value float64
	/**
	 * Qualifies the unit by which the amount is measured. Optional because a measure may be unit-less (e.g. when representing a ratio between amounts in the same unit).
	 */
	Unit UnitType
}

/**
 * A set of measures, all in the same unit, where the values are defined through a schedule of steps. The initial value may be defined either as part of the steps, or using the single amount attribute.
 */
type MeasureSchedule struct {
	/**
	 * Specifies the value of the measure as a number. Optional because in a measure vector or schedule, this single value may be omitted.
	 */
	Value float64
	/**
	 * Qualifies the unit by which the amount is measured. Optional because a measure may be unit-less (e.g. when representing a ratio between amounts in the same unit).
	 */
	Unit UnitType
	/**
	 * A schedule of step date and value pairs. On each step date the associated step value becomes effective. The step dates are used to order the steps by ascending order. This attribute is optional so the data type may be used to define a schedule with a single value.
	 */
	DatedValue []DatedValue
}

/**
 * Defines the underlier for both the Purchaser Company and the Acquired Compagny involved in Merger corporate action, as well as th respective price observation for each.
 */
type Merger struct {
	/**
	 * Defines the underlier for the Purchaser Company involved in Merger corporate action.
	 */
	PurchaserSecurity Security
	/**
	 * The price observation that relates to underlier for the Purchaser Company.
	 */
	PurchaserSecurityPrice Price
	/**
	 * Defines the underlier for the Acquired Company involved in Merger corporate action.
	 */
	AcquiredSecurity Security
	/**
	 * The price observation that relates to underlier for the Acquired Company.
	 */
	AcquiredSecurityPrice Price
}

/**
 * This class corresponds to the components of the FpML MessageHeader.model.
 */
type MessageInformation struct {
	/**
	 * A unique identifier assigned to the message.
	 */
	MessageId FieldWithMeta
	/**
	 * The identifier for the originator of a message instance.
	 */
	SentBy FieldWithMeta
	/**
	 * The identifier(s) for the recipient(s) of a message instance.
	 */
	SentTo []FieldWithMeta
	/**
	 * A unique identifier (within the specified coding scheme) giving the details of some party to whom a copy of this message will be sent for reference.
	 */
	CopyTo []FieldWithMeta
}

/**
 * Represents the threshold specified in the agreement below which collateral need not be delivered, and represents the smallest incremental value of exposure that must be reached before the parties are obliged to transfer collateral.
 */
type MinimumTransferAmount struct {
	/**
	 * The party specific elections for the Minimum Transfer Amount clause.
	 */
	PartyElection []MinimumTransferAmountElection
}

/**
 *  A class to specify whether Amendment to Minimum Transfer Amount language is applicable or not.
 */
type MinimumTransferAmountAmendment struct {
	/**
	 * The definition of Minimum Transfer Amount in any Other Regulatory CSA will be amended when applicable.
	 */
	IsApplicable bool
	/**
	 * The effective date of the Amendment to Termination Currency.
	 */
	EffectiveDate AmendmentEffectiveDate
	/**
	 * The party elective amounts.
	 */
	PartyElections []ElectiveAmountElection
}

/**
 * Specifies the method by which the minimum transfer amount is defined, such as a fixed amount, ratings based, or an alternative basis set out in the agreement.
 */
type MinimumTransferAmountElection struct {
	/**
	 * The party to which the minimum transfer amount applies.
	 */
	Party CounterpartyRoleEnum
	/**
	 * Defines that the Minimum Transfer Amount (MTA) is a Fixed Amount.
	 */
	FixedAmount ThresholdMinimumTransferAmountFixedAmount
	/**
	 * Defines that the Minimum Transfer Amount (MTA) is based on a Ratings condition(s).
	 */
	RatingsBased MinimumTransferAmountRatingsBased
	/**
	 * Utilised where the clause data structure is not able to capture a material aspect of the clause.
	 */
	Other string
}

/**
 * Defines that the Minimum Transfer Amount (MTA) is based on a Ratings condition(s).
 */
type MinimumTransferAmountRatingsBased struct {
	/**
	 * Indicates whether a trigger applies for the Threshold or Minimum Transfer Amount to fall to zero. If 'True' then such a trigger exists.
	 */
	ZeroEvent bool
	/**
	 * Specifies the relevant trigger for the Threshold or Minimum Transfer Amount to fall to zero.
	 */
	Event []ZeroEventEnum
	/**
	 * The minimum transfer amount currency code.
	 */
	Currency ISOCurrencyCodeEnum
	/**
	 * The relevant rating type.
	 */
	RatingType RatingTypeEnum
	/**
	 * Defines a combination of Rating Agency, Rating Value, amount and Currency code.
	 */
	VariableSet []CSAMinimumTransferAmountVariableSet
	/**
	 * The party to which a rating applies.
	 */
	RatedParty RatedPartyEnum
	/**
	 * Details the Named Entity where the Rated Party is Named Entity.
	 */
	NamedEntity string
	/**
	 * Details the Named Affiliate where the Rated Party is Named Affiliate.
	 */
	NamedAffiliate string
	/**
	 * Where two ratings are specified whether the higher or lower rating prevails.
	 */
	Compare CreditNotationMismatchResolutionEnum
	/**
	 * What conditions apply where a party has no rating.
	 */
	NoRating bool
	/**
	 * Defines where conditions apply if no Rating where ratings may not exist.
	 */
	NotRatedBy NotRatedByEnum
	/**
	 * Defines the number of Rating Agencies that the Party must be rated by.
	 */
	NumberOfRatingAgencies NumberOfRatingAgenciesEnum
}

/**
 * Defines a monetary amount in a specified currency.
 */
type Money struct {
	/**
	 * Specifies the value of the measure as a number. Optional because in a measure vector or schedule, this single value may be omitted.
	 */
	Value float64
	/**
	 * Qualifies the unit by which the amount is measured. Optional because a measure may be unit-less (e.g. when representing a ratio between amounts in the same unit).
	 */
	Unit UnitType
	/**
	 * A schedule of step date and value pairs. On each step date the associated step value becomes effective. The step dates are used to order the steps by ascending order. This attribute is optional so the data type may be used to define a schedule with a single value.
	 */
	DatedValue []DatedValue
	/**
	 * Defines an optional number that the quantity should be multiplied by to derive a total quantity. This number is associated to a unit. For example in the case of the Coal (API2) CIF ARA (ARGUS-McCloskey) Futures Contract on the CME, where the unit would be contracts, the multiplier value would 1,000 and the mulitiplier unit would be 1,000 MT (Metric Tons).
	 */
	Multiplier Measure
	/**
	 * Defines the frequency to be used when defining a quantity. For example a quantity may be specified as a number of barrels of oil per day, which needs multiplying by the number of days in the relevant period to get the total quantity as a number of barrels.
	 */
	Frequency Frequency
	Meta      MetaFields
}

/**
 * The money bound is defined as a money amount and whether the bound is inclusive.
 */
type MoneyBound struct {
	/**
	 * The money amount to be used as the bound, e.g. 1,000 USD.
	 */
	Money Money
	/**
	 * Whether the money amount bound is inclusive, e.g. for a lower bound, false would indicate greater than, whereas true would indicate greater than or equal to.
	 */
	Inclusive bool
}

/**
 * The money range defined as either a lower and upper money bound, or both.
 */
type MoneyRange struct {
	/**
	 * The lower bound of a money range, e.g. greater than or equal to 1,000 USD.
	 */
	LowerBound MoneyBound
	/**
	 * The upper bound of a money range, e.g. less than 10,000 USD.
	 */
	UpperBound MoneyBound
}

/**
 * Represetns a class to specify multiple credit notations alongside a conditional 'any' or 'all' qualifier.
 */
type MultipleCreditNotations struct {
	/**
	 * An enumerated element, to qualify whether All or Any credit notation applies.
	 */
	Condition QuantifierEnum
	/**
	 * At least two credit notations much be specified.
	 */
	CreditNotation     []FieldWithMeta
	MismatchResolution CreditNotationMismatchResolutionEnum
	ReferenceAgency    CreditRatingAgencyEnum
}

/**
 * Represents a class to specify multiple credit debt types alongside a conditional 'any' or 'all' qualifier.
 */
type MultipleDebtTypes struct {
	/**
	 * An enumerated attribute, to qualify whether All or Any debt type applies.
	 */
	Condition QuantifierEnum
	/**
	 * The type of debt, e.g. long term debt, deposit, ... FpML doesn't specific a scheme value, hence no enumeration is specified as part of the CDM. At least two debt types much be specified.
	 */
	DebtType []FieldWithMeta
}

/**
 * A class defining multiple exercises. As defined in the 2000 ISDA Definitions, Section 12.4. Multiple Exercise, the buyer of the option has the right to exercise all or less than all the unexercised notional amount of the underlying swap on one or more days in the exercise period, but on any such day may not exercise less than the minimum notional amount or more than the maximum notional amount, and if an integral multiple amount is specified, the notional exercised must be equal to or, be an integral multiple of, the integral multiple amount. In FpML, MultipleExercise is built upon the PartialExercise.model.
 */
type MultipleExercise struct {
	/**
	 * A pointer style reference to the associated notional schedule defined elsewhere in the document. This element has been made optional as part of its integration in the OptionBaseExtended, because not required for the options on securities.
	 */
	NotionalReference ReferenceWithMeta
	/**
	 * A notional amount which restricts the amount of notional that can be exercised when partial exercise or multiple exercise is applicable. The integral multiple amount defines a lower limit of notional that can be exercised and also defines a unit multiple of notional that can be exercised, i.e. only integer multiples of this amount can be exercised.
	 */
	IntegralMultipleAmount float64
	/**
	 * The minimum notional amount that can be exercised on a given exercise date. See multipleExercise.
	 */
	MinimumNotionalAmount float64
	/**
	 * The minimum number of options that can be exercised on a given exercise date.
	 */
	MinimumNumberOfOptions int
	/**
	 * The maximum notional amount that can be exercised on a given exercise date.
	 */
	MaximumNotionalAmount float64
	/**
	 * The maximum number of options that can be exercised on a given exercise date. If the number is not specified, it means that the maximum number of options corresponds to the remaining unexercised options.
	 */
	MaximumNumberOfOptions int
}

type MultipleValuationDates struct {
	/**
	 * A number of business days. Its precise meaning is dependant on the context in which this element is used. ISDA 2003 Term: Business Day.
	 */
	BusinessDays int
	/**
	 * The number of business days between successive valuation dates when multiple valuation dates are applicable for cash settlement. ISDA 2003 Term: Business Days thereafter.
	 */
	BusinessDaysThereafter int
	/**
	 * Where multiple valuation dates are specified as being applicable for cash settlement, this element specifies (a) the number of applicable valuation dates, and (b) the number of business days after satisfaction of all conditions to settlement when the first such valuation date occurs, and (c) the number of business days thereafter of each successive valuation date. ISDA 2003 Term: Multiple Valuation Dates.
	 */
	NumberValuationDates int
}

type Nationalization struct {
	NationalizationIsApplicable bool
	RelevantAffiliate           []AncillaryEntity
	EscrowArrangement           EscrowArrangement
	AssignmentOfClaim           bool
	MaximumDaysOfDisruption     ValuationPostponement
}

/**
 * A class to represent the attributes that are specific to a natural person.
 */
type NaturalPerson struct {
	/**
	 * The identifier associated with a person, e.g. the internal identification code.
	 */
	PersonId []FieldWithMeta
	/**
	 * An honorific title, such as Mr., Ms., Dr. etc.
	 */
	Honorific string
	/**
	 * The natural person's first name. It is optional in FpML.
	 */
	FirstName string
	/**
	 * The natural person's middle name(s). If a middle name is provided then an initial should be absent.
	 */
	MiddleName []string
	/**
	 * The natural person's middle initial(s). If a middle initial is provided then a name should be absent.
	 */
	Initial []string
	/**
	 * The natural person's surname.
	 */
	Surname string
	/**
	 * Name suffix, such as Jr., III, etc.
	 */
	Suffix string
	/**
	 * The natural person's date of birth.
	 */
	DateOfBirth time.Time
	/**
	 * The contact information for such person, when different from the contact information associated with the party.
	 */
	ContactInformation ContactInformation
	Meta               MetaFields
}

/**
 * A class to specify the role(s) that natural person(s) may have in relation to the contract.
 */
type NaturalPersonRole struct {
	/**
	 * A reference to the natural person to whom the role refers to.
	 */
	PersonReference ReferenceWithMeta
	/**
	 * FpML specifies a person role that is distinct from the party role.
	 */
	Role []FieldWithMeta
}

/**
 * Used to apply a NOT logic condition to a single Collateral Criteria.
 */
type NegativeCriteria struct {
	NegativeCriteria CollateralCriteria
}

type New struct {
	TxId            string
	ExctgPty        string
	InvstmtPtyInd   string
	SubmitgPty      string
	Buyr            Buyr
	Sellr           Sellr
	OrdrTrnsmssn    OrdrTrnsmssn
	Tx              *Tx
	FinInstrm       FinInstrm
	InvstmtDcsnPrsn InvstmtDcsnPrsn
	ExctgPrsn       ExctgPrsn
	AddtlAttrbts    AddtlAttrbts
}

type Nm struct {
	RefRate RefRate
	Term    Term
}

type NonDeliverableSubstitute struct {
	NonDeliverableSubstituteIsApplicable bool
	ValuationDate                        AdjustedRelativeDateOffset
	SettlementDate                       AdjustedRelativeDateOffset
	SettlementRateOption                 SettlementRateOption
}

/**
 * Specifies a non-legal entity that exists in the agreement with a defined relationship to a Legal entity to the agreement
 */
type NonLegalEntity struct {
	/**
	 * Specifies the unique identifier value for a non-legal entity such as a sleeve, or sub account, that is a party to the agreement and has a relationship with a legal entity defined on the agreement.
	 */
	Identifier string
	/**
	 * Specifies the Identifier type for a non-Legal entity to the agreement, for example the use of an alert code to uniquely identify a non-Legal entity.
	 */
	IdentifierType string
}

/**
 * Specifies a quantity as a non-negative number, which condition is enforced through a data rule that only applies to the extending class.
 */
type NonNegativeQuantity struct {
	/**
	 * Specifies the value of the measure as a number. Optional because in a measure vector or schedule, this single value may be omitted.
	 */
	Value float64
	/**
	 * Qualifies the unit by which the amount is measured. Optional because a measure may be unit-less (e.g. when representing a ratio between amounts in the same unit).
	 */
	Unit UnitType
	/**
	 * A schedule of step date and value pairs. On each step date the associated step value becomes effective. The step dates are used to order the steps by ascending order. This attribute is optional so the data type may be used to define a schedule with a single value.
	 */
	DatedValue []DatedValue
	/**
	 * Defines an optional number that the quantity should be multiplied by to derive a total quantity. This number is associated to a unit. For example in the case of the Coal (API2) CIF ARA (ARGUS-McCloskey) Futures Contract on the CME, where the unit would be contracts, the multiplier value would 1,000 and the mulitiplier unit would be 1,000 MT (Metric Tons).
	 */
	Multiplier Measure
	/**
	 * Defines the frequency to be used when defining a quantity. For example a quantity may be specified as a number of barrels of oil per day, which needs multiplying by the number of days in the relevant period to get the total quantity as a number of barrels.
	 */
	Frequency Frequency
}

type NonNegativeQuantitySchedule struct {
	/**
	 * Specifies the value of the measure as a number. Optional because in a measure vector or schedule, this single value may be omitted.
	 */
	Value float64
	/**
	 * Qualifies the unit by which the amount is measured. Optional because a measure may be unit-less (e.g. when representing a ratio between amounts in the same unit).
	 */
	Unit UnitType
	/**
	 * A schedule of step date and value pairs. On each step date the associated step value becomes effective. The step dates are used to order the steps by ascending order. This attribute is optional so the data type may be used to define a schedule with a single value.
	 */
	DatedValue []DatedValue
	/**
	 * Defines an optional number that the quantity should be multiplied by to derive a total quantity. This number is associated to a unit. For example in the case of the Coal (API2) CIF ARA (ARGUS-McCloskey) Futures Contract on the CME, where the unit would be contracts, the multiplier value would 1,000 and the mulitiplier unit would be 1,000 MT (Metric Tons).
	 */
	Multiplier Measure
	/**
	 * Defines the frequency to be used when defining a quantity. For example a quantity may be specified as a number of barrels of oil per day, which needs multiplying by the number of days in the relevant period to get the total quantity as a number of barrels.
	 */
	Frequency Frequency
}

/**
 * A class defining a step date and non-negative step value pair. This step definitions are used to define varying rate or amount schedules, e.g. a notional amortisation or a step-up coupon schedule.
 */
type NonNegativeStep struct {
	/**
	 * The date on which the associated stepValue becomes effective. This day may be subject to adjustment in accordance with a business day convention.
	 */
	StepDate time.Time
	/**
	 * The non-negative rate or amount which becomes effective on the associated stepDate. A rate of 5% would be represented as 0.05.
	 */
	StepValue float64
	Meta      MetaFields
}

/**
 * A data type to specify the financial product's economic terms, alongside the product identification and product taxonomy. The non-transferable product data type represents a product that can be traded (as part of a TradableProduct) but cannot be transferred to others.  It is meant to be used across the pre-execution, execution and (as part of the Contract) post-execution lifecycle contexts.
 */
type NonTransferableProduct struct {
	/**
	 * Comprises a identifier and a source to uniquely identify the nonTransferableProduct.
	 */
	Identifier []ProductIdentifier
	/**
	 * Specifies the product taxonomy, which is composed of a taxonomy value and a taxonomy source.
	 */
	Taxonomy []ProductTaxonomy
	/**
	 * The price forming features, including payouts and provisions.
	 */
	EconomicTerms EconomicTerms
	Meta          MetaFields
}

/**
 * A class to specify the ISDA 2003 Term: Not Domestic Currency.
 */
type NotDomesticCurrency struct {
	/**
	 * Indicates whether the Not Domestic Currency provision is applicable.
	 */
	Applicable bool
	/**
	 * An explicit specification of the domestic currency. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	Currency FieldWithMeta
}

/**
 * Specifies notice information including address and, optionally, an associated person. This type also supports the ISDA CSA representation as a single string, through the address attribute in Contact Information.
 */
type NoticeContactInformation struct {
	/**
	 * The telephone number.
	 */
	Telephone []TelephoneNumber
	/**
	 * The street/postal address.
	 */
	Address []Address
	/**
	 * The email address.
	 */
	Email []string
	/**
	 * The web page. This attribute is not specified as part of the FpML ContactInformation complex type.
	 */
	WebPage []string
	/**
	 * Optional information about people involved in a transaction or business process.
	 */
	NaturalPerson []NaturalPerson
	/**
	 * Additional information on the recipient.
	 */
	AdditionalInformation string
}

/**
 * Specifies the party reference and the associated notice and contact information.
 */
type NoticeInformationElection struct {
	/**
	 * The party to which the election and details apply.
	 */
	PartyReference CounterpartyRoleEnum
	/**
	 * The main notice/address information of the party.
	 */
	PrimaryContactInformation ContactInformation
	/**
	 * The main notice/address information of the party.
	 */
	/**
	 * Additional notice/address information of the party.
	 */
	AdditionalContactInformation []NoticeContactInformation
}

/**
 * Specifies the time by which a demand for the Transfer of Eligible Credit Support or Posted Credit Support needs to be made in order for the transfer to take place in accordance with the Transfer Timing provisions.
 */
type NotificationTime struct {
	/**
	 * The parties' Notification Time election.
	 */
	PartyElections []NotificationTimeElection
	/**
	 * The determination of whether reference is made to dispute resolution notification timing in the agreement.
	 */
	DisputeNotificationReference bool
	/**
	 * The determination of whether transfer timing language is applicable or not.
	 */
	TransferTimingProviso bool
}

/**
 * A class to specify the notification time election by the respective parties to the agreement.
 */
type NotificationTimeElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The Notification Time as a time that is qualified as a standard business center.
	 */
	NotificationTime BusinessCenterTime
	/**
	 * The Notification Time as a bespoke election.
	 */
	CustomNotification string
	/**
	 * Whether the Notification Time is precluded on the day being a local business day.
	 */
	LocalBusinessDay bool
}

/**
 * The number bound is defined as a number and whether the bound is inclusive.
 */
type NumberBound struct {
	/**
	 * The number to be used as the bound, e.g. 5.
	 */
	Number float64
	/**
	 * Whether the number bound is inclusive, e.g. for a lower bound, false would indicate greater than, whereas true would indicate greater than or equal to.
	 */
	Inclusive bool
}

/**
 * The number range defined as either a lower and upper number bound, or both.
 */
type NumberRange struct {
	/**
	 * The lower bound of a number range, e.g. greater than or equal to 5.
	 */
	LowerBound NumberBound
	/**
	 * The upper bound of a number range, e.g. less than 10.
	 */
	UpperBound NumberBound
}

/**
 * A class to specify the underlying obligations of the reference entity on which protection is purchased or sold through the Credit Default Swap.
 */
type Obligations struct {
	/**
	 * Used in both obligations and deliverable obligations to represent a class or type of securities which apply. ISDA 2003 Term: Obligation Category/Deliverable Obligation Category.
	 */
	Category ObligationCategoryEnum
	/**
	 * An obligation and deliverable obligation characteristic. An obligation that ranks at least equal with the most senior Reference Obligation in priority of payment or, if no Reference Obligation is specified in the related Confirmation, the obligations of the Reference Entity that are senior. ISDA 2003 Term: Not Subordinated.
	 */
	NotSubordinated bool
	/**
	 * An obligation and deliverable obligation characteristic. The currency or currencies in which an obligation or deliverable obligation must be payable. ISDA 2003 Term: Specified Currency.
	 */
	SpecifiedCurrency SpecifiedCurrency
	/**
	 * An obligation and deliverable obligation characteristic. Any obligation that is not primarily (majority) owed to a Sovereign or Supranational Organisation. ISDA 2003 Term: Not Sovereign Lender.
	 */
	NotSovereignLender bool
	/**
	 * An obligation and deliverable obligation characteristic. Any obligation that is payable in any currency other than the domestic currency. Domestic currency is either the currency so specified or, if no currency is specified, the currency of (a) the reference entity, if the reference entity is a sovereign, or (b) the jurisdiction in which the relevant reference entity is organised, if the reference entity is not a sovereign. ISDA 2003 Term: Not Domestic Currency.
	 */
	NotDomesticCurrency NotDomesticCurrency
	/**
	 * An obligation and deliverable obligation characteristic. If the reference entity is a Sovereign, this means any obligation that is not subject to the laws of the reference entity. If the reference entity is not a sovereign, this means any obligation that is not subject to the laws of the jurisdiction of the reference entity. ISDA 2003 Term: Not Domestic Law.
	 */
	NotDomesticLaw bool
	/**
	 * An obligation and deliverable obligation characteristic. Indicates whether or not the obligation is quoted, listed or ordinarily purchased and sold on an exchange. ISDA 2003 Term: Listed.
	 */
	Listed bool
	/**
	 * An obligation and deliverable obligation characteristic. Any obligation other than an obligation that was intended to be offered for sale primarily in the domestic market of the relevant Reference Entity. This specifies that the obligation must be an internationally recognised bond. ISDA 2003 Term: Not Domestic Issuance.
	 */
	NotDomesticIssuance bool
	/**
	 * An obligation and deliverable obligation characteristic. Defined in the ISDA published additional provisions for U.S. Municipal as Reference Entity. ISDA 2003 Term: Full Faith and Credit Obligation Liability.
	 */
	FullFaithAndCreditObLiability bool
	/**
	 * An obligation and deliverable obligation characteristic. Defined in the ISDA published additional provisions for U.S. Municipal as Reference Entity. ISDA 2003 Term: General Fund Obligation Liability.
	 */
	GeneralFundObligationLiability bool
	/**
	 * An obligation and deliverable obligation characteristic. Defined in the ISDA published additional provisions for U.S. Municipal as Reference Entity. ISDA 2003 Term: Revenue Obligation Liability.
	 */
	RevenueObligationLiability bool
	/**
	 * OTE: Only allowed as an obligation characteristic under ISDA Credit 1999. In essence Not Contingent means the repayment of principal cannot be dependent on a formula/index, i.e. to prevent the risk of being delivered an instrument that may never pay any element of principal, and to ensure that the obligation is interest bearing (on a regular schedule). ISDA 2003 Term: Not Contingent.
	 */
	NotContingent bool
	/**
	 * A free format string to specify any excluded obligations or deliverable obligations, as the case may be, of the reference entity or excluded types of obligations or deliverable obligations. ISDA 2003 Term: Excluded Obligations/Excluded Deliverable Obligations.
	 */
	Excluded string
	/**
	 * This element is used to specify any other obligations of a reference entity in both obligations and deliverable obligations. The obligations can be specified free-form. ISDA 2003 Term: Other Obligations of a Reference Entity.
	 */
	OthReferenceEntityObligations string
	/**
	 * Applies to Loan CDS, to indicate what lien level is appropriate for a deliverable obligation. Applies to European Loan CDS, to indicate the Ranking of the obligation. Example: a 2nd lien Loan CDS would imply that the deliverable obligations are 1st or 2nd lien loans.
	 */
	DesignatedPriority FieldWithMeta
	/**
	 * An obligation and deliverable obligation characteristic. Defined in the ISDA published Standard Terms Supplement for use with CDS Transactions on Leveraged Loans. ISDA 2003 Term: Cash Settlement Only.
	 */
	CashSettlementOnly bool
	/**
	 * An obligation and deliverable obligation characteristic. Defined in the ISDA published Standard Terms Supplement for use with CDS Transactions on Leveraged Loans. ISDA 2003 Term: Delivery of Commitments.
	 */
	DeliveryOfCommitments bool
	/**
	 * An obligation and deliverable obligation characteristic. Defined in the ISDA published Standard Terms Supplement for use with CDS Transactions on Leveraged Loans. ISDA 2003 Term: Continuity.
	 */
	Continuity bool
}

/**
 * Specifies the object to be observed for a price, it could be an asset or a reference.
 */
type Observable struct {
	/**
	 * The object to be observed is an Asset, ie something that can be owned and transferred in the financial markets.
	 */
	Asset Asset
	/**
	 * The object to be observed is a Basket, ie a collection of Observables with an identifier and optional weightings.
	 */
	Basket Basket
	/**
	 * The object to be observed is an Index, ie an observable computed on the prices, rates or valuations of a number of assets.
	 */
	Index Index
}

/**
 * Defines a single, numerical value that was observed in the marketplace. Observations of market data are made independently to business events or trade life-cycle events, so data instances of Observation can be created independently of any other model type, hence it is annotated as a root type. Observations will be broadly reused in many situations, so references to Observation are supported via the 'key' annotation.
 */
type Observation struct {
	/**
	 * Specifies the observed value as a number.
	 */
	ObservedValue Price
	/**
	 * Represents the observation was made i.e. how to uniquely identify the observed value among the population of all available market data.
	 */
	ObservationIdentifier ObservationIdentifier
	Meta                  MetaFields
}

/**
 * Specifies a single date on which market observations take place and specifies optional associated weighting.
 */
type ObservationDate struct {
	/**
	 * A date subject to adjustment.
	 */
	UnadjustedDate time.Time
	/**
	 * The date once the adjustment has been performed. (Note that this date may change if the business center holidays change).
	 */
	AdjustedDate time.Time
	/**
	 * Specifies the degree of importance of the observation.
	 */
	Weight float64
	/**
	 * Specifies an identification key for the market observation. This attribute can be used as a reference to assign weights to a series of dates defined in a parametricSchedule.
	 */
	ObservationReference string
	Meta                 MetaFields
}

/**
 * Describes date details for a set of observation dates in parametric or non-parametric form.
 */
type ObservationDates struct {
	/**
	 * Specifies a schedule of dates (non-parametric) on which market observations take place, and allows for the optional definition of weights where applicable.  When no weight is specified, then weight of each date is assumed to be 1.0
	 */
	ObservationSchedule ObservationSchedule
	/**
	 * Specifies the date range and frequency on which market observations take place.  Weights can be assigned to dates in the schedule by assigning the weight and corresponding observationReference in the observationSchedule.
	 */
	PeriodicSchedule PeriodicDates
	/**
	 * Specifies parametric terms to determine which days within a given calculation period the price would be observed. Typically associated with Commodities.
	 */
	ParametricDates ParametricDates
}

/**
 * Specifies the necessary information to create any observation event.
 */
type ObservationEvent struct {
	/**
	 * Specifies the necessary information to create a credit event.
	 */
	CreditEvent CreditEvent
	/**
	 * Specifies the necessary information to create a corporate action.
	 */
	CorporateAction CorporateAction
}

/**
 * Defines the parameters needed to uniquely identify a piece of data among the population of all available market data.
 */
type ObservationIdentifier struct {
	/**
	 * Represents the asset or rate to which the observation relates.
	 */
	Observable Observable
	/**
	 * Specifies the date value to use when resolving the market data.
	 */
	ObservationDate time.Time
	/**
	 * Represents the time and time-zone.
	 */
	ObservationTime TimeZone
	/**
	 * Represents where the market data published and should be observed.
	 */
	InformationSource InformationSource
	/**
	 * Specifies the method according to which an amount or a date is determined.
	 */
	DeterminationMethodology DeterminationMethodology
}

/**
 * Specifies inputs needed to process an observation.
 */
type ObservationInstruction struct {
	/**
	 * Contains all information related to an observation.
	 */
	ObservationEvent ObservationEvent
}

/**
 * Parameters on daily observed computed rates, specifically daily caps and floors. This type is used to represent modular computed rates in interestRatePayouts.
 */
type ObservationParameters struct {
	/**
	 * A daily observation cap rate.
	 */
	ObservationCapRate float64
	/**
	 * A daily observation floor rate.
	 */
	ObservationFloorRate float64
}

/**
 * Specifies a single date on which market observations take place and specifies optional associated weighting.
 */
type ObservationSchedule struct {
	/**
	 * Specifies an adjusted or unadjusted date for a market observation.
	 */
	ObservationDate []ObservationDate
	/**
	 * The business day convention and financial business centers used for adjusting the date if it would otherwise fall on a day that is not a business date in the specified business centers.
	 */
	DateAdjustments BusinessDayAdjustments
}

/**
 * Parameters to describe the observation shift for a daily compounded or averaged floating rate. This type is used to represent modular computed rates in interestRatePayouts.
 */
type ObservationShiftCalculation struct {
	/**
	 * The number of days of observation shift.
	 */
	OffsetDays int
	/**
	 * Whether the rate is calculated in advance, in arrears, or relative to a reset date.
	 */
	CalculationBase ObservationPeriodDatesEnum
	/**
	 * Any additional business days that be applicable.
	 */
	AdditionalBusinessDays BusinessCenters
}

/**
 * Class containing terms that are associated with observing a price/benchmark/index across either single or multiple observations.
 */
type ObservationTerms struct {
	/**
	 * Defines time in respect to a business calendar location that the price/benchmark/index is observed
	 */
	ObservationTime BusinessCenterTime
	/**
	 * The enumerated values to specify points in the day when option exercise and valuation can occur.
	 */
	ObservationTimeType TimeTypeEnum
	/**
	 * The information source where a published or displayed market rate will be obtained, e.g. Telerate Page 3750.
	 */
	InformationSource FxSpotRateSource
	/**
	 * Defines rounding rules and precision to be used in the rounding of observations.
	 */
	Precision Rounding
	/**
	 * Defines parameters used to generate the calculation period dates schedule, including the specification of any initial or final stub calculation periods. A calculation period schedule consists of an optional initial stub calculation period, one or more regular calculation periods and an optional final stub calculation period. In the absence of any initial or final stub calculation periods, the regular part of the calculation period schedule is assumed to be between the effective date and the termination date. No implicit stubs are allowed, i.e. stubs must be explicitly specified using an appropriate combination of firstPeriodStartDate, firstRegularPeriodStartDate and lastRegularPeriodEndDate.
	 */
	CalculationPeriodDates CalculationPeriodDates
	/**
	 * Describes date details for a set of observation dates in parametric or non-parametric form.
	 */
	ObservationDates ObservationDates
	/**
	 * The number of observation dates between observation start date and observation end date.
	 */
	NumberOfObservationDates int
}

/**
 * A class defining an offset used in calculating a new date relative to a reference date, e.g. calendar days, business days, commodity Business days, etc.
 */
type Offset struct {
	/**
	 * A time period multiplier, e.g. 1, 2 or 3 etc. A negative value can be used when specifying an offset relative to another date, e.g. -2 days.
	 */
	PeriodMultiplier int
	/**
	 * A time period, e.g. a day, week, month or year of the stream. If the periodMultiplier value is 0 (zero) then period must contain the value D (day).
	 */
	Period PeriodEnum
	Meta   MetaFields
	/**
	 * In the case of an offset specified as a number of days, this element defines whether consideration is given as to whether a day is a good business day or not. If a day type of business days is specified then non-business days are ignored when calculating the offset. The financial business centers to use for determination of business days are implied by the context in which this element is used. This element must only be included when the offset is specified as a number of days. If the offset is zero days then the dayType element should not be included.
	 */
	DayType DayTypeEnum
}

/**
 * Defines business day shifts for daily componded or averaged rates.  This type is used for lookback and lockout rates. This type is used to represent modular computed rates in interestRatePayouts.
 */
type OffsetCalculation struct {
	/**
	 * The number of business days offset.
	 */
	OffsetDays int
}

/**
 * A class to specify whether One Way Provisions apply.
 */
type OneWayProvisions struct {
	/**
	 * The determination of whether the One Way Provisions are applicable (true) or not applicable (false).
	 */
	IsApplicable bool
	/**
	 * The Posting Party for the purposes of One Way Provisions. It is specified in the case where the One Way Provision is deemed applicable.
	 */
	PostingParty CounterpartyRoleEnum
}

/**
 * Defines additional optional features that can be included in an option contract.
 */
type OptionFeature struct {
	/**
	 * Describes a quanto or composite FX feature.
	 */
	FxFeature []FxFeature
	/**
	 * Defines a simple strategy feature.
	 */
	StrategyFeature StrategyFeature
	/**
	 * Defines an option feature in which an average market observation price is determined on valuation and compared to the strike to determine a settlement amount.
	 */
	AveragingFeature AveragingFeature
	/**
	 * Specifies a barrier feature.
	 */
	Barrier Barrier
	/**
	 * Specifies a knock in or knock out feature.
	 */
	Knock Knock
	/**
	 * Specifies the rules for pass-through payments from the underlier, such as dividends.
	 */
	PassThrough PassThrough
}

/**
 *  The option payout specification terms. The associated globalKey denotes the ability to associate a hash value to the respective OptionPayout instantiation for the purpose of model cross-referencing, in support of functionality such as the event effect and the lineage.
 */
type OptionPayout struct {
	/**
	 * Canonical representation of the payer and receiver parties applicable to each payout leg.
	 */
	PayerReceiver PayerReceiver
	/**
	 * Each payout leg must implement the quantity concept as a 'resolvable' type, which allows for different payout legs to be linked to each other (e.g. in the case of cross-curreny products).
	 */
	PriceQuantity ResolvablePriceQuantity
	/**
	 * The specification of the principal exchange. Optional as only applicable in the case of cross-currency or zero-coupon swaps with a final payment.
	 */
	PrincipalPayment PrincipalPayments
	/**
	 * Each payout leg must specifies its settlement terms, including the delivery type (i.e. cash vs physical, and their respective terms), the transfer type (DvP etc.) and settlement date, if any.
	 */
	SettlementTerms SettlementTerms
	BuyerSeller     BuyerSeller
	/**
	 * The option feature, such as quanto, Asian, barrier, knock.
	 */
	Feature OptionFeature
	/**
	 * Class containing terms that are associated with observing a price/benchmark/index across either single or multple observations. To be used for option contracts that reference a benchmark price.
	 */
	ObservationTerms ObservationTerms
	/**
	 * Allows the full representation of a payout by defining a set of schedule periods. It supports standard schedule customization by expressing all the dates, quantities, and pricing data in a non-parametric way.
	 */
	Schedule CalculationSchedule
	/**
	 * Contains the information relative to the delivery of the asset.
	 */
	Delivery AssetDeliveryInformation
	/**
	 * The financial product underlying the option, which can be of any type including an Asset, Basket, Index or a NonTransferableProduct.
	 */
	Underlier Underlier
	/**
	 * The type of option transaction. From a usage standpoint, put/call is the default option type, while payer/receiver indicator is used for options on index credit default swaps, consistently with the industry practice. Straddle is used for the case of straddle strategy, that combine a call and a put with the same strike.
	 */
	OptionType OptionTypeEnum
	/**
	 * The terms for exercising the option, which include the option style (e.g. American style option), the exercise procedure (e.g. manual exercise) and the settlement terms (e.g. physical vs. cash).
	 */
	ExerciseTerms ExerciseTerms
	/**
	 * Specifies the strike of the option
	 */
	Strike OptionStrike
}

/**
 * Defines the strike price of an option.
 */
type OptionStrike struct {
	/**
	 * Defines the strike of an option in the form of a price that could be a cash price, interestRate, or other types.
	 */
	StrikePrice Price
	/**
	 * Defines the strike of an option in reference to the spread of the underlying swap (typical practice in the case of an option on a credit single name swaps).
	 */
	StrikeReference ReferenceWithMeta
	/**
	 * Defines the strike of an option when expressed by reference to a swap curve (Typically the case for a convertible bond option).
	 */
	ReferenceSwapCurve ReferenceSwapCurve
	/**
	 * Defines an  option strike that is calculated from an average of observed market prices.
	 */
	AveragingStrikeFeature AveragingStrikeFeature
}

/**
 * A data defining:  an early termination provision where either or both parties have the right to exercise.
 */
type OptionalEarlyTermination struct {
	/**
	 * If optional early termination is not available to both parties then this component specifies the buyer and seller of the option. In FpML, this attribute is of type SinglePsrtyOption, which actually consists of the BuyerSeller.model.
	 */
	SinglePartyOption BuyerSeller
	/**
	 * Used for specifying whether the Mutual Early Termination Right that is detailed in the Master Confirmation will apply.
	 */
	MutualEarlyTermination bool
	/**
	 * Definition of the party to whom notice of exercise should be given.
	 */
	ExerciseNotice []ExerciseNotice
	/**
	 * A flag to indicate whether follow-up confirmation of exercise (written or electronic) is required following telephonic notice by the buyer to the seller or seller's agent.
	 */
	FollowUpConfirmation bool
	/**
	 * The ISDA Calculation Agent responsible for performing duties associated with an optional early termination.
	 */
	CalculationAgent CalculationAgent
	/**
	 * If specified, this means that cash settlement is applicable to the transaction and defines the parameters associated with the cash settlement procedure. If not specified, then physical settlement is applicable.
	 */
	CashSettlement SettlementTerms
	/**
	 * An early termination provision to terminate the trade at fair value where one or both parties have the right to decide on termination.
	 */
	OptionalEarlyTerminationAdjustedDates OptionalEarlyTerminationAdjustedDates
	/**
	 * The exercise terms associated with the optional early termination, including details such as exercise style, exercise fees, and any other relevant conditions or terms.
	 */
	ExerciseTerms ExerciseTerms
}

/**
 * A data defining:  the adjusted dates associated with an optional early termination provision.
 */
type OptionalEarlyTerminationAdjustedDates struct {
	/**
	 * The adjusted dates associated with an individual early termination date.
	 */
	EarlyTerminationEvent []EarlyTerminationEvent
}

type OrdrTrnsmssn struct {
	TrnsmssnInd string
}

/**
 * A class for defining an agreement executed between parties.
 */
type OtherAgreement struct {
	/**
	 * An identifier that has been created to identify the agreement.
	 */
	Identifier FieldWithMeta
	/**
	 * The agreement executed between the parties and intended to govern product-specific derivatives transactions between those parties.
	 */
	OtherAgreementType FieldWithMeta
	/**
	 * The version of the agreement.
	 */
	Version FieldWithMeta
	/**
	 * The date on which the agreement was signed.
	 */
	Date time.Time
}

/**
 * A class to specify a related legal agreement. For example, ISDA 2016 Credit Support Annex for Initial Margin, paragraph 13, General Principles, (s): Other CSA and Japanese Law CSA (VM). | ISDA 2016 Credit Support Annex for Variation Margin, paragraph 13, (o): Other CSA.
 */
type OtherAgreementTerms struct {
	/**
	 * The qualification of whether some other related agreement is specified (True) or not (False).
	 */
	IsSpecified bool
	/**
	 * The specification of this other agreement, when the qualification is True.
	 */
	LegalDocument string
}

/**
 * The bespoke definition of other agreements or other credit support annex terms as specified by the parties to the agreement.
 */
type OtherAgreements struct {
	/**
	 * The bespoke definition of Other CSA as specified by the parties to the agreement.
	 */
	OtherCSA OtherAgreementTerms
	/**
	 * The bespoke definition of whether Japanese Law CSA (VM) are specified by the parties to the agreement.
	 */
	JapaneseLawCsa OtherAgreementTerms
}

/**
 * Specifies the Other Eligible Support elections.
 */
type OtherEligibleAndPostedSupport struct {
	/**
	 * The definition of 'Value' with respect to Other Eligible Support and Other Posted Support.
	 */
	ApplicableValue bool
	/**
	 * The definition of 'Transfer' with respect to Other Eligible Support and Other Posted Support.
	 */
	ApplicableTransfer bool
	/**
	 * Details how particular types of collateral are valued (other than in respect of a default).
	 */
	Value CollateralValueMethod
}

/**
 * Specification of a user-defined index that does not meet the criteria of other Index data types.
 */
type OtherIndex struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * A description of the Index.
	 */
	Name FieldWithMeta
	/**
	 * The organisation that creates or maintains the Index.
	 */
	Provider LegalEntity
	/**
	 * The Asset Class of the Index.
	 */
	AssetClass AssetClassEnum
	/**
	 * A description that defines the OtherIndex.
	 */
	Description string
}

type Othr struct {
	FinInstrmGnlAttrbts FinInstrmGnlAttrbts
	DerivInstrmAttrbts  DerivInstrmAttrbts
	Id                  string
	SchmeNm             SchmeNm
}

/**
 * A class to specify the Partial Cash Deliverable Obligation Characteristic.
 */
type PCDeliverableObligationCharac struct {
	/**
	 * Indicates whether the provision is applicable.
	 */
	Applicable bool
	/**
	 * Specifies whether either 'Partial Cash Settlement of Assignable Loans', 'Partial Cash Settlement of Consent Required Loans' or 'Partial Cash Settlement of Participations' is applicable. If this element is specified and Assignable Loan is a Deliverable Obligation Characteristic, any Assignable Loan that is deliverable, but where a non-receipt of Consent by the Physical Settlement Date has occurred, the Loan can be cash settled rather than physically delivered. If this element is specified and Consent Required Loan is a Deliverable Obligation Characteristic, any Consent Required Loan that is deliverable, but where a non-receipt of Consent by the Physical Settlement Date has occurred, the Loan can be cash settled rather than physically delivered. If this element is specified and Direct Loan Participation is a Deliverable Obligation Characteristic, any Participation that is deliverable, but where this participation has not been effected (has not come into effect) by the Physical Settlement Date, the participation can be cash settled rather than physically delivered.
	 */
	PartialCashSettlement bool
}

/**
 * Defines rules for the dates on which the price will be determined.
 */
type ParametricDates struct {
	/**
	 * Denotes the enumerated values to specify the day type classification used in counting the number of days between two dates.
	 */
	DayType DayTypeEnum
	/**
	 * Denotes the method by which the pricing days are distributed across the pricing period.
	 */
	DayDistribution DayDistributionEnum
	/**
	 * Indicates the days of the week on which the price will be determined.
	 */
	DayOfWeek []DayOfWeekEnum
	/**
	 * Defines the occurrence of the dayOfWeek within the pricing period on which pricing will take place, e.g. the 3rd Friday within each Calculation Period. If omitted, every dayOfWeek will be a pricing day.
	 */
	DayFrequency float64
	/**
	 * The pricing period per calculation period if the pricing days do not wholly fall within the respective calculation period.
	 */
	Lag Lag
	/**
	 * The enumerated values to specify the business centers.
	 */
	BusinessCenters BusinessCenters
}

/**
 * A class defining partial exercise. As defined in the 2000 ISDA Definitions, Section 12.3 Partial Exercise, the buyer of the option may exercise all or less than all the notional amount of the underlying swap but may not be less than the minimum notional amount (if specified) and must be an integral multiple of the integral multiple amount if specified.
 */
type PartialExercise struct {
	/**
	 * A pointer style reference to the associated notional schedule defined elsewhere in the document. This element has been made optional as part of its integration in the OptionBaseExtended, because not required for the options on securities.
	 */
	NotionalReference ReferenceWithMeta
	/**
	 * A notional amount which restricts the amount of notional that can be exercised when partial exercise or multiple exercise is applicable. The integral multiple amount defines a lower limit of notional that can be exercised and also defines a unit multiple of notional that can be exercised, i.e. only integer multiples of this amount can be exercised.
	 */
	IntegralMultipleAmount float64
	/**
	 * The minimum notional amount that can be exercised on a given exercise date. See multipleExercise.
	 */
	MinimumNotionalAmount float64
	/**
	 * The minimum number of options that can be exercised on a given exercise date.
	 */
	MinimumNumberOfOptions int
}

/**
 * A class to specify a party, without a qualification as to whether this party is a legal entity or a natural person, although the model provides the ability to associate a person (or set of persons) to a party, which use case would imply that such party would be a legal entity (even if not formally specified as such).
 */
type Party struct {
	/**
	 * The identifier associated with a party, e.g. the 20 digits LEI code.
	 */
	PartyId []PartyIdentifier
	/**
	 * The party name.
	 */
	Name FieldWithMeta
	/**
	 * Optional organization unit information used to describe the organization units (e.g. trading desks) involved in a transaction or business process, incl. the contact information (when relevant).
	 */
	BusinessUnit []BusinessUnit
	/**
	 * The person(s) who might be associated with the party as part of the execution, contract or legal document.
	 */
	Person []NaturalPerson
	/**
	 * The role of the person(s)
	 */
	PersonRole []NaturalPersonRole
	/**
	 * The account that might be associated with the party. At most one account can be specified, as it is expected that this information is used in the context of a contract or legal document where only one account per party can be associated with such object.
	 */
	Account Account
	/**
	 * The postal/street address, telephone number, email address and/or web page. If the contact information is specific to the associated business unit(s) or person (s), it should be associated with those.
	 */
	ContactInformation ContactInformation
	Meta               MetaFields
}

/**
 * A class defining a legal agreement identifier issued by the indicated party.
 */
type PartyAgreementIdentifier struct {
	/**
	 * Party that issued the document identifier.
	 */
	PartyReference ReferenceWithMeta
	/**
	 * While FpML specifies the document identifier with a value and an associated scheme, the CDM makes use of the Identifier, which has an explicit issuer. The issuer of this identifier is not necessarily the same as the party reference.
	 */
	DocumentIdentifier []FieldWithMeta
	Meta               MetaFields
}

/**
 * Specifies instruction to change the party on a trade. This primitive instruction is used in a number of scenarios including: clearing, allocation and novation. The instrution must include a trade identifier, because a change of party effectively results in a different trade.
 */
type PartyChangeInstruction struct {
	/**
	 * The new counterparty who is stepping into the trade. The stepping out counterparty is inferred based on the counterparty role that is being updated.
	 */
	Counterparty Counterparty
	/**
	 * Specifies an ancillary party to be added onto the new transaction, e.g. the original executing party in an allocation.
	 */
	AncillaryParty AncillaryParty
	/**
	 * Specifies an additional party roles to be added on to the new transaction.
	 */
	PartyRole PartyRole
	/**
	 * The identifier to be assigned to the new trade post change of party.
	 */
	TradeId []TradeIdentifier
}

/**
 * A class to specify a party-related, non-standardized data in a generic form.
 */
type PartyCustomisedWorkflow struct {
	/**
	 * Reference to the party to which the workflow pertains to.
	 */
	PartyReference ReferenceWithMeta
	/**
	 * The party name to which the workflow pertains to.
	 */
	PartyName string
	/**
	 * Non-standardized data in a generic form.
	 */
	CustomisedWorkflow []CustomisedWorkflow
}

/**
 * Comprises an identifier and a source. The associated metadata key denotes the ability to associate a hash value to the PartyIdentifier instantiations for the purpose of model cross-referencing, in support of functionality such as the event effect and the lineage.
 */
type PartyIdentifier struct {
	/**
	 * Provides an identifier associated with a party. The identifier is unique within the public source specified in the source attribute.
	 */
	Identifier FieldWithMeta
	/**
	 * Defines the source of the identifier.
	 */
	IdentifierType PartyIdentifierTypeEnum
	Meta           MetaFields
}

/**
 * Specifies mechanism for Termination currency to be selected by the Non-defaulting Party/party which is not the Affected Party.
 */
type PartyOptionTerminationCurrency struct {
	/**
	 * Specifies termination currency where there are two Affected Parties and they cannot agree on the termination currency.
	 */
	BothAffectedTermCurrencyOption string
	/**
	 * Specifies alternative conditions for selection of the termination currency.
	 */
	TerminationCurrencySpecifiedCondition string
	/**
	 * Specifies the enumerated conditions for selection of the termination currency.
	 */
	TerminationCurrencyCondition TerminationCurrencyConditionEnum
}

/**
 * Specifies the parties responsible for making and receiving payments defined by this structure.
 */
type PartyReferencePayerReceiver struct {
	/**
	 * The party responsible for making the payments defined by this structure.
	 */
	PayerPartyReference ReferenceWithMeta
	/**
	 * A reference to the account responsible for making the payments defined by this structure.
	 */
	PayerAccountReference ReferenceWithMeta
	/**
	 * The party that receives the payments corresponding to this structure.
	 */
	ReceiverPartyReference ReferenceWithMeta
	/**
	 * A reference to the account that receives the payments corresponding to this structure.
	 */
	ReceiverAccountReference ReferenceWithMeta
}

/**
 * A class to specify the role(s) that party(ies) may have in relation to the execution, contract or other legal agreement.
 */
type PartyRole struct {
	/**
	 * A reference to the party to which the role refers to.
	 */
	PartyReference ReferenceWithMeta
	/**
	 * The party role.
	 */
	Role PartyRoleEnum
	/**
	 * A reference to the party that has ownership of this party role information. FpML specifies that For shared trade information, this attribute will reference the originator of the data (for example, an execution facility or clearing house).
	 */
	OwnershipPartyReference ReferenceWithMeta
}

/**
 * Specifies the termination currency to be used by a party when it is the Non-Defaulting Party or the Party which is not the Affected Party.
 */
type PartyTerminationCurrencySelection struct {
	/**
	 * The elective party.
	 */
	Party Party
	/**
	 * Specifies termination Currency.
	 */
	StatedPartyCurrency string
}

/**
 * Type which contains pass through payments.
 */
type PassThrough struct {
	/**
	 * One to many pass through payment items.
	 */
	PassThroughItem []PassThroughItem
}

/**
 * Class to represent a single pass through payment.
 */
type PassThroughItem struct {
	/**
	 * This attribute doesn't exists in the FpML construct, which makes use of the PayerReceiver.model group.
	 */
	PayerReceiver PayerReceiver
	/**
	 * Percentage of payments from the underlier which are passed through.
	 */
	PassThroughPercentage float64
}

/**
 * Specifies the parties responsible for making and receiving payments defined by this structure.
 */
type PayerReceiver struct {
	/**
	 * Specifies the counterparty responsible for making the payments defined by this structure.  The party is one of the two principal parties to the transaction.
	 */
	Payer CounterpartyRoleEnum
	/**
	 * Specifies the party that receives the payments corresponding to this structure.  The party is one of the two counterparties to the transaction.
	 */
	Receiver CounterpartyRoleEnum
}

/**
 * A data defining:  the adjusted payment date and associated calculation period parameters required to calculate the actual or projected payment amount. This data forms:  part of the cashflow representation of a swap stream.
 */
type PaymentCalculationPeriod struct {
	/**
	 * The unadjusted payment date.
	 */
	UnadjustedPaymentDate time.Time
	/**
	 * The adjusted payment date. This date should already be adjusted for any applicable business day convention. This component is not intended for use in trade confirmation but may be specified to allow the fee structure to also serve as a cashflow type component.
	 */
	AdjustedPaymentDate time.Time
	/**
	 * The parameters used in the calculation of a fixed or floating rate calculation period amount. A list of calculation period elements may be ordered in the document by ascending start date. An FpML document which contains an unordered list of calculation periods is still regarded as a conformant document.
	 */
	CalculationPeriod []CalculationPeriod
	/**
	 * A known fixed payment amount.
	 */
	FixedPaymentAmount Money
	/**
	 * A decimal value representing the discount factor used to calculate the present value of cash flow.
	 */
	DiscountFactor float64
	/**
	 * A monetary amount representing the forecast of the future value of the payment.
	 */
	ForecastPaymentAmount Money
	/**
	 * A monetary amount representing the present value of the forecast payment.
	 */
	PresentValueAmount Money
	Meta               MetaFields
}

/**
 * The payment dates when specified as relative to a set of dates specified somewhere else in the instance document/transaction, e.g. the valuation dates as typically the case for equity swaps, or when specified as a calculation period schedule.
 */
type PaymentDateSchedule struct {
	InterimPaymentDates []AdjustableRelativeOrPeriodicDates
	/**
	 * The last payment when specified as an adjustable or relative date, as in the FpML total return construct.
	 */
	FinalPaymentDate AdjustableOrRelativeDate
}

/**
 * Specifies the parameters to generate the payment date schedule, either through a parametric representation or by reference to specified dates.
 */
type PaymentDates struct {
	/**
	 * The frequency at which regular payment dates occur. If the payment frequency is equal to the frequency defined in the calculation period dates component then one calculation period contributes to each payment amount. If the payment frequency is less frequent than the frequency defined in the calculation period dates component then more than one calculation period will contribute to the payment amount. A payment frequency more frequent than the calculation period frequency or one that is not a multiple of the calculation period frequency is invalid. If the payment frequency is of value T (term), the period is defined by the effectiveDate and the terminationDate.
	 */
	PaymentFrequency Frequency
	/**
	 * The first unadjusted payment date. This day may be subject to adjustment in accordance with any business day convention specified in paymentDatesAdjustments. This element must only be included if there is an initial stub. This date will normally correspond to an unadjusted calculation period start or end date. This is true even if early or delayed payment is specified to be applicable since the actual first payment date will be the specified number of days before or after the applicable adjusted calculation period start or end date with the resulting payment date then being adjusted in accordance with any business day convention specified in paymentDatesAdjustments.
	 */
	FirstPaymentDate time.Time
	/**
	 * The last regular payment date when specified as a date, as in the FpML interest rate construct. FpML specifies that this date may be subject to adjustment in accordance with any business day convention specified in the paymentDatesAdjustments attribute.
	 */
	LastRegularPaymentDate time.Time
	/**
	 * The payment dates when specified as relative to a set of dates specified somewhere else in the instance document/transaction, e.g. the valuation dates as typically the case for equity swaps, or when specified as a calculation period schedule.
	 */
	PaymentDateSchedule PaymentDateSchedule
	/**
	 * Specifies whether the payments occur relative to each adjusted calculation period start date or end date, each reset date, valuation date or the last pricing date. Calculation period start date means relative to the start of the first calculation period contributing to a given payment. Similarly, calculation period end date means the end of the last calculation period contributing to a given payment. The valuation date is applicable for Brazilian-CDI and equity swaps.
	 */
	PayRelativeTo PayRelativeToEnum
	/**
	 * If early payment or delayed payment is required, specifies the number of days offset that the payment occurs relative to what would otherwise be the unadjusted payment date. The offset can be specified in terms of either calendar or business days. Even in the case of a calendar days offset, the resulting payment date, adjusted for the specified calendar days offset, will still be adjusted in accordance with the specified payment dates adjustments. This element should only be included if early or delayed payment is applicable, i.e. if the periodMultiplier element value is not equal to zero. An early payment would be indicated by a negative periodMultiplier element value and a delayed payment (or payment lag) would be indicated by a positive periodMultiplier element value.
	 */
	PaymentDaysOffset Offset
	/**
	 * The definition of the business day convention and financial business centers used for adjusting the payment date if it would otherwise fall on a day that is not a business day in the specified business center.
	 */
	PaymentDatesAdjustments BusinessDayAdjustments
	Meta                    MetaFields
}

type PaymentDetail struct {
	PaymentDate AdjustableOrRelativeDate
	/**
	 * The calculation rule.
	 */
	PaymentRule PaymentRule
	/**
	 * A fixed payment amount.
	 */
	PaymentAmount Money
	Meta          MetaFields
}

/**
 * This class corresponds to the FpML PaymentDiscounting.model group for representing the discounting elements that can be associated with a payment.
 */
type PaymentDiscounting struct {
	/**
	 * The value representing the discount factor used to calculate the present value of the cash flow.
	 */
	DiscountFactor float64
	/**
	 * The amount representing the present value of the forecast payment.
	 */
	PresentValueAmount Money
}

/**
 * A class defining the payment calculation rule. As of FpML 5.10, percentage rule is the only calculation rule that has been specified as part of the standard.
 */
type PaymentRule struct {
	/**
	 * This attribute is not present as part of the FpML construct, as the payment rule is specialised by means of runtime type extension through the xsi:type.
	 */
	PercentageRule PercentageRule
}

/**
 * Represents the set of future cashflow methodologies in the form of specific payout data type(s) which result from the financial product.  Examples: a trade in a cash asset will use only a settlement payout; for derivatives, two interest rate payouts can be combined to specify an interest rate swap; one interest rate payout can be combined with a credit default payout to specify a credit default swap.
 */
type Payout struct {
	/**
	 * Defines the assets and movements in a security financing transaction.
	 */
	AssetPayout AssetPayout
	/**
	 * Defines the payout for the floating leg of a Commodity Swap.
	 */
	CommodityPayout CommodityPayout
	/**
	 * The credit default payout, which provides the details necessary for determining when a credit payout will be triggered as well as the parameters for calculating the payout and the settlement terms.
	 */
	CreditDefaultPayout CreditDefaultPayout
	/**
	 * Defines a payout in which one or more payouts are defined as a fixed price.
	 */
	FixedPricePayout FixedPricePayout
	/**
	 * All of the terms necessary to define and calculate a cash flow based on a fixed, a floating or an inflation index rate. The interest rate payout can be applied to interest rate swaps and FRA (which both have two associated interest rate payouts), credit default swaps (to represent the fee leg when subject to periodic payments) and equity swaps (to represent the funding leg).
	 */
	InterestRatePayout InterestRatePayout
	/**
	 * The option payout.
	 */
	OptionPayout OptionPayout
	/**
	 * The performance payout, which encompasses the equity price returns, dividend returns, volatility return, variance return and correlation provisions.
	 */
	PerformancePayout PerformancePayout
	/**
	 * Represents a forward settling payout. The 'Underlier' attribute captures the underlying payout, which is settled according to the 'SettlementTerms' attribute. Both FX Spot and FX Forward should use this component.
	 */
	SettlementPayout SettlementPayout
	Meta             MetaFields
}

/**
 * A data type that contains the common attributes (e.g. payer and receiver parties) and validation conditions that apply across all payout types
 */
type PayoutBase struct {
	/**
	 * Canonical representation of the payer and receiver parties applicable to each payout leg.
	 */
	PayerReceiver PayerReceiver
	/**
	 * Each payout leg must implement the quantity concept as a 'resolvable' type, which allows for different payout legs to be linked to each other (e.g. in the case of cross-curreny products).
	 */
	PriceQuantity ResolvablePriceQuantity
	/**
	 * The specification of the principal exchange. Optional as only applicable in the case of cross-currency or zero-coupon swaps with a final payment.
	 */
	PrincipalPayment PrincipalPayments
	/**
	 * Each payout leg must specifies its settlement terms, including the delivery type (i.e. cash vs physical, and their respective terms), the transfer type (DvP etc.) and settlement date, if any.
	 */
	SettlementTerms SettlementTerms
}

/**
 * A class defining a content model for a calculation rule defined as percentage of the notional amount.
 */
type PercentageRule struct {
	/**
	 * A percentage of the notional amount.
	 */
	PaymentPercent float64
	/**
	 * A reference to the notional amount.
	 */
	NotionalAmountReference ReferenceWithMeta
}

/**
 * Contains the necessary specifications for all performance payouts, encompassing equity return, dividend, variance, volatility and correlation products.
 */
type PerformancePayout struct {
	/**
	 * Canonical representation of the payer and receiver parties applicable to each payout leg.
	 */
	PayerReceiver PayerReceiver
	/**
	 * Each payout leg must implement the quantity concept as a 'resolvable' type, which allows for different payout legs to be linked to each other (e.g. in the case of cross-curreny products).
	 */
	PriceQuantity ResolvablePriceQuantity
	/**
	 * The specification of the principal exchange. Optional as only applicable in the case of cross-currency or zero-coupon swaps with a final payment.
	 */
	PrincipalPayment PrincipalPayments
	/**
	 * Each payout leg must specifies its settlement terms, including the delivery type (i.e. cash vs physical, and their respective terms), the transfer type (DvP etc.) and settlement date, if any.
	 */
	SettlementTerms SettlementTerms
	/**
	 * Defines how and when a performance type option or performance type swap is to be observed.
	 */
	ObservationTerms ObservationTerms
	/**
	 * Defines how and when a performance type option or performance type swap is to be valued, including both interim and final valuation.
	 */
	ValuationDates ValuationDates
	/**
	 * Defines the payment date schedule, as defined by the parameters that are needed to specify it, either in a parametric way or by reference to another schedule of dates (e.g. the valuation dates).
	 */
	PaymentDates PaymentDates
	/**
	 * Identifies the underlying product that is referenced for pricing of the applicable leg in a swap.  Referenced in the '2018 ISDA CDM Equity Confirmation for Security Equity Swap' as Security.
	 */
	Underlier Underlier
	/**
	 * Defines quanto or composite FX features that are included in the swap leg.
	 */
	FxFeature []FxFeature
	/**
	 * Specifies the type of return of a performance payout.
	 */
	ReturnTerms ReturnTerms
	/**
	 * Specifies an individual type of return of a Performance Payout, when such individual return is part of an aggregation of multiple similar returns, at Performance Payout level
	 */
	PortfolioReturnTerms []PortfolioReturnTerms
	/**
	 * Specifies the net initial valuation price(s) of the underlier at Performance Payout level. This price can be expressed either as an actual amount/currency, as a determination method, or by reference to another value specified in the swap document.
	 */
	InitialValuationPrice []ReferenceWithMeta
	/**
	 * Specifies the net initial valuation price(s) of the underlier at Performance Payout level. This price can be expressed either as an actual amount/currency, as a determination method, or by reference to another value specified in the swap document.
	 */
	InterimValuationPrice []ReferenceWithMeta
	/**
	 * Specifies the net final valuation price(s) of the underlier at Performance Payout level. This price can be expressed either as an actual amount/currency, as a determination method, or by reference to another value specified in the swap document.
	 */
	FinalValuationPrice []ReferenceWithMeta
}

/**
 * Defines how and when a performance type option or performance type swap is to be valued.
 */
type PerformanceValuationDates struct {
	/**
	 * Specifies the method according to which an amount or a date is determined.
	 */
	DeterminationMethod DeterminationMethodEnum
	/**
	 * 2018 ISDA CDM Equity Confirmation for Security Equity Swap: Pricing Date
	 */
	ValuationDates AdjustableRelativeOrPeriodicDates
	/**
	 * 2018 ISDA CDM Equity Confirmation for Security Equity Swap: Pricing Date
	 */
	ValuationDate AdjustableOrRelativeDate
	/**
	 * The specific time of day at which the calculation agent values the underlying. The SpecificTime is the only case when the valuationTime (time + business center location  e.g. 10:00:00 USNY) should be provided. You should be able to provide just the valuationTime without valuationTimeType, which infer that this is a specific time.
	 */
	ValuationTime BusinessCenterTime
	/**
	 * The time of day at which the calculation agent values the underlying, for example the official closing time of the exchange.
	 */
	ValuationTimeType TimeTypeEnum
	Meta              MetaFields
}

/**
 * A class to define recurring periods or time offsets.
 */
type Period struct {
	/**
	 * A time period multiplier, e.g. 1, 2 or 3 etc. A negative value can be used when specifying an offset relative to another date, e.g. -2 days.
	 */
	PeriodMultiplier int
	/**
	 * A time period, e.g. a day, week, month or year of the stream. If the periodMultiplier value is 0 (zero) then period must contain the value D (day).
	 */
	Period PeriodEnum
	Meta   MetaFields
}

/**
 * Indicator to specify if the period bound is defined as a period and whether the bound is inclusive.
 */
type PeriodBound struct {
	/**
	 * Specifies the period is to be used as the bound, e.g. 5Y.
	 */
	Period Period
	/**
	 * Specifies whether the period bound is inclusive, e.g. for a lower bound, false would indicate greater than, whereas true would indicate greater than or equal to.
	 */
	Inclusive bool
}

/**
 * Indicates The period range defined as either a lower and upper period bound, or both.
 */
type PeriodRange struct {
	/**
	 * Specifies the lower bound of a period range, e.g. greater than or equal to 5Y.
	 */
	LowerBound PeriodBound
	/**
	 * Specifies the upper bound of a period range, e.g. less than to 10Y.
	 */
	UpperBound PeriodBound
}

/**
 * A class for specifying a calculation period schedule.
 */
type PeriodicDates struct {
	/**
	 * The start date of the calculation period. FpML specifies that for interest rate swaps this date must only be specified if it is not equal to the effective date. It is always specified in the case of equity swaps and credit default swaps with periodic payments. This date may be subject to adjustment in accordance with a business day convention.
	 */
	StartDate AdjustableOrRelativeDate
	/**
	 * The end date of the calculation period. FpML specifies that for interest rate swaps this date must only be specified if it is not equal to the termination date. It is always specified in the case of equity swaps with periodic payments. This date may be subject to adjustment in accordance with a business day convention.
	 */
	EndDate AdjustableOrRelativeDate
	/**
	 * The frequency at which calculation period end dates occur with the regular part of the calculation period schedule and their roll date convention.
	 */
	PeriodFrequency CalculationPeriodFrequency
	/**
	 * The specification of the business day convention and financial business centers used for adjusting any calculation period date if it would otherwise fall on a day that is not a business day in the specified business center.
	 */
	PeriodDatesAdjustments BusinessDayAdjustments
	/**
	 * Denotes the enumerated values to specify the day type classification used in counting the number of days between two dates.
	 */
	DayType DayTypeEnum
}

/**
 * Comprises an identifier and a source. The associated metadata key denotes the ability to associate a hash value to the PersonIdentifier instantiations for the purpose of model cross-referencing, in support of functionality such as the event effect and the lineage.
 */
type PersonIdentifier struct {
	/**
	 * Provides an identifier associated with a person. The identifier is unique within the public source specified in the source attribute.
	 */
	Identifier FieldWithMeta
	/**
	 * Defines the source of the identifier.
	 */
	IdentifierType PersonIdentifierTypeEnum
	/**
	 * The ISO 3166 standard code for the country issuing the identifier.
	 */
	Country FieldWithMeta
	Meta    MetaFields
}

type PhysicalSettlementPeriod struct {
	/**
	 * An explicit indication that a number of business days are not specified and therefore ISDA fallback provisions should apply.
	 */
	BusinessDaysNotSpecified bool
	/**
	 * A number of business days. Its precise meaning is dependant on the context in which this element is used. ISDA 2003 Term: Business Day.
	 */
	BusinessDays int
	/**
	 * A maximum number of business days. Its precise meaning is dependant on the context in which this element is used. Intended to be used to limit a particular ISDA fallback provision.
	 */
	MaximumBusinessDays int
}

/**
 * Specifies Physical Settlement Terms characteristics for the settlement of a Credit Default Swap or Option.
 */
type PhysicalSettlementTerms struct {
	/**
	 * Specifies whether the swap resulting from physical settlement of the swaption transaction will clear through a clearing house. The meaning of Cleared Physical Settlement is defined in the 2006 ISDA Definitions, Section 15.2 (published in Supplement number 28).
	 */
	ClearedPhysicalSettlement bool
	/**
	 * Specifies the clearing organization (CCP, DCO) to which the trade should be cleared.
	 */
	PredeterminedClearingOrganizationParty AncillaryRoleEnum
	/**
	 * The number of business days used in the determination of the physical settlement date. The physical settlement date is this number of business days after all applicable conditions to settlement are satisfied. If a number of business days is not specified fallback provisions apply for determining the number of business days. If Section 8.5/8.6 of the 1999/2003 ISDA Definitions are to apply the businessDaysNotSpecified element should be included. If a specified number of business days are to apply these should be specified in the businessDays element. If Section 8.5/8.6 of the 1999/2003 ISDA Definitions are to apply but capped at a maximum number of business days then the maximum number should be specified in the maximumBusinessDays element. ISDA 2003 Term: Physical Settlement Period.
	 */
	PhysicalSettlementPeriod PhysicalSettlementPeriod
	/**
	 * This element contains all the ISDA terms relevant to defining the deliverable obligations.
	 */
	DeliverableObligations DeliverableObligations
	/**
	 * If this element is specified and set to 'true', indicates that physical settlement must take place through the use of an escrow agent. (For Canadian counterparties this is always 'Not Applicable'. ISDA 2003 Term: Escrow.
	 */
	Escrow bool
	/**
	 * If this element is specified and set to 'true', for a transaction documented under the 2003 ISDA Credit Derivatives Definitions, has the effect of incorporating the language set forth below into the confirmation. The section references are to the 2003 ISDA Credit Derivatives Definitions. Notwithstanding Section 1.7 or any provisions of Sections 9.9 or 9.10 to the contrary, but without prejudice to Section 9.3 and (where applicable) Sections 9.4, 9.5 and 9.6, if the Termination Date has not occurred on or prior to the date that is 60 Business Days following the Physical Settlement Date, such 60th Business Day shall be deemed to be the Termination Date with respect to this Transaction except in relation to any portion of the Transaction (an 'Affected Portion') in respect of which: (1) a valid notice of Buy-in Price has been delivered that is effective fewer than three Business Days prior to such 60th Business Day, in which case the Termination Date for that Affected Portion shall be the third Business Day following the date on which such notice is effective; or (2) Buyer has purchased but not Delivered Deliverable Obligations validly specified by Seller pursuant to Section 9.10(b), in which case the Termination Date for that Affected Portion shall be the tenth Business Day following the date on which Seller validly specified such Deliverable Obligations to Buyer.
	 */
	SixtyBusinessDaySettlementCap bool
	Meta                          MetaFields
}

/**
 * The terms of the Rider for the ISDA Euroclear 2019 Collateral Transfer Agreement with respect to the use of a Pledgee Representative attached to this Agreement.
 */
type PledgeeRepresentativeRider struct {
	/**
	 * Identification of whether the representative CTA provisions are applicable (True) or not applicable (False).
	 */
	IsApplicable bool
	/**
	 * Identification of the represented party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The specific representative terms applicable when specified.
	 */
	RepresentativeTerms string
	/**
	 * The specification of whether the representative event terms are applicable.
	 */
	RepresentativeEvent ExceptionEnum
	/**
	 * The specific representative event terms applicable when specified.
	 */
	RepresentativeEventTerms string
	/**
	 * The definition of representative end date in relation to a representative event.
	 */
	RepresentativeEndDate CustomisableOffset
}

/**
 *  A Portfolio represents an aggregation of multiple Positions, by describing the parameters that this Portfolio should be aggregated based on. The resulting PortfolioState is calculated using these aggregation parameters as inputs, by aggregating all the Events that are relevant to this Portfolio. The concept of Portfolio works at all levels in the model: from the highest for a given LegalEntity for instance, to the lowest to account for security substitutions in a secutity financing transaction. As such, Portfolio can be used either above or below the Contract level.
 */
type Portfolio struct {
	/**
	 * Describes the portfolio by describing how to aggregate all its relevant Events.
	 */
	AggregationParameters AggregationParameters
	/**
	 * Describes the state of the Portfolio as a list of Positions resulting from the aggregation.
	 */
	PortfolioState PortfolioState
}

/**
 * Specifies an individual type of return of a Performance Payout, when such individual return is part of an aggregation of multiple similar returns, at Performance Payout level.
 */
type PortfolioReturnTerms struct {
	/**
	 * Return terms based upon the underlier's observed price.
	 */
	PriceReturnTerms PriceReturnTerms
	/**
	 * Return terms based upon dividend payments associated to the underlier.
	 */
	DividendReturnTerms DividendReturnTerms
	/**
	 * Return terms based upon the observed variance of the underlier's price.
	 */
	VarianceReturnTerms VarianceReturnTerms
	/**
	 * Return terms based upon the observed volatility of the underlier's price.
	 */
	VolatilityReturnTerms VolatilityReturnTerms
	/**
	 * Return terms based upon the observed correlation between the components of the underlying basket.
	 */
	CorrelationReturnTerms CorrelationReturnTerms
	/**
	 * Canonical representation of the payer and receiver parties applicable to each individual return leg.
	 */
	PayerReceiver PayerReceiver
	/**
	 * Defines the product that is the subject of a tradable product definition, an underlying product definition, a physical exercise, a position, or other purposes.
	 */
	Underlier ReferenceWithMeta
	/**
	 * Specifies a quantity schedule for the underlier, which applies to each individual return leg.
	 */
	Quantity ReferenceWithMeta
	/**
	 * Specifies the initial valuation price(s) of the underlier. This price can be expressed either as an actual amount/currency, as a determination method, or by reference to another value specified in the swap document.
	 */
	InitialValuationPrice []ReferenceWithMeta
	/**
	 * Specifies the initial valuation price(s) of the underlier. This price can be expressed either as an actual amount/currency, as a determination method, or by reference to another value specified in the swap document.
	 */
	InterimValuationPrice []ReferenceWithMeta
	/**
	 * 2018 ISDA CDM Equity Confirmation for Security Equity Swap: Final Price | Specifies the final valuation price of the underlier. This price can be expressed either as an actual amount/currency, as a determination method, or by reference to another value specified in the swap document.
	 */
	FinalValuationPrice []ReferenceWithMeta
	Meta                MetaFields
}

/**
 * State-full representation of a Portfolio that describes all the positions held at a given time, in various states which can be either traded, settled, etc., with lineage information to the previous state
 */
type PortfolioState struct {
	/**
	 * The list of positions, each containing a Quantity and a Product.
	 */
	Positions []Position
	/**
	 * Pointer to the previous PortfolioState and new Event(s) leading to the current (new) state. Previous PortfolioState in the Lineage can be Null in case this is the start of the chain of Events.
	 */
	Lineage Lineage
	Meta    MetaFields
}

/**
 * A Position describes how much of a given Product is being held and constitutes the atomic element of a Portfolio.
 */
type Position struct {
	/**
	 * Position with many price quantities.
	 */
	PriceQuantity []PriceQuantity
	/**
	 * The product underlying the position.
	 */
	Product Product
	/**
	 * The asset that underlies the position.
	 */
	Asset Asset
	/**
	 * The aggregate cost of proceeds.
	 */
	CashBalance Money
	/**
	 * Reference to the Contract, in case product is contractual and the contract has been formed.
	 */
	TradeReference ReferenceWithMeta
	/**
	 * The product underlying the position.
	 */
}

/**
 * Base class to describe a Position, which could be a group of products (e.g. trade portfolio) or a group of assets (e.g. collateral portfolio).
 */
type PositionBase struct {
	/**
	 * Position with many price quantities.
	 */
	PriceQuantity []PriceQuantity
	/**
	 * The product underlying the position.
	 */
	Product Product
	/**
	 * The asset that underlies the position.
	 */
	Asset Asset
	/**
	 * The aggregate cost of proceeds.
	 */
	CashBalance Money
	/**
	 * Reference to the Contract, in case product is contractual and the contract has been formed.
	 */
	TradeReference ReferenceWithMeta
}

/**
 * Defines a position identifier as a special case of the generic identifier type, that also includes the position identifier class.
 */
type PositionIdentifier struct {
	/**
	 * The identifier issuer, when specified by reference to a party specified as part of the transaction.
	 */
	IssuerReference ReferenceWithMeta
	/**
	 * The identifier issuer, when specified explicitly alongside the identifier value (instead of being specified by reference to a party).
	 */
	Issuer FieldWithMeta
	/**
	 * The identifier value. This level of indirection between the issuer and the identifier and its version provides the ability to associate multiple identifiers to one issuer, consistently with the FpML PartyTradeIdentifier.
	 */
	AssignedIdentifier []AssignedIdentifier
	Meta               MetaFields
	/**
	 * The enumerated classification of the identifier. Optional as a position identifier may be party-specific, in which case it may not correspond to any established classification.
	 */
	IdentifierType TradeIdentifierTypeEnum
}

/**
 * Posted Credit Support item with corresponding Valuation Percentage, FX Haircut Percentage and any related disputed Posted Credit Support valuation.
 */
type PostedCreditSupportItem struct {
	/**
	 * The Base Currency Equivalent of Cash or Security.
	 */
	CashOrSecurityValue Money
	/**
	 * Valuation Percentage means, for any item of Eligible Collateral (IM), the percentage specified in accordance with Paragraph 13.
	 */
	HaircutPercentage float64
	/**
	 * FX Haircut Percentage means, for any item of Eligible Collateral (IM), the percentage specified in accordance with Paragraph 13.
	 */
	FxHaircutPercentage float64
	/**
	 * Percentage value of any additional haircut to be applied to a collateral asset,the percentage value is expressed as the discount haircut to the value of the collateral- as an example a 5% haircut would be expressed as 0.05.
	 */
	AdditionalHaircutPercentage float64
	/**
	 * Paragraph 5. If a party (a 'Disputing Party') disputes the Value of any Posted Credit Support (IM).
	 */
	DisputedCashOrSecurityValue Money
}

/**
 * A class to specify the collateral posting obligations of the security provider or security providers as specified in the corresponding agreement.
 */
type PostingObligations struct {
	/**
	 * The security provider party(ies) to which the posting obligations apply to, which can be either one of the parties to the legal agreement, or both of those.
	 */
	SecurityProvider []CounterpartyRoleEnum
	/**
	 * The specification of the collateral posting obligations for the security provider party(ies).
	 */
	PartyElection []PostingObligationsElection
}

/**
 * Specifies the collateral posting obligations for the security provider party(ies).
 */
type PostingObligationsElection struct {
	/**
	 * Specifies the party providing the security.
	 */
	Party CounterpartyRoleEnum
	/**
	 * If set to True, the Control Agreement is a Credit Support Document with respect to the party(ies).
	 */
	AsPermitted bool
	/**
	 * The eligible collateral as specified in relation to the pledgor/chargor/obligor(s) posting obligation.
	 */
	EligibleCollateral []EligibleCollateralCriteria
	/**
	 * The excluded collateral as specified in relation to the pledgor/chargor/obligor(s) posting obligation.
	 */
	ExcludedCollateral string
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * This class corresponds to the FpML Premium.model group for representing the option premium when expressed in a way other than an amount.
 */
type PremiumExpression struct {
	/**
	 * Forward start premium type
	 */
	PremiumType PremiumTypeEnum
	/**
	 * The amount of premium to be paid expressed as a function of the number of options.
	 */
	PricePerOption Money
	/**
	 * The amount of premium to be paid expressed as a percentage of the notional value of the transaction. A percentage of 5% would be expressed as 0.05.
	 */
	PercentageOfNotional float64
}

type Pric struct {
	Pric    *Pric
	BsisPts string
}

/**
 * Specifies a price as a single value to be associated to a financial product. This data type extends PriceSchedule and requires that only the amount value exists.
 */
type Price struct {
	/**
	 * Specifies the value of the measure as a number. Optional because in a measure vector or schedule, this single value may be omitted.
	 */
	Value float64
	/**
	 * Qualifies the unit by which the amount is measured. Optional because a measure may be unit-less (e.g. when representing a ratio between amounts in the same unit).
	 */
	Unit UnitType
	/**
	 * A schedule of step date and value pairs. On each step date the associated step value becomes effective. The step dates are used to order the steps by ascending order. This attribute is optional so the data type may be used to define a schedule with a single value.
	 */
	DatedValue []DatedValue
	/**
	 * Provides an attribute to define the unit of the thing being priced. For example, {amount, unitOfAmount, PerUnitOfAmount} = [10, EUR, Shares] = (10.00 EUR/SHARE) * (300,000 SHARES) = EUR 3,000,000.00 (Shares cancel out in the calculation).
	 */
	PerUnitOf UnitType
	/**
	 * Specifies the price type as an enumeration: interest rate, exchange rate, asset price etc. This attribute is mandatory so that prices can always be clasiffied according to their type. The price type implies some constraints on the price's units.
	 */
	PriceType PriceTypeEnum
	/**
	 * Allows further classification of the chosen price type.
	 */
	PriceSubType PriceSubTypeEnum
	/**
	 * (Optionally) Specifies whether the price is expressed in absolute or percentage terms.
	 */
	PriceExpression PriceExpressionEnum
	/**
	 * (Optionally) Specifies the underlying price components if the price can be expressed as a composite: e.g. dirty price = clean price + accrued.
	 */
	Composite PriceComposite
	/**
	 * (Optionally) When the price is to be understood as an operator to apply to an observable, i.e. a spread, multiplier or min/max.
	 */
	ArithmeticOperator ArithmeticOperationEnum
	/**
	 * (Optionally) Additional attribute that can further define any premium to the price.
	 */
	PremiumType PremiumTypeEnum
}

/**
 * Defines the inputs required to calculate a price as a simple composite of 2 other values. The inputs consist of 2 numbers and a simple arithmetic operator. This generic data type applies to a variety of use cases where a price is obtained by simple composition, e.g. dirty = clean + accrued (Bond), forward rate = spot rate + forward point (FX) etc.
 */
type PriceComposite struct {
	/**
	 * The 1st value in the arithmetic operation, which may be non-commutative in some cases: Subtract, Divide). This 1st operand is called 'baseValue' as it refers to the price anchor in the arithmetic operation: e.g. the clean price (Bond) or the spot rate (FX).
	 */
	BaseValue float64
	/**
	 * The 2nd value in the arithmetic operation, which may be non-commutative in some cases: Subtract, Divide). The 2nd operand is called 'operand' to distinguish it from the 1st one which is the price anchor.
	 */
	Operand float64
	/**
	 * Specifies the arithmetic operator via an enumeration.
	 */
	ArithmeticOperator ArithmeticOperationEnum
	/**
	 * Optionally qualifies the type of operand: e.g. accrued or forward point.
	 */
	OperandType PriceOperandEnum
}

type PriceMateriality struct {
	PriceMaterialityIsApplicable bool
	EscrowArrangement            EscrowArrangement
	/**
	 * The method, prioritised by the order it is listed in this element, to get a replacement rate for the disrupted settlement rate option.
	 */
	FallbackReferencePrice SettlementRateOption
}

/**
 * Defines a settlement as an exchange between two parties of a specified quantity of an asset (the quantity) against a specified quantity of another asset (the price). The settlement is optional and can be either cash or physical. The quantity can additionally be specified in terms of one or more currency amounts. In the case of non-cash products, the settlement of the price/quantity would not be specified here and instead would be delegated to the product mechanics, as parameterised by the price/quantity values.
 */
type PriceQuantity struct {
	/**
	 * Specifies a price to be used for trade amounts and other purposes.
	 */
	Price []FieldWithMeta
	/**
	 * Specifies a quantity to be associated with an event, for example a trade amount.
	 */
	Quantity []FieldWithMeta
	/**
	 * Specifies the object to be observed for a price, it could be an asset or an index. The cardinality is optional as some quantity / price cases have no observable (e.g. a fixed rate in a given currency).
	 */
	Observable FieldWithMeta
	/**
	 * Specifies the date at which the price and quantity become effective. This day may be subject to adjustment in accordance with a business day convention, or could be specified as relative to a trade date, for instance. Optional cardinality, as the effective date is usually specified in the product definition, so it may only need to be specified as part of the PriceQuantity in an increase/decrease scenario for an existing trade.
	 */
	EffectiveDate AdjustableOrRelativeDate
	Meta          MetaFields
}

type PriceReturnTerms struct {
	/**
	 * The type of return associated with the equity swap.
	 */
	ReturnType ReturnTypeEnum
	/**
	 * Defines the conversion applied if the quantity unit on contract is different from unit on referenced underlier.
	 */
	ConversionFactor float64
	/**
	 * Performance calculation, in accordance with Part 1 Section 12 of the 2018 ISDA CDM Equity Confirmation for Security Equity Swap, Para 75. 'Equity Performance'. Cumulative performance is used as a notional multiplier factor on both legs of an Equity Swap.
	 */
	Performance string
}

/**
 * Specifies the price of a financial instrument in a trade as a schedule of measures. A price generically expresses the value of an exchange as a ratio: it measures the amount of one thing needed to be exchanged for 1 unit of another thing (e.g. cash in a specific currency in exchange for a bond or share). This generic representation can be used to support any type of financial price beyond just cash price: e.g. an interest rate, a foreign exchange rate, etc. This data type is generically based on a schedule and can also be used to represent a price as a single value.
 */
type PriceSchedule struct {
	/**
	 * Specifies the value of the measure as a number. Optional because in a measure vector or schedule, this single value may be omitted.
	 */
	Value float64
	/**
	 * Qualifies the unit by which the amount is measured. Optional because a measure may be unit-less (e.g. when representing a ratio between amounts in the same unit).
	 */
	Unit UnitType
	/**
	 * A schedule of step date and value pairs. On each step date the associated step value becomes effective. The step dates are used to order the steps by ascending order. This attribute is optional so the data type may be used to define a schedule with a single value.
	 */
	DatedValue []DatedValue
	/**
	 * Provides an attribute to define the unit of the thing being priced. For example, {amount, unitOfAmount, PerUnitOfAmount} = [10, EUR, Shares] = (10.00 EUR/SHARE) * (300,000 SHARES) = EUR 3,000,000.00 (Shares cancel out in the calculation).
	 */
	PerUnitOf UnitType
	/**
	 * Specifies the price type as an enumeration: interest rate, exchange rate, asset price etc. This attribute is mandatory so that prices can always be clasiffied according to their type. The price type implies some constraints on the price's units.
	 */
	PriceType PriceTypeEnum
	/**
	 * Allows further classification of the chosen price type.
	 */
	PriceSubType PriceSubTypeEnum
	/**
	 * (Optionally) Specifies whether the price is expressed in absolute or percentage terms.
	 */
	PriceExpression PriceExpressionEnum
	/**
	 * (Optionally) Specifies the underlying price components if the price can be expressed as a composite: e.g. dirty price = clean price + accrued.
	 */
	Composite PriceComposite
	/**
	 * (Optionally) When the price is to be understood as an operator to apply to an observable, i.e. a spread, multiplier or min/max.
	 */
	ArithmeticOperator ArithmeticOperationEnum
	/**
	 * (Optionally) Additional attribute that can further define any premium to the price.
	 */
	PremiumType PremiumTypeEnum
}

/**
 * Specifies a publication that provides the commodity price, including, where applicable, the details of where in the publication the price is published.
 */
type PriceSource struct {
	/**
	 * Defines a publication in which the price can be found. (e.g Gas Daily, Platts Bloomberg. Commodity publishers can be found at this URL:  http://www.fpml.org/coding-scheme/commodity-information-provider>
	 */
	PricePublisher FieldWithMeta
	/**
	 * Specifies the location of the price which may be a specific page, electornic screen name, or a code (e.g. a RIC code) where the price can be found.
	 */
	PriceSourceLocation string
	/**
	 * Specifies the heading or field name for the price  on a given page or screen, where applicable.
	 */
	PriceSourceHeading string
	/**
	 * Specifies the time at which the price should be observed.
	 */
	PriceSourceTime time.Time
}

/**
 * A data defining:  the parameters used to get a price quote to replace the settlement rate option that is disrupted.
 */
type PriceSourceDisruption struct {
	/**
	 * The method, prioritised by the order it is listed in this element, to get a replacement rate for the disrupted settlement rate option.
	 */
	FallbackReferencePrice FallbackReferencePrice
}

/**
 * Specifies specific dates or parametric rules for the dates on which the price will be determined
 */
type PricingDates struct {
	/**
	 * Defines specified dates on which the price will be determined.
	 */
	SpecifiedDates []AdjustableDates
	/**
	 * Defines rules for the dates on which the price will be determined.
	 */
	ParametricDates ParametricDates
}

/**
 * A Primitive Instruction describes the inputs required to pass into the corresponding PrimitiveEvent function.
 */
type PrimitiveInstruction struct {
	/**
	 * Specifies instructions describing an contract formation primitive event.
	 */
	ContractFormation ContractFormationInstruction
	/**
	 * Specifies instructions describing an execution primitive event.
	 */
	Execution ExecutionInstruction
	/**
	 * Specifies instructions describing an exercise primitive event.
	 */
	Exercise *ExerciseInstruction
	/**
	 * Specifies instructions describing a party change primitive event.
	 */
	PartyChange PartyChangeInstruction
	/**
	 * Specifies instructions describing an quantity change primitive event.
	 */
	QuantityChange QuantityChangeInstruction
	/**
	 * Specifies instructions describing a reset event.
	 */
	Reset ResetInstruction
	/**
	 * Specifies instructions to split a trade into multiple branches.
	 */
	Split SplitInstruction
	/**
	 * Specifies instructions describing a terms change primitive event.
	 */
	TermsChange TermsChangeInstruction
	/**
	 * Specifies instructions describing a transfer primitive event.
	 */
	Transfer TransferInstruction
	/**
	 * Specifies inputs needed to process a Index Transition business event.
	 */
	IndexTransition IndexTransitionInstruction
	/**
	 * Specifies inputs needed to process a Stock Split business event.
	 */
	StockSplit StockSplitInstruction
	/**
	 * Specifies inputs needed to process an observation.
	 */
	Observation ObservationInstruction
	/**
	 * Specifies inputs needed to process an update of a valuation.
	 */
	Valuation ValuationInstruction
}

/**
 * Any kind of principal payments when the amount is known and thus fixed.
 */
type PrincipalPayment struct {
	/**
	 * The date where the PrincipalPayment shall be settled.
	 */
	PrincipalPaymentDate AdjustableDate
	/**
	 * Specifies the parties responsible for making and receiving payments defined by this structure.
	 */
	PayerReceiver PayerReceiver
	/**
	 * When known at the time the transaction is made, the cash amount to be paid.
	 */
	PrincipalAmount Money
	/**
	 * The value representing the discount factor used to calculate the present value of the principal payment amount.
	 */
	DiscountFactor float64
	/**
	 * The amount representing the present value of the principal payment.
	 */
	PresentValuePrincipalAmount Money
	Meta                        MetaFields
}

/**
 * Describe dates schedules for Principal Exchanges and related role of the parties when known.
 */
type PrincipalPaymentSchedule struct {
	/**
	 * Principal Payment made at Trade inception.
	 */
	InitialPrincipalPayment PrincipalPayment
	/**
	 * Principal Payment as part of the Trade lifecycle e.g. as part of notional reset adjustements in a Cross Currency Swap with a varying notional leg.
	 */
	IntermediatePrincipalPayment AdjustableRelativeOrPeriodicDates
	/**
	 * Principal Payment at Trade maturity
	 */
	FinalPrincipalPayment PrincipalPayment
}

/**
 * A class defining which principal exchanges occur for the stream.
 */
type PrincipalPayments struct {
	/**
	 * A true/false flag to indicate whether there is an initial exchange of principal on the effective date.
	 */
	InitialPayment bool
	/**
	 * A true/false flag to indicate whether there is a final exchange of principal on the termination date.
	 */
	FinalPayment bool
	/**
	 * A true/false flag to indicate whether there are intermediate or interim exchanges of principal during the term of the swap.
	 */
	IntermediatePayment bool
	/**
	 * Indicate the Payout legs which nominal amount may vary in regards of FX Fixing dates as determined in the product terms.
	 */
	VaryingLegNotionalCurrency []string
	/**
	 * Describe dates schedules for Principal Exchanges and related role of the parties when known.
	 */
	PrincipalPaymentSchedule PrincipalPaymentSchedule
	Meta                     MetaFields
}

/**
 * Specifies the Process Agent that might be appointed by the parties as part of a Credit Support Annex/Deed or Collateral Transfer Agreement.
 */
type ProcessAgent struct {
	/**
	 * The parties' Process Agent election.
	 */
	PartyElection []ProcessAgentElection
}

/**
 * Specifies the parties' respective elections with respect to the Process Agent.
 */
type ProcessAgentElection struct {
	/**
	 * The party to which the election and details apply.
	 */
	PartyReference CounterpartyRoleEnum
	/**
	 * The main notice/address information of the party.
	 */
	PrimaryContactInformation ContactInformation
	/**
	 * The qualification of whether the Process Agent is applicable (True) or not applicable (False).
	 */
	IsApplicable bool
	/**
	 * Specification of the process agent entity.
	 */
	Entity LegalEntity
	/**
	 * Additional information relating to the Process Agent.
	 */
	AdditionalInformation string
}

/**
 * Enables either a TransferableProduct or a NonTransferableProduct to be used in an underlier.
 */
type Product struct {
	/**
	 * A TransferableProduct is a type of financial product which can be held or transferred, represented as an Asset with the addition of specific EconomicTerms.
	 */
	TransferableProduct TransferableProduct
	/**
	 * The non-transferable product data type represents a product that can be traded (as part of a TradableProduct) but cannot be transferred to others.
	 */
	NonTransferableProduct NonTransferableProduct
}

/**
 * Comprises an identifier and a source. The associated metadata key denotes the ability to associate a hash value to the ProductIdentifier instantiations for the purpose of model cross-referencing, in support of functionality such as the event effect and the lineage.
 */
type ProductIdentifier struct {
	/**
	 * Provides an identifier associated with a specific product.  The identifier is unique within the public source specified in the source attribute.
	 */
	Identifier FieldWithMeta
	/**
	 * Defines the source of the identifier.
	 */
	Source ProductIdTypeEnum
	Meta   MetaFields
}

/**
 * Specifies the product taxonomy, which is composed of a taxonomy value and a taxonomy source.
 */
type ProductTaxonomy struct {
	/**
	 * The source of the taxonomy that defines the rules for classifying the object. The taxonomy source is taken from a enumerated list of taxonomy names. Optional as the taxonomy source may not be provided.
	 */
	Source TaxonomySourceEnum
	/**
	 * The value according to that taxonomy. Optional as it may not be possible to classify the object in that taxonomy.
	 */
	Value TaxonomyValue
	/**
	 * Classifies the most important risk class of the trade.
	 */
	PrimaryAssetClass FieldWithMeta
	/**
	 *  Classifies additional risk classes of the trade, if any.
	 */
	SecondaryAssetClass []FieldWithMeta
	/**
	 * Derived from the product payout features using a CDM product qualification function that determines the product type based on the product payout features.
	 */
	ProductQualifier string
}

/**
 * A class to specify the terms for calculating a payout to protect the buyer of the swap in the case of a qualified credit event. These terms include the applicable credit events, the reference obligation, and in the case of a CDS on mortgage-backed securities, the floatingAmountEvents.
 */
type ProtectionTerms struct {
	/**
	 * Specifies the applicable Credit Events that would trigger a settlement, as specified in the related Confirmation and defined in the ISDA 2014 Credit Definition article IV section 4.1.
	 */
	CreditEvents CreditEvents
	/**
	 * The underlying obligations of the reference entity on which you are buying or selling protection. The credit events Failure to Pay, Obligation Acceleration, Obligation Default, Restructuring, Repudiation/Moratorium are defined with respect to these obligations.
	 */
	Obligations Obligations
	/**
	 * This element contains the ISDA terms relating to the floating rate payment events and the implied additional fixed payments, applicable to the credit derivatives transactions on mortgage-backed securities with pay-as-you-go or physical settlement.
	 */
	FloatingAmountEvents FloatingAmountEvents
	Meta                 MetaFields
}

type Prsn struct {
	CtryOfBrnch string
	Othr        Othr
}

type PubliclyAvailableInformation struct {
	/**
	 * If this element is specified and set to 'true', indicates that ISDA defined Standard Public Sources are applicable.
	 */
	StandardPublicSources bool
	/**
	 * A public information source, e.g. a particular newspaper or electronic news service, that may publish relevant information used in the determination of whether or not a credit event has occurred. ISDA 2003 Term: Public Source.
	 */
	PublicSource []string
	/**
	 * The minimum number of the specified public information sources that must publish information that reasonably confirms that a credit event has occurred. The market convention is two. ISDA 2003 Term: Specified Number.
	 */
	SpecifiedNumber int
}

type Qty struct {
	Unit string
}

/**
 * Specifies a quantity as a single value to be associated to a financial product, for example a transfer amount resulting from a trade. This data type extends QuantitySchedule and requires that only the single amount value exists.
 */
type Quantity struct {
	/**
	 * Specifies the value of the measure as a number. Optional because in a measure vector or schedule, this single value may be omitted.
	 */
	Value float64
	/**
	 * Qualifies the unit by which the amount is measured. Optional because a measure may be unit-less (e.g. when representing a ratio between amounts in the same unit).
	 */
	Unit UnitType
	/**
	 * A schedule of step date and value pairs. On each step date the associated step value becomes effective. The step dates are used to order the steps by ascending order. This attribute is optional so the data type may be used to define a schedule with a single value.
	 */
	DatedValue []DatedValue
	/**
	 * Defines an optional number that the quantity should be multiplied by to derive a total quantity. This number is associated to a unit. For example in the case of the Coal (API2) CIF ARA (ARGUS-McCloskey) Futures Contract on the CME, where the unit would be contracts, the multiplier value would 1,000 and the mulitiplier unit would be 1,000 MT (Metric Tons).
	 */
	Multiplier Measure
	/**
	 * Defines the frequency to be used when defining a quantity. For example a quantity may be specified as a number of barrels of oil per day, which needs multiplying by the number of days in the relevant period to get the total quantity as a number of barrels.
	 */
	Frequency Frequency
}

/**
 * Instructions required to create a Quantity Change Primitive Event, which can be either an increase, a decrease or a replacement. An increase adds a new trade lot to the original trade, whereas a decrease subtracts from an existing trade lot's quantity. A replacement updates the quantity of an existing trade lot to the new value.
 */
type QuantityChangeInstruction struct {
	/**
	 * Quantity by which the trade is being increased, decreased or replaced, and the price at which such quantity change is agreed. The quantity change should always be specified as a positive number, with the direction (increase/decrease/replacement) being specified by the direction enumeration. A fee can also be associated to the quantity change by specifying a Price component of type CashPrice, including the corresponding settlement date and direction.
	 */
	Change []PriceQuantity
	/**
	 * Direction of the quantity change specified as either an increase, decrease or replacement.
	 */
	Direction QuantityChangeDirectionEnum
	/**
	 * Identifier for the new lot (in case of increase) or for the existing lot to be changed(in case of decrease or replacement). This optional attribute is mandatory in case of a decrease or replacement if the initial trade state contains multiple trade lots.
	 */
	LotIdentifier []Identifier
}

/**
 *  Class to specify a mechanism for a quantity to be set as a multiplier to another (reference) quantity, based on a price observation. At the moment this class only supports FX or Equity-linked notional and re-uses existing building blocks for those 2 cases, until such time when component can be made more generic. This captures the case of resetting cross-currency swaps and resetting equity swaps.
 */
type QuantityMultiplier struct {
	/**
	 * Multiplier specified as an FX-linked schedule, e.g. for a resetting cross-currency swap..
	 */
	FxLinkedNotionalSchedule FxLinkedNotionalSchedule
	MultiplierValue          float64
}

/**
 * Specifies a quantity schedule to be associated to a financial product to represent a trade amount. This data type extends MeasureSchedule with several unit or multiplier attributes that are used to define financial quantities. This data type is generically based on a schedule and can also be used to represent a quantity as a single value.
 */
type QuantitySchedule struct {
	/**
	 * Specifies the value of the measure as a number. Optional because in a measure vector or schedule, this single value may be omitted.
	 */
	Value float64
	/**
	 * Qualifies the unit by which the amount is measured. Optional because a measure may be unit-less (e.g. when representing a ratio between amounts in the same unit).
	 */
	Unit UnitType
	/**
	 * A schedule of step date and value pairs. On each step date the associated step value becomes effective. The step dates are used to order the steps by ascending order. This attribute is optional so the data type may be used to define a schedule with a single value.
	 */
	DatedValue []DatedValue
	/**
	 * Defines an optional number that the quantity should be multiplied by to derive a total quantity. This number is associated to a unit. For example in the case of the Coal (API2) CIF ARA (ARGUS-McCloskey) Futures Contract on the CME, where the unit would be contracts, the multiplier value would 1,000 and the mulitiplier unit would be 1,000 MT (Metric Tons).
	 */
	Multiplier Measure
	/**
	 * Defines the frequency to be used when defining a quantity. For example a quantity may be specified as a number of barrels of oil per day, which needs multiplying by the number of days in the relevant period to get the total quantity as a number of barrels.
	 */
	Frequency Frequency
}

/**
 * Determines the currency rate that the seller of the equity amounts will apply at each valuation date for converting the respective amounts into a currency that is different from the currency denomination of the underlier.
 */
type Quanto struct {
	/**
	 * Specifies a currency conversion rate.
	 */
	FxRate []FxRate
	/**
	 * Specifies the methodology (reference source and, optionally, fixing time) to be used for determining a currency conversion rate.
	 */
	FxSpotRateSource FxSpotRateSource
	/**
	 * The time at which the spot currency exchange rate will be observed. It is specified as a time in a business day calendar location, e.g. 11:00am London time.
	 */
	FixingTime BusinessCenterTime
}

/**
 * Represents a class to allow specification of different types of Quasi Government collateral.
 */
type QuasiGovernmentIssuerType struct {
	/**
	 * True if sovereign entity (e.g. not separate legal personality from sovereign) or false if non-sovereign entity (e.g. separate legal personality from sovereign).
	 */
	SovereignEntity bool
	/**
	 * Applies to non-sovereign entity (e.g. separate legal personality from sovereign).  True if entity has recourse to sovereign (e.g. debt guaranteed by government).  False if entity does not have recourse to sovereign.
	 */
	SovereignRecourse bool
}

/**
 * A class that describes the composition of a rate that has been quoted or is to be quoted. This includes the two currencies and the quotation relationship between the two currencies and is used as a building block throughout the FX specification.
 */
type QuotedCurrencyPair struct {
	/**
	 * The first currency specified when a pair of currencies is to be evaluated.
	 */
	Currency1 FieldWithMeta
	/**
	 * The second currency specified when a pair of currencies is to be evaluated.
	 */
	Currency2 FieldWithMeta
	/**
	 * The method by which the exchange rate is quoted.
	 */
	QuoteBasis QuoteBasisEnum
}

/**
 * A class defining parameters associated with an individual observation or fixing. This class forms part of the cashflow representation of a stream.
 */
type RateObservation struct {
	/**
	 * The reset date.
	 */
	ResetDate time.Time
	/**
	 * The adjusted fixing date, i.e. the actual date the rate is observed. The date should already be adjusted for any applicable business day convention.
	 */
	AdjustedFixingDate time.Time
	/**
	 * The actual observed rate before any required rate treatment is applied, e.g. before converting a rate quoted on a discount basis to an equivalent yield. An observed rate of 5% would be represented as 0.05.
	 */
	ObservedRate float64
	/**
	 * The observed rate after any required rate treatment is applied. A treated rate of 5% would be represented as 0.05.
	 */
	TreatedRate float64
	/**
	 * The number of days weighting to be associated with the rate observation, i.e. the number of days such rate is in effect. This is applicable in the case of a weighted average method of calculation where more than one reset date is established for a single calculation period.
	 */
	ObservationWeight int
	/**
	 * A pointer style reference to a floating rate component defined as part of a stub calculation period amount component. It is only required when it is necessary to distinguish two rate observations for the same fixing date which could occur when linear interpolation of two different rates occurs for a stub calculation period.
	 */
	RateReference ReferenceWithMeta
	/**
	 * The value representing the forecast rate used to calculate the forecast future value of the accrual period.A value of 1% should be represented as 0.01.
	 */
	ForecastRate float64
	/**
	 * The value representing the forecast rate after applying rate treatment rules. A value of 1% should be represented as 0.01.
	 */
	TreatedForecastRate float64
	Meta                MetaFields
}

/**
 * A class defining a schedule of rates or amounts in terms of an initial value and then a series of step date and value pairs. On each step date the rate or amount changes to the new step value. The series of step date and value pairs are optional. If not specified, this implies that the initial value remains unchanged over time.
 */
type RateSchedule struct {
	/**
	 * The initial rate. An initial rate of 5% would be represented as 0.05.
	 */
	Price ReferenceWithMeta
}

/**
 *  A data type to specify the fixed interest rate, floating interest rate or inflation rate.
 */
type RateSpecification struct {
	/**
	 * The fixed rate or fixed rate specification expressed as explicit fixed rates and dates.
	 */
	FixedRateSpecification FixedRateSpecification
	/**
	 * The floating interest rate specification, which includes the definition of the floating rate index. the tenor, the initial value, and, when applicable, the spread, the rounding convention, the averaging method and the negative interest rate treatment.
	 */
	FloatingRateSpecification FloatingRateSpecification
	/**
	 * An inflation rate calculation definition.
	 */
	InflationRateSpecification InflationRateSpecification
}

/**
 * A logical container to hold a defined set of related data. In this case the collation of Rating Information and an associated Independent Amount.
 */
type RatingAgencyAmount struct {
	/**
	 * Allows a grouping of Rating agencies and Rating values.
	 */
	RatingVariableSet []CreditNotation
	/**
	 * Specifies the notional or collateral Independent Amount.
	 */
	Amount float64
}

/**
 * A logical container designed to hold a set of related data. In this case the Rating Agency, Rating Value and Rating Type.
 */
type RatingAgencyValue struct {
	/**
	 * Specifies The credit agency to which the other variables (notation, scale, debt type) refer to.
	 */
	Agency CreditRatingAgencyEnum
	/**
	 * Specifies The credit rating notation. As it varies among credit rating agencies, FpML doesn't specify a default scheme.
	 */
	Notation FieldWithMeta
	/**
	 * Specifies the credit rating scale, with a typical distinction between short term, long term. FpML doesn't specify a default scheme, which is hence not specified as an enumeration as part of the CDM.
	 */
	Scale FieldWithMeta
	/**
	 * Specifies the credit rating debt type (e.g. long term, high yield, deposits, ...) associated with the credit rating notation and scale.
	 */
	Debt CreditRatingDebt
	/**
	 * Assesses the potential direction of a long-term credit rating over the intermediate term, which is generally up to two years for investment grade and generally up to one year for speculative grade.
	 */
	Outlook CreditRatingOutlookEnum
	/**
	 * Indicates the potential direction of a short-term or long-term rating. It focuses on identifiable events and short-term trends that cause ratings to be placed under special surveillance.
	 */
	CreditWatch CreditRatingCreditWatchEnum
	/**
	 * The relevant rating type.
	 */
	RatingType RatingTypeEnum
}

/**
 * A logical container to hold a defined set of related data. In this case the collation of Ratings Information and associated exposure multiplier.
 */
type RatingMultiplier struct {
	/**
	 * Allows a grouping of Rating agencies and Rating values.
	 */
	RatingVariableSet []CreditNotation
	/**
	 * Specifies the multiplier applied to exposure based on the credit rating of the rated party.
	 */
	ExposureMultiplier float64
}

/**
 * A class to specify terms for Recalculation of the Market Value of Posted Collateral when a dispute has been failed to be resolved by Resolution Time.
 */
type RecalculationOfValue struct {
	/**
	 * The parties' Recalculation of Value terms.
	 */
	PartyElection []RecalculationOfValueElection
}

/**
 * A class to specify Recalculation of Value terms that will be applicable.
 */
type RecalculationOfValueElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The procedure for Recalculation of Value.
	 */
	RecalculationOfValueElection RecalculationOfValueElectionEnum
	/**
	 * Additional Recalculation of Value terms when specified.
	 */
	RecalculationOfValueTerms string
}

type RefRate struct {
	Indx string
	Nm   string
}

/**
 * A class to describe an institution (party) identified by means of a coding scheme and an optional name.
 */
type ReferenceBank struct {
	/**
	 * An institution (party) identifier, e.g. a bank identifier code (BIC). FpML specifies a referenceBankIdScheme.
	 */
	ReferenceBankId FieldWithMeta
	/**
	 * The name of the institution (party). A free format string. FpML does not define usage rules for the element.
	 */
	ReferenceBankName string
}

/**
 * A class defining the list of reference institutions polled for relevant rates or prices when determining the cash settlement amount for a product where cash settlement is applicable.
 */
type ReferenceBanks struct {
	/**
	 * An institution (party) identified by means of a coding scheme and an optional name.
	 */
	ReferenceBank []ReferenceBank
}

/**
 * A class specifying the Credit Default Swap Reference Information.
 */
type ReferenceInformation struct {
	/**
	 * The corporate or sovereign entity which is subject to the swap transaction and any successor that assumes all or substantially all of its contractual and other obligations. Reference Entities cannot be senior or subordinated. It is the obligations of the Reference Entities that can be senior or subordinated. ISDA 2014 Credit definitions article II section 2.1: `Reference Entity` means the entity specified as such in the related Confirmation.
	 */
	ReferenceEntity LegalEntity
	/**
	 * The Reference Obligation is a financial instrument that is either issued or guaranteed by the reference entity. It serves to clarify the precise reference entity protection is being offered upon, and its legal position with regard to other related firms (parents/subsidiaries). Furthermore the Reference Obligation is ALWAYS deliverable and establishes the Pari Passu ranking (as the deliverable bonds must rank equal to the reference obligation). ISDA 2003 Term: Reference Obligation.
	 */
	ReferenceObligation []ReferenceObligation
	/**
	 * Used to indicate that there is no Reference Obligation associated with this Credit Default Swap and that there will never be one.
	 */
	NoReferenceObligation bool
	/**
	 * Used to indicate that the Reference obligation associated with the Credit Default Swap is currently not known. This is not valid for Legal Confirmation purposes, but is valid for earlier stages in the trade life cycle (e.g. Broker Confirmation).
	 */
	UnknownReferenceObligation bool
	/**
	 * Indicates whether an obligation of the Reference Entity, guaranteed by the Reference Entity on behalf of a non-Affiliate, is to be considered an Obligation for the purpose of the transaction. It will be considered an obligation if allGuarantees is applicable (true) and not if allGuarantees is inapplicable (false). ISDA 2003 Term: All Guarantees.
	 */
	AllGuarantees bool
	/**
	 * Used to determine (a) for physically settled trades, the Physical Settlement Amount, which equals the Floating Rate Payer Calculation Amount times the Reference Price and (b) for cash settled trades, the Cash Settlement Amount, which equals the greater of (i) the difference between the Reference Price and the Final Price and (ii) zero. ISDA 2003 Term: Reference Price.
	 */
	ReferencePrice Price
	/**
	 * Applicable to the transactions on mortgage-backed security, which can make use of a reference policy. Presence of the element with value set to 'true' indicates that the reference policy is applicable; absence implies that it is not.
	 */
	ReferencePolicy bool
	/**
	 * With respect to any day, the list of Syndicated Secured Obligations of the Designated Priority of the Reference Entity published by Markit Group Limited or any successor thereto appointed by the Specified Dealers (the 'Secured List Publisher') on or most recently before such day, which list is currently available at [http://www.markit.com]. ISDA 2003 Term: Relevant Secured List.
	 */
	SecuredList bool
}

/**
 * A class to specify the reference obligation that is associated with a credit derivative instrument.
 */
type ReferenceObligation struct {
	/**
	 * Identifies the underlying asset when it is a security, such as a bond or convertible bond. The security data type requires one or more productIdentifiers, specificaiton of the security type (e.g. debt), and includes optional attributes to specify a debt class, such as asset-backed, as well as seniority.
	 */
	Security Security
	/**
	 * Identifies the underlying asset when it is a loan.
	 */
	Loan Loan
	/**
	 * The entity primarily responsible for repaying debt to a creditor as a result of borrowing or issuing bonds. ISDA 2003 Term: Primary Obligor.
	 */
	PrimaryObligor LegalEntity
	/**
	 * A pointer style reference to a reference entity defined elsewhere in the document. Used when the reference entity is the primary obligor.
	 */
	PrimaryObligorReference ReferenceWithMeta
	/**
	 * The party that guarantees by way of a contractual arrangement to pay the debts of an obligor if the obligor is unable to make the required payments itself. ISDA 2003 Term: Guarantor.
	 */
	Guarantor LegalEntity
	/**
	 * A pointer style reference to a reference entity defined elsewhere in the document. Used when the reference entity is the guarantor.
	 */
	GuarantorReference string
	/**
	 * Indicates if the reference obligation is a Standard Reference Obligation. ISDA 2014 Term: Standard Reference Obligation.
	 */
	StandardReferenceObligation bool
}

type ReferencePair struct {
	/**
	 * The corporate or sovereign entity on which you are buying or selling protection and any successor that assumes all or substantially all of its contractual and other obligations. It is vital to use the correct legal name of the entity and to be careful not to choose a subsidiary if you really want to trade protection on a parent company. Please note, Reference Entities cannot be senior or subordinated. It is the obligations of the Reference Entities that can be senior or subordinated. ISDA 2003 Term: Reference Entity.
	 */
	ReferenceEntity LegalEntity
	/**
	 * The Reference Obligation is a financial instrument that is either issued or guaranteed by the reference entity. It serves to clarify the precise reference entity protection is being offered upon, and its legal position with regard to other related firms (parents/subsidiaries). Furthermore the Reference Obligation is ALWAYS deliverable and establishes the Pari Passu ranking (as the deliverable bonds must rank equal to the reference obligation). ISDA 2003 Term: Reference Obligation.
	 */
	ReferenceObligation ReferenceObligation
	/**
	 * Used to indicate that there is no Reference Obligation associated with this Credit Default Swap and that there will never be one.
	 */
	NoReferenceObligation bool
	/**
	 * Defines the reference entity types corresponding to a list of types in the ISDA First to Default documentation.
	 */
	EntityType FieldWithMeta
}

/**
 * This type contains all the reference pool items to define the reference entity and reference obligation(s) in the basket.
 */
type ReferencePool struct {
	/**
	 * This type contains all the constituent weight and reference information.
	 */
	ReferencePoolItem []ReferencePoolItem
}

/**
 * This type contains all the constituent weight and reference information.
 */
type ReferencePoolItem struct {
	/**
	 * Describes the weight of each of the constituents within the basket. If not provided, it is assumed to be equal weighted.
	 */
	ConstituentWeight ConstituentWeight
	ReferencePair     ReferencePair
	/**
	 * Reference to the documentation terms applicable to this item.
	 */
	ProtectionTermsReference ReferenceWithMeta
	/**
	 * Reference to the cash settlement terms applicable to this item.
	 */
	CashSettlementTermsReference ReferenceWithMeta
	/**
	 * Reference to the physical settlement terms applicable to this item.
	 */
	PhysicalSettlementTermsReference ReferenceWithMeta
}

/**
 * A complex type used to specify the option and convertible bond option strike when expressed in reference to a swap curve.
 */
type ReferenceSwapCurve struct {
	SwapUnwindValue SwapCurveValuation
	/**
	 * Amount to be paid by the buyer of the option if the option is exercised prior to the Early Call Date. (The market practice in the convertible bond option space being that the buyer should be penalised if he/she exercises the option early on.)
	 */
	MakeWholeAmount MakeWholeAmount
}

/**
 * A class to specify one or more regimes that may be specified as relevant to a legal agreement.
 */
type Regime struct {
	/**
	 * A class to specify the regime(s) that parties to a legal agreement, such as the ISDA 2016 and 2018 CSA for Initial Margin, might agree to apply to one or both parties when acting as collateral taker, and specific terms associated with that application.
	 */
	ApplicableRegime []ApplicableRegime
	/**
	 * The specification of the number of days after effective delivery of notice that Mandatory method fallback applies. Specification is only required when one or more Regimes have Fall Back to Mandatory Method elected as a SIMM exception.
	 */
	FallbackToMandatoryMethodDays float64
}

/**
 * A class that is used by the ApplicableRegime and the AdditionalRegime classes to specify the regulatory regime terms.
 */
type RegimeTerms struct {
	/**
	 * The party for which the regime terms are being specified when acting as collateral taker.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The specification of whether the regime is elected as applicable to the party when acting as collateral taker.
	 */
	IsApplicable ExceptionEnum
	/**
	 * The bespoke party specific Regime term elections applicable when specified.
	 */
	AsSpecified string
	/**
	 * The election for SIMM exception to the regulatory regime clause.
	 */
	SimmException SimmException
	/**
	 * Specifies the retrospective effect exception to the regulatory regime clause of Initial Margin documents as either a normalized value specified as part of an enumeration or a customized value specified of type string.
	 */
	RetrospectiveEffect RetrospectiveEffect
}

/**
 * Represents a class to allow specification of different type of Regional government collateral.
 */
type RegionalGovernmentIssuerType struct {
	/**
	 * Applies to regional governments, local authorities or municipals.  True if entity has recourse to sovereign (e.g. debt guaranteed by government).  False if entity does not have recourse to sovereign.
	 */
	SovereignRecourse bool
}

type RelatedParty struct {
	/**
	 * Reference to a party.
	 */
	PartyReference ReferenceWithMeta
	/**
	 * Reference to an account.
	 */
	AccountReference ReferenceWithMeta
	/**
	 * The category of the relationship. The related party performs the role specified in this field for the base party. For example, if the role is ,Guarantor, the related party acts as a guarantor for the base party.
	 */
	Role PartyRoleEnum
}

/**
 * A class defining a date (referred to as the derived date) as a relative offset from another date (referred to as the anchor date). If the anchor date is itself an adjustable date then the offset is assumed to be calculated from the adjusted anchor date. A number of different scenarios can be supported, namely; 1) the derived date may simply be a number of calendar periods (days, weeks, months or years) preceding or following the anchor date; 2) the unadjusted derived date may be a number of calendar periods (days, weeks, months or years) preceding or following the anchor date with the resulting unadjusted derived date subject to adjustment in accordance with a specified business day convention, i.e. the derived date must fall on a good business day; 3) the derived date may be a number of business days preceding or following the anchor date. Note that the businessDayConvention specifies any required adjustment to the unadjusted derived date. A negative or positive value in the periodMultiplier indicates whether the unadjusted derived precedes or follows the anchor date. The businessDayConvention should contain a value NONE if the day type element contains a value of Business (since specifying a negative or positive business days offset would already guarantee that the derived date would fall on a good business day in the specified business centers).
 */
type RelativeDateOffset struct {
	/**
	 * A time period multiplier, e.g. 1, 2 or 3 etc. A negative value can be used when specifying an offset relative to another date, e.g. -2 days.
	 */
	PeriodMultiplier int
	/**
	 * A time period, e.g. a day, week, month or year of the stream. If the periodMultiplier value is 0 (zero) then period must contain the value D (day).
	 */
	Period PeriodEnum
	Meta   MetaFields
	/**
	 * In the case of an offset specified as a number of days, this element defines whether consideration is given as to whether a day is a good business day or not. If a day type of business days is specified then non-business days are ignored when calculating the offset. The financial business centers to use for determination of business days are implied by the context in which this element is used. This element must only be included when the offset is specified as a number of days. If the offset is zero days then the dayType element should not be included.
	 */
	DayType DayTypeEnum
	/**
	 * The convention for adjusting a date if it would otherwise fall on a day that is not a business day, as specified by an ISDA convention (e.g. Following, Precedent).
	 */
	BusinessDayConvention BusinessDayConventionEnum
	BusinessCenters       BusinessCenters
	/**
	 * A pointer style reference to a set of financial business centers defined elsewhere in the document. This set of business centers is used to determine whether a particular day is a business day or not.
	 */
	BusinessCentersReference ReferenceWithMeta
	/**
	 * Specifies the anchor as an href attribute. The href attribute value is a pointer style reference to the element or component elsewhere in the document where the anchor date is defined.
	 */
	DateRelativeTo ReferenceWithMeta
	/**
	 * The date once the adjustment has been performed. (Note that this date may change if the business center holidays change).
	 */
	AdjustedDate time.Time
}

/**
 * A class describing a set of dates defined as relative to another set of dates.
 */
type RelativeDates struct {
	/**
	 * A time period multiplier, e.g. 1, 2 or 3 etc. A negative value can be used when specifying an offset relative to another date, e.g. -2 days.
	 */
	PeriodMultiplier int
	/**
	 * A time period, e.g. a day, week, month or year of the stream. If the periodMultiplier value is 0 (zero) then period must contain the value D (day).
	 */
	Period PeriodEnum
	Meta   MetaFields
	/**
	 * In the case of an offset specified as a number of days, this element defines whether consideration is given as to whether a day is a good business day or not. If a day type of business days is specified then non-business days are ignored when calculating the offset. The financial business centers to use for determination of business days are implied by the context in which this element is used. This element must only be included when the offset is specified as a number of days. If the offset is zero days then the dayType element should not be included.
	 */
	DayType DayTypeEnum
	/**
	 * The convention for adjusting a date if it would otherwise fall on a day that is not a business day, as specified by an ISDA convention (e.g. Following, Precedent).
	 */
	BusinessDayConvention BusinessDayConventionEnum
	BusinessCenters       BusinessCenters
	/**
	 * A pointer style reference to a set of financial business centers defined elsewhere in the document. This set of business centers is used to determine whether a particular day is a business day or not.
	 */
	BusinessCentersReference ReferenceWithMeta
	/**
	 * Specifies the anchor as an href attribute. The href attribute value is a pointer style reference to the element or component elsewhere in the document where the anchor date is defined.
	 */
	DateRelativeTo ReferenceWithMeta
	/**
	 * The date once the adjustment has been performed. (Note that this date may change if the business center holidays change).
	 */
	AdjustedDate time.Time
	/**
	 * The number of periods in the referenced date schedule that are between each date in the relative date schedule. Thus a skip of 2 would mean that dates are relative to every second date in the referenced schedule. If present this should have a value greater than 1.
	 */
	PeriodSkip int
	/**
	 * The first and last dates of a schedule. This can be used to restrict the range of values in a reference series of dates.
	 */
	ScheduleBounds DateRange
}

type Representations struct {
	/**
	 * If true, then non reliance is applicable.
	 */
	NonReliance bool
	/**
	 * If true, then agreements regarding hedging are applicable.
	 */
	AgreementsRegardingHedging bool
	/**
	 * If present and true, then index disclaimer is applicable.
	 */
	IndexDisclaimer bool
	/**
	 * If true, then additional acknowledgements are applicable.
	 */
	AdditionalAcknowledgements bool
	AdditionalBespokeTerms     []Clause
}

/**
 * Defines the reset value or fixing value produced in cashflow calculations, during the life-cycle of a financial instrument. The reset process defined in Create_Reset function joins product definition details with observations to compute the reset value.
 */
type Reset struct {
	/**
	 * Specifies the reset or fixing value. The fixing value could be a cash price, interest rate, or other value.
	 */
	ResetValue Price
	/**
	 * Specifies the date on which the reset occurred.
	 */
	ResetDate time.Time
	/**
	 * Specifies the 'Rate Record Day' for a Fallback rate.  Fallback rate fixing processes typically set the fixing rate in arrears, i.e., the Fallback Rate corresponding to a Rate Record Date is set at the end of the interest accural period.  When this applies, Reset->resetDate occurs at the end of the interest period, and the Reset->rateRecordDate occurs near the start of the interest period.  The Reset->rateRecordDate and Reset->observations->observationIdentifier->observationDate will differ if a Fallback rate is unavailable on the Rate Record Date, and the latest previous available rate is used as the observation.
	 */
	RateRecordDate time.Time
	/**
	 * Represents an audit of the observations used to produce the reset value. If multiple observations were necessary to produce the reset value, the aggregation method should be defined on the payout.
	 */
	Observations []ReferenceWithMeta
	/**
	 * Identifies the aggregation method to use in the case where multiple observations are used to compute the reset value and the method is not defined in a payout.
	 */
	AveragingMethodology AveragingCalculation
	Meta                 MetaFields
}

/**
 * A data defining:  the parameters used to generate the reset dates schedule and associated fixing dates. The reset dates are the dates on which the new index value (which is observed on the fixing date) is applied for each period and on which the interest rate hence begins to accrue.
 */
type ResetDates struct {
	/**
	 * A pointer style reference to the associated calculation period dates component defined elsewhere in the document.
	 */
	CalculationPeriodDatesReference ReferenceWithMeta
	/**
	 * Specifies whether the reset dates are determined with respect to each adjusted calculation period start date or adjusted calculation period end date. If the reset frequency is specified as daily this element must not be included.
	 */
	ResetRelativeTo ResetRelativeToEnum
	/**
	 * The initial fixing date.
	 */
	InitialFixingDate InitialFixingDate
	/**
	 * The fixing dates are the dates on which the index values are observed. The fixing dates are specified by reference to the reset date through business days offset and an associated set of financial business centers. Normally these offset calculation rules will be those specified in the ISDA definition for the relevant floating rate index (ISDA's Floating Rate Option). However, non-standard offset calculation rules may apply for a trade if mutually agreed by the principal parties to the transaction.
	 */
	FixingDates RelativeDateOffset
	/**
	 * This attribute is not part of the FpML ResetDate, and has been added as part of the CDM to support the credit derivatives final fixing date.
	 */
	FinalFixingDate AdjustableDate
	/**
	 * Specifies the number of business days before the period end date when the rate cut-off date is assumed to apply. The financial business centers associated with determining the rate cut-off date are those specified in the reset dates adjustments. The rate cut-off number of days must be a negative integer (a value of zero would imply no rate cut off applies in which case the rateCutOffDaysOffset element should not be included). The relevant rate for each reset date in the period from, and including, a rate cut-off date to, but excluding, the next applicable period end date (or, in the case of the last calculation period, the termination date) will (solely for purposes of calculating the floating amount payable on the next applicable payment date) be deemed to be the relevant rate in effect on that rate cut-off date. For example, if rate cut-off days for a daily averaging deal is -2 business days, then the refix rate applied on (period end date - 2 days) will also be applied as the reset on (period end date - 1 day), i.e. the actual number of reset dates remains the same but from the rate cut-off date until the period end date, the same refix rate is applied. Note that in the case of several calculation periods contributing to a single payment, the rate cut-off is assumed only to apply to the final calculation period contributing to that payment. The day type associated with the offset must imply a business days offset.
	 */
	RateCutOffDaysOffset Offset
	/**
	 * The frequency at which the reset dates occur. In the case of a weekly reset frequency, also specifies the day of the week that the reset occurs. If the reset frequency is greater than the calculation period frequency then this implies that more than one reset is established for each calculation period and some form of rate averaging is applicable.
	 */
	ResetFrequency ResetFrequency
	/**
	 * The definition of the business day convention and financial business centers used for adjusting the reset date if it would otherwise fall on a day that is not a business day in the specified business center.
	 */
	ResetDatesAdjustments BusinessDayAdjustments
	Meta                  MetaFields
}

/**
 * A class defining the reset frequency. In the case of a weekly reset, also specifies the day of the week that the reset occurs. If the reset frequency is greater than the calculation period frequency the this implies that more or more reset dates is established for each calculation period and some form of rate averaging is applicable. The specific averaging method of calculation is specified in FloatingRateCalculation. In case the reset frequency is of value T (term), the period is defined by the swap/swapStream/calculationPerioDates/effectiveDate and the swap/swapStream/calculationPerioDates/terminationDate.
 */
type ResetFrequency struct {
	/**
	 * A time period multiplier, e.g. 1, 2, or 3. If the period value is T (Term) then period multiplier must contain the value 1.
	 */
	PeriodMultiplier int
	/**
	 * A time period, e.g. a day, week, month, year or term of the stream.
	 */
	Period PeriodExtendedEnum
	Meta   MetaFields
	/**
	 * The day of the week on which a weekly reset date occurs. This element must be included if the reset frequency is defined as weekly and not otherwise.
	 */
	WeeklyRollConvention WeeklyRollConventionEnum
}

/**
 * Defines the information needed to create a Reset Business Event.
 */
type ResetInstruction struct {
	Payout []ReferenceWithMeta
	/**
	 * Specifies the 'Rate Record Day' for a Fallback rate.  Fallback rate fixing processes typically set the fixing rate in arrears, i.e., the Fallback Rate corresponding to a Rate Record Date is set at the end of the interest accural period.  When this applies, Reset->resetDate occurs at the end of the interest period, and the Reset->rateRecordDate occurs near the start of the interest period.  The Reset->rateRecordDate and Reset->observations->observationIdentifier->observationDate will differ if a Fallback rate is unavailable on the Rate Record Date, and the latest previous available rate is used as the observation.
	 */
	RateRecordDate time.Time
	/**
	 * Specifies the date on which the reset is occuring.
	 */
	ResetDate time.Time
}

/**
 * Methodology for resolving disputes in respect of the determination of the value of collateral.
 */
type ResolutionValue struct {
	/**
	 * Details of how cash collateral is valued when resolving disputes.
	 */
	Cash ValueCashEnum
	/**
	 * Details of how securities collateral is valued when resolving disputes.
	 */
	Securities ValueSecuritiesEnum
	/**
	 * Represents whether a fallback calculation method is applicable. 'True' if fallback calculation method is applicable.
	 */
	Fallback bool
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * Generic class to specify the quantity for different payout legs in a contractual product, when that quantity can vary across payout legs or across time. A resolvable quantity can always be resolved into a single quantity from the quantity notation which has a corresponding asset identifier. In addition to the base case, where quantity is directly specified as a number as part of the quantity notation, the other use cases are: (i) quantity based on some pre-defined schedule (eg amortising notional), (ii) quantity based on some pre-defined events (eg resetting cross-currency notional), or quantity set as reference to another quantity (eg equity notional as no. securities x price).
 */
type ResolvablePriceQuantity struct {
	/**
	 * A product's quantity as a single, non-negative amount.  When specified as part of a product definition, this quantity attribute would not be set.  Instead it is specified on the quantity notation along with an asset identifier matching this payout's asset identifier.  This allows the quantity to be resolved for a payout leg, which can then be specified here for convenience during data processing.  There needs to be at least one resolvable quantity across payout legs of a product to define an anchor that other payout quantities can refer to.  This attribute is ignored when mapping existing FpML messages.
	 */
	ResolvedQuantity Quantity
	/**
	 * A payout's quantity specified as a schedule, which may also contain a single value if that quantity is constant. There can only be a single quantity schedule applicable to a payout: e.g. the notional for an interest rate leg. The quantity must be specified outside of the payout in a PriceQuantity object and only referenced inside the payout using an address.
	 */
	QuantitySchedule ReferenceWithMeta
	/**
	 * Reference quantity when resolvable quantity is defined as relative to another (resolvable) quantity. A resolvable quantity needs to contain either an absolute quantity or a reference to another (resolvable) quantity. This requirement is captured by a choice rule on the class.
	 */
	QuantityReference ReferenceWithMeta
	/**
	 * Quantity multiplier is specified on top of a reference quantity and is used as a multiplying factor when resolving the quantity. A quantity multiplier can only exist when the resolvable quantity specifies a reference quantity.
	 */
	QuantityMultiplier QuantityMultiplier
	/**
	 * Whether the quantity is resettable
	 */
	Reset bool
	/**
	 * The future value notional is specific to BRL CDI swaps, and is specified alongside the notional amount. The value is calculated as follows: Future Value Notional = Notional Amount * (1 + Fixed Rate) ^ (Fixed Rate Day Count Fraction). The currency should always match that expressed in the notional schedule. The value date should match the adjusted termination date.
	 */
	FutureValueNotional FutureValueAmount
	/**
	 * A payout's price specified as a schedule, which may also contain a single value if that price is constant. There may be multiple prices specified for a single payout: e.g. a floating interest rate leg may specify a spread, a cap and/or floor and a multiplier. The price must be specified outside of the payout in a PriceQuantity object and only referenced inside the payout using an address.
	 */
	PriceSchedule []ReferenceWithMeta
	Meta          MetaFields
}

/**
 * Describes the resource that contains the media representation of a business event (i.e used for stating the Publicly Available Information). For example, can describe a file or a URL that represents the event. This type is an extended version of a type defined by RIXML (www.rixml.org).  Rosetta restricts the FpML implementation by not providing the ability to associated a document in hexadecimalBinary or base64Binary until such time that actual use cases will come up.
 */
type Resource struct {
	/**
	 * The unique identifier of the resource within the event. FpML specifies this element of type resourceIdScheme but with no specified value.
	 */
	ResourceId FieldWithMeta
	/**
	 * A description of the type of the resource, e.g. a confirmation.
	 */
	ResourceType FieldWithMeta
	/**
	 * Indicates the language of the resource, described using the ISO 639-2/T Code.
	 */
	Language FieldWithMeta
	/**
	 * Indicates the size of the resource in bytes. It could be used by the end user to estimate the download time and storage needs.
	 */
	SizeInBytes float64
	/**
	 * Indicates the length of the resource. For example, if the resource were a PDF file, the length would be in pages.
	 */
	Length ResourceLength
	/**
	 * Indicates the type of media used to store the content. mimeType is used to determine the software product(s) that can read the content. MIME Types are described in RFC 2046.
	 */
	MimeType FieldWithMeta
	/**
	 * The name of the resource.  It is specified as a NormalizedString in FpML.
	 */
	Name string
	/**
	 * Any additional comments that are deemed necessary. For example, which software version is required to open the document? Or, how does this resource relate to the others for this event?
	 */
	Comments string
	/**
	 * Provides extra information as string. In case the extra information is in XML format, a CDATA section must be placed around the source message to prevent its interpretation as XML content.
	 */
	String string
	/**
	 * Indicates where the resource can be found, as a URL that references the information on a web server accessible to the message recipient.
	 */
	Url string
}

/**
 * A class to indicate the length of the resource.
 */
type ResourceLength struct {
	/**
	 * The length unit of the resource. For example, pages (pdf, text documents) or time (audio, video files).
	 */
	LengthUnit LengthUnitEnum
	/**
	 * The length value of the resource.
	 */
	LengthValue float64
}

type Restructuring struct {
	/**
	 * Indicates whether the restructuring provision is applicable.
	 */
	Applicable bool
	/**
	 * Specifies the type of restructuring that is applicable.
	 */
	RestructuringType FieldWithMeta
	/**
	 * In relation to a restructuring credit event, unless multiple holder obligation is not specified restructurings are limited to multiple holder obligations. A multiple holder obligation means an obligation that is held by more than three holders that are not affiliates of each other and where at least two thirds of the holders must agree to the event that constitutes the restructuring credit event. ISDA 2003 Term: Multiple Holder Obligation.
	 */
	MultipleHolderObligation bool
	/**
	 * Presence of this element and value set to 'true' indicates that Section 3.9 of the 2003 Credit Derivatives Definitions shall apply. Absence of this element indicates that Section 3.9 shall not apply. NOTE: Not allowed under ISDA Credit 1999.
	 */
	MultipleCreditEventNotices bool
}

/**
 * A class to specify the retrospective effect exception to the regulatory regime clause of Initial Margin documents as either a normalized value specified as part of an enumeration or a customized value specified of type string.
 */
type RetrospectiveEffect struct {
	/**
	 * The Standard Initial Margin Model exception when specified by the party according to one of the enumerated values.
	 */
	StandardisedException ExceptionEnum
	/**
	 * The Standard Initial Margin Model exception when specified as a customized approach by the party.
	 */
	AsSpecified string
}

/**
 * A class to specify the application of Interest Amount with respect the Return Amount.
 */
type ReturnAmount struct {
	/**
	 * Default language is included when True, and excluded when False.
	 */
	IncludesDefaultLanguage bool
	/**
	 * Custom election that might be specified by the parties to the agreement.
	 */
	CustomElection string
}

/**
 * Specifies the information required to create the return of a Security Finance Transaction.
 */
type ReturnInstruction struct {
	/**
	 * Specifies the quantity of shares and cash to be returned in a partial return event.
	 */
	Quantity []Quantity
}

/**
 * Specifies the type of return of a performance payout.
 */
type ReturnTerms struct {
	/**
	 * Return terms based upon the underlier's observed price.
	 */
	PriceReturnTerms PriceReturnTerms
	/**
	 * Return terms based upon dividend payments associated to the underlier.
	 */
	DividendReturnTerms DividendReturnTerms
	/**
	 * Return terms based upon the observed variance of the underlier's price.
	 */
	VarianceReturnTerms VarianceReturnTerms
	/**
	 * Return terms based upon the observed volatility of the underlier's price.
	 */
	VolatilityReturnTerms VolatilityReturnTerms
	/**
	 * Return terms based upon the observed correlation between the components of the underlying basket.
	 */
	CorrelationReturnTerms CorrelationReturnTerms
}

/**
 * Contains all common elements in variance, volatility and correlation return Terms.
 */
type ReturnTermsBase struct {
	/**
	 * Contains all non-date valuation information.
	 */
	ValuationTerms ValuationTerms
	/**
	 * This specifies the numerator of an annualization factor. Frequently this number is equal to the number of observations of prices in a year e.g. 252.
	 */
	AnnualizationFactor int
	/**
	 * The parameters which define whether dividends are applicable
	 */
	DividendApplicability DividendApplicability
	/**
	 * Contains Equity Underlyer provisions regarding jurisdiction and fallbacks.
	 */
	EquityUnderlierProvisions EquityUnderlierProvisions
	/**
	 * Indicates whether the price of shares is adjusted for dividends or not.
	 */
	SharePriceDividendAdjustment bool
	/**
	 * Expected number of trading days.
	 */
	ExpectedN int
	/**
	 * Contract will strike off this initial level. Providing just the initialLevel without initialLevelSource, infers that this is AgreedInitialPrice - a specified Initial Index Level.
	 */
	InitialLevel float64
	/**
	 * In this context, this is AgreedInitialPrice - a specified Initial Index Level.
	 */
	InitialLevelSource DeterminationMethodEnum
	/**
	 * Specifies whether Mean Adjustment is applicable or not in the calculation of the Realized Volatility, Variance or Correlation
	 */
	MeanAdjustment bool
	/**
	 * Performance calculation, in accordance with Part 1 Section 12 of the 2018 ISDA CDM Equity Confirmation for Security Equity Swap, Para 75. 'Equity Performance'. Cumulative performance is used as a notional multiplier factor on both legs of an Equity Swap.
	 */
	Performance string
}

/**
 * A class to specify the rights of Security Taker and/or Security Provider when an Early Termination or Access Condition event has occurred.
 */
type RightsEvents struct {
	/**
	 * The bespoke provisions that might be specified by the parties to the agreement applicable to a Security Taker Rights Event.
	 */
	SecurityTakerRightsEvent SecuredPartyRightsEvent
	/**
	 * The bespoke provisions that might be specified by the parties to the agreement applicable to a Notice of Exclusive Control Event.
	 */
	ControlAgreementNecEvent ControlAgreementNecEvent
	/**
	 * The bespoke provisions that might be specified by the parties to the agreement applicable to a Security Provider Rights Event.
	 */
	SecurityProviderRightsEvent SecurityProviderRightsEvent
	/**
	 * The specification of whether Delivery In Lieu language is applicable to the agreement (true) or not (false).
	 */
	DeliveryInLieuRight bool
	/**
	 * The Additional Rights Event election.
	 */
	AdditionalRightsEvent AdditionalRightsEvent
}

/**
 * Used in conjunction with an exchange-based pricing source. Identifies a way in which the futures contracts referenced will roll between periods.
 */
type RollFeature struct {
	/**
	 * Used in conjunction with an exchange-based pricing source. Identifies a date source calendar from which the pricing dates and thus roll to the next contract will be based off (e.g. pricing is based on the NYMEX WTI First Nearby Futures Contract, if Future is chosen, the pricing will roll to the next futures contract on expiration, if ListedOption is chosen, the pricing will roll to the next futures contract on the Option expiration date which is three business days before the expiration of the NYMEX WTI futures contract.) Omitting this element will result in the default behavior expected with the pricing source described within the commodity element.
	 */
	RollSourceCalendar RollSourceCalendarEnum
	/**
	 * Specifies, for a Commodity Transaction that references a delivery date for a listed future, the day on which the specified future will roll to the next nearby month prior to the expiration of the referenced future. If the future will not roll at all - i.e. the price will be taken from the expiring contract, 0 should be specified here. If the future will roll to the next nearby on the last trading day - i.e. the price will be taken from the next nearby on the last trading day, then 1 should be specified and so on.
	 */
	DeliveryDateRollConvention Offset
}

/**
 * Defines rounding rules and precision to be used in the rounding of a number.
 */
type Rounding struct {
	/**
	 * Specifies the rounding rounding rule as up, down, or nearest.
	 */
	RoundingDirection RoundingDirectionEnum
	/**
	 * Specifies the rounding precision in terms of a number of decimal places when the number is evaluated in decimal form (not percentage), e.g. 0.09876543 rounded to the nearest 5 decimal places is  0.0987654.
	 */
	Precision int
}

/**
 * A class defining a schedule of rates or amounts in terms of an initial value and then a series of step date and value pairs. On each step date the rate or amount changes to the new step value. The series of step date and value pairs are optional. If not specified, this implies that the initial value remains unchanged over time.
 */
type Schedule struct {
	/**
	 * The initial rate or amount, as the case may be. An initial rate of 5% would be represented as 0.05.
	 */
	Value float64
	/**
	 * The schedule of step date and value pairs. On each step date the associated step value becomes effective. A list of steps may be ordered in the document by ascending step date. An FpML document containing an unordered list of steps is still regarded as a conformant document.
	 */
	DatedValue []DatedValue
}

/**
 * A class that defines the period of a schedule. The period contains a set of start and end dates, quantities, fixing, and pricing data.
 */
type SchedulePeriod struct {
	/**
	 * Period for which the payment is generated.
	 */
	CalculationPeriod DateRange
	/**
	 * Adjusted payment date.
	 */
	PaymentDate time.Time
	/**
	 * Period over which the underlying price is observed.
	 */
	FixingPeriod DateRange
	/**
	 * Period and time profile over which the delivery takes place.
	 */
	DeliveryPeriod CalculationScheduleDeliveryPeriods
}

type ScheduledTransfer struct {
	/**
	 * Specifies a transfer created from a scheduled or contingent event on a contract, e.g. Exercise, Performance, Credit Event
	 */
	TransferType                ScheduledTransferEnum
	CorporateActionTransferType CorporateActionTypeEnum
}

type SchmeNm struct {
	Prtry string
}

type SecuredDebt struct {
	/**
	 * Specifies the type of secured debt product.
	 */
	SecuredType SecuredTypeEnum
	/**
	 * Specifies the type of Asset Backed Security.
	 */
	AssetBacked AssetBackedEnum
	/**
	 * Specifies the type of Collateralized Obligation.
	 */
	CollateralizedObligation CollateralizedObligationEnum
	/**
	 * Specifies the type of Covered Bond.
	 */
	CoveredBond CoveredBondEnum
	/**
	 * Specifies the type of property when the security is linked to a property asset.
	 */
	PropertyType PropertyTypeEnum
}

/**
 * A class to specify Secured Party Rights Event language.
 */
type SecuredPartyRightsEvent struct {
	SecuredPartyRightsEventElection []SecuredPartyRightsEventElection
	/**
	 * A boolean attribute to specify whether Failure to Pay Early Termination language is included (True) or excluded (False) from the agreement.
	 */
	EarlyTerminationDateOptionalLanguage bool
	/**
	 * A boolean attribute to specify whether Failure to Pay Early Termination language in the agreement is deemed applicable or not.
	 */
	FailureToPayEarlyTermination bool
}

/**
 * A class to specify party specific Secured Party Rights Event language.
 */
type SecuredPartyRightsEventElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * A boolean attribute to specify whether a Secured Party Rights Event will only occur upon the occurrence of one or more of the event specified in a Control Agreement.
	 */
	RightsEvent bool
}

/**
 * Identifies a security by referencing an identifier and by specifying the sector.
 */
type Security struct {
	/**
	 * Asset Identifiers are used to uniquely identify an Asset, using a specified Asset Identifier Type.
	 */
	Identifier []AssetIdentifier
	/**
	 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object.
	 */
	Taxonomy []Taxonomy
	/**
	 * Defines whether the Asset is listed on a public exchange.
	 */
	IsExchangeListed bool
	/**
	 * If the Asset is listed, defines the public exchange of the listing.
	 */
	Exchange LegalEntity
	/**
	 * Provides the related Exchanges, if applicable.
	 */
	RelatedExchange []LegalEntity
	/**
	 * Identifies the type of an instrument using an enumerated list.
	 */
	InstrumentType InstrumentTypeEnum
	/**
	 * Identifies the type of debt and selected debt economics.
	 */
	DebtType DebtType
	/**
	 * Identifies the type of equity.
	 */
	EquityType EquityType
	/**
	 * Identifies the type of fund.
	 */
	FundType FundProductTypeEnum
}

/**
 * The set of elections which specify a Security Agremeent.
 */
type SecurityAgreementElections struct {
	/**
	 * The pledged account associated with the agreement.
	 */
	PledgedAccount Account
	/**
	 * Enforcement Events specific to the agreement.
	 */
	EnforcementEvent EnforcementEvent
	/**
	 * Delivery In Lieu rights.
	 */
	DeliveryInLieuRight bool
	/**
	 * Full Discharge condition.
	 */
	FullDischarge bool
	/**
	 * The election for the Valuation of Appropriate Collateral.
	 */
	AppropriatedCollateralValuation AppropriatedCollateralValuation
	/**
	 * The Process Agent that might be appointed by the parties to the agreement.
	 */
	ProcessAgent ProcessAgent
	/**
	 * The jurisdiction specific terms.
	 */
	JurisdictionRelatedTerms JurisdictionRelatedTerms
	/**
	 * Any additional amendments that might be specified by the parties to the agreement.
	 */
	AdditionalAmendments string
	/**
	 * Any additional terms that might be specified applicable.
	 */
	AdditionalBespokeTerms string
	/**
	 * The location and language of execution to determine duty to be paid.
	 */
	ExecutionTerms ExecutionTerms
}

/**
 * Security Interest for Obligations clause applicable to 1994 NY CSA, 1995 English Law CSA, 1995 English Law Credit Support Deed, 2016 English Law VM CSA and 2016 New York Law VM CSA.
 */
type SecurityInterestForObligations struct {
	/**
	 * The elective party to which the Security Interest for Obligations is applicable.
	 */
	Party []CounterpartyRoleEnum
	/**
	 * A description of any additional obligations secured by the credit support arrangements.
	 */
	Obligations SecurityInterestObligationsEnum
	/**
	 * The party to whom obligations under the Master Agreement are owed and secured by the credit support arrangements.
	 */
	Obligee SecurityInterestObligeeEnum
	/**
	 * Exception value.
	 */
	Other string
}

/**
 * Specifies the information required for inclusion in a securities lending billing invoice.
 */
type SecurityLendingInvoice struct {
	/**
	 * The party issuing the invoice
	 */
	SendingParty Party
	/**
	 * The party receiving the invoice
	 */
	ReceivingParty Party
	/**
	 * The starting date of the period described by this invoice
	 */
	BillingStartDate time.Time
	/**
	 * The ending date of the period described by this invoice
	 */
	BillingEndDate time.Time
	/**
	 * The billing records contained within the invoice
	 */
	BillingRecord []BillingRecord
	/**
	 * The billing summaries contained within the invoice
	 */
	BillingSummary []BillingSummary
	Meta           MetaFields
}

/**
 * A locate is an approval from a broker that needs to be obtained prior to effecting a short sale in an equity security. Similar to security availability, a borrower can request a single or multiple securities, but at least one must be requested.
 */
type SecurityLocate struct {
	/**
	 * Defines the purpose of this inventory.
	 */
	AvailableInventoryType AvailableInventoryTypeEnum
	/**
	 * Allows details related to the availability messaging use case to be defined
	 */
	MessageInformation MessageInformation
	/**
	 * Defines all parties involved for the list of inventory records in this set of inventory. For example, when used to describe securities lending availability, this could hold the sender of the availability, the intended recipient, the beneficial owner(s), the lender (which may differ from the sender as the lender may have the same piece of availability going through multiple agents), an agent or a venue.
	 */
	Party []Party
	/**
	 * Defines the role(s) that party(ies) may have in relation to the inventory.
	 */
	PartyRole []PartyRole
	/**
	 * An array holding the list of inventory being described. Each element in the inventoryRecord array represents an individual piece of inventory i.e. a security.
	 */
	AvailableInventoryRecord []AvailableInventoryRecord
}

/**
 * A class to specify the Pledgor/Obligor/Chargor Rights Event election.
 */
type SecurityProviderRightsEvent struct {
	PartyElection []SecurityProviderRightsEventElection
	/**
	 * The Pledgor/Obligor/Chargor Rights Event election includes cooling off language when the attribute is set of True.
	 */
	IncludeCoolingOffLanguage bool
	/**
	 * If specified as applicable here, a Pledgor/Obligor/Chargor Rights Event will not occur unless the Pledgor/Obligor/Chargor (A) has provided a statement to the Secured Party in respect of such Early Termination Date.
	 */
	FullDischarge bool
	/**
	 * The Automatic Set-Off provision applies when the value is set to True.
	 */
	AutomaticSetOff bool
	/**
	 * A custom Pledgor/Obligor/Chargor Rights Event election might be specified by the parties.
	 */
	CustomElection string
}

/**
 * A class to specify party specific Secured Party Rights Event language.
 */
type SecurityProviderRightsEventElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * If True, the Rights Event language applies for the specified Party.
	 */
	RightsEvent bool
}

type Sellr struct {
	AcctOwnr AcctOwnr
}

/**
 * A class to specificy methodologies to compute sensitivities specific to the agreement.
 */
type SensitivityMethodologies struct {
	/**
	 * Party specific elections for Sensitivity Methodology for Equity and Commodities.
	 */
	PartyElection []SensitivityMethodologiesPartyElection
}

/**
 * A class to specificy methodologies to compute sensitivities specific to the agreement.
 */
type SensitivityMethodologiesPartyElection struct {
	/**
	 * The party to which the election applies.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The methodology to compute sensitivities to equity indices, funds and ETFs for the purpose of Initial Margin agreements.
	 */
	SensitivityToEquity SensitivityToEquity
	/**
	 * The methodology to compute sensitivities to commodity indices for the purpose of Initial Margin agreements.
	 */
	SensitivityToCommodity SensitivityMethodology
}

/**
 * A class to specify the methodology according to which sensitivities are computed.
 */
type SensitivityMethodology struct {
	/**
	 * The methodology according to which sensitivities will be computed, when specified through a normalized election.
	 */
	SpecifiedMethodology SensitivitiesEnum
	/**
	 * The methodology according to which sensitivities will be computed, when specified through a custom election.
	 */
	CustomMethodology string
}

/**
 * Represents risk sensitivity calculated with respect to the value of an individual equity instrument.
 */
type SensitivityToEquity struct {
	/**
	 * Represents risk sensitivity calculated with respect to the level of an equity or market index.
	 */
	SensitivityToIndices SensitivityMethodology
	/**
	 * Represents risk sensitivity calculated with respect to the value of a fund, such as a mutual fund or collective investment vehicle.
	 */
	SensitivityToFunds SensitivityMethodology
	/**
	 * Represents risk sensitivity calculated with respect to the value of an exchange-traded fund (ETF).
	 */
	SensitivityToETFs SensitivityMethodology
}

/**
 * A class to specify the Relevant Settled Entity Matrix.
 */
type SettledEntityMatrix struct {
	/**
	 * Relevant settled entity matrix source.
	 */
	MatrixSource FieldWithMeta
	/**
	 * Specifies the publication date of the applicable version of the matrix. When this element is omitted, the Standard Terms Supplement defines rules for which version of the matrix is applicable.
	 */
	PublicationDate time.Time
}

/**
 * A base class to be extended by the SettlementTerms class.
 */
type SettlementBase struct {
	/**
	 * Whether the settlement will be cash, physical, by election, ...
	 */
	SettlementType SettlementTypeEnum
	/**
	 * The qualification as to how the transfer will settle, e.g. a DvP settlement.
	 */
	TransferSettlementType TransferSettlementEnum
	/**
	 * The settlement currency is to be specified when the Settlement Amount cannot be known in advance. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	SettlementCurrency FieldWithMeta
	/**
	 * The date on which the settlement amount will be paid, subject to adjustment in accordance with any applicable business day convention. This component would not be present for a mandatory early termination provision where the cash settlement payment date is the mandatory early termination date.
	 */
	SettlementDate SettlementDate
	/**
	 * Optional settlement centre as an enumerated list: Euroclear, Clearstream.
	 */
	SettlementCentre SettlementCentreEnum
	/**
	 * Optionally defines the parameters that regulate a settlement.
	 */
	SettlementProvision SettlementProvision
	/**
	 * Settlement Style.
	 */
	StandardSettlementStyle StandardSettlementStyleEnum
	Meta                    MetaFields
}

/**
 * A data defining the settlement date(s) for cash or physical settlement as either a set of explicit dates, together with applicable adjustments, or as a date relative to some other (anchor) date, or as any date in a range of contiguous business days. This data type provides a level of abstraction on top of the different legacy methods used to specify a settlement / payment date, which vary across product types, asset classes and delivery types.
 */
type SettlementDate struct {
	/**
	 * A single settlement date subject to adjustment or specified as relative to another date (e.g. the trade date). This attribute was formerly part of 'SettlementTerms', which is now being harmonised to include a common 'SettlementDate', as inherited from 'SettlementBase'.
	 */
	AdjustableOrRelativeDate AdjustableOrAdjustedOrRelativeDate
	/**
	 * The settlement date for a forward settling product. For Foreign Exchange contracts, this represents a common settlement date between both currency legs. To specify different settlement dates for each currency leg, see the ForeignExchange class. This attribute was formerly part of 'SettlementTerms', which is now being harmonised to include a common 'SettlementDate', as inherited from 'SettlementBase'.
	 */
	ValueDate time.Time
	/**
	 * A series of dates that shall be subject to adjustment if they would otherwise fall on a day that is not a business day in the specified business centers, together with the convention for adjusting the date. This attributes was formerly part of 'CashSettlementPaymentDate' as included into 'OptionCashSettlement' (which is now merged into a unique 'CashSettlementTerms' data type.
	 */
	AdjustableDates AdjustableDates
	/**
	 * A range of contiguous business days. This attribute is meant to be merged with the 'settlementDate' at some future point once we refactor 'Date' to use a single complex type across the model. This attributes was formerly part of 'CashSettlementPaymentDate', as included into 'OptionCashSettlement' (which is now merged into a unique 'CashSettlementTerms' data type.
	 */
	BusinessDateRange BusinessDateRange
	/**
	 * The number of business days used in the determination of the cash settlement payment date. If a cash settlement amount is specified, the cash settlement payment date will be this number of business days following the calculation of the final price. If a cash settlement amount is not specified, the cash settlement payment date will be this number of business days after all conditions to settlement are satisfied. ISDA 2003 Term: Cash Settlement Date. This attribute was formerly part of 'CashSettlementTerms' as used for credit event settlement, which now includes a common 'SettlementDate' attribute.
	 */
	CashSettlementBusinessDays int
	/**
	 * Applicable to CDS on MBS to specify whether payment delays are applicable to the fixed Amount. RMBS typically have a payment delay of 5 days between the coupon date of the reference obligation and the payment date of the synthetic swap. CMBS do not, on the other hand, with both payment dates being on the 25th of each month.
	 */
	PaymentDelay bool
	Meta         MetaFields
}

/**
 * Represents a forward settling payout. The underlier attribute captures the underlying payout, which is settled according to the settlementTerms attribute (which is part of PayoutBase). Both FX Spot and FX Forward should use this component.
 */
type SettlementPayout struct {
	/**
	 * Canonical representation of the payer and receiver parties applicable to each payout leg.
	 */
	PayerReceiver PayerReceiver
	/**
	 * Each payout leg must implement the quantity concept as a 'resolvable' type, which allows for different payout legs to be linked to each other (e.g. in the case of cross-curreny products).
	 */
	PriceQuantity ResolvablePriceQuantity
	/**
	 * The specification of the principal exchange. Optional as only applicable in the case of cross-currency or zero-coupon swaps with a final payment.
	 */
	PrincipalPayment PrincipalPayments
	/**
	 * Each payout leg must specifies its settlement terms, including the delivery type (i.e. cash vs physical, and their respective terms), the transfer type (DvP etc.) and settlement date, if any.
	 */
	SettlementTerms SettlementTerms
	/**
	 * The underlying financial product that will be physically or cash settled, which can be of any type, eg an asset such as cash or a security, or the cash settlement of an index rate.
	 */
	Underlier Underlier
	/**
	 * Also called contract month or delivery month. However, it's not always a month. It is usually expressed using a code, e.g. Z23 would be the Dec 2023 contract, (Z = December). For crude oil, the corresponding contract might be called CLZ23.
	 */
	DeliveryTerm string
	/**
	 * Contains the information relative to the delivery of the asset.
	 */
	Delivery AssetDeliveryInformation
	/**
	 * Allows the full representation of a payout by defining a set of schedule periods. It supports standard schedule customization by expressing all the dates, quantities, and pricing data in a non-parametric way.
	 */
	Schedule CalculationSchedule
}

/**
 * Defines parameters that regulate a settlement, for instance whether this settlement should be netted with other ones or broken-down into smaller amounts.
 */
type SettlementProvision struct {
	/**
	 * Defines the parameters that are necessary to 'shape' a settlement, i.e. break it down into smaller amounts.
	 */
	ShapingProvisions ShapingProvision
}

/**
 * Defines the settlement rate option to use for fixing in case of cash settlement. Currently only applicable to foreign exchange fixing in case of cross-currency settlement.
 */
type SettlementRateOption struct {
	/**
	 * The rate source for the conversion to the settlement currency. This source is specified through a scheme that reflects the terms of the Annex A to the 1998 FX and Currency Option Definitions.
	 */
	SettlementRateOption FieldWithMeta
	/**
	 * An attribute defining the parameters to get a new quote when a settlement rate option is disrupted.
	 */
	PriceSourceDisruption PriceSourceDisruption
}

/**
 * Specifies the settlement terms, which can either be cash, physical, or fx-based cash-settlement. This class can be used for the settlement of options and forwards, cash transactions (e.g. securities or foreign exchange), or in case of credit event.
 */
type SettlementTerms struct {
	/**
	 * Whether the settlement will be cash, physical, by election, ...
	 */
	SettlementType SettlementTypeEnum
	/**
	 * The qualification as to how the transfer will settle, e.g. a DvP settlement.
	 */
	TransferSettlementType TransferSettlementEnum
	/**
	 * The settlement currency is to be specified when the Settlement Amount cannot be known in advance. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	SettlementCurrency FieldWithMeta
	/**
	 * The date on which the settlement amount will be paid, subject to adjustment in accordance with any applicable business day convention. This component would not be present for a mandatory early termination provision where the cash settlement payment date is the mandatory early termination date.
	 */
	SettlementDate SettlementDate
	/**
	 * Optional settlement centre as an enumerated list: Euroclear, Clearstream.
	 */
	SettlementCentre SettlementCentreEnum
	/**
	 * Optionally defines the parameters that regulate a settlement.
	 */
	SettlementProvision SettlementProvision
	/**
	 * Settlement Style.
	 */
	StandardSettlementStyle StandardSettlementStyleEnum
	Meta                    MetaFields
	/**
	 * Specifies the parameters associated with the cash settlement procedure.
	 */
	CashSettlementTerms []CashSettlementTerms
	/**
	 * Specifies the physical settlement terms which apply to the transaction.
	 */
	PhysicalSettlementTerms PhysicalSettlementTerms
}

/**
 * Defines the applicable settlement limits that may require a settlement to be 'shaped', i.e. broken-down into smaller amounts.
 */
type ShapingProvision struct {
	/**
	 * Defines applicable settlement limits in each currency.
	 */
	ShapeSchedule []Money
}

/**
 * A class to specify the SIMM Calculation Currency elections by each party to the agreement.
 */
type SimmCalculationCurrency struct {
	/**
	 * The parties' SIMM Calculation Currency election.
	 */
	PartyElection []CalculationCurrencyElection
}

/**
 * A class to specify the SIMM exception to the regulatory regime clause.
 */
type SimmException struct {
	/**
	 * The Standard Initial Margin Model exception when specified by the party according to one of the enumerated values.
	 */
	StandardisedException ExceptionEnum
	/**
	 * The Standard Initial Margin model exception approach applicable when specified by the party according to one of the enumerated values.
	 */
	SimmExceptionApplicable SimmExceptionApplicableEnum
	/**
	 * The Standard Initial Margin Model exception when specified as a customized approach by the party.
	 */
	AsSpecified string
}

/**
 * A class to specify the ISDA SIMM version that applies to the ISDA 2018 CSA for Initial Margin.
 */
type SimmVersion struct {
	/**
	 * A boolean attribute to determine whether the SIMM version is specified for the purpose of the legal agreement.
	 */
	IsSpecified bool
	/**
	 * The party which the specified SIMM version applies to.
	 */
	PartyVersion CounterpartyRoleEnum
	/**
	 * The SIMM version exception when specified as a customized approach by the party.
	 */
	AsSpecified string
}

/**
 * Where only a single  party is explicitly designated to post collateral, the party which is specified as the sole poster of collateral.
 */
type SinglePostingParty struct {
	/**
	 * The party acting as the Single Posting Party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * Utilised where the clause data structure is not able to capture a material aspect of the clause.
	 */
	Other string
}

/**
 * A class to specify the number of business days after satisfaction of all conditions to settlement.
 */
type SingleValuationDate struct {
	/**
	 * A number of business days. Its precise meaning is dependant on the context in which this element is used. ISDA 2003 Term: Business Day.
	 */
	BusinessDays int
}

type Sngl struct {
	Isin string
	Indx Indx
}

type SovereignAgencyRating struct {
	/**
	 * Represents an agency rating based on default risk of the country of the issuer.
	 */
	SovereignAgencyRating AgencyRatingCriteria
}

/**
 * Represents a class to allow specification of different types of special purpose vehicle (SPV) collateral.
 */
type SpecialPurposeVehicleIssuerType struct {
	/**
	 * Indicates tranched or untranched credit risk.
	 */
	CreditRisk CreditRiskEnum
}

/**
 * A single, specifically identified Asset chosen from the Asset data type
 */
type SpecificAsset struct {
	/**
	 * An Asset that consists solely of a monetary holding in a currency.
	 */
	Cash Cash
	/**
	 * An Asset comprised of raw or refined materials or agricultural products, eg gold, oil or wheat.
	 */
	Commodity Commodity
	/**
	 * An Asset that exists only in digital form, eg Bitcoin or Ethereum; excludes the digital representation of other Assets.
	 */
	DigitalAsset DigitalAsset
	/**
	 * An asset that is issued by one party to one or more others; Instrument is also a choice data type.
	 */
	Instrument Instrument
}

type SpecificInconvertibility struct {
	SpecificInconvertibilityIsApplicable bool
	MinimumAmount                        Quantity
	RelevantClass                        []string
	LocalSubstituteProvisionType         FxSubstitutionProvisionTypeEnum
	/**
	 * Specifies how long to wait to get a quote from a settlement rate option upon a price source disruption.
	 */
	MaximumDaysOfDisruption  ValuationPostponement
	NonDeliverableSubstitute NonDeliverableSubstitute
	/**
	 * Refers to the segregation deposit procedure parties may want to commmit into during the time a Disruption Event is effective
	 */
	EscrowArrangement EscrowArrangement
}

type SpecificNonTransferability struct {
	SpecificNonTransferabilityIsApplicable bool
	RelevantClass                          []string
	LocalSubstituteProvisionType           FxSubstitutionProvisionTypeEnum
	MaximumDaysOfDisruption                ValuationPostponement
	NonDeliverableSubstitute               NonDeliverableSubstitute
	EscrowArrangement                      EscrowArrangement
}

/**
 * Specifies the events elected by the parties that are deemed an Access Condition (for Initial Margin CSA) or a Specified Condition (for Variation Margin and Legacy CSA), which, if triggered, may temporarily or permanently suspend rights to rehypothecate, transfer, or substitute collateral.
 */
type SpecifiedConditionOrAccessCondition struct {
	/**
	 * Indicates whether events that could temporarily or permanently suspend rights to rehypothecate, transfer or substitute collateral are applicable. If applicable,'True'. If not applicable,'False'.
	 */
	IsApplicable bool
	/**
	 * The party specific elections.
	 */
	PartyElection []SpecifiedOrAccessConditionPartyElection
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

type SpecifiedCurrency struct {
	/**
	 * Indicates whether the specified currency provision is applicable.
	 */
	Applicable bool
	/**
	 * The currency in which the specified currency is denominated. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	Currency FieldWithMeta
}

/**
 * A provision that allows each party to specify its Specified Entities for certain Events of Default and Termination Events.
 */
type SpecifiedEntities struct {
	/**
	 * The Event of Default or Termination event for which Specified Entities terms are being defined.
	 */
	SpecifiedEntityClause SpecifiedEntityClauseEnum
	/**
	 * The party specific election of Specified Entities for the Event of Default or Termination Event specified.
	 */
	SpecifiedEntity []SpecifiedEntity
}

/**
 * The party specific election of Specified Entities for the Event of Default or Termination Event specified.
 */
type SpecifiedEntity struct {
	/**
	 * The elective party.
	 */
	Party Party
	/**
	 * The specified entity terms for the Event of Default or Termination Event specified.
	 */
	SpecifiedEntityTerms SpecifiedEntityTermsEnum
	/**
	 * The specified entities for the Event of Default or Termination Event specified.
	 */
	SpecifiedEntity []LegalEntity
	/**
	 * The meaning of Material Subsidiary for the Event of Default or Termination Event specified.
	 */
	MaterialSubsidiaryTerms string
	/**
	 * The non standard terms for the Event of Default or Termination Event specified.
	 */
	OtherSpecifiedEntityTerms string
}

/**
 * The party specific elections in respect to the Specified Condition and Access Condition clause.
 */
type SpecifiedOrAccessConditionPartyElection struct {
	/**
	 * Identifies the party making the election.
	 */
	Party CounterpartyRoleEnum
	/**
	 * Specifies events that may temporarily or permanently suspend a partys rights to rehypothecate, transfer, or substitute collateral.
	 */
	SpecifiedOrAccessCondition []CSASpecifiedOrAccessConditionEnum
	/**
	 * Specifies events that may lead to the early termination of the Master Agreement, including suspension of the affected party's rights to rehypothecate, transfer, or substitute collateral.
	 */
	AdditionalTerminationEvent []AdditionalTerminationEventEnum
	/**
	 * Specifies any bespoke Additional Termination Event(s) that are set out in the related Specified Condition clause.
	 */
	SpecifiedAdditionalTerminationEvent []string
}

/**
 * Defines the underlier for both the Parent Company and the Child Compagny involved in Spin Off corporate action, as well as th respective price observation for each.
 */
type SpinOff struct {
	/**
	 * Defines the underlier for the Parent Company involved in Spin Off corporate action.
	 */
	ParentSecurity Security
	/**
	 * The price observation that relates to underlier for the Parent Company.
	 */
	ParentSecurityPrice Price
	/**
	 * Defines the underlier for the Child Company involved in Spin Off corporate action.
	 */
	ChildSecurity Security
	/**
	 * The price observation that relates to underlier for the Child Company.
	 */
	ChildSecurityPrice Price
}

/**
 * Specifies instructions for a split, consisting of a breakdown of instructions to be applied to each branch of the split. This instruction can be used to duplicate a trade, as in a clearing scenario, or to split a trade into smaller quantities (in which case each breakdown instruction needs to include a quantity change), as in an allocation.
 */
type SplitInstruction struct {
	/**
	 * Each split breakdown specifies the set of primitive instructions to be applied to a single branch of that split. N split breakdowns result in N output trades, which include the original trade. Instructions for how to handle the original trade (e.g. if it must be closed) must be specified in one of the breakdowns.
	 */
	Breakdown []PrimitiveInstruction
}

/**
 * Adds an optional spread type element to the Schedule to identify a long or short spread value.
 */
type SpreadSchedule struct {
	/**
	 * The initial rate. An initial rate of 5% would be represented as 0.05.
	 */
	Price ReferenceWithMeta
	/**
	 * An element which purpose is to identify a long or short spread value.
	 */
	SpreadScheduleType FieldWithMeta
}

type StandardizedSchedule struct {
	AssetClass       StandardizedScheduleAssetClassEnum
	ProductClass     StandardizedScheduleProductClassEnum
	Notional         float64
	NotionalCurrency string
	DurationInYears  float64
}

type StandardizedScheduleInitialMargin struct {
	TradeInfo        []StandardizedScheduleTradeInfo
	NetInitialMargin Money
}

type StandardizedScheduleTradeInfo struct {
	AssetClass         StandardizedScheduleAssetClassEnum
	ProductClass       StandardizedScheduleProductClassEnum
	GrossInitialMargin Money
	MarkToMarketValue  Money
}

/**
 * Defines the state of a trade at a point in the Trade's life cycle. Trades have many state dimensions, all of which are represented here. For example, states useful for position keeping are represented alongside those needed for regulatory reporting.
 */
type State struct {
	/**
	 * Represents the qualification of what led to the trade's closure alongside the dates on which this closure took effect.
	 */
	ClosedState ClosedState
	/**
	 * Identifies the state of the position, to distinguish if just executed, formed, already settled, closed, etc.
	 */
	PositionState PositionStatusEnum
}

/**
 * Data required to perform a stock split business event.
 */
type StockSplitInstruction struct {
	/**
	 * The number that denotes the cumulative quantity of post-split shares issued to shareholders versus the quantity of pre-split shares previously issued to shareholders.  This number will be multiplied by existing shares in an equity derivative contract or other positions to determine the post-split number of shares.  With regard to any reference to price, the pre-split reference price will be divided by this number to determine the post-split reference price.
	 */
	AdjustmentRatio float64
	/**
	 * The effective date of the stock split, also known as the ex-date. This is the date on which the additional shares are paid to the shareholders, or in the case of a reverse stock split, the number shares held by each shareholder is proportionally reduced.  Equity derivative transactions can be amended in firms' internal systems on such date.   In most markets, the listed stock price is reduced (or increased for a reverse stock split) to account for the split on the same date, but in some markets the price adjustment occurs on a later date.  In either case, equity derivative transactions should be amended on the date that the stocks are paid to the shareholders (or consolidated).
	 */
	EffectiveDate time.Time
}

/**
 * A class for defining option strategy features.
 */
type StrategyFeature struct {
	/**
	 * Definition of the upper strike in a strike spread.
	 */
	StrikeSpread StrikeSpread
	/**
	 * Definition of the later expiration date in a calendar spread.
	 */
	CalendarSpread CalendarSpread
}

/**
 * A class describing a single cap or floor rate.
 */
type Strike struct {
	/**
	 * The rate for a cap or floor.
	 */
	StrikeRate float64
	/**
	 * The buyer of the option.
	 */
	Buyer PayerReceiverEnum
	/**
	 * The party that has sold.
	 */
	Seller PayerReceiverEnum
	Meta   MetaFields
}

/**
 * A class describing a schedule of cap or floor rates.
 */
type StrikeSchedule struct {
	/**
	 * The initial rate. An initial rate of 5% would be represented as 0.05.
	 */
	Price ReferenceWithMeta
	/**
	 * The buyer of the option.
	 */
	Buyer PayerReceiverEnum
	/**
	 * The party that has sold.
	 */
	Seller PayerReceiverEnum
}

/**
 * A class for defining a strike spread feature.
 */
type StrikeSpread struct {
	/**
	 * Upper strike in a strike spread.
	 */
	UpperStrike OptionStrike
	/**
	 * Number of options at the upper strike price in a strike spread.
	 */
	UpperStrikeNumberOfOptions float64
}

/**
 * A data defining:  how the initial or final stub calculation period amounts is calculated. For example, the rate to be applied to the initial or final stub calculation period may be the linear interpolation of two different tenors for the floating rate index specified in the calculation period amount component, e.g. A two month stub period may used the linear interpolation of a one month and three month floating rate. The different rate tenors would be specified in this component. Note that a maximum of two rate tenors can be specified. If a stub period uses a single index tenor and this is the same as that specified in the calculation period amount component then the initial stub or final stub component, as the case may be, must not be included.
 */
type StubCalculationPeriodAmount struct {
	/**
	 * A pointer style reference to the associated calculation period dates component defined elsewhere in the document.
	 */
	CalculationPeriodDatesReference ReferenceWithMeta
	/**
	 * Specifies how the initial stub amount is calculated. A single floating rate tenor different to that used for the regular part of the calculation periods schedule may be specified, or two floating tenors may be specified. If two floating rate tenors are specified then Linear Interpolation (in accordance with the 2000 ISDA Definitions, Section 8.3. Interpolation) is assumed to apply. Alternatively, an actual known stub rate or stub amount may be specified.
	 */
	InitialStub StubValue
	/**
	 * Specifies how the final stub amount is calculated. A single floating rate tenor different to that used for the regular part of the calculation periods schedule may be specified, or two floating tenors may be specified. If two floating rate tenors are specified then Linear Interpolation (in accordance with the 2000 ISDA Definitions, Section 8.3. Interpolation) is assumed to apply. Alternatively, an actual known stub rate or stub amount may be specified.
	 */
	FinalStub StubValue
}

/**
 * A class defining a floating rate.
 */
type StubFloatingRate struct {
	/**
	 * The floating rate index.
	 */
	FloatingRateIndex FloatingRateIndexEnum
	/**
	 * The ISDA Designated Maturity, i.e. the tenor of the floating rate.
	 */
	IndexTenor Period
	/**
	 * A rate multiplier or multiplier schedule to apply to the floating rate. A multiplier schedule is expressed as explicit multipliers and dates. In the case of a schedule, the step dates may be subject to adjustment in accordance with any adjustments specified in the calculationPeriodDatesAdjustments. The multiplier can be a positive or negative decimal. This element should only be included if the multiplier is not equal to 1 (one) for the term of the stream.
	 */
	FloatingRateMultiplierSchedule Schedule
	/**
	 * The ISDA Spread or a Spread schedule expressed as explicit spreads and dates. In the case of a schedule, the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The spread is a per annum rate, expressed as a decimal. For purposes of determining a calculation period amount, if positive the spread will be added to the floating rate and if negative the spread will be subtracted from the floating rate. A positive 10 basis point (0.1%) spread would be represented as 0.001.
	 */
	SpreadSchedule []SpreadSchedule
	/**
	 * The specification of any rate conversion which needs to be applied to the observed rate before being used in any calculations. The two common conversions are for securities quoted on a bank discount basis which will need to be converted to either a Money Market Yield or Bond Equivalent Yield. See the Annex to the 2000 ISDA Definitions, Section 7.3. Certain General Definitions Relating to Floating Rate Options, paragraphs (g) and (h) for definitions of these terms.
	 */
	RateTreatment RateTreatmentEnum
	/**
	 * The cap rate or cap rate schedule, if any, which applies to the floating rate. The cap rate (strike) is only required where the floating rate on a swap stream is capped at a certain level. A cap rate schedule is expressed as explicit cap rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The cap rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A cap rate of 5% would be represented as 0.05.
	 */
	CapRateSchedule []StrikeSchedule
	/**
	 * The floor rate or floor rate schedule, if any, which applies to the floating rate. The floor rate (strike) is only required where the floating rate on a swap stream is floored at a certain strike level. A floor rate schedule is expressed as explicit floor rates and dates and the step dates may be subject to adjustment in accordance with any adjustments specified in calculationPeriodDatesAdjustments. The floor rate is assumed to be exclusive of any spread and is a per annum rate, expressed as a decimal. A floor rate of 5% would be represented as 0.05.
	 */
	FloorRateSchedule []StrikeSchedule
}

/**
 *  A class defining how the initial or final stub calculation period amounts is calculated. For example, the rate to be applied to the initial or final stub calculation period may be the linear interpolation of two different tenors for the floating rate index specified in the calculation period amount component, e.g. A two month stub period may used the linear interpolation of a one month and three month floating rate. The different rate tenors would be specified in this component. Note that a maximum of two rate tenors can be specified. If a stub period uses a single index tenor and this is the same as that specified in the calculation period amount component then the initial stub or final stub component, as the case may be, must not be included.
 */
type StubPeriod struct {
	/**
	 * A pointer style reference to the associated calculation period dates component defined elsewhere in the document.
	 */
	CalculationPeriodDatesReference ReferenceWithMeta
	/**
	 * Specifies how the initial stub amount is calculated. A single floating rate tenor different to that used for the regular part of the calculation periods schedule may be specified, or two floating tenors may be specified. If two floating rate tenors are specified then Linear Interpolation (in accordance with the 2000 ISDA Definitions, Section 8.3. Interpolation) is assumed to apply. Alternatively, an actual known stub rate or stub amount may be specified.
	 */
	InitialStub StubValue
	/**
	 * Specifies how the final stub amount is calculated. A single floating rate tenor different to that used for the regular part of the calculation periods schedule may be specified, or two floating tenors may be specified. If two floating rate tenors are specified then Linear Interpolation (in accordance with the 2000 ISDA Definitions, Section 8.3. Interpolation) is assumed to apply. Alternatively, an actual known stub rate or stub amount may be specified.
	 */
	FinalStub StubValue
}

/**
 * A type defining how a stub calculation period amount is calculated. A single floating rate tenor different to that used for the regular part of the calculation periods schedule may be specified, or two floating rate tenors many be specified. If two floating rate tenors are specified then Linear Interpolation (in accordance with the 2000 ISDA Definitions, Section 8.3 Interpolation) is assumed to apply. Alternatively, an actual known stub rate or stub amount may be specified.
 */
type StubValue struct {
	/**
	 * The rates to be applied to the initial or final stub may be the linear interpolation of two different rates. While the majority of the time, the rate indices will be the same as that specified in the stream and only the tenor itself will be different, it is possible to specift two different rates. For example, a 2 month stub period may use the linear interpolation of a 1 month and 3 month rate. The different rates would be specified in this component. Note that a maximum of two rates can be specified. If a stub period uses the same floating rate index, including tenor, as the regular calculation periods then this should not be specified again within this component, i.e. the stub calculation period amount component may not need to be specified even if there is an initial or final stub period. If a stub period uses a different floating rate index compared to the regular calculation periods then this should be specified within this component. If specified here, they are likely to have id attributes, allowing them to be referenced from within the cashflows component.
	 */
	FloatingRate []StubFloatingRate
	/**
	 * An actual rate to apply for the initial or final stub period may have been agreed between the principal parties (in a similar way to how an initial rate may have been agreed for the first regular period). If an actual stub rate has been agreed then it would be included in this component. It will be a per annum rate, expressed as a decimal. A stub rate of 5% would be represented as 0.05.
	 */
	StubRate float64
	/**
	 * An actual amount to apply for the initial or final stub period may have been agreed between the two parties. If an actual stub amount has been agreed then it would be included in this component.
	 */
	StubAmount Money
}

/**
 * A class to specify each party's election with respect to the Substituted Regimes that will be applicable...
 */
type SubstitutedRegime struct {
	/**
	 * The applicable regulatory regime, as specified through an enumeration.
	 */
	Regime RegulatoryRegimeEnum
	/**
	 * The additional regulatory regime as specified by the parties.
	 */
	AdditionalRegime string
	/**
	 * Specifies the applicability of the Substituted Regime as denoted in the Substituted Regime Table as part of certain legal agreements, such as the ISDA 2016 and 2018 CSA for Initial Margin.
	 */
	RegimeTerms []SubstitutedRegimeTerms
}

/**
 * Specifies the applicability of the Substituted Regime as denoted in the Substituted Regime Table as part of certain legal agreements, such as such as the ISDA 2016 and 2018 CSA for Initial Margin.
 */
type SubstitutedRegimeTerms struct {
	/**
	 * The party for which the regime terms are being specified when acting as collateral taker.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The specification of whether the regime is elected as applicable to the party when acting as collateral taker.
	 */
	IsApplicable bool
}

/**
 * A class to specify the conditions under which the Security Provider can substitute posted collateral. The details required for 'Legacy' agreements are substantially different to those required for newer agreements; where the agreement is a 1994 or 1995 CSA, or 1995 CSD, the legacyConsent attribute needs to be used which contains additional values beyond a simple true/false value.
 */
type Substitution struct {
	/**
	 * Party specific elections for substitution of posted collateral.
	 */
	PartyElection []SubstitutionPartyElection
}

/**
 * Party specific elections for substitution of posted collateral.
 */
type SubstitutionPartyElection struct {
	/**
	 * The party that is acting as the Pledgor/Obligor/Chargor/Security Provider.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The election as to whether the Pledgor/Obligor/Chargor/Security Provider must obtain the Secured Partys consent for any collateral substitution.
	 */
	NeedsConsent bool
	/**
	 * Substitution Date has the meaning specified in Paragraph4(d)(ii), unless otherwise specified.
	 */
	SubstitutionDateLanguage string
	/**
	 * Specific consent language might be specified by the parties.
	 */
	SpecificConsentLanguage string
	/**
	 * The process for substituting/exchanging one form of collateral held by a party for a different form of collateral, used with legacy agreements.
	 */
	LegacyConsent LegacyConsentEnum
}

/**
 * Defines collateral substitution provisions such as how many and with how much notice are substitutions allowed.
 */
type SubstitutionProvisions struct {
	/**
	 * Specifies if 1 or more substitutions are allowed.
	 */
	NumberOfSubstitutionsAllowed int
	/**
	 * Defines the min period for notify of a substitution.
	 */
	NoticeDeadlinePeriod Period
	/**
	 * A specific date and time for the notice deadline
	 */
	NoticeDeadlineDateTime time.Time
}

/**
 * A class to specify a valuation swap curve, which is used as part of the strike construct for the bond and convertible bond options.
 */
type SwapCurveValuation struct {
	FloatingRateIndex FloatingRateIndexEnum
	/**
	 * The ISDA Designated Maturity, i.e. the tenor of the floating rate.
	 */
	IndexTenor Period
	/**
	 * Spread in basis points over the floating rate index.
	 */
	Spread float64
	/**
	 * The side (bid/mid/ask) of the measure.
	 */
	Side QuotationSideEnum
}

type Swp struct {
	SwpIn  SwpIn
	SwpOut SwpOut
}

type SwpIn struct {
	Sngl Sngl
}

type SwpOut struct {
	Sngl Sngl
}

/**
 * Defines the taxonomy of an object by combining a taxonomy source (i.e. the rules to classify the object) and a value (i.e. the output of those rules on the object).
 */
type Taxonomy struct {
	/**
	 * The source of the taxonomy that defines the rules for classifying the object. The taxonomy source is taken from a enumerated list of taxonomy names. Optional as the taxonomy source may not be provided.
	 */
	Source TaxonomySourceEnum
	/**
	 * The value according to that taxonomy. Optional as it may not be possible to classify the object in that taxonomy.
	 */
	Value TaxonomyValue
}

type TaxonomyClassification struct {
	/**
	 * The name defined by the classification system for a specific attribute in the taxonomy
	 */
	ClassName string
	/**
	 * The value set by the taxonomy that is specific to the className attribute.
	 */
	Value string
	/**
	 * A description of the class.
	 */
	Description string
	/**
	 * In the case of multi-layered hierarchical classification systems such as commodity classification, the layer the value and className occupy in the classification hierarchy, where 1 represents the top-layer.
	 */
	Ordinal int
}

/**
 * Defines a taxonomy value as either a simple string or a more granular expression with class names and values for each class.
 */
type TaxonomyValue struct {
	/**
	 * Specifies the taxonomy value as a simple string, which may be associated to an external scheme.
	 */
	Name FieldWithMeta
	/**
	 * Specifies the taxonomy value as a set of class names and values for each class.
	 */
	Classification []TaxonomyClassification
}

/**
 * A class to specify a telephone number as a type of phone number (e.g. work, personal, ...) alongside with the actual number.
 */
type TelephoneNumber struct {
	/**
	 * The type of telephone number, e.g. work, mobile.
	 */
	TelephoneNumberType TelephoneTypeEnum
	/**
	 * The actual telephone number.
	 */
	Number string
}

type Term struct {
	Unit string
	Val  string
}

/**
 * Specifies how the Termination Currency for the agreement will be determined.
 */
type TerminationCurrency struct {
	/**
	 * Allows for specific Termination Currency(ies) and a fallback Termination Currency to be selected.
	 */
	StatedTerminationCurrency TerminationCurrencySelection
	/**
	 * Provides that the Termination Currency will be determined by reference to a contractual mechanism when closing out the Agreement.
	 */
	PartyOptionTerminationCurrency PartyOptionTerminationCurrency
}

/**
 * A class to specify the Amendment to Termination Currency elections by the parties to the agreement.
 */
type TerminationCurrencyAmendment struct {
	/**
	 * The qualification of whether the Amendment to Termination Currency is deemed applicable by the parties (True) or not (False).
	 */
	IsApplicable bool
	/**
	 * The effective date of the Amendment to Termination Currency. This date can be specified as either an actual date, a specific date (e.g. the annex date) or as a custom provision.
	 */
	EffectiveDate AmendmentEffectiveDate
	/**
	 * The parties' Amendment Currency election.
	 */
	PartyElection []TerminationCurrencyElection
}

/**
 * A class to specify the Amendment to Termination Currency election by the parties to the agreement.
 */
type TerminationCurrencyElection struct {
	/**
	 * The elective party.
	 */
	Party []CounterpartyRoleEnum
	/**
	 * The qualification of whether the Termination Currency is specified in this document (True) or in an Eligible Support Credit Support (IM) Schedule (False).
	 */
	IsSpecified bool
	/**
	 * The Termination Currency associated with the party that referenced as part of this class. The list of valid currencies is not presently positioned as an enumeration as part of the CDM because that scope is limited to the values specified by ISDA and FpML. As a result, implementers have to make reference to the relevant standard, such as the ISO 4217 standard for currency codes.
	 */
	Currency FieldWithMeta
}

/**
 * Specifies Termination Currency where a currency is stated at the time the agreement is entered into.
 */
type TerminationCurrencySelection struct {
	/**
	 * Specifies a single Termination Currency for the agreement.
	 */
	StatedCurrency string
	/**
	 * Specifies a single fallback Termination Currency should the stated currency not be freely available.
	 */
	FallbackCurrency string
	/**
	 * Specifies different termination currencies to apply depending on which party or parties are the Defaulting Party Affected Party(ies).
	 */
	PartyElection []PartyTerminationCurrencySelection
	/**
	 * Specifies fallback Termination Currency where both parties are Affected Parties.
	 */
	BothAffected string
}

/**
 * A class for defining option provisions.
 */
type TerminationProvision struct {
	/**
	 * A provision that allows the specification of an embedded option within a swap giving the buyer of the option the right to terminate the swap, in whole or in part, on the early termination date.
	 */
	CancelableProvision CancelableProvision
	/**
	 * Parameters specifying provisions relating to the optional and mandatory early termination of a swap transaction.
	 */
	EarlyTerminationProvision EarlyTerminationProvision
	/**
	 * A data defining: the right of a party to exercise an Evergreen option
	 */
	EvergreenProvision EvergreenProvision
	/**
	 * A provision that allows the specification of an embedded option with a swap giving the buyer of the option the right to extend the swap, in whole or in part, to the extended termination date.
	 */
	ExtendibleProvision ExtendibleProvision
}

/**
 * Specifies instructions for terms change consisting of the new transaction terms, and the renegotiation fee.
 */
type TermsChangeInstruction struct {
	/**
	 * product to be changed
	 */
	Product NonTransferableProduct
	/**
	 * ancillary party to be changed
	 */
	AncillaryParty []AncillaryParty
	Adjustment     NotionalAdjustmentEnum
}

/**
 * Represents the amount of unsecured risk that a party is willing to tolerate without holding any posted collateral provided by the other party.
 */
type Threshold struct {
	/**
	 * The party specific election for the threshold clause.
	 */
	PartyElection []ThresholdElection
}

/**
 * Specifies whether the threshold is rating based, a fixed amount, or infinity.
 */
type ThresholdElection struct {
	/**
	 * The party to which the threshold applies.
	 */
	Party CounterpartyRoleEnum
	/**
	 * Defines that the Threshold is a Fixed Amount.
	 */
	FixedAmount ThresholdMinimumTransferAmountFixedAmount
	/**
	 * Defines that the Threshold is based on a Ratings condition(s).
	 */
	RatingsBased ThresholdRatingsBased
	/**
	 * Specifies that the Threshold is Infinity. If 'True' then the threshold for the elective party is Infinity.
	 */
	Infinity bool
	/**
	 * Utilised where the clause data structure is not able to capture a material aspect of the clause.
	 */
	Other string
}

/**
 * Defines the Threshold or the Minimum Transfer Amount (MTA) as a fixed amount.
 */
type ThresholdMinimumTransferAmountFixedAmount struct {
	/**
	 * Indicates whether a trigger applies for the Threshold or Minimum Transfer Amount to fall to zero. If 'True' then such a trigger exists.
	 */
	ZeroEvent bool
	/**
	 * Specifies the relevant trigger for the Threshold or Minimum Transfer Amount to fall to zero.
	 */
	Event []ZeroEventEnum
	/**
	 * Specifies the monetary value of the Threshold or the Minimum Transfer Amount (MTA) applicable to collateral transfer obligations.
	 */
	Amount Money
}

/**
 * A logical container designed to hold a set of related data. In this case, designed to hold a set of information required to specify conditions under which Threshold or Minimum Transfer Amount can fall to zero.
 */
type ThresholdMinmumTransferAmountBase struct {
	/**
	 * Indicates whether a trigger applies for the Threshold or Minimum Transfer Amount to fall to zero. If 'True' then such a trigger exists.
	 */
	ZeroEvent bool
	/**
	 * Specifies the relevant trigger for the Threshold or Minimum Transfer Amount to fall to zero.
	 */
	Event []ZeroEventEnum
}

/**
 * Defines that the Threshold is based on Ratings condition(s).
 */
type ThresholdRatingsBased struct {
	/**
	 * Indicates whether a trigger applies for the Threshold or Minimum Transfer Amount to fall to zero. If 'True' then such a trigger exists.
	 */
	ZeroEvent bool
	/**
	 * Specifies the relevant trigger for the Threshold or Minimum Transfer Amount to fall to zero.
	 */
	Event []ZeroEventEnum
	/**
	 * The threshold currency code.
	 */
	Currency ISOCurrencyCodeEnum
	/**
	 * The relevant rating type.
	 */
	RatingType RatingTypeEnum
	/**
	 * Defines a combination of Rating Agency, Rating Value, Threshold amount and Currency code.
	 */
	VariableSet []CSAThresholdVariableSet
	/**
	 * The party to which a rating applies.
	 */
	RatedParty RatedPartyEnum
	/**
	 * Details the Named Entity where the Rated Party is Named Entity.
	 */
	NamedEntity string
	/**
	 * Details the Named Affiliate where the Rated Party is Named Affiliate.
	 */
	NamedAffiliate string
	/**
	 * Where two ratings are specified whether the higher or lower rating prevails.
	 */
	Compare CreditNotationMismatchResolutionEnum
	/**
	 * What conditions apply where a party has no rating.
	 */
	NoRating bool
	/**
	 * Defines where conditions apply if no Rating where ratings may not exist.
	 */
	NotRatedBy NotRatedByEnum
	/**
	 * Defines the number of Rating Agencies that the Party must be rated by.
	 */
	NumberOfRatingAgencies NumberOfRatingAgenciesEnum
}

/**
 * The time alongside with the timezone location information. This class makes use of the FpML TimezoneLocation construct.
 */
type TimeZone struct {
	/**
	 * The observation time.
	 */
	Time time.Time
	/**
	 * FpML specifies the timezoneLocationScheme by reference to the Time Zone Database (a.k.a. tz database) maintained by IANA, the Internet Assigned Numbers Authority.
	 */
	Location FieldWithMeta
}

/**
 * Definition of a product as ready to be traded, i.e. included in an execution or contract, by associating a specific price and quantity to this product plus an (optional) mechanism for any potential future quantity adjustment.
 */
type TradableProduct struct {
	/**
	 * The underlying product to be included in a contract or execution.
	 */
	Product NonTransferableProduct
	/**
	 * Specifies the price, quantity and effective date of each trade lot, when the same product may be traded multiple times in different lots with the same counterparty. In a trade increase, a new trade lot is added to the list, with the corresponding effective date. In a trade decrease, the existing trade lot(s) are decreased of the corresponding quantity (and an unwind fee may have to be settled). The multiple cardinality and the ability to increase existing trades is used for Equity Swaps in particular.
	 */
	TradeLot []TradeLot
	/**
	 * Specifies the parties which are the two counterparties to the transaction.  The product is agnostic to the actual parties to the transaction, with the party references abstracted away from the product definition and replaced by the counterparty enum (e.g. CounterpartyEnum values Party1 or Party2). The counterparty enum can then be positioned in the product (e.g. to specify which counterparty is the payer, receiver etc) and this counterparties attribute, which is positioned outside of the product definition, allows the counterparty enum to be associated with an actual party reference.
	 */
	Counterparty []Counterparty
	/**
	 * Specifies the parties with ancillary roles in the transaction. The product is agnostic to the actual parties involved in the transaction, with the party references abstracted away from the product definition and replaced by the AncillaryRoleEnum. The AncillaryRoleEnum can then be positioned in the product and this AncillaryParty type, which is positioned outside of the product definition, allows the AncillaryRoleEnum to be associated with an actual party reference.
	 */
	AncillaryParty []AncillaryParty
	/**
	 * Specifies the conditions that govern the adjustment to the quantity of a product being traded: e.g. execution, portfolio rebalancing etc. It is typically used in the context of Equity Swaps.
	 */
	Adjustment NotionalAdjustmentEnum
}

/**
 * Defines the output of a financial transaction between parties - a Business Event. A Trade impacts the financial position (i.e. the balance sheet) of involved parties.
 */
type Trade struct {
	/**
	 * The underlying product to be included in a contract or execution.
	 */
	Product NonTransferableProduct
	/**
	 * Specifies the price, quantity and effective date of each trade lot, when the same product may be traded multiple times in different lots with the same counterparty. In a trade increase, a new trade lot is added to the list, with the corresponding effective date. In a trade decrease, the existing trade lot(s) are decreased of the corresponding quantity (and an unwind fee may have to be settled). The multiple cardinality and the ability to increase existing trades is used for Equity Swaps in particular.
	 */
	TradeLot []TradeLot
	/**
	 * Specifies the parties which are the two counterparties to the transaction.  The product is agnostic to the actual parties to the transaction, with the party references abstracted away from the product definition and replaced by the counterparty enum (e.g. CounterpartyEnum values Party1 or Party2). The counterparty enum can then be positioned in the product (e.g. to specify which counterparty is the payer, receiver etc) and this counterparties attribute, which is positioned outside of the product definition, allows the counterparty enum to be associated with an actual party reference.
	 */
	Counterparty []Counterparty
	/**
	 * Specifies the parties with ancillary roles in the transaction. The product is agnostic to the actual parties involved in the transaction, with the party references abstracted away from the product definition and replaced by the AncillaryRoleEnum. The AncillaryRoleEnum can then be positioned in the product and this AncillaryParty type, which is positioned outside of the product definition, allows the AncillaryRoleEnum to be associated with an actual party reference.
	 */
	AncillaryParty []AncillaryParty
	/**
	 * Specifies the conditions that govern the adjustment to the quantity of a product being traded: e.g. execution, portfolio rebalancing etc. It is typically used in the context of Equity Swaps.
	 */
	Adjustment NotionalAdjustmentEnum
	/**
	 * Represents the identifier(s) that uniquely identify a trade for an identity issuer. A trade can include multiple identifiers, for example a trade that is reportable to both the CFTC and ESMA, and then has an associated USI (Unique Swap Identifier) UTI (Unique Trade Identifier).
	 */
	TradeIdentifier []TradeIdentifier
	/**
	 * Specifies the date which the trade was agreed.
	 */
	TradeDate FieldWithMeta
	/**
	 * Denotes the trade time and timezone as agreed by the parties to the trade.
	 */
	TradeTime FieldWithMeta
	/**
	 * Represents the parties to the trade. The cardinality is optional to address the case where the trade is defined within a BusinessEvent data type, in which case the party is specified in BusinessEvent.
	 */
	Party []Party
	/**
	 * Represents the role each specified party takes in the trade. further to the principal roles, payer and receiver.
	 */
	PartyRole []PartyRole
	/**
	 * Represents information specific to trades that arose from executions.
	 */
	ExecutionDetails ExecutionDetails
	/**
	 * Represents information specific to trades involving contractual products.
	 */
	ContractDetails ContractDetails
	/**
	 * Specifies the date on which a trade is cleared (novated) through a central counterparty clearing service.
	 */
	ClearedDate time.Time
	/**
	 * Represents the collateral obligations of a party.
	 */
	Collateral Collateral
	/**
	 * Represents a party's granular account information, which may be used in subsequent internal processing.
	 */
	Account []Account
	Meta    MetaFields
}

/**
 * Defines a trade identifier as a special case of the generic identifier type, that also includes the trade identifier class.
 */
type TradeIdentifier struct {
	/**
	 * The identifier issuer, when specified by reference to a party specified as part of the transaction.
	 */
	IssuerReference ReferenceWithMeta
	/**
	 * The identifier issuer, when specified explicitly alongside the identifier value (instead of being specified by reference to a party).
	 */
	Issuer FieldWithMeta
	/**
	 * The identifier value. This level of indirection between the issuer and the identifier and its version provides the ability to associate multiple identifiers to one issuer, consistently with the FpML PartyTradeIdentifier.
	 */
	AssignedIdentifier []AssignedIdentifier
	Meta               MetaFields
	/**
	 * The enumerated classification of the identifier. Optional as a trade identifier may be party-specific, in which case it may not correspond to any established classification.
	 */
	IdentifierType TradeIdentifierTypeEnum
}

/**
 * Specifies the price and quantity of a trade lot, where the same product could be traded multiple times with the same counterparty but in different lots (at a different date, in a different quantity and at a different price). One trade lot combined with a product definition specifies the entire economics of a trade. The lifecycle mechanics of each such trade lot (e.g. cashflow payments) is independent of the other lots.
 */
type TradeLot struct {
	/**
	 * Specifies one or more identifiers for the lot, if any.
	 */
	LotIdentifier []Identifier
	/**
	 * Specifies the settlement characteristics of a trade lot: price, quantity, observable (optionally) and the settlement terms. This attribute has a multiple cardinality to allow to specify the price, quantity and observable of different legs in a single, composite product (e.g. a Swap).
	 */
	PriceQuantity []PriceQuantity
}

/**
 * The attributes that are specific for consensus based pricing reporting.
 */
type TradePricingReport struct {
	/**
	 * Represents the cosensus based pricing parameters on a trade basis.
	 */
	Trade Trade
	/**
	 * The regional exchange close time for the underlying contract,including time zone, at which the trades should be priced. This provides an indication for which regional snapshot should be used for pricing primarily for Global markets where there are multiple regional close times.
	 */
	PricingTime TimeZone
	/**
	 * It specifies the interest payable on collateral delivered under a CSA covering the trade.
	 */
	DiscountingIndex FloatingRateIndexEnum
}

/**
 * Defines the fundamental financial information that can be changed by a Primitive Event and by extension any business or life-cycle event. Each TradeState specifies where a Trade is in its life-cycle. TradeState is a root type and as such, can be created independently to any other CDM data type, but can also be used as part of the CDM Event Model.
 */
type TradeState struct {
	/**
	 * Represents the Trade that has been effected by a business or life-cycle event.
	 */
	Trade Trade
	/**
	 * Represents the State of the Trade through its life-cycle.
	 */
	State State
	/**
	 * Represents the updated Trade attributes which can change as the result of a reset event. Only the changed values are captured, leaving the remaining data attributes empty. See Create_Reset function for further details on how TradeState is used in the Reset event. The TradeState data type is used to maintain backwards compatibility with the current Reset mechanism.
	 */
	ResetHistory []Reset
	/**
	 * Represents the updated Trade attributes which can change as the result of a transfer event.
	 */
	TransferHistory []TransferState
	/**
	 * Represents the observed events related to a particular product or process, such as credit events or corporate actions.
	 */
	ObservationHistory []ObservationEvent
	ValuationHistory   []Valuation
	Meta               MetaFields
}

/**
 * The class to represent a CDS Tranche.
 */
type Tranche struct {
	/**
	 * Lower bound percentage of the loss that the Tranche can endure, expressed as a decimal. An attachment point of 5% would be represented as 0.05. The difference between Attachment and Exhaustion points is called the width of the Tranche.
	 */
	AttachmentPoint float64
	/**
	 * Upper bound percentage of the loss that the Tranche can endure, expressed as a decimal. An exhaustion point of 5% would be represented as 0.05. The difference between Attachment and Exhaustion points is call the width of the Tranche.
	 */
	ExhaustionPoint float64
	/**
	 * Outstanding Swap Notional Amount is defined at any time on any day, as the greater of: (a) Zero; If Incurred Recovery Amount Applicable: (b) The Original Swap Notional Amount minus the sum of all Incurred Loss Amounts and all Incurred Recovery Amounts (if any) determined under this Confirmation at or prior to such time.Incurred Recovery Amount not populated: (b) The Original Swap Notional Amount minus the sum of all Incurred Loss Amounts determined under this Confirmation at or prior to such time.
	 */
	IncurredRecoveryApplicable bool
}

/**
 *  A class to represent the transacted price attributes that are positioned as part of the FpML FeeLeg.
 */
type TransactedPrice struct {
	/**
	 * An optional element that only has meaning in a credit index trade. This element contains the credit spread ('fair value') at which the trade was executed. Unlike the fixedRate of an index, the marketFixedRate varies over the life of the index depending on market conditions. The marketFixedRate is the price of the index as quoted by trading desks.
	 */
	MarketFixedRate float64
	/**
	 * An optional element that contains the up-front points expressed as a percentage of the notional. An initialPoints value of 5% would be represented as 0.05. The initialPoints element is an alternative to marketFixedRate in quoting the traded level of a trade. When initialPoints is used, the traded level is the sum of fixedRate and initialPoints. The initialPoints is one of the items that are factored into the initialPayment calculation and is payable by the Buyer to the Seller. Note that initialPoints and marketFixedRate may both be present in the same document when both implied values are desired.
	 */
	InitialPoints float64
	/**
	 * An optional element that only has meaning in a credit index trade. This element contains the price at which the trade was executed and is used instead of marketFixedRate on credit trades on certain indicies which are quoted using a price rather than a spread.
	 */
	MarketPrice float64
	/**
	 * An optional element that contains the up-front points expressed as a percentage of the notional. An initialPoints value of 5% would be represented as 0.05. The initialPoints element is an alternative to marketFixedRate in quoting the traded level of a trade. When initialPoints is used, the traded level is the sum of fixedRate and initialPoints. The initialPoints is one of the items that are factored into the initialPayment calculation and is payable by the Buyer to the Seller. Note that initialPoints and marketFixedRate may both be present in the same document when both implied values are desired.
	 */
	QuotationStyle QuotationStyleEnum
}

/**
 * Additional specification for the extraordinary events that may affect a trade and the related contractual rights and obligation of the parties when this happens. Such terms are typically required to extend the economics terms, for the purpose of producing the final legal contractual form of the Transaction.
 */
type TransactionAdditionalTerms struct {
	EquityAdditionalTerms          EquityAdditionalTerms
	ForeignExchangeAdditionalTerms FxAdditionalTerms
	CommoditiesAdditionalTerms     string
	CreditAdditionalTerms          string
	InterestRateAdditionalTerms    string
	DigitalAssetAdditionalTerms    string
}

/**
 * Defines the movement of an Asset (eg cash, securities or commodities) between two parties on a date.
 */
type Transfer struct {
	/**
	 * Represents the amount of the asset to be transferred. The cashflow amount is always a positive number, as the cashflow direction is implied by the payer/receiver attribute.
	 */
	Quantity NonNegativeQuantity
	/**
	 * Represents the object that is subject to the transfer, it could be an asset or a reference.
	 */
	Asset Asset
	/**
	 * Represents the date on which the transfer to due.
	 */
	SettlementDate AdjustableOrAdjustedOrRelativeDate
	/**
	 * Represents a unique reference to the transfer.
	 */
	Identifier []FieldWithMeta
	/**
	 * Represents the parties to the transfer and their role.
	 */
	PayerReceiver PartyReferencePayerReceiver
	/**
	 * Represents the origin to the transfer as a reference for lineage purposes, whether it originated from trade level settlement terms or from payment terms on an economic payout.
	 */
	SettlementOrigin ReferenceWithMeta
	/**
	 * Represents the reset and observation values that were used to determine the transfer amount.
	 */
	ResetOrigin Reset
	/**
	 * Specifies a transfer expression (cash price, performance amount, scheduled payment amount, etc.) to define the nature of the transfer amount and its source.
	 */
	TransferExpression TransferExpression
}

/**
 * Specifies transfer information including address and, optionally, an associated account. This type also supports the ISDA CSA representation as a single string, through the address attribute in Contact Information.
 */
type TransferContactInformation struct {
	/**
	 * The telephone number.
	 */
	Telephone []TelephoneNumber
	/**
	 * The street/postal address.
	 */
	Address []Address
	/**
	 * The email address.
	 */
	Email []string
	/**
	 * The web page. This attribute is not specified as part of the FpML ContactInformation complex type.
	 */
	WebPage []string
	/**
	 * Specifies the account infomation for transfers to a party.
	 */
	Account []Account
	/**
	 * Additional transfer information.
	 */
	AdditionalInformation string
}

/**
 * Specifies a transfer expression (cash price, performance amount, scheduled payment amount, etc.) to define the nature of the transfer amount and its source.
 */
type TransferExpression struct {
	/**
	 * Specifies a transfer amount exchanged as a price or fee for entering into a Business Event, e.g. Premium, Termination fee, Novation fee.
	 */
	PriceTransfer FeeTypeEnum
	/**
	 * Specifies a transfer created from a scheduled or contingent event on a contract, e.g. Exercise, Performance, Credit Event
	 */
	ScheduledTransfer ScheduledTransfer
}

/**
 * Specifies the party reference and the associated transfer & contact information.
 */
type TransferInformationElection struct {
	/**
	 * The party to which the election and details apply.
	 */
	PartyReference CounterpartyRoleEnum
	/**
	 * The main notice/address information of the party.
	 */
	PrimaryContactInformation ContactInformation
	/**
	 * The main notice/address and account information of the party.
	 */
}

/**
 * Defines the payout on which to create a Transfer along with all necessary resets.
 */
type TransferInstruction struct {
	/**
	 * Specifies the terms and state of a transfers.
	 */
	TransferState []TransferState
}

/**
 * Specification of transfer / settlement timing for cash and securities collateral assuming that the notice is provided before the Notification Time (and with a 1LBD lag introduced if the notice is delivered after the Notification Time), and that securities allow for a further Local Business Day for transfer / settlement.
 */
type TransferSettlementTiming struct {
	/**
	 * Details the day on which collateral is required to be transferred relative to the Notification Time.
	 */
	CashCollateralTransferSettlementTime CashCTSTimeEnum
	/**
	 * Details the day on which collateral is required to be transferred relative to the Notification Time.
	 */
	SecurititesCollateralTransferSettlementTime SecuritiesCTSTimeEnum
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * Defines the fundamental financial information associated with a Transfer event. Each TransferState specifies where a Transfer is in its life-cycle. TransferState is a root type and as such, can be created independently to any other CDM data type, but can also be used as part of the CDM Event Model.
 */
type TransferState struct {
	/**
	 * Represents the Transfer that has been effected by a business or life-cycle event.
	 */
	Transfer Transfer
	/**
	 * Represents the State of the Transfer through its life-cycle.
	 */
	TransferStatus TransferStatusEnum
	Meta           MetaFields
}

/**
 * A TransferableProduct is a type of financial product which can be held or transferred, represented as an Asset with the addition of specific EconomicTerms.
 */
type TransferableProduct struct {
	/**
	 * An Asset that consists solely of a monetary holding in a currency.
	 */
	Cash Cash
	/**
	 * An Asset comprised of raw or refined materials or agricultural products, eg gold, oil or wheat.
	 */
	Commodity Commodity
	/**
	 * An Asset that exists only in digital form, eg Bitcoin or Ethereum; excludes the digital representation of other Assets.
	 */
	DigitalAsset DigitalAsset
	/**
	 * An asset that is issued by one party to one or more others; Instrument is also a choice data type.
	 */
	Instrument Instrument
	/**
	 * The price forming features, including payouts and provisions.
	 */
	EconomicTerms EconomicTerms
}

/**
 * Trigger point at which feature is effective.
 */
type Trigger struct {
	/**
	 * The trigger level.
	 */
	Level                 []PriceSchedule
	CreditEvents          CreditEvents
	CreditEventsReference ReferenceWithMeta
	/**
	 * The Triggering condition.
	 */
	TriggerType TriggerTypeEnum
	/**
	 * The valuation time type of knock condition.
	 */
	TriggerTimeType TriggerTimeTypeEnum
}

/**
 * Observation point for trigger.
 */
type TriggerEvent struct {
	/**
	 * A derivative schedule.
	 */
	Schedule []AveragingSchedule
	/**
	 * The trigger Dates.
	 */
	TriggerDates DateList
	/**
	 * The trigger level
	 */
	Trigger Trigger
	/**
	 * The feature payment, i.e. the payment made following trigger occurrence.
	 */
	FeaturePayment FeaturePayment
}

type Tx struct {
	NewTx       New
	TradDt      string
	TradgCpcty  string
	Qty         Qty
	Pric        Pric
	TradVn      string
	CtryOfBrnch string
}

/**
 * Specifies the set of agreements, the parties of which are set within UmbrellaAgreementSet, beyond the two contracting parties to that agreement.
 */
type UmbrellaAgreement struct {
	/**
	 * The language associated with the umbrella agreement, and which applies to all the parties to the umbrella agreement.
	 */
	AgreementSet []UmbrellaAgreementSet
}

/**
 * Specifies the entities that are part of the umbrella agreement.
 */
type UmbrellaAgreementParty struct {
	/**
	 * Specifies the party to the umbrella agreement
	 */
	Party Party
	/**
	 * Specifies the CounterpartyRoleEnum, e.g. either Party1 or Party2, that is associated to the party.
	 */
	ActingAs CounterpartyRoleEnum
	/**
	 * Represents the legal role that each defined entity to the agreement performs within the structure of that agreement.
	 */
	PartyRole UmbrellaPartyRoleEnum
	/**
	 * Specifies a non-legal entity that exists in the agreement with a defined relationship to a Legal entity to the agreement.
	 */
	NonLegalEntity NonLegalEntity
	/**
	 * Represents the identification of a parent child relationship between two entities in the Umbrella agreement, allowing non-legal entities to be linked to the appropriate legal entity and allowing funds, portfolio or managed accounts to be linked to their applicable Investment Manager or Agent within the agreement.
	 */
	ParentParty Party
}

/**
 * Represents the groups of agreement term elections that exist within the agreement and to which entities that particular group of elections apply. Where all entities under the umbrella have the same set of elections only one agreement set will exist. At least one agreement set must be defined for an Umbrella agreement
 */
type UmbrellaAgreementSet struct {
	/**
	 * Specification of the standard set of terms that define a legal agreement.
	 */
	Agreement Agreement
	/**
	 * Specifies the underlying parties to the agreement that either act as legal parties to the agreement or represent legal constructs that are managed by a party to the agreement but do not represent a Legal Entity as of themselves.
	 */
	Party []UmbrellaAgreementParty
	/**
	 * Represents any additional language that may need to be captured regarding the structure or application of terms that may exist for the agreement.
	 */
	AdditionalLanguage string
}

/**
 * The underlying financial product that will be physically or cash settled, which can be of any type, eg an asset such as cash or a security, a product, or the cash settlement of an index rate.  Conditions are usually applied when used in a data type, such as a payout, to ensure this aligns with the use case.
 */
type Underlier struct {
	/**
	 * Specifies the object to be observed for a price, it could be an asset or a reference.
	 */
	Observable ReferenceWithMeta
	/**
	 * Enables either a TransferableProduct or a NonTransferableProduct to be used in an underlier.
	 */
	Product Product
}

/**
 * Where parties describe any substitution terms.
 */
type UnderlierSubstitutionProvision struct {
	/**
	 * Designates which Counterparty to the transaction who has the right to trigger a substitution or to provide related determination e.g. for instance to qualify the effectiveness of an Event which may be a trigger for substitution, determine the replacement Share to substitute, etc. ; cardinality of this object is 2, in case parties jointly have this role.
	 */
	WhoMaySubstitute []CounterpartyRoleEnum
	/**
	 * Where parties describe any substitution terms e.g. for instance the election criteria for an Asset to be eligible as the Substitute Asset to the prior Affected Asset in terms of sector of activity, currency, market capitalisation, liquidity, volatility, or any additional features that parties would agree to take into considerations, etc.
	 */
	SubstitutionBeSpokeTerms []Clause
	/**
	 * Where the parties may optionnally explictly specify the list of Events to be considered as a trigger for a Substitution.
	 */
	SubstitutionTriggerEvents []ExtraordinaryEvents
	/**
	 * Where the party who is not granted with the substitution role at least has a right to dispute the determination given by the counterparty with such role. As an example, a given PartyA is the unique Counterparty with the Role of WhoMaySubstitute, yet PartyB could be Disputing Party in regard of such Role.
	 */
	DisputingParty CounterpartyRoleEnum
}

type UndrlygInstrm struct {
	Swp Swp
}

/**
 * Defines the unit to be used for price, quantity, or other purposes
 */
type UnitType struct {
	/**
	 * Provides an enumerated value for a capacity unit, generally used in the context of defining quantities for commodities.
	 */
	CapacityUnit CapacityUnitEnum
	/**
	 * Provides an enumerated values for a weather unit, generally used in the context of defining quantities for commodities.
	 */
	WeatherUnit WeatherUnitEnum
	/**
	 * Provides an enumerated value for financial units, generally used in the context of defining quantities for securities.
	 */
	FinancialUnit FinancialUnitEnum
	/**
	 * Defines the currency to be used as a unit for a price, quantity, or other purpose.
	 */
	Currency FieldWithMeta
}

/**
 * Defines the value of an investment, asset, or security
 */
type Valuation struct {
	/**
	 * Current value of the outstanding contract
	 */
	Amount Money
	/**
	 * Date and time of the last valuation marked to market, provided by the central counterparty (CCP) or calculated using the current or last available market price of the inputs.
	 */
	Timestamp time.Time
	/**
	 * Method used for the valuation of the transaction by the valuation party.
	 */
	Method ValuationTypeEnum
	/**
	 * Source of the valuation of the transaction by the valuation party.
	 */
	Source ValuationSourceEnum
	/**
	 *  Defines the scope of the valuation, what it applies to e.g. the collateral or the trade
	 */
	Scope ValuationScopeEnum
	/**
	 * The ratio of the change in the price of a derivative transaction to the change in the price of the underlying. This field is applicable only to options and swaptions.
	 */
	Delta float64
	/**
	 * Denotes when the valuation was sourced during a business day.
	 */
	ValuationTiming PriceTimingEnum
	/**
	 * Denotes the price used to compute the valuation.
	 */
	PriceComponent Price
}

/**
 * Details of the party valuating the collateral to be delivered or returned.
 */
type ValuationAgent struct {
	/**
	 * Specification of which party calculating the value of collateral to be delivered or returned.
	 */
	Party CalculationValuationAgentPartyEnum
	/**
	 * Specifies how the Valuation Agent is determined.
	 */
	ValuationAgent ValuationAgentDeterminationEnum
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

/**
 * Specification of the Calculation Date Location for the respective parties to the legal agreement.
 */
type ValuationCalculationDateLocation struct {
	/**
	 * The parties' calculation date location election.
	 */
	PartyElection []ValuationCalculationDateLocationElection
}

/**
 * A class to specify each of the party elections with respect to the Calculation Date Location.
 */
type ValuationCalculationDateLocationElection struct {
	/**
	 * The elective party.
	 */
	Party CounterpartyRoleEnum
	/**
	 * The Calculation Date Location when specified as a business center which corresponds to the FpML list of business centers or can be mapped to it.
	 */
	BusinessCenter FieldWithMeta
	/**
	 * The Calculation Date Location when specified a location which doesn't correspond to the FpML list of business centers or cannot be mapped to it.
	 */
	CustomLocation string
}

/**
 * A single object that represents the different methods to specify a valuation date, as used for cash settlement. The Single / Multiple ValuationDate is used for the determination of recovery in a credit event, the RelativeDateOffset is used for cash-settled option, and FxFixingDate is used for cross-currency settlement.
 */
type ValuationDate struct {
	/**
	 * Where single valuation date is specified as being applicable for cash settlement, this element specifies the number of business days after satisfaction of all conditions to settlement when such valuation date occurs. ISDA 2003 Term: Single Valuation Date.
	 */
	SingleValuationDate SingleValuationDate
	/**
	 * Where multiple valuation dates are specified as being applicable for cash settlement, this element specifies (a) the number of applicable valuation dates, and (b) the number of business days after satisfaction of all conditions to settlement when the first such valuation date occurs, and (c) the number of business days thereafter of each successive valuation date. ISDA 2003 Term: Multiple Valuation Dates.
	 */
	MultipleValuationDates MultipleValuationDates
	/**
	 * The date on which the cash settlement amount will be determined according to the cash settlement method if the parties have not otherwise been able to agree the cash settlement amount. This attribute was formerly part of 'OptionCashSettlement', which is now being harmonised into a common 'CashSettlementTerms' that includes a 'ValuationDate'.
	 */
	ValuationDate RelativeDateOffset
	/**
	 * The date on which the currency rate will be determined for the purpose of specifying the amount in deliverable currency. This attribute was formerly part of 'NonDeliverableSettlement', which is now being harmonised into a common 'CashSettlementTerms' that includes a 'ValuationDate'.
	 */
	FxFixingDate FxFixingDate
	/**
	 * The date, when expressed as a schedule of date(s), on which the currency rate will be determined for the purpose of specifying the amount in deliverable currency. This attribute was formerly part of 'NonDeliverableSettlement', which is now being harmonised into a common 'CashSettlementTerms' that includes a 'ValuationDate'.
	 */
	FxFixingSchedule AdjustableDates
}

/**
 * Defines how and when a performance type option or performance type swap is to be valued, including initial, interim and final valuation dates.
 */
type ValuationDates struct {
	/**
	 * Specifies the initial valuation dates of the underlyer.
	 */
	InitialValuationDate PerformanceValuationDates
	/**
	 * Specifies the interim valuation dates of the underlyer.
	 */
	InterimValuationDate PerformanceValuationDates
	/**
	 * Specifies the final valuation dates of the underlyer.
	 */
	FinalValuationDate PerformanceValuationDates
}

/**
 * Specifies inputs needed to process a valuation.
 */
type ValuationInstruction struct {
	/**
	 * Contains all information related to a valuation.
	 */
	Valuation []Valuation
	/**
	 * Specifies whether the previous valuation tracks in the valuation history are removed (True) or kept (False).
	 */
	Replace bool
}

/**
 * Specifies the parameters required to obtain a valuation, including the source, quotation method (bid, mid etc.) and any applicable quotation amount.
 */
type ValuationMethod struct {
	/**
	 * The source for obtaining a valuation. This may come from some information source (e.g. Reuters), from a rate option fixing (e.g. FX fixing for cross-currency settlement), or from a set of reference banks. This is a mandatory attribute as the valuation method relies on one of those sources to be specified.
	 */
	ValuationSource ValuationSource
	/**
	 * The type of price quotations to be requested from dealers when determining the market value of the reference obligation for purposes of cash settlement, or which rate quote is to be observed for a fixing. For example, Bid, Offer, Mid-market or Exercising Party Pays. ISDA 2003 Term: Quotation Method. The meaning of Exercising Party Pays is defined in the 2000 ISDA Definitions, Section 17.2. Certain Definitions Relating to Cash Settlement, paragraph (j).
	 */
	QuotationMethod QuotationRateTypeEnum
	/**
	 * The ISDA defined methodology for determining the final price of the reference obligation for purposes of cash settlement. (ISDA 2003 Term: Valuation Method). For example, Market, Highest etc.
	 */
	ValuationMethod ValuationMethodEnum
	/**
	 * In the determination of a cash settlement amount, if weighted average quotations are to be obtained, the quotation amount specifies an upper limit to the outstanding principal balance of the reference obligation for which the quote should be obtained. If not specified, the ISDA definitions provide for a fallback amount equal to the floating rate payer calculation amount. ISDA 2003 Term: Quotation Amount.
	 */
	QuotationAmount Money
	/**
	 * In the determination of a cash settlement amount, if weighted average quotations are to be obtained, the minimum quotation amount specifies a minimum intended threshold amount of outstanding principal balance of the reference obligation for which the quote should be obtained. If not specified, the ISDA definitions provide for a fallback amount of the lower of either USD 1,000,000 (or its equivalent in the relevant obligation currency) or the quotation amount. ISDA 2003 Term: Minimum Quotation Amount.
	 */
	MinimumQuotationAmount Money
	/**
	 * Specifies the parameters representing several mid-market valuation and replacement value methods.
	 */
	CashCollateralValuationMethod CashCollateralValuationMethod
}

/**
 * Specifies how long to wait to get a quote from a settlement rate option upon a price source disruption.
 */
type ValuationPostponement struct {
	/**
	 * The maximum number of days to wait for a quote from the disrupted settlement rate option before proceeding to the next method.
	 */
	MaximumDaysOfPostponement int
}

/**
 * A class describing the method for obtaining a settlement rate, specified through either an information source (page), a settlement rate option (fixing) or by using quotes from reference banks.
 */
type ValuationSource struct {
	/**
	 * Defines the two currencies for an FX trade and the quotation relationship between the two currencies.  This attribute was formerly part of 'fxSettlementTerms', which is now being harmonised into a common 'CashSettlementTerms' that includes a 'ValuationDate'.
	 */
	QuotedCurrencyPair ReferenceWithMeta
	/**
	 * The information source where a published or displayed market rate will be obtained, e.g. Telerate Page 3750.
	 */
	InformationSource FxSpotRateSource
	/**
	 * The rate option to use for the fixing. Currently only applicable to foreign exchange fixing in case of cross-currency settlement.
	 */
	SettlementRateOption SettlementRateOption
	/**
	 * A container for a set of reference institutions that may be called upon to provide rate quotations as part of the method to determine the applicable cash settlement amount. If institutions are not specified, it is assumed that reference institutions will be agreed between the parties on the exercise date, or in the case of swap transaction to which mandatory early termination is applicable, the cash settlement valuation date.
	 */
	ReferenceBanks ReferenceBanks
	/**
	 * Holds an identifier for the reference entity that is agreed by both parties as a basis for cash settlement calculations. This could be a dealer from whom quotations are obtained by the calculation agent on the reference obligation for purposes of cash settlement in a credit event. ISDA 2003 Term: Dealer. This could be the clearing organization (CCP, DCO) to which the trade should be cleared, as applicable for cash-settled swaptions.
	 */
	DealerOrCCP AncillaryEntity
}

type ValuationTerms struct {
	/**
	 * The official settlement price as announced by the related exchange is applicable, in accordance with the ISDA 2002 definitions.
	 */
	FuturesPriceValuation bool
	/**
	 * The official settlement price as announced by the related exchange is applicable, in accordance with the ISDA 2002 definitions
	 */
	OptionsPriceValuation bool
	/**
	 * The number of valuation dates between valuation start date and valuation end date.
	 */
	NumberOfValuationDates int
	/**
	 * Specifies the dividend valuation dates of the swap.
	 */
	DividendValuationDates AdjustableRelativeOrPeriodicDates
	/**
	 * Specifies the fallback provisions for Hedging Party in the determination of the Final Price.
	 */
	FPVFinalPriceElectionFallback FPVFinalPriceElectionFallbackEnum
	/**
	 * For an index option transaction, a flag to indicate whether a relevant Multiple Exchange Index Annex is applicable to the transaction. This annex defines additional provisions which are applicable where an index is comprised of component securities that are traded on multiple exchanges.
	 */
	MultipleExchangeIndexAnnexFallback bool
	/**
	 * For an index option transaction, a flag to indicate whether a relevant Component Security Index Annex is applicable to the transaction.
	 */
	ComponentSecurityIndexAnnexFallback bool
}

/**
 * Specifies the time by which the value of the collateral and obligations to transfer or return collateral are to be calculated relative to the Valuation Date.
 */
type ValuationTime struct {
	/**
	 * The day in respect of which the Valuation Time is being specified. If not specified, addional language must be used to define the methodology.
	 */
	Day ValuationTimeDayEnum
	/**
	 * The location relevant to the timezone for the Valuation Time.
	 */
	Location []BusinessCenter
	/**
	 * The type of time specified for the Valuation Time.
	 */
	Time ValuationTimeEnum
	/**
	 * A time specified in a 24-hour notation, e.g. 3pm would be represented as 15:00.
	 */
	Hourminutetime time.Time
	/**
	 * Any relevant time zone where specified.
	 */
	Timezone BusinessCenter
	/**
	 * The additional language that might be specified by the parties to the legal agreement.
	 */
	AdditionalLanguage string
}

type VarianceCapFloor struct {
	/**
	 * If present and true, then variance cap is applicable.
	 */
	VarianceCap bool
	/**
	 * For use when varianceCap is applicable. Contains the scaling factor of the Variance Cap that can differ on a trade-by-trade basis in the European market. For example, a Variance Cap of 2.5^2 x Variance Strike Price has an unadjustedVarianceCap of 2.5.
	 */
	UnadjustedVarianceCap float64
	/**
	 * Conditions which bound variance. The contract specifies one or more boundary levels. These levels are expressed as prices for confirmation purposes Underlyer price must be equal to or higher than Lower Barrier is known as Up Conditional Swap Underlyer price must be equal to or lower than Upper Barrier is known as Down Conditional Swap Underlyer price must be equal to or higher than Lower Barrier and must be equal to or lower than Upper Barrier is known as Barrier Conditional Swap.
	 */
	BoundedVariance BoundedVariance
}

type VarianceReturnTerms struct {
	/**
	 * Contains all non-date valuation information.
	 */
	ValuationTerms ValuationTerms
	/**
	 * This specifies the numerator of an annualization factor. Frequently this number is equal to the number of observations of prices in a year e.g. 252.
	 */
	AnnualizationFactor int
	/**
	 * The parameters which define whether dividends are applicable
	 */
	DividendApplicability DividendApplicability
	/**
	 * Contains Equity Underlyer provisions regarding jurisdiction and fallbacks.
	 */
	EquityUnderlierProvisions EquityUnderlierProvisions
	/**
	 * Indicates whether the price of shares is adjusted for dividends or not.
	 */
	SharePriceDividendAdjustment bool
	/**
	 * Expected number of trading days.
	 */
	ExpectedN int
	/**
	 * Contract will strike off this initial level. Providing just the initialLevel without initialLevelSource, infers that this is AgreedInitialPrice - a specified Initial Index Level.
	 */
	InitialLevel float64
	/**
	 * In this context, this is AgreedInitialPrice - a specified Initial Index Level.
	 */
	InitialLevelSource DeterminationMethodEnum
	/**
	 * Specifies whether Mean Adjustment is applicable or not in the calculation of the Realized Volatility, Variance or Correlation
	 */
	MeanAdjustment bool
	/**
	 * Performance calculation, in accordance with Part 1 Section 12 of the 2018 ISDA CDM Equity Confirmation for Security Equity Swap, Para 75. 'Equity Performance'. Cumulative performance is used as a notional multiplier factor on both legs of an Equity Swap.
	 */
	Performance string
	/**
	 * Variance Strike Price in accordance with the ISDA 2011 Equity Derivatives Definitions.
	 */
	VarianceStrikePrice Price
	/**
	 * Volatility Strike Price in accordance with the ISDA 2011 Equity Derivatives Definitions.
	 */
	VolatilityStrikePrice Price
	/**
	 * Contains possible barriers for variance products, both variance-based and underlier price based
	 */
	VarianceCapFloor VarianceCapFloor
	/**
	 * Contains containing volatility-based barriers
	 */
	VolatilityCapFloor VolatilityCapFloor
	/**
	 * Vega Notional represents the approximate gain/loss at maturity for a 1% difference between RVol (realised vol) and KVol (strike vol). It does not necessarily represent the Vega Risk of the trade.
	 */
	VegaNotionalAmount NonNegativeQuantitySchedule
	/**
	 * Specification of the exchange traded contract nearest.
	 */
	ExchangeTradedContractNearest ReferenceWithMeta
}

type Velocity struct {
	PeriodMultiplier int
	Period           PeriodTimeEnum
}

/**
 * Contains volatility-based barriers. Volatility Cap needs to be specified in accordance with the ISDA 2011 Equity Derivatives Definitions.
 */
type VolatilityCapFloor struct {
	/**
	 * Indicates whether the volatility cap is applicable in accordance with the ISDA 2011 Equity Derivatives Definitions. Setting the element 'applicable' to 'False' - means No Volatility Cap and no 'totalVolatilityCap' or 'volatilityCapFactor' should be provided. Setting the element 'applicable' to 'True' - means Volatility Cap election, then 'totalVolatilityCap' or 'volatilityCapFactor' should be provided, otherwise it defaults to volatilityCapFactor=2.5.
	 */
	Applicable bool
	/**
	 * Volatility Cap Amount in accordance with the ISDA 2011 Equity Derivatives Definitions. This means the Volatility Cap Amount election is a number.
	 */
	TotalVolatilityCap float64
	/**
	 * Volatility Cap Amount in accordance with the ISDA 2011 Equity Derivatives Definitions. The Calculated VolCapAmt can be optionally provided.
	 */
	VolatilityCapFactor float64
}

type VolatilityReturnTerms struct {
	/**
	 * Contains all non-date valuation information.
	 */
	ValuationTerms ValuationTerms
	/**
	 * This specifies the numerator of an annualization factor. Frequently this number is equal to the number of observations of prices in a year e.g. 252.
	 */
	AnnualizationFactor int
	/**
	 * The parameters which define whether dividends are applicable
	 */
	DividendApplicability DividendApplicability
	/**
	 * Contains Equity Underlyer provisions regarding jurisdiction and fallbacks.
	 */
	EquityUnderlierProvisions EquityUnderlierProvisions
	/**
	 * Indicates whether the price of shares is adjusted for dividends or not.
	 */
	SharePriceDividendAdjustment bool
	/**
	 * Expected number of trading days.
	 */
	ExpectedN int
	/**
	 * Contract will strike off this initial level. Providing just the initialLevel without initialLevelSource, infers that this is AgreedInitialPrice - a specified Initial Index Level.
	 */
	InitialLevel float64
	/**
	 * In this context, this is AgreedInitialPrice - a specified Initial Index Level.
	 */
	InitialLevelSource DeterminationMethodEnum
	/**
	 * Specifies whether Mean Adjustment is applicable or not in the calculation of the Realized Volatility, Variance or Correlation
	 */
	MeanAdjustment bool
	/**
	 * Performance calculation, in accordance with Part 1 Section 12 of the 2018 ISDA CDM Equity Confirmation for Security Equity Swap, Para 75. 'Equity Performance'. Cumulative performance is used as a notional multiplier factor on both legs of an Equity Swap.
	 */
	Performance string
	/**
	 * Volatility Strike Price in accordance with the ISDA 2011 Equity Derivatives Definitions.
	 */
	VolatilityStrikePrice Price
	/**
	 * Contains volatility-based barriers
	 */
	VolatilityCapFloor VolatilityCapFloor
	/**
	 * Specification of the exchange traded contract nearest.
	 */
	ExchangeTradedContractNearest ListedDerivative
}

/**
 * A single weighted averaging observation.
 */
type WeightedAveragingObservation struct {
	/**
	 * Observation date time, which should be used when literal observation dates are required. The CDM specifies that the zoned date time is to be expressed in accordance with ISO 8601, either as UTC as an offset to UTC.
	 */
	DateTime time.Time
	/**
	 * Observation number, which should be unique, within a series generated by a date schedule.
	 */
	ObservationNumber int
	/**
	 * Observation weight, which is used as a multiplier for the observation value.
	 */
	Weight float64
}

/**
 * A collection of workflow steps which together make up an entire workflow sequence.
 */
type Workflow struct {
	Steps []WorkflowStep
}

/**
 * A class to specify workflow information, which is conceptually applicable to all lifecycle events.
 */
type WorkflowState struct {
	/**
	 * The workflow status indicator, e.g. Accepted, Rejected, ...
	 */
	WorkflowStatus WorkflowStatusEnum
	/**
	 * A comment field to be associated with the workflow, e.g. to specify why a transaction event was rejected by a party.
	 */
	Comment string
	/**
	 * Workflow data that is specific to certain market participants and is expressed as part of the CDM in a very generic manner, which can be party-specific. The initial use cases have been derived from the CME clearing and the DTCC TIW submissions.
	 */
	PartyCustomisedWorkflow []PartyCustomisedWorkflow
	/**
	 * The identity of the warehouse, if any, that is executing that workflow step.
	 */
	WarehouseIdentity WarehouseIdentityEnum
}

/**
 * A workflow step represents the state of a business event. The workflow step contains a reference to a previous WorkflowStep in order to preserve lineage. A workflow step is accepted if it contains a business event, proposed if proposedEvent is present and is rejected if the rejected flag is set.
 */
type WorkflowStep struct {
	/**
	 * Life cycle event for the step.
	 */
	BusinessEvent BusinessEvent
	/**
	 * Documents the life cycle event for a position.
	 */
	CounterpartyPositionBusinessEvent CounterpartyPositionBusinessEvent
	/**
	 * The proposed event for a workflow step.
	 */
	ProposedEvent EventInstruction
	/**
	 * Flags this step as rejected.
	 */
	Rejected bool
	/**
	 * Optional party approvals for the current workflow step. A workflow step can have any number of parties associated to it, thus this object is represented as a list. All parties that are expected to provide approval should have an item in this list that references them.
	 */
	Approval []WorkflowStepApproval
	/**
	 * Optional previous workflow step that provides lineage to workflow steps that precedes it.
	 */
	PreviousWorkflowStep ReferenceWithMeta
	/**
	 * The intended next event can be specified, even if the instructions are not known yet.
	 */
	NextEvent EventInstruction
	/**
	 * Contains all information pertaining the FpML messaging header
	 */
	MessageInformation MessageInformation
	/**
	 * The set of timestamp(s) associated with the event as a collection of [dateTime, qualifier].
	 */
	Timestamp []EventTimestamp
	/**
	 * The identifier(s) that uniquely identify a lifecycle event. The unbounded cardinality is meant to provide the ability to associate identifiers that are issued by distinct parties. As an example, each of the parties to the event may choose to associate their own identifiers to the event.
	 */
	EventIdentifier []Identifier
	/**
	 * Specifies whether the event is a new, a correction or a cancellation.
	 */
	Action ActionEnum
	/**
	 * The specification of the event parties. This attribute is optional, as not applicable to certain events (e.g. most of the observations).
	 */
	Party []Party
	/**
	 * Optional account information that could be associated to the event.
	 */
	Account []Account
	/**
	 * The lineage attribute provides a linkage among lifecycle events through the globalKey hash value. One example is when a given lifecycle event is being corrected or cancelled. In such case, each subsequent event will have lineage into the prior version of that event. The second broad use case is when an event has a dependency upon either another event (e.g. the regular payment associated with a fix/float swap will have a lineage into the reset event, which will in turn have a lineage into the observation event for the floating rate and the contract) or a contract (e.g. the exercise of an option has a lineage into that option).
	 */
	Lineage                Lineage
	CreditLimitInformation CreditLimitInformation
	/**
	 * The event workflow information, i.e. the workflow status, the associated comment and the partyCustomisedWorkflow which purpose is to provide the ability to associate custom workflow information to the CDM.
	 */
	WorkflowState WorkflowState
	Meta          MetaFields
}

/**
 * Party approvals associated to the current WorkflowStep.
 */
type WorkflowStepApproval struct {
	/**
	 * Flag denoting whether the workflow step is approved or not
	 */
	Approved bool
	/**
	 * Reference to the Party who is approving/rejecting this workflow step
	 */
	Party ReferenceWithMeta
	/**
	 * Optional reason for rejecting the workflow step
	 */
	RejectedReason string
	/**
	 * Timestamp of the approval
	 */
	Timestamp EventTimestamp
	Meta      MetaFields
}
