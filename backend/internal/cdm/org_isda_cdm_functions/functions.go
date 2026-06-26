
/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm_functions  

import "time"
import cdm "github.com/hondyman/semlayer/backend/internal/cdm/org_isda_cdm"    

//Pointer type args used when the latter are optional
func Abs(arg float64 ) float64 {    
/**
 * Function definition for Abs
 */
return 0
}

func AddBusinessDays(originalDate time.Time, offsetBusinessDays int, businessCenters *cdm.BusinessCenter ) time.Time {    
/**
 * Function definition for AddBusinessDays
 */
return time.Time{}
}

func AddDays(inputDate time.Time, numDays int ) time.Time {    
/**
 * Function definition for AddDays
 */
return time.Time{}
}

func AddTradeLot(tradableProduct cdm.TradableProduct, newTradeLot cdm.TradeLot ) cdm.TradableProduct {    
/**
 * Function definition for AddTradeLot
 */
return cdm.TradableProduct{}
}

func AdjustableDateResolution(adjustableDate cdm.AdjustableDate ) *time.Time {    
/**
 * Function definition for AdjustableDateResolution
 */
return &time.Time{}
}

func AdjustableDatesResolution(adjustableDates cdm.AdjustableDates ) *time.Time {    
/**
 * Function definition for AdjustableDatesResolution
 */
return &time.Time{}
}

func AdjustableOrAdjustedOrRelativeDateResolution(adjustableDate cdm.AdjustableOrAdjustedOrRelativeDate ) *time.Time {    
/**
 * Function definition for AdjustableOrAdjustedOrRelativeDateResolution
 */
return &time.Time{}
}

func AdjustedValuationDates(valuationDates cdm.ValuationDates ) *time.Time {    
/**
 * Function definition for AdjustedValuationDates
 */
return &time.Time{}
}

func AppendDateToList(origDates *time.Time, newDate time.Time ) *time.Time {    
/**
 * Function definition for AppendDateToList
 */
return &time.Time{}
}

func AppendToVector(vector *float64, value float64 ) *float64 {    
/**
 * Function definition for AppendToVector
 */
return nil
}

func ApplyAveragingFormula(observations *float64, weights *float64 ) cdm.CalculatedRateDetails {    
/**
 * Function definition for ApplyAveragingFormula
 */
return cdm.CalculatedRateDetails{}
}

func ApplyCapsAndFloors(processing cdm.FloatingRateProcessingParameters, inputRate float64 ) float64 {    
/**
 * Function definition for ApplyCapsAndFloors
 */
return 0
}

func ApplyCompoundingFormula(observations *float64, weights *float64, yearFrac float64 ) cdm.CalculatedRateDetails {    
/**
 * Function definition for ApplyCompoundingFormula
 */
return cdm.CalculatedRateDetails{}
}

func ApplyFinalRateRounding(baseRate float64, finalRateRounding *cdm.Rounding ) float64 {    
/**
 * Function definition for ApplyFinalRateRounding
 */
return 0
}

func ApplyFloatingRatePostSpreadProcessing(inputRate float64, processing cdm.FloatingRateProcessingParameters ) float64 {    
/**
 * Function definition for ApplyFloatingRatePostSpreadProcessing
 */
return 0
}

func ApplyFloatingRateProcessing(processing cdm.FloatingRateProcessingParameters, rawRate float64, calculationPeriod cdm.CalculationPeriodBase, isInitialPeriod bool ) cdm.FloatingRateProcessingDetails {    
/**
 * Function definition for ApplyFloatingRateProcessing
 */
return cdm.FloatingRateProcessingDetails{}
}

func ApplyFloatingRateSetting(interestRatePayout cdm.InterestRatePayout, calculationPeriod cdm.CalculationPeriodBase, isInitialPeriod bool, suppliedNotional *float64, suppliedRate *float64, floatingRateSetting *cdm.FloatingRateSettingDetails ) cdm.FloatingAmountCalculationDetails {    
/**
 * Function definition for ApplyFloatingRateSetting
 */
return cdm.FloatingAmountCalculationDetails{}
}

func ApplyUSRateTreatment(baseRate float64, rateTreatment cdm.RateTreatmentEnum, calculationPeriod cdm.CalculationPeriodBase ) float64 {    
/**
 * Function definition for ApplyUSRateTreatment
 */
return 0
}

func ArithmeticOperation(n1 float64, op cdm.ArithmeticOperationEnum, n2 float64 ) float64 {    
/**
 * Function definition for ArithmeticOperation
 */
return 0
}

func AssetIdentifierByType(identifiers *cdm.AssetIdentifier, idType cdm.AssetIdTypeEnum ) *cdm.AssetIdentifier {    
/**
 * Function definition for AssetIdentifierByType
 */
return &cdm.AssetIdentifier{}
}

func AuxiliarEffectiveDate(trade cdm.Trade ) *time.Time {    
/**
 * Function definition for AuxiliarEffectiveDate
 */
return &time.Time{}
}

func AuxiliarTerminationDate(trade cdm.Trade ) *time.Time {    
/**
 * Function definition for AuxiliarTerminationDate
 */
return &time.Time{}
}

func BuildStandardizedSchedule(trade cdm.Trade ) cdm.StandardizedSchedule {    
/**
 * Function definition for BuildStandardizedSchedule
 */
return cdm.StandardizedSchedule{}
}

func BusinessCenterHolidays(businessCenter cdm.BusinessCenter ) *time.Time {    
/**
 * Function definition for BusinessCenterHolidays
 */
return &time.Time{}
}

func BusinessCenterHolidaysMultiple(businessCenters *cdm.BusinessCenter ) *time.Time {    
/**
 * Function definition for BusinessCenterHolidaysMultiple
 */
return &time.Time{}
}

func CalculateFloatingCashFlow(interestRatePayout cdm.InterestRatePayout, calculationPeriod cdm.CalculationPeriodBase, notional *float64, currency *string, floatingRateSetting *cdm.FloatingRateSettingDetails, processedRateDetails cdm.FloatingRateProcessingDetails ) cdm.FloatingAmountCalculationDetails {    
/**
 * Function definition for CalculateFloatingCashFlow
 */
return cdm.FloatingAmountCalculationDetails{}
}

func CalculateTransfer(instruction cdm.CalculateTransferInstruction ) *cdm.Transfer {    
/**
 * Function definition for CalculateTransfer
 */
return &cdm.Transfer{}
}

func CalculateYearFraction(interestRatePayout cdm.InterestRatePayout, dcf cdm.DayCountFractionEnum, calculationPeriod cdm.CalculationPeriodBase ) float64 {    
/**
 * Function definition for CalculateYearFraction
 */
return 0
}

func CalculationPeriod(calculationPeriodDates cdm.CalculationPeriodDates, date time.Time ) cdm.CalculationPeriodData {    
/**
 * Function definition for CalculationPeriod
 */
return cdm.CalculationPeriodData{}
}

func CalculationPeriodRange(startDate *time.Time, endDate *time.Time, dateAdjustments *cdm.BusinessDayAdjustments ) cdm.CalculationPeriodData {    
/**
 * Function definition for CalculationPeriodRange
 */
return cdm.CalculationPeriodData{}
}

func CalculationPeriods(calculationPeriodDates cdm.CalculationPeriodDates ) *cdm.CalculationPeriodData {    
/**
 * Function definition for CalculationPeriods
 */
return &cdm.CalculationPeriodData{}
}

func CapRateAmount(interestRatePayout cdm.InterestRatePayout, calculationPeriod cdm.CalculationPeriodBase ) *float64 {    
/**
 * Function definition for CapRateAmount
 */
return nil
}

func CashPriceQuantityNoOfUnitsTriangulation(quantity *cdm.NonNegativeQuantitySchedule, price *cdm.PriceSchedule ) bool {    
/**
 * Function definition for CashPriceQuantityNoOfUnitsTriangulation
 */
return false
}

func CheckAgencyRating(agencyRatings *cdm.AgencyRatingCriteria, query cdm.EligibilityQuery ) bool {    
/**
 * Function definition for CheckAgencyRating
 */
return false
}

func CheckAssetType(collateralAssetTypes *cdm.AssetType, query cdm.EligibilityQuery ) bool {    
/**
 * Function definition for CheckAssetType
 */
return false
}

func CheckCountryOfOrigin(countryOfOrigin *cdm.ISOCountryCodeEnum, query cdm.EligibilityQuery ) bool {    
/**
 * Function definition for CheckCountryOfOrigin
 */
return false
}

func CheckCriteria(inputCriteria cdm.CollateralCriteria, query cdm.EligibilityQuery ) bool {    
/**
 * Function definition for CheckCriteria
 */
return false
}

func CheckDenominatedCurrency(denominatedCurrency *cdm.CurrencyCodeEnum, query cdm.EligibilityQuery ) bool {    
/**
 * Function definition for CheckDenominatedCurrency
 */
return false
}

func CheckEligibilityByDetails(specification cdm.EligibleCollateralSpecification, query cdm.EligibilityQuery ) cdm.CheckEligibilityResult {    
/**
 * Function definition for CheckEligibilityByDetails
 */
return cdm.CheckEligibilityResult{}
}

func CheckEligibilityForProduct(specifications cdm.EligibleCollateralSpecification, product *cdm.TransferableProduct ) *cdm.CheckEligibilityResult {    
/**
 * Function definition for CheckEligibilityForProduct
 */
return &cdm.CheckEligibilityResult{}
}

func CheckIssuerName(issuerName *cdm.IssuerName, query cdm.EligibilityQuery ) bool {    
/**
 * Function definition for CheckIssuerName
 */
return false
}

func CheckIssuerType(issuerType *cdm.CollateralIssuerType, query cdm.EligibilityQuery ) bool {    
/**
 * Function definition for CheckIssuerType
 */
return false
}

func CheckMaturity(maturityRange *cdm.AssetMaturity, query cdm.EligibilityQuery ) bool {    
/**
 * Function definition for CheckMaturity
 */
return false
}

func CloneEligibleCollateralWithChangedTreatment(inputSpecification cdm.EligibleCollateralSpecification, changedCriteria cdm.CollateralCriteria, changedTreatment cdm.CollateralTreatment ) cdm.EligibleCollateralSpecification {    
/**
 * Function definition for CloneEligibleCollateralWithChangedTreatment
 */
return cdm.EligibleCollateralSpecification{}
}

func CommodityPayoutOnlyExists(payouts *cdm.Payout ) bool {    
/**
 * Function definition for CommodityPayoutOnlyExists
 */
return false
}

func CompareNumbers(n1 float64, op cdm.CompareOp, n2 float64 ) bool {    
/**
 * Function definition for CompareNumbers
 */
return false
}

func CompareQuantityByUnitOfAmount(quantity1 *cdm.Quantity, op cdm.CompareOp, quantity2 *cdm.Quantity, unitOfAmount cdm.UnitType ) bool {    
/**
 * Function definition for CompareQuantityByUnitOfAmount
 */
return false
}

func CompareTradeLot(tradeLot1 cdm.TradeLot, op cdm.CompareOp, tradeLot2 cdm.TradeLot ) bool {    
/**
 * Function definition for CompareTradeLot
 */
return false
}

func CompareTradeLotToAmount(tradeLot cdm.TradeLot, op cdm.CompareOp, amount float64 ) bool {    
/**
 * Function definition for CompareTradeLotToAmount
 */
return false
}

func CompareTradeStatesToAmount(tradeStates *cdm.TradeState, op cdm.CompareOp, amount float64 ) bool {    
/**
 * Function definition for CompareTradeStatesToAmount
 */
return false
}

func ComputeCalculationPeriod(calculationPeriod cdm.CalculationPeriodBase, priorCalculationPeriod *cdm.CalculationPeriodBase, calculateRelativeTo *cdm.ObservationPeriodDatesEnum, resetDates *cdm.ResetDates ) cdm.CalculationPeriodBase {    
/**
 * Function definition for ComputeCalculationPeriod
 */
return cdm.CalculationPeriodBase{}
}

func ConvertToAdjustableOrAdjustedOrRelativeDate(adjustableOrRelativeDate *cdm.AdjustableOrRelativeDate ) *cdm.AdjustableOrAdjustedOrRelativeDate {    
/**
 * Function definition for ConvertToAdjustableOrAdjustedOrRelativeDate
 */
return &cdm.AdjustableOrAdjustedOrRelativeDate{}
}

func ConvertToAdjustableOrRelativeDate(adjustableOrAdjustedOrRelativeDate *cdm.AdjustableOrAdjustedOrRelativeDate ) *cdm.AdjustableOrRelativeDate {    
/**
 * Function definition for ConvertToAdjustableOrRelativeDate
 */
return &cdm.AdjustableOrRelativeDate{}
}

func CreateAndCriteria(inputCriteria cdm.CollateralCriteria ) cdm.CollateralCriteria {    
/**
 * Function definition for CreateAndCriteria
 */
return cdm.CollateralCriteria{}
}

func CreateOrCriteria(inputCriteria cdm.CollateralCriteria ) cdm.CollateralCriteria {    
/**
 * Function definition for CreateOrCriteria
 */
return cdm.CollateralCriteria{}
}

func Create_AcceptedWorkflowStep(messageInformation *cdm.MessageInformation, timestamp cdm.EventTimestamp, eventIdentifier cdm.Identifier, party *cdm.Party, account *cdm.Account, proposedWorkflowStep cdm.WorkflowStep, businessEvent cdm.BusinessEvent ) cdm.WorkflowStep {    
/**
 * Function definition for Create_AcceptedWorkflowStep
 */
return cdm.WorkflowStep{}
}

func Create_AcceptedWorkflowStepFromInstruction(proposedWorkflowStep cdm.WorkflowStep ) cdm.WorkflowStep {    
/**
 * Function definition for Create_AcceptedWorkflowStepFromInstruction
 */
return cdm.WorkflowStep{}
}

func Create_AdjustmentPrimitiveInstruction(tradeState cdm.TradeState, newAllinPrice float64, newAssetQuantity float64, effectiveRepriceDate cdm.AdjustableOrRelativeDate ) cdm.PrimitiveInstruction {    
/**
 * Function definition for Create_AdjustmentPrimitiveInstruction
 */
return cdm.PrimitiveInstruction{}
}

func Create_AssetPayoutTradeStateWithObservations(billingInstruction cdm.BillingRecordInstruction ) cdm.TradeState {    
/**
 * Function definition for Create_AssetPayoutTradeStateWithObservations
 */
return cdm.TradeState{}
}

func Create_AssetReset(assetPayout cdm.AssetPayout, observation cdm.Observation, resetDate time.Time ) cdm.Reset {    
/**
 * Function definition for Create_AssetReset
 */
return cdm.Reset{}
}

func Create_AssetTransfer(instruction cdm.CalculateTransferInstruction ) cdm.Transfer {    
/**
 * Function definition for Create_AssetTransfer
 */
return cdm.Transfer{}
}

func Create_BillingRecord(billingInstruction cdm.BillingRecordInstruction ) cdm.BillingRecord {    
/**
 * Function definition for Create_BillingRecord
 */
return cdm.BillingRecord{}
}

func Create_BillingRecords(billingInstruction cdm.BillingRecordInstruction ) cdm.BillingRecord {    
/**
 * Function definition for Create_BillingRecords
 */
return cdm.BillingRecord{}
}

func Create_BillingSummary(billingRecord cdm.BillingRecord ) cdm.BillingSummary {    
/**
 * Function definition for Create_BillingSummary
 */
return cdm.BillingSummary{}
}

func Create_BusinessEvent(instruction cdm.Instruction, intent *cdm.EventIntentEnum, eventDate time.Time, effectiveDate time.Time ) cdm.BusinessEvent {    
/**
 * Function definition for Create_BusinessEvent
 */
return cdm.BusinessEvent{}
}

func Create_CalculationPeriodBase(calcPeriodData cdm.CalculationPeriodData ) cdm.CalculationPeriodBase {    
/**
 * Function definition for Create_CalculationPeriodBase
 */
return cdm.CalculationPeriodBase{}
}

func Create_CancellationPrimitiveInstruction(tradeState cdm.TradeState, newRepurchasePrice *float64, cancellationDate cdm.AdjustableOrRelativeDate ) cdm.PrimitiveInstruction {    
/**
 * Function definition for Create_CancellationPrimitiveInstruction
 */
return cdm.PrimitiveInstruction{}
}

func Create_CancellationTermChangeInstruction(product cdm.NonTransferableProduct, cancellationDate cdm.AdjustableOrRelativeDate ) cdm.TermsChangeInstruction {    
/**
 * Function definition for Create_CancellationTermChangeInstruction
 */
return cdm.TermsChangeInstruction{}
}

func Create_CashTransfer(instruction cdm.CalculateTransferInstruction ) cdm.Transfer {    
/**
 * Function definition for Create_CashTransfer
 */
return cdm.Transfer{}
}

func Create_CashflowFromSettlementPayout(payout cdm.SettlementPayout ) []cdm.Cashflow {    
/**
 * Function definition for Create_CashflowFromSettlementPayout
 */
return []cdm.Cashflow{}
}

func Create_ContractFormation(instruction cdm.ContractFormationInstruction, execution cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_ContractFormation
 */
return cdm.TradeState{}
}

func Create_ContractFormationInstruction(legalAgreement *cdm.LegalAgreement ) cdm.ContractFormationInstruction {    
/**
 * Function definition for Create_ContractFormationInstruction
 */
return cdm.ContractFormationInstruction{}
}

func Create_EffectiveOrTerminationDateTermChangeInstruction(product cdm.NonTransferableProduct, effectiveRollDate *cdm.AdjustableOrRelativeDate, terminationDate *cdm.AdjustableOrRelativeDate ) cdm.TermsChangeInstruction {    
/**
 * Function definition for Create_EffectiveOrTerminationDateTermChangeInstruction
 */
return cdm.TermsChangeInstruction{}
}

func Create_Execution(instruction cdm.ExecutionInstruction ) cdm.TradeState {    
/**
 * Function definition for Create_Execution
 */
return cdm.TradeState{}
}

func Create_Exercise(exerciseInstruction cdm.ExerciseInstruction, originalTrade cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_Exercise
 */
return cdm.TradeState{}
}

func Create_ExposureFromTrades(trades *cdm.TradeState ) *cdm.Exposure {    
/**
 * Function definition for Create_ExposureFromTrades
 */
return &cdm.Exposure{}
}

func Create_IndexTransitionTermsChange(instruction cdm.IndexTransitionInstruction, tradeState cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_IndexTransitionTermsChange
 */
return cdm.TradeState{}
}

func Create_NonTransferableProduct(underlier cdm.Underlier, payerReceiver cdm.PayerReceiver ) cdm.NonTransferableProduct {    
/**
 * Function definition for Create_NonTransferableProduct
 */
return cdm.NonTransferableProduct{}
}

func Create_Observation(instruction cdm.ObservationInstruction, before cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_Observation
 */
return cdm.TradeState{}
}

func Create_OnDemandInterestPaymentPrimitiveInstruction(tradeState cdm.TradeState, interestAmount cdm.Money, settlementDate cdm.SettlementDate ) cdm.PrimitiveInstruction {    
/**
 * Function definition for Create_OnDemandInterestPaymentPrimitiveInstruction
 */
return cdm.PrimitiveInstruction{}
}

func Create_OnDemandRateChangePriceChangeInstruction(priceQuantity cdm.PriceQuantity, newRate float64 ) cdm.QuantityChangeInstruction {    
/**
 * Function definition for Create_OnDemandRateChangePriceChangeInstruction
 */
return cdm.QuantityChangeInstruction{}
}

func Create_OnDemandRateChangePrimitiveInstruction(tradeState cdm.TradeState, effectiveDate cdm.AdjustableOrRelativeDate, agreedRate float64 ) cdm.PrimitiveInstruction {    
/**
 * Function definition for Create_OnDemandRateChangePrimitiveInstruction
 */
return cdm.PrimitiveInstruction{}
}

func Create_OnDemandRateChangeTermsChangeInstruction(product cdm.NonTransferableProduct, effectiveDate cdm.AdjustableOrRelativeDate ) cdm.TermsChangeInstruction {    
/**
 * Function definition for Create_OnDemandRateChangeTermsChangeInstruction
 */
return cdm.TermsChangeInstruction{}
}

func Create_PackageExecutionDetails(executionDetails *cdm.ExecutionDetails, listId cdm.Identifier, componentId cdm.Identifier ) cdm.ExecutionDetails {    
/**
 * Function definition for Create_PackageExecutionDetails
 */
return cdm.ExecutionDetails{}
}

func Create_PairOffInstruction(tradeState cdm.TradeState, pairReference cdm.Identifier ) cdm.Instruction {    
/**
 * Function definition for Create_PairOffInstruction
 */
return cdm.Instruction{}
}

func Create_PartialDeliveryPrimitiveInstruction(tradeState cdm.TradeState, deliveredPriceQuantity cdm.PriceQuantity ) cdm.PrimitiveInstruction {    
/**
 * Function definition for Create_PartialDeliveryPrimitiveInstruction
 */
return cdm.PrimitiveInstruction{}
}

func Create_PartyChange(counterparty cdm.Counterparty, ancillaryParty *cdm.AncillaryParty, partyRole *cdm.PartyRole, tradeId cdm.TradeIdentifier, originalTrade cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_PartyChange
 */
return cdm.TradeState{}
}

func Create_ProposedWorkflowStep(messageInformation *cdm.MessageInformation, timestamp cdm.EventTimestamp, eventIdentifier cdm.Identifier, party *cdm.Party, account *cdm.Account, previousWorkflowStep *cdm.WorkflowStep, action cdm.ActionEnum, proposedEvent cdm.EventInstruction, approval *cdm.WorkflowStepApproval ) cdm.WorkflowStep {    
/**
 * Function definition for Create_ProposedWorkflowStep
 */
return cdm.WorkflowStep{}
}

func Create_QuantityChange(instruction cdm.QuantityChangeInstruction, tradeState cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_QuantityChange
 */
return cdm.TradeState{}
}

func Create_RejectedWorkflowStep(messageInformation *cdm.MessageInformation, timestamp cdm.EventTimestamp, eventIdentifier cdm.Identifier, proposedWorkflowStep cdm.WorkflowStep ) cdm.WorkflowStep {    
/**
 * Function definition for Create_RejectedWorkflowStep
 */
return cdm.WorkflowStep{}
}

func Create_RepricePrimitiveInstruction(tradeState cdm.TradeState, newAllinPrice float64, newCashValue float64, effectiveRepriceDate cdm.AdjustableOrRelativeDate ) cdm.PrimitiveInstruction {    
/**
 * Function definition for Create_RepricePrimitiveInstruction
 */
return cdm.PrimitiveInstruction{}
}

func Create_Reset(instruction cdm.ResetInstruction, tradeState cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_Reset
 */
return cdm.TradeState{}
}

func Create_Return(tradeState cdm.TradeState, returnInstruction cdm.ReturnInstruction, returnDate time.Time ) cdm.BusinessEvent {    
/**
 * Function definition for Create_Return
 */
return cdm.BusinessEvent{}
}

func Create_RollPrimitiveInstruction(tradeState cdm.TradeState, effectiveRollDate cdm.AdjustableOrRelativeDate, terminationDate cdm.AdjustableOrRelativeDate, priceQuantity cdm.PriceQuantity ) cdm.PrimitiveInstruction {    
/**
 * Function definition for Create_RollPrimitiveInstruction
 */
return cdm.PrimitiveInstruction{}
}

func Create_RollTermChangeInstruction(product cdm.NonTransferableProduct, effectiveRollDate cdm.AdjustableOrRelativeDate, terminationDate cdm.AdjustableOrRelativeDate ) cdm.TermsChangeInstruction {    
/**
 * Function definition for Create_RollTermChangeInstruction
 */
return cdm.TermsChangeInstruction{}
}

func Create_SecurityLendingInvoice(instruction cdm.BillingInstruction ) cdm.SecurityLendingInvoice {    
/**
 * Function definition for Create_SecurityLendingInvoice
 */
return cdm.SecurityLendingInvoice{}
}

func Create_ShapingInstruction(tradeState cdm.TradeState, tradeLots cdm.TradeLot, shapeIdentifier cdm.Identifier ) cdm.PrimitiveInstruction {    
/**
 * Function definition for Create_ShapingInstruction
 */
return cdm.PrimitiveInstruction{}
}

func Create_Split(breakdown cdm.PrimitiveInstruction, originalTrade cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_Split
 */
return cdm.TradeState{}
}

func Create_StockSplit(stockSplitInstruction cdm.StockSplitInstruction, before cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_StockSplit
 */
return cdm.TradeState{}
}

func Create_SubstitutionInstruction(product cdm.NonTransferableProduct, effectiveDate cdm.AdjustableOrRelativeDate, newCollateralPortfolio cdm.CollateralPortfolio ) cdm.TermsChangeInstruction {    
/**
 * Function definition for Create_SubstitutionInstruction
 */
return cdm.TermsChangeInstruction{}
}

func Create_SubstitutionPrimitiveInstruction(tradeState cdm.TradeState, effectiveDate cdm.AdjustableOrRelativeDate, newCollateralPortfolio cdm.CollateralPortfolio, priceQuantity cdm.PriceQuantity ) cdm.PrimitiveInstruction {    
/**
 * Function definition for Create_SubstitutionPrimitiveInstruction
 */
return cdm.PrimitiveInstruction{}
}

func Create_TerminationInstruction(tradeState cdm.TradeState ) cdm.PrimitiveInstruction {    
/**
 * Function definition for Create_TerminationInstruction
 */
return cdm.PrimitiveInstruction{}
}

func Create_TermsChange(termsChange cdm.TermsChangeInstruction, before cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_TermsChange
 */
return cdm.TradeState{}
}

func Create_TradeState(primitiveInstruction *cdm.PrimitiveInstruction, before *cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_TradeState
 */
return cdm.TradeState{}
}

func Create_Transfer(instruction cdm.TransferInstruction, tradeState cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_Transfer
 */
return cdm.TradeState{}
}

func Create_Valuation(instruction cdm.ValuationInstruction, before cdm.TradeState ) cdm.TradeState {    
/**
 * Function definition for Create_Valuation
 */
return cdm.TradeState{}
}

func Create_Workflow(steps cdm.WorkflowStep ) cdm.Workflow {    
/**
 * Function definition for Create_Workflow
 */
return cdm.Workflow{}
}

func Create_WorkflowStep(messageInformation *cdm.MessageInformation, timestamp cdm.EventTimestamp, eventIdentifier cdm.Identifier, party *cdm.Party, account *cdm.Account, previousWorkflowStep *cdm.WorkflowStep, action cdm.ActionEnum, businessEvent *cdm.BusinessEvent ) cdm.WorkflowStep {    
/**
 * Function definition for Create_WorkflowStep
 */
return cdm.WorkflowStep{}
}

func CreditSupportAmount(marginAmount cdm.Money, threshold cdm.Money, marginApproach cdm.MarginApproachEnum, marginAmountIA *cdm.Money, baseCurrency string ) cdm.Money {    
/**
 * Function definition for CreditSupportAmount
 */
return cdm.Money{}
}

func CriteriaMatchesAssetType(inputCriteria *cdm.CollateralCriteria, assetType *cdm.InstrumentTypeEnum ) bool {    
/**
 * Function definition for CriteriaMatchesAssetType
 */
return false
}

func DateDifference(firstDate time.Time, secondDate time.Time ) int {    
/**
 * Function definition for DateDifference
 */
return 0
}

func DateDifferenceYears(firstDate time.Time, secondDate time.Time ) float64 {    
/**
 * Function definition for DateDifferenceYears
 */
return 0
}

func DayCountBasis(dcf cdm.DayCountFractionEnum ) int {    
/**
 * Function definition for DayCountBasis
 */
return 0
}

func DayOfWeek(date time.Time ) cdm.DayOfWeekEnum {    
/**
 * Function definition for DayOfWeek
 */
return 0
}

func DefaultFloatingRate(suppliedRate float64 ) cdm.FloatingRateProcessingDetails {    
/**
 * Function definition for DefaultFloatingRate
 */
return cdm.FloatingRateProcessingDetails{}
}

func DeliveryAmount(postedCreditSupportItems *cdm.PostedCreditSupportItem, priorDeliveryAmountAdjustment cdm.Money, priorReturnAmountAdjustment cdm.Money, disputedTransferredPostedCreditSupportAmount cdm.Money, marginAmount cdm.Money, threshold cdm.Money, marginApproach cdm.MarginApproachEnum, marginAmountIA *cdm.Money, minimumTransferAmount cdm.Money, rounding cdm.CollateralRounding, disputedDeliveryAmount cdm.Money, baseCurrency string ) cdm.Money {    
/**
 * Function definition for DeliveryAmount
 */
return cdm.Money{}
}

func DetermineFixingDate(resetDates cdm.ResetDates, resetDate time.Time ) time.Time {    
/**
 * Function definition for DetermineFixingDate
 */
return time.Time{}
}

func DetermineFloatingRateReset(interestRatePayout cdm.InterestRatePayout, calcPeriod cdm.CalculationPeriodBase ) cdm.FloatingRateSettingDetails {    
/**
 * Function definition for DetermineFloatingRateReset
 */
return cdm.FloatingRateSettingDetails{}
}

func DetermineObservationPeriod(adjustedCalculationPeriod cdm.CalculationPeriodBase, calculationParams cdm.FloatingRateCalculationParameters ) cdm.CalculationPeriodBase {    
/**
 * Function definition for DetermineObservationPeriod
 */
return cdm.CalculationPeriodBase{}
}

func DetermineResetDate(resetDates cdm.ResetDates, calculationPeriod cdm.CalculationPeriodBase ) time.Time {    
/**
 * Function definition for DetermineResetDate
 */
return time.Time{}
}

func DetermineWeightingDates(calculationParams cdm.FloatingRateCalculationParameters, observationDates *time.Time, observationPeriod cdm.CalculationPeriodBase, adjustedCalculationPeriod cdm.CalculationPeriodBase, lockoutDays int ) *time.Time {    
/**
 * Function definition for DetermineWeightingDates
 */
return &time.Time{}
}

func DividendCashSettlementAmount(numberOfSecurities float64, declaredDividend float64 ) float64 {    
/**
 * Function definition for DividendCashSettlementAmount
 */
return 0
}

func EmptyExecutionDetails() *cdm.ExecutionDetails {    
/**
 * Function definition for EmptyExecutionDetails
 */
return &cdm.ExecutionDetails{}
}

func EmptyTransferHistory() *cdm.TransferState {    
/**
 * Function definition for EmptyTransferHistory
 */
return &cdm.TransferState{}
}

func EquityCashSettlementAmount(tradeState cdm.TradeState, date time.Time ) cdm.Transfer {    
/**
 * Function definition for EquityCashSettlementAmount
 */
return cdm.Transfer{}
}

func EquityNotionalAmount(numberOfSecurities float64, price cdm.Price ) float64 {    
/**
 * Function definition for EquityNotionalAmount
 */
return 0
}

func EquityPerformance(trade cdm.Trade, observation cdm.Price, date time.Time ) float64 {    
/**
 * Function definition for EquityPerformance
 */
return 0
}

func EvaluateCalculatedRate(interestRateIndex cdm.InterestRateIndex, calculationParameters cdm.FloatingRateCalculationParameters, resetDates *cdm.ResetDates, calculationPeriod cdm.CalculationPeriodBase, priorCalculationPeriod *cdm.CalculationPeriodBase, dayCount cdm.DayCountFractionEnum ) cdm.FloatingRateSettingDetails {    
/**
 * Function definition for EvaluateCalculatedRate
 */
return cdm.FloatingRateSettingDetails{}
}

func EvaluatePortfolioState(portfolio cdm.Portfolio ) cdm.PortfolioState {    
/**
 * Function definition for EvaluatePortfolioState
 */
return cdm.PortfolioState{}
}

func EvaluateScreenRate(rateDef cdm.FloatingRate, resetDates cdm.ResetDates, calculationPeriod cdm.CalculationPeriodBase ) cdm.FloatingRateSettingDetails {    
/**
 * Function definition for EvaluateScreenRate
 */
return cdm.FloatingRateSettingDetails{}
}

func ExtractAfterTrade(businessEvent cdm.BusinessEvent ) *cdm.Trade {    
/**
 * Function definition for ExtractAfterTrade
 */
return &cdm.Trade{}
}

func ExtractAncillaryPartyByRole(ancillaryParties cdm.AncillaryParty, roleEnumToExtract cdm.AncillaryRoleEnum ) *cdm.AncillaryParty {    
/**
 * Function definition for ExtractAncillaryPartyByRole
 */
return &cdm.AncillaryParty{}
}

func ExtractBeforeEconomicTerms(businessEvent cdm.BusinessEvent ) *cdm.EconomicTerms {    
/**
 * Function definition for ExtractBeforeEconomicTerms
 */
return &cdm.EconomicTerms{}
}

func ExtractBeforeTrade(businessEvent cdm.BusinessEvent ) *cdm.Trade {    
/**
 * Function definition for ExtractBeforeTrade
 */
return &cdm.Trade{}
}

func ExtractCounterpartyByRole(counterparties cdm.Counterparty, roleEnumToExtract cdm.CounterpartyRoleEnum ) *cdm.Counterparty {    
/**
 * Function definition for ExtractCounterpartyByRole
 */
return &cdm.Counterparty{}
}

func ExtractFixedLeg(interestRatePayouts *cdm.InterestRatePayout ) *cdm.InterestRatePayout {    
/**
 * Function definition for ExtractFixedLeg
 */
return &cdm.InterestRatePayout{}
}

func ExtractOpenEconomicTerms(businessEvent cdm.BusinessEvent ) *cdm.EconomicTerms {    
/**
 * Function definition for ExtractOpenEconomicTerms
 */
return &cdm.EconomicTerms{}
}

func ExtractTradeCollateralPrice(tradableProduct cdm.TradableProduct ) *float64 {    
/**
 * Function definition for ExtractTradeCollateralPrice
 */
return nil
}

func ExtractTradeCollateralQuantity(tradableProduct cdm.TradableProduct ) *float64 {    
/**
 * Function definition for ExtractTradeCollateralQuantity
 */
return nil
}

func ExtractTradePurchasePrice(tradableProduct cdm.TradableProduct ) *float64 {    
/**
 * Function definition for ExtractTradePurchasePrice
 */
return nil
}

func FXFarLeg(product cdm.NonTransferableProduct ) *cdm.SettlementPayout {    
/**
 * Function definition for FXFarLeg
 */
return &cdm.SettlementPayout{}
}

func FilterCashTransfers(transfers *cdm.Transfer ) *cdm.Transfer {    
/**
 * Function definition for FilterCashTransfers
 */
return &cdm.Transfer{}
}

func FilterChangePriceQuantity(priceQuantity *cdm.PriceQuantity, change *cdm.PriceQuantity ) *cdm.PriceQuantity {    
/**
 * Function definition for FilterChangePriceQuantity
 */
return &cdm.PriceQuantity{}
}

func FilterClosedTradeStates(tradeStates *cdm.TradeState ) *cdm.TradeState {    
/**
 * Function definition for FilterClosedTradeStates
 */
return &cdm.TradeState{}
}

func FilterInvalidFloatingRateIndexTradeDate(tradeState cdm.TradeState ) *cdm.FloatingRateIndexEnum {    
/**
 * Function definition for FilterInvalidFloatingRateIndexTradeDate
 */
return new(cdm.FloatingRateIndexEnum)
}

func FilterOpenTradeStates(tradeStates *cdm.TradeState ) *cdm.TradeState {    
/**
 * Function definition for FilterOpenTradeStates
 */
return &cdm.TradeState{}
}

func FilterPartyRole(partyRoles *cdm.PartyRole, partyRoleEnum cdm.PartyRoleEnum ) *cdm.PartyRole {    
/**
 * Function definition for FilterPartyRole
 */
return &cdm.PartyRole{}
}

func FilterPrice(prices *cdm.PriceSchedule, priceType cdm.PriceTypeEnum, arithmeticOperators *cdm.ArithmeticOperationEnum, priceExpression *cdm.PriceExpressionEnum ) *cdm.PriceSchedule {    
/**
 * Function definition for FilterPrice
 */
return &cdm.PriceSchedule{}
}

func FilterQuantity(quantities *cdm.Quantity, unit cdm.UnitType ) *cdm.Quantity {    
/**
 * Function definition for FilterQuantity
 */
return &cdm.Quantity{}
}

func FilterQuantityByCurrency(quantities *cdm.QuantitySchedule, currency string ) *cdm.QuantitySchedule {    
/**
 * Function definition for FilterQuantityByCurrency
 */
return &cdm.QuantitySchedule{}
}

func FilterQuantityByCurrencyExists(quantities *cdm.QuantitySchedule ) *cdm.QuantitySchedule {    
/**
 * Function definition for FilterQuantityByCurrencyExists
 */
return &cdm.QuantitySchedule{}
}

func FilterQuantityByFinancialUnit(quantities *cdm.QuantitySchedule, financialUnit cdm.FinancialUnitEnum ) *cdm.QuantitySchedule {    
/**
 * Function definition for FilterQuantityByFinancialUnit
 */
return &cdm.QuantitySchedule{}
}

func FilterRelatedPartyByRole(relatedParties *cdm.RelatedParty, partyRoleEnum cdm.PartyRoleEnum ) *cdm.RelatedParty {    
/**
 * Function definition for FilterRelatedPartyByRole
 */
return &cdm.RelatedParty{}
}

func FilterSecurityTransfers(transfers *cdm.Transfer ) *cdm.Transfer {    
/**
 * Function definition for FilterSecurityTransfers
 */
return &cdm.Transfer{}
}

func FilterTradeLot(tradeLots *cdm.TradeLot, lotIdentifier *cdm.Identifier ) *cdm.TradeLot {    
/**
 * Function definition for FilterTradeLot
 */
return &cdm.TradeLot{}
}

func FindMatchingIndexTransitionInstruction(instructions cdm.PriceQuantity, priceQuantity cdm.PriceQuantity ) *cdm.PriceQuantity {    
/**
 * Function definition for FindMatchingIndexTransitionInstruction
 */
return &cdm.PriceQuantity{}
}

func FixedAmount(interestRatePayout cdm.InterestRatePayout, notional *float64, date *time.Time, calculationPeriodData *cdm.CalculationPeriodData ) float64 {    
/**
 * Function definition for FixedAmount
 */
return 0
}

func FixedAmountCalculation(interestRatePayout cdm.InterestRatePayout, calculationPeriod cdm.CalculationPeriodBase, notional *float64 ) cdm.FixedAmountCalculationDetails {    
/**
 * Function definition for FixedAmountCalculation
 */
return cdm.FixedAmountCalculationDetails{}
}

func FloatingAmount(interestRatePayout cdm.InterestRatePayout, rate *float64, notional *float64, date *time.Time, calculationPeriodData *cdm.CalculationPeriodData ) float64 {    
/**
 * Function definition for FloatingAmount
 */
return 0
}

func FloatingAmountCalculation(interestRatePayout cdm.InterestRatePayout, calculationPeriod cdm.CalculationPeriodBase, isInitialPeriod bool, suppliedNotional *float64, suppliedRate *float64 ) cdm.FloatingAmountCalculationDetails {    
/**
 * Function definition for FloatingAmountCalculation
 */
return cdm.FloatingAmountCalculationDetails{}
}

func FloatingRateIndexMetadata(floatingRateIndexName cdm.FloatingRateIndexEnum ) *cdm.FloatingRateIndexDefinition {    
/**
 * Function definition for FloatingRateIndexMetadata
 */
return &cdm.FloatingRateIndexDefinition{}
}

func FloorRateAmount(interestRatePayout cdm.InterestRatePayout, calculationPeriod cdm.CalculationPeriodBase ) *float64 {    
/**
 * Function definition for FloorRateAmount
 */
return nil
}

func FpmlIrd8(trade cdm.Trade, accounts *cdm.Account ) bool {    
/**
 * Function definition for FpmlIrd8
 */
return false
}

func FxMarkToMarket(trade cdm.Trade ) float64 {    
/**
 * Function definition for FxMarkToMarket
 */
return 0
}

func GenerateDateList(startDate time.Time, endDate time.Time, businessCenters *cdm.BusinessCenter ) *time.Time {    
/**
 * Function definition for GenerateDateList
 */
return &time.Time{}
}

func GenerateObservationDates(observationPeriod cdm.CalculationPeriodBase, businessCenters *cdm.BusinessCenter, lockoutDays *int ) *time.Time {    
/**
 * Function definition for GenerateObservationDates
 */
return &time.Time{}
}

func GenerateObservationDatesAndWeights(calculationParams cdm.FloatingRateCalculationParameters, resetDates *cdm.ResetDates, calculationPeriod cdm.CalculationPeriodBase, priorCalculationPeriod *cdm.CalculationPeriodBase ) cdm.CalculatedRateObservationDatesAndWeights {    
/**
 * Function definition for GenerateObservationDatesAndWeights
 */
return cdm.CalculatedRateObservationDatesAndWeights{}
}

func GenerateObservationPeriod(calculationPeriod cdm.CalculationPeriodBase, businessCenters *cdm.BusinessCenter, shiftDays *int ) cdm.CalculationPeriodBase {    
/**
 * Function definition for GenerateObservationPeriod
 */
return cdm.CalculationPeriodBase{}
}

func GenerateWeightings(calculationParams cdm.FloatingRateCalculationParameters, observationDates *time.Time, observationPeriod cdm.CalculationPeriodBase, adjustedCalculationPeriod cdm.CalculationPeriodBase, lockoutDays int ) *float64 {    
/**
 * Function definition for GenerateWeightings
 */
return nil
}

func GenerateWeights(weightingDates *time.Time ) *float64 {    
/**
 * Function definition for GenerateWeights
 */
return nil
}

func GetAllBusinessCenters(businessCenters cdm.BusinessCenters ) *cdm.BusinessCenter {    
/**
 * Function definition for GetAllBusinessCenters
 */
return &cdm.BusinessCenter{}
}

func GetCalculatedFROCalculationParameters(resetDates cdm.ResetDates, calcMethod cdm.CalculationMethodEnum ) cdm.FloatingRateCalculationParameters {    
/**
 * Function definition for GetCalculatedFROCalculationParameters
 */
return cdm.FloatingRateCalculationParameters{}
}

func GetCashCurrency(cash cdm.Cash ) cdm.CurrencyCodeEnum {    
/**
 * Function definition for GetCashCurrency
 */
return 0
}

func GetFixedRate(interestRatePayout cdm.InterestRatePayout, calculationPeriod cdm.CalculationPeriodBase ) *float64 {    
/**
 * Function definition for GetFixedRate
 */
return nil
}

func GetFloatingRateProcessingParameters(interestRatePayout cdm.InterestRatePayout, calculationPeriod cdm.CalculationPeriodBase ) cdm.FloatingRateProcessingParameters {    
/**
 * Function definition for GetFloatingRateProcessingParameters
 */
return cdm.FloatingRateProcessingParameters{}
}

func GetFloatingRateProcessingType(rateDef cdm.FloatingRateSpecification ) cdm.FloatingRateIndexProcessingTypeEnum {    
/**
 * Function definition for GetFloatingRateProcessingType
 */
return 0
}

func GetGrossInitialMarginFromStandardizedSchedule(standardizedSchedule cdm.StandardizedSchedule ) *cdm.Money {    
/**
 * Function definition for GetGrossInitialMarginFromStandardizedSchedule
 */
return &cdm.Money{}
}

func GetNetInitialMarginFromExposure(exposure *cdm.Exposure ) *cdm.StandardizedScheduleInitialMargin {    
/**
 * Function definition for GetNetInitialMarginFromExposure
 */
return &cdm.StandardizedScheduleInitialMargin{}
}

func GetNotionalAmount(interestRatePayout cdm.InterestRatePayout, calculationPeriod cdm.CalculationPeriodBase ) cdm.Money {    
/**
 * Function definition for GetNotionalAmount
 */
return cdm.Money{}
}

func GetQuantityScheduleStepValues(schedule cdm.NonNegativeQuantitySchedule, periodStartDate time.Time ) *float64 {    
/**
 * Function definition for GetQuantityScheduleStepValues
 */
return nil
}

func GetRateScheduleAmount(schedule cdm.RateSchedule, periodStartDate time.Time ) float64 {    
/**
 * Function definition for GetRateScheduleAmount
 */
return 0
}

func GetRateScheduleStepValues(schedule cdm.RateSchedule, periodStartDate time.Time ) *float64 {    
/**
 * Function definition for GetRateScheduleStepValues
 */
return nil
}

func GetStandardizedScheduleMarginRate(assetClass cdm.StandardizedScheduleAssetClassEnum, durationInYears float64 ) float64 {    
/**
 * Function definition for GetStandardizedScheduleMarginRate
 */
return 0
}

func IndexValueObservation(observationDate time.Time, interestRateIndex cdm.InterestRateIndex ) float64 {    
/**
 * Function definition for IndexValueObservation
 */
return 0
}

func IndexValueObservationMultiple(observationDate *time.Time, interestRateIndex cdm.InterestRateIndex ) *float64 {    
/**
 * Function definition for IndexValueObservationMultiple
 */
return nil
}

func InterestCashSettlementAmount(tradeState cdm.TradeState, payout cdm.Payout, resets cdm.Reset, date time.Time ) cdm.Transfer {    
/**
 * Function definition for InterestCashSettlementAmount
 */
return cdm.Transfer{}
}

func InterestRateObservableCondition(pq cdm.PriceQuantity ) *bool {    
/**
 * Function definition for InterestRateObservableCondition
 */
return nil
}

func InterestRatePayoutCurrency(interestRatePayouts *cdm.InterestRatePayout ) *string {    
/**
 * Function definition for InterestRatePayoutCurrency
 */
return new(string)
}

func InterestRatePayoutOnlyExists(payouts *cdm.Payout ) bool {    
/**
 * Function definition for InterestRatePayoutOnlyExists
 */
return false
}

func InterpolateForwardRate(settlementPayout cdm.SettlementPayout ) float64 {    
/**
 * Function definition for InterpolateForwardRate
 */
return 0
}

func IsBusinessDay(date time.Time, businessCenters *cdm.BusinessCenter ) bool {    
/**
 * Function definition for IsBusinessDay
 */
return false
}

func IsHoliday(checkDate time.Time, businessCenters *cdm.BusinessCenter ) bool {    
/**
 * Function definition for IsHoliday
 */
return false
}

func IsValidPartyRole(partyRoles *cdm.PartyRole, validRoles cdm.PartyRoleEnum ) bool {    
/**
 * Function definition for IsValidPartyRole
 */
return false
}

func IsWeekend(date time.Time, businessCenters *cdm.BusinessCenter ) bool {    
/**
 * Function definition for IsWeekend
 */
return false
}

func LeapYearDateDifference(firstDate time.Time, secondDate time.Time ) int {    
/**
 * Function definition for LeapYearDateDifference
 */
return 0
}

func LoadCodeList(domain string ) *cdm.CodeList {    
/**
 * Function definition for LoadCodeList
 */
return &cdm.CodeList{}
}

func Max(a float64, b float64 ) float64 {    
/**
 * Function definition for Max
 */
return 0
}

func Min(a float64, b float64 ) float64 {    
/**
 * Function definition for Min
 */
return 0
}

func MultiplierAmount(interestRatePayout cdm.InterestRatePayout, calculationPeriod cdm.CalculationPeriodBase ) *float64 {    
/**
 * Function definition for MultiplierAmount
 */
return nil
}

func NewEquitySwapProduct(security cdm.Security, masterConfirmation *cdm.EquitySwapMasterConfirmation2018 ) cdm.NonTransferableProduct {    
/**
 * Function definition for NewEquitySwapProduct
 */
return cdm.NonTransferableProduct{}
}

func NewFloatingPayout(masterConfirmation *cdm.EquitySwapMasterConfirmation2018 ) cdm.InterestRatePayout {    
/**
 * Function definition for NewFloatingPayout
 */
return cdm.InterestRatePayout{}
}

func NewSingleNameEquityPerformancePayout(security cdm.Security, masterConfirmation *cdm.EquitySwapMasterConfirmation2018 ) cdm.PerformancePayout {    
/**
 * Function definition for NewSingleNameEquityPerformancePayout
 */
return cdm.PerformancePayout{}
}

func NewTradeInstructionOnlyExists(primitiveInstruction cdm.PrimitiveInstruction ) bool {    
/**
 * Function definition for NewTradeInstructionOnlyExists
 */
return false
}

func Now() time.Time {    
/**
 * Function definition for Now
 */
return time.Time{}
}

func ObservableIsCommodity(observable *cdm.Observable ) bool {    
/**
 * Function definition for ObservableIsCommodity
 */
return false
}

func ObservableQualification(observable *cdm.Observable, securityType *cdm.InstrumentTypeEnum, assetClass *cdm.AssetClassEnum ) bool {    
/**
 * Function definition for ObservableQualification
 */
return false
}

func OptionPayoutOnlyExists(payouts *cdm.Payout ) bool {    
/**
 * Function definition for OptionPayoutOnlyExists
 */
return false
}

func PaymentDate(economicTerms cdm.EconomicTerms ) *time.Time {    
/**
 * Function definition for PaymentDate
 */
return &time.Time{}
}

func PerformancePayoutAndFixedPricePayoutOnlyExists(payouts *cdm.Payout ) bool {    
/**
 * Function definition for PerformancePayoutAndFixedPricePayoutOnlyExists
 */
return false
}

func PerformancePayoutAndInterestRatePayoutOnlyExists(payouts *cdm.Payout ) bool {    
/**
 * Function definition for PerformancePayoutAndInterestRatePayoutOnlyExists
 */
return false
}

func PerformancePayoutOnlyExists(payouts *cdm.Payout ) bool {    
/**
 * Function definition for PerformancePayoutOnlyExists
 */
return false
}

func PeriodsInYear(frequency cdm.CalculationPeriodFrequency ) int {    
/**
 * Function definition for PeriodsInYear
 */
return 0
}

func PopOffDateList(dates *time.Time ) *time.Time {    
/**
 * Function definition for PopOffDateList
 */
return &time.Time{}
}

func PostedCreditSupportItemAmount(postedItem cdm.PostedCreditSupportItem, baseCurrency string ) cdm.Money {    
/**
 * Function definition for PostedCreditSupportItemAmount
 */
return cdm.Money{}
}

func PriceQuantityTriangulation(tradeLots *cdm.TradeLot ) bool {    
/**
 * Function definition for PriceQuantityTriangulation
 */
return false
}

func PriceUnitEquals(p1 *cdm.PriceSchedule, p2 *cdm.PriceSchedule ) bool {    
/**
 * Function definition for PriceUnitEquals
 */
return false
}

func ProcessFloatingRateReset(interestRatePayout cdm.InterestRatePayout, calcPeriod cdm.CalculationPeriodBase, processingType cdm.FloatingRateIndexProcessingTypeEnum ) cdm.FloatingRateSettingDetails {    
/**
 * Function definition for ProcessFloatingRateReset
 */
return cdm.FloatingRateSettingDetails{}
}

func ProcessObservations(calculationParameters cdm.FloatingRateCalculationParameters, rawObservations *float64 ) *float64 {    
/**
 * Function definition for ProcessObservations
 */
return nil
}

func Qualify_Adjustment(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Adjustment
 */
return false
}

func Qualify_Allocation(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Allocation
 */
return false
}

func Qualify_AssetClass_Commodity(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_AssetClass_Commodity
 */
return false
}

func Qualify_AssetClass_Credit(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_AssetClass_Credit
 */
return false
}

func Qualify_AssetClass_Equity(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_AssetClass_Equity
 */
return false
}

func Qualify_AssetClass_ForeignExchange(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_AssetClass_ForeignExchange
 */
return false
}

func Qualify_AssetClass_InterestRate(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_AssetClass_InterestRate
 */
return false
}

func Qualify_BaseProduct_CrossCurrency(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_BaseProduct_CrossCurrency
 */
return false
}

func Qualify_BaseProduct_EquityForward(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_BaseProduct_EquityForward
 */
return false
}

func Qualify_BaseProduct_EquitySwap(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_BaseProduct_EquitySwap
 */
return false
}

func Qualify_BaseProduct_Fra(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_BaseProduct_Fra
 */
return false
}

func Qualify_BaseProduct_IRSwap(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_BaseProduct_IRSwap
 */
return false
}

func Qualify_BaseProduct_Inflation(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_BaseProduct_Inflation
 */
return false
}

func Qualify_BuySellBack(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_BuySellBack
 */
return false
}

func Qualify_Cancellation(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Cancellation
 */
return false
}

func Qualify_CashAndSecurityTransfer(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_CashAndSecurityTransfer
 */
return false
}

func Qualify_CashTransfer(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_CashTransfer
 */
return false
}

func Qualify_ClearedTrade(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_ClearedTrade
 */
return false
}

func Qualify_Commodity_Forward(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Commodity_Forward
 */
return false
}

func Qualify_Commodity_Option(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Commodity_Option
 */
return false
}

func Qualify_Commodity_Option_Cash(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Commodity_Option_Cash
 */
return false
}

func Qualify_Commodity_Option_NonStandard(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Commodity_Option_NonStandard
 */
return false
}

func Qualify_Commodity_Option_Physical(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Commodity_Option_Physical
 */
return false
}

func Qualify_Commodity_Swap_Basis(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Commodity_Swap_Basis
 */
return false
}

func Qualify_Commodity_Swap_FixedFloat(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Commodity_Swap_FixedFloat
 */
return false
}

func Qualify_Commodity_Swaption(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Commodity_Swaption
 */
return false
}

func Qualify_Compression(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Compression
 */
return false
}

func Qualify_ContractFormation(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_ContractFormation
 */
return false
}

func Qualify_CorporateActionDetermined(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_CorporateActionDetermined
 */
return false
}

func Qualify_CreditDefaultSwap_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_CreditDefaultSwap_Basket
 */
return false
}

func Qualify_CreditDefaultSwap_Index(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_CreditDefaultSwap_Index
 */
return false
}

func Qualify_CreditDefaultSwap_IndexTranche(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_CreditDefaultSwap_IndexTranche
 */
return false
}

func Qualify_CreditDefaultSwap_Loan(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_CreditDefaultSwap_Loan
 */
return false
}

func Qualify_CreditDefaultSwap_SingleName(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_CreditDefaultSwap_SingleName
 */
return false
}

func Qualify_CreditDefaultSwaption(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_CreditDefaultSwaption
 */
return false
}

func Qualify_CreditEventDetermined(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_CreditEventDetermined
 */
return false
}

func Qualify_Credit_NthToDefault(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Credit_NthToDefault
 */
return false
}

func Qualify_Credit_Option_NonStandard(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Credit_Option_NonStandard
 */
return false
}

func Qualify_EquityOption_ParameterReturnCorrelation_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_ParameterReturnCorrelation_Basket
 */
return false
}

func Qualify_EquityOption_ParameterReturnDividend_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_ParameterReturnDividend_Basket
 */
return false
}

func Qualify_EquityOption_ParameterReturnDividend_Index(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_ParameterReturnDividend_Index
 */
return false
}

func Qualify_EquityOption_ParameterReturnDividend_SingleName(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_ParameterReturnDividend_SingleName
 */
return false
}

func Qualify_EquityOption_ParameterReturnVariance_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_ParameterReturnVariance_Basket
 */
return false
}

func Qualify_EquityOption_ParameterReturnVariance_Index(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_ParameterReturnVariance_Index
 */
return false
}

func Qualify_EquityOption_ParameterReturnVariance_SingleName(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_ParameterReturnVariance_SingleName
 */
return false
}

func Qualify_EquityOption_ParameterReturnVolatility_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_ParameterReturnVolatility_Basket
 */
return false
}

func Qualify_EquityOption_ParameterReturnVolatility_Index(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_ParameterReturnVolatility_Index
 */
return false
}

func Qualify_EquityOption_ParameterReturnVolatility_SingleName(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_ParameterReturnVolatility_SingleName
 */
return false
}

func Qualify_EquityOption_PriceReturnBasicPerformance_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_PriceReturnBasicPerformance_Basket
 */
return false
}

func Qualify_EquityOption_PriceReturnBasicPerformance_Index(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_PriceReturnBasicPerformance_Index
 */
return false
}

func Qualify_EquityOption_PriceReturnBasicPerformance_SingleName(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquityOption_PriceReturnBasicPerformance_SingleName
 */
return false
}

func Qualify_EquitySwap_ParameterReturnCorrelation_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_ParameterReturnCorrelation_Basket
 */
return false
}

func Qualify_EquitySwap_ParameterReturnDispersion(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_ParameterReturnDispersion
 */
return false
}

func Qualify_EquitySwap_ParameterReturnDividend_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_ParameterReturnDividend_Basket
 */
return false
}

func Qualify_EquitySwap_ParameterReturnDividend_Index(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_ParameterReturnDividend_Index
 */
return false
}

func Qualify_EquitySwap_ParameterReturnDividend_SingleName(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_ParameterReturnDividend_SingleName
 */
return false
}

func Qualify_EquitySwap_ParameterReturnVariance_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_ParameterReturnVariance_Basket
 */
return false
}

func Qualify_EquitySwap_ParameterReturnVariance_Index(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_ParameterReturnVariance_Index
 */
return false
}

func Qualify_EquitySwap_ParameterReturnVariance_SingleName(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_ParameterReturnVariance_SingleName
 */
return false
}

func Qualify_EquitySwap_ParameterReturnVolatility_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_ParameterReturnVolatility_Basket
 */
return false
}

func Qualify_EquitySwap_ParameterReturnVolatility_Index(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_ParameterReturnVolatility_Index
 */
return false
}

func Qualify_EquitySwap_ParameterReturnVolatility_SingleName(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_ParameterReturnVolatility_SingleName
 */
return false
}

func Qualify_EquitySwap_PriceReturnBasicPerformance_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_PriceReturnBasicPerformance_Basket
 */
return false
}

func Qualify_EquitySwap_PriceReturnBasicPerformance_Index(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_PriceReturnBasicPerformance_Index
 */
return false
}

func Qualify_EquitySwap_PriceReturnBasicPerformance_SingleName(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_PriceReturnBasicPerformance_SingleName
 */
return false
}

func Qualify_EquitySwap_TotalReturnBasicPerformance_Basket(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_TotalReturnBasicPerformance_Basket
 */
return false
}

func Qualify_EquitySwap_TotalReturnBasicPerformance_Index(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_TotalReturnBasicPerformance_Index
 */
return false
}

func Qualify_EquitySwap_TotalReturnBasicPerformance_SingleName(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_EquitySwap_TotalReturnBasicPerformance_SingleName
 */
return false
}

func Qualify_Equity_OtherForward(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Equity_OtherForward
 */
return false
}

func Qualify_Equity_OtherOption(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Equity_OtherOption
 */
return false
}

func Qualify_Equity_Swap_NonStandard(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Equity_Swap_NonStandard
 */
return false
}

func Qualify_Execution(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Execution
 */
return false
}

func Qualify_Exercise(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Exercise
 */
return false
}

func Qualify_ForeignExchange_NDF(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_ForeignExchange_NDF
 */
return false
}

func Qualify_ForeignExchange_NDO(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_ForeignExchange_NDO
 */
return false
}

func Qualify_ForeignExchange_NDS(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_ForeignExchange_NDS
 */
return false
}

func Qualify_ForeignExchange_ParameterReturnCorrelation(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_ForeignExchange_ParameterReturnCorrelation
 */
return false
}

func Qualify_ForeignExchange_ParameterReturnVariance(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_ForeignExchange_ParameterReturnVariance
 */
return false
}

func Qualify_ForeignExchange_ParameterReturnVolatility(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_ForeignExchange_ParameterReturnVolatility
 */
return false
}

func Qualify_ForeignExchange_Spot_Forward(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_ForeignExchange_Spot_Forward
 */
return false
}

func Qualify_ForeignExchange_Swap(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_ForeignExchange_Swap
 */
return false
}

func Qualify_ForeignExchange_VanillaOption(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_ForeignExchange_VanillaOption
 */
return false
}

func Qualify_FullReturn(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_FullReturn
 */
return false
}

func Qualify_Increase(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Increase
 */
return false
}

func Qualify_IndexTransition(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_IndexTransition
 */
return false
}

func Qualify_InstrumentTypeEquity(instrument cdm.Instrument ) bool {    
/**
 * Function definition for Qualify_InstrumentTypeEquity
 */
return false
}

func Qualify_InterestRate_CapFloor(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_CapFloor
 */
return false
}

func Qualify_InterestRate_CrossCurrency_Basis(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_CrossCurrency_Basis
 */
return false
}

func Qualify_InterestRate_CrossCurrency_FixedFixed(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_CrossCurrency_FixedFixed
 */
return false
}

func Qualify_InterestRate_CrossCurrency_FixedFloat(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_CrossCurrency_FixedFloat
 */
return false
}

func Qualify_InterestRate_Forward_Debt(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_Forward_Debt
 */
return false
}

func Qualify_InterestRate_Fra(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_Fra
 */
return false
}

func Qualify_InterestRate_IRSwap_Basis(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_IRSwap_Basis
 */
return false
}

func Qualify_InterestRate_IRSwap_Basis_OIS(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_IRSwap_Basis_OIS
 */
return false
}

func Qualify_InterestRate_IRSwap_FixedFixed(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_IRSwap_FixedFixed
 */
return false
}

func Qualify_InterestRate_IRSwap_FixedFloat(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_IRSwap_FixedFloat
 */
return false
}

func Qualify_InterestRate_IRSwap_FixedFloat_OIS(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_IRSwap_FixedFloat_OIS
 */
return false
}

func Qualify_InterestRate_IRSwap_FixedFloat_ZeroCoupon(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_IRSwap_FixedFloat_ZeroCoupon
 */
return false
}

func Qualify_InterestRate_InflationSwap_Basis_YearOn_Year(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_InflationSwap_Basis_YearOn_Year
 */
return false
}

func Qualify_InterestRate_InflationSwap_Basis_ZeroCoupon(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_InflationSwap_Basis_ZeroCoupon
 */
return false
}

func Qualify_InterestRate_InflationSwap_FixedFloat_YearOn_Year(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_InflationSwap_FixedFloat_YearOn_Year
 */
return false
}

func Qualify_InterestRate_InflationSwap_FixedFloat_ZeroCoupon(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_InflationSwap_FixedFloat_ZeroCoupon
 */
return false
}

func Qualify_InterestRate_Option_DebtOption(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_Option_DebtOption
 */
return false
}

func Qualify_InterestRate_Option_Swaption(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_Option_Swaption
 */
return false
}

func Qualify_InterestRate_SwapWithCallableBermudanRightToEnterExitSwaps(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_SwapWithCallableBermudanRightToEnterExitSwaps
 */
return false
}

func Qualify_InterestRate_Swaption_Straddle(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_InterestRate_Swaption_Straddle
 */
return false
}

func Qualify_Novation(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Novation
 */
return false
}

func Qualify_OnDemandPayment(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_OnDemandPayment
 */
return false
}

func Qualify_OnDemandRateChange(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_OnDemandRateChange
 */
return false
}

func Qualify_OpenOfferClearedTrade(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_OpenOfferClearedTrade
 */
return false
}

func Qualify_PairOff(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_PairOff
 */
return false
}

func Qualify_PartialDelivery(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_PartialDelivery
 */
return false
}

func Qualify_PartialNovation(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_PartialNovation
 */
return false
}

func Qualify_PartialTermination(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_PartialTermination
 */
return false
}

func Qualify_PortfolioRebalancing(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_PortfolioRebalancing
 */
return false
}

func Qualify_Reallocation(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Reallocation
 */
return false
}

func Qualify_Renegotiation(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Renegotiation
 */
return false
}

func Qualify_Reprice(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Reprice
 */
return false
}

func Qualify_Repurchase(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Repurchase
 */
return false
}

func Qualify_RepurchaseAgreement(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_RepurchaseAgreement
 */
return false
}

func Qualify_Reset(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Reset
 */
return false
}

func Qualify_Roll(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Roll
 */
return false
}

func Qualify_SecurityLending(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_SecurityLending
 */
return false
}

func Qualify_SecuritySettlement(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_SecuritySettlement
 */
return false
}

func Qualify_SecurityTransfer(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_SecurityTransfer
 */
return false
}

func Qualify_Shaping(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Shaping
 */
return false
}

func Qualify_StockSplit(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_StockSplit
 */
return false
}

func Qualify_SubProduct_Basis(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_SubProduct_Basis
 */
return false
}

func Qualify_SubProduct_FixedFixed(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_SubProduct_FixedFixed
 */
return false
}

func Qualify_SubProduct_FixedFloat(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_SubProduct_FixedFloat
 */
return false
}

func Qualify_Substitution(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Substitution
 */
return false
}

func Qualify_Termination(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_Termination
 */
return false
}

func Qualify_TotalReturnSwap_Index(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_TotalReturnSwap_Index
 */
return false
}

func Qualify_Transaction_OIS(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Transaction_OIS
 */
return false
}

func Qualify_Transaction_YoY(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Transaction_YoY
 */
return false
}

func Qualify_Transaction_ZeroCoupon(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Transaction_ZeroCoupon
 */
return false
}

func Qualify_Transaction_ZeroCoupon_KnownAmount(economicTerms cdm.EconomicTerms ) bool {    
/**
 * Function definition for Qualify_Transaction_ZeroCoupon_KnownAmount
 */
return false
}

func Qualify_UnderlierObservable_Equity(observable cdm.Observable ) bool {    
/**
 * Function definition for Qualify_UnderlierObservable_Equity
 */
return false
}

func Qualify_ValuationUpdate(businessEvent cdm.BusinessEvent ) bool {    
/**
 * Function definition for Qualify_ValuationUpdate
 */
return false
}

func QuantityDecreased(before cdm.TradeState, after *cdm.TradeState ) bool {    
/**
 * Function definition for QuantityDecreased
 */
return false
}

func QuantityDecreasedToZero(before *cdm.TradeState, after *cdm.TradeState ) bool {    
/**
 * Function definition for QuantityDecreasedToZero
 */
return false
}

func QuantityIncreased(before cdm.TradeState, after *cdm.TradeState ) bool {    
/**
 * Function definition for QuantityIncreased
 */
return false
}

func RateOfReturn(initialPrice cdm.PriceSchedule, finalPrice cdm.PriceSchedule ) float64 {    
/**
 * Function definition for RateOfReturn
 */
return 0
}

func ReplaceParty(parties *cdm.Party, oldParty cdm.Party, newParty cdm.Party ) *cdm.Party {    
/**
 * Function definition for ReplaceParty
 */
return &cdm.Party{}
}

func ReplaceTradeLot(tradeLots *cdm.TradeLot, newTradeLot cdm.TradeLot ) *cdm.TradeLot {    
/**
 * Function definition for ReplaceTradeLot
 */
return &cdm.TradeLot{}
}

func ResolveAdjustableDate(adjustableOrRelativeDate cdm.AdjustableOrRelativeDate ) *time.Time {    
/**
 * Function definition for ResolveAdjustableDate
 */
return &time.Time{}
}

func ResolveAdjustableDates(adjustableRelativeOrPeriodicDates cdm.AdjustableRelativeOrPeriodicDates ) *time.Time {    
/**
 * Function definition for ResolveAdjustableDates
 */
return &time.Time{}
}

func ResolveCashSettlementDate(tradeState cdm.TradeState ) time.Time {    
/**
 * Function definition for ResolveCashSettlementDate
 */
return time.Time{}
}

func ResolveEquityInitialPrice(price *cdm.PriceSchedule ) *cdm.PriceSchedule {    
/**
 * Function definition for ResolveEquityInitialPrice
 */
return &cdm.PriceSchedule{}
}

func ResolveInterestRateObservationIdentifiers(payout cdm.InterestRatePayout, date time.Time ) cdm.ObservationIdentifier {    
/**
 * Function definition for ResolveInterestRateObservationIdentifiers
 */
return cdm.ObservationIdentifier{}
}

func ResolveInterestRateReset(payouts cdm.InterestRatePayout, observation cdm.Observation, resetDate time.Time, rateRecordDate *time.Time ) cdm.Reset {    
/**
 * Function definition for ResolveInterestRateReset
 */
return cdm.Reset{}
}

func ResolveObservation(identifiers cdm.ObservationIdentifier, averagingMethod *cdm.AveragingCalculationMethod ) cdm.Observation {    
/**
 * Function definition for ResolveObservation
 */
return cdm.Observation{}
}

func ResolveObservationAverage(observations cdm.Observation ) cdm.Price {    
/**
 * Function definition for ResolveObservationAverage
 */
return cdm.Price{}
}

func ResolvePerformanceObservationIdentifiers(payout cdm.PerformancePayout, adjustedDate time.Time ) cdm.ObservationIdentifier {    
/**
 * Function definition for ResolvePerformanceObservationIdentifiers
 */
return cdm.ObservationIdentifier{}
}

func ResolvePerformancePeriodStartPrice(performancePayout cdm.PerformancePayout, price *cdm.PriceSchedule, observable *cdm.Observable, adjustedDate time.Time ) cdm.PriceSchedule {    
/**
 * Function definition for ResolvePerformancePeriodStartPrice
 */
return cdm.PriceSchedule{}
}

func ResolvePerformanceReset(performancePayout cdm.PerformancePayout, observation cdm.Observation, date time.Time ) cdm.Reset {    
/**
 * Function definition for ResolvePerformanceReset
 */
return cdm.Reset{}
}

func ResolvePerformanceValuationTime(valuationTime *cdm.BusinessCenterTime, valuationTimeType *cdm.TimeTypeEnum, assetIdentifier cdm.AssetIdentifier, determinationMethod cdm.DeterminationMethodEnum ) cdm.TimeZone {    
/**
 * Function definition for ResolvePerformanceValuationTime
 */
return cdm.TimeZone{}
}

func ResolveRateIndex(index cdm.FloatingRateIndexEnum ) float64 {    
/**
 * Function definition for ResolveRateIndex
 */
return 0
}

func ResolveRepurchaseTransferInstruction(tradeState cdm.TradeState, repurchaseDate time.Time ) cdm.EventInstruction {    
/**
 * Function definition for ResolveRepurchaseTransferInstruction
 */
return cdm.EventInstruction{}
}

func ResolveReset(tradeState cdm.TradeState, date time.Time ) cdm.Reset {    
/**
 * Function definition for ResolveReset
 */
return cdm.Reset{}
}

func ResolveSecurityFinanceBillingAmount(tradeState cdm.TradeState, reset cdm.Reset, recordStartDate time.Time, recordEndDate time.Time, transferDate time.Time ) cdm.Transfer {    
/**
 * Function definition for ResolveSecurityFinanceBillingAmount
 */
return cdm.Transfer{}
}

func ResolveTimeZoneFromTimeType(assetIdentifier cdm.AssetIdentifier, timeType cdm.TimeTypeEnum, determinationMethod cdm.DeterminationMethodEnum ) cdm.TimeZone {    
/**
 * Function definition for ResolveTimeZoneFromTimeType
 */
return cdm.TimeZone{}
}

func ResolveTransfer(instruction cdm.CalculateTransferInstruction ) cdm.Transfer {    
/**
 * Function definition for ResolveTransfer
 */
return cdm.Transfer{}
}

func ReturnAmount(postedCreditSupportItems *cdm.PostedCreditSupportItem, priorDeliveryAmountAdjustment cdm.Money, priorReturnAmountAdjustment cdm.Money, disputedTransferredPostedCreditSupportAmount cdm.Money, marginAmount cdm.Money, threshold cdm.Money, marginApproach cdm.MarginApproachEnum, marginAmountIA *cdm.Money, minimumTransferAmount cdm.Money, rounding cdm.CollateralRounding, disputedReturnAmount cdm.Money, baseCurrency string ) cdm.Money {    
/**
 * Function definition for ReturnAmount
 */
return cdm.Money{}
}

func RoundToNearest(value float64, nearest float64, roundingMode cdm.RoundingModeEnum ) float64 {    
/**
 * Function definition for RoundToNearest
 */
return 0
}

func RoundToPrecision(value float64, precision int, roundingMode cdm.RoundingDirectionEnum, removeTrailingZeros bool ) float64 {    
/**
 * Function definition for RoundToPrecision
 */
return 0
}

func RoundToSignificantFigures(value float64, significantFigures int, roundingMode cdm.RoundingDirectionEnum ) float64 {    
/**
 * Function definition for RoundToSignificantFigures
 */
return 0
}

func SecurityFinanceCashSettlementAmount(tradeState cdm.TradeState, date time.Time, quantity *cdm.Quantity, payerReceiver *cdm.PayerReceiver ) cdm.Transfer {    
/**
 * Function definition for SecurityFinanceCashSettlementAmount
 */
return cdm.Transfer{}
}

func SetCashCurrency(cash *cdm.Cash, currency cdm.CurrencyCodeEnum ) cdm.Cash {    
/**
 * Function definition for SetCashCurrency
 */
return cdm.Cash{}
}

func SettlementPayoutOnlyExists(payouts *cdm.Payout ) bool {    
/**
 * Function definition for SettlementPayoutOnlyExists
 */
return false
}

func SplitQuantityChange(changeList cdm.PriceQuantity ) cdm.PriceQuantity {    
/**
 * Function definition for SplitQuantityChange
 */
return cdm.PriceQuantity{}
}

func SpreadAmount(interestRatePayout cdm.InterestRatePayout, calculationPeriod cdm.CalculationPeriodBase ) *float64 {    
/**
 * Function definition for SpreadAmount
 */
return nil
}

func StandardizedScheduleAssetClass(trade cdm.Trade ) *cdm.StandardizedScheduleAssetClassEnum {    
/**
 * Function definition for StandardizedScheduleAssetClass
 */
return new(cdm.StandardizedScheduleAssetClassEnum)
}

func StandardizedScheduleCommodityForwardNotionalAmount(economicTerms *cdm.EconomicTerms ) *float64 {    
/**
 * Function definition for StandardizedScheduleCommodityForwardNotionalAmount
 */
return nil
}

func StandardizedScheduleCommoditySwapFixedFloatNotionalAmount(economicTerms *cdm.EconomicTerms ) *float64 {    
/**
 * Function definition for StandardizedScheduleCommoditySwapFixedFloatNotionalAmount
 */
return nil
}

func StandardizedScheduleDuration(trade cdm.Trade, assetClass cdm.StandardizedScheduleAssetClassEnum, productClass cdm.StandardizedScheduleProductClassEnum ) *float64 {    
/**
 * Function definition for StandardizedScheduleDuration
 */
return nil
}

func StandardizedScheduleEquityForwardNotionalAmount(settlementPayout *cdm.SettlementPayout ) *float64 {    
/**
 * Function definition for StandardizedScheduleEquityForwardNotionalAmount
 */
return nil
}

func StandardizedScheduleFXSwapNotional(farLeg *cdm.SettlementPayout, tradeLot *cdm.TradeLot ) *cdm.NonNegativeQuantitySchedule {    
/**
 * Function definition for StandardizedScheduleFXSwapNotional
 */
return &cdm.NonNegativeQuantitySchedule{}
}

func StandardizedScheduleFXVarianceNotionalAmount(performancePayout *cdm.PerformancePayout ) *float64 {    
/**
 * Function definition for StandardizedScheduleFXVarianceNotionalAmount
 */
return nil
}

func StandardizedScheduleMonetaryNotionalCurrencyFromResolvablePQ(priceQuantity *cdm.ResolvablePriceQuantity ) *string {    
/**
 * Function definition for StandardizedScheduleMonetaryNotionalCurrencyFromResolvablePQ
 */
return new(string)
}

func StandardizedScheduleMonetaryNotionalFromResolvablePQ(priceQuantity *cdm.ResolvablePriceQuantity ) *float64 {    
/**
 * Function definition for StandardizedScheduleMonetaryNotionalFromResolvablePQ
 */
return nil
}

func StandardizedScheduleNotional(trade cdm.Trade, assetClass cdm.StandardizedScheduleAssetClassEnum, productClass cdm.StandardizedScheduleProductClassEnum ) *float64 {    
/**
 * Function definition for StandardizedScheduleNotional
 */
return nil
}

func StandardizedScheduleNotionalCurrency(trade cdm.Trade, assetClass cdm.StandardizedScheduleAssetClassEnum, productClass cdm.StandardizedScheduleProductClassEnum ) *string {    
/**
 * Function definition for StandardizedScheduleNotionalCurrency
 */
return new(string)
}

func StandardizedScheduleOptionNotionalAmount(optionPayout *cdm.OptionPayout ) *float64 {    
/**
 * Function definition for StandardizedScheduleOptionNotionalAmount
 */
return nil
}

func StandardizedScheduleProductClass(trade cdm.Trade ) *cdm.StandardizedScheduleProductClassEnum {    
/**
 * Function definition for StandardizedScheduleProductClass
 */
return new(cdm.StandardizedScheduleProductClassEnum)
}

func StandardizedScheduleVarianceSwapNotionalAmount(performancePayout *cdm.PerformancePayout ) *float64 {    
/**
 * Function definition for StandardizedScheduleVarianceSwapNotionalAmount
 */
return nil
}

func StringEquals(s1 *string, s2 *string ) bool {    
/**
 * Function definition for StringEquals
 */
return false
}

func TimeZoneFromBusinessCenterTime(time cdm.BusinessCenterTime ) cdm.TimeZone {    
/**
 * Function definition for TimeZoneFromBusinessCenterTime
 */
return cdm.TimeZone{}
}

func ToDateTime(date *time.Time ) *time.Time {    
/**
 * Function definition for ToDateTime
 */
return &time.Time{}
}

func ToMoney(quantity cdm.Quantity ) cdm.Money {    
/**
 * Function definition for ToMoney
 */
return cdm.Money{}
}

func ToTime(hours int, minutes int, seconds int ) time.Time {    
/**
 * Function definition for ToTime
 */
return time.Time{}
}

func Today() time.Time {    
/**
 * Function definition for Today
 */
return time.Time{}
}

func TradeNoExecutionDetails(trade cdm.Trade ) cdm.Trade {    
/**
 * Function definition for TradeNoExecutionDetails
 */
return cdm.Trade{}
}

func TransfersForDate(transfers *cdm.Transfer, date time.Time ) *cdm.Transfer {    
/**
 * Function definition for TransfersForDate
 */
return &cdm.Transfer{}
}

func UnderlierForOptionOrForwardProduct(product cdm.NonTransferableProduct ) cdm.Underlier {    
/**
 * Function definition for UnderlierForOptionOrForwardProduct
 */
return cdm.Underlier{}
}

func UnderlierQualification(underlier cdm.Underlier, securityType *cdm.InstrumentTypeEnum, assetClass *cdm.AssetClassEnum ) bool {    
/**
 * Function definition for UnderlierQualification
 */
return false
}

func UndisputedAdjustedPostedCreditSupportAmount(postedCreditSupportItems *cdm.PostedCreditSupportItem, priorDeliveryAmountAdjustment cdm.Money, priorReturnAmountAdjustment cdm.Money, disputedTransferredPostedCreditSupportAmount cdm.Money, baseCurrency string ) cdm.Money {    
/**
 * Function definition for UndisputedAdjustedPostedCreditSupportAmount
 */
return cdm.Money{}
}

func UnitEquals(u1 *cdm.UnitType, u2 *cdm.UnitType ) bool {    
/**
 * Function definition for UnitEquals
 */
return false
}

func UpdateAmount(oldAmount *float64, changeAmount *float64, direction cdm.QuantityChangeDirectionEnum ) *float64 {    
/**
 * Function definition for UpdateAmount
 */
return nil
}

func UpdateAmountForEachMatchingQuantity(priceQuantityList cdm.PriceQuantity, change cdm.PriceQuantity, direction cdm.QuantityChangeDirectionEnum ) cdm.PriceQuantity {    
/**
 * Function definition for UpdateAmountForEachMatchingQuantity
 */
return cdm.PriceQuantity{}
}

func UpdateDatedValues(datedValues *cdm.DatedValue, changeAmount *float64, direction cdm.QuantityChangeDirectionEnum, effectiveDate *time.Time ) *cdm.DatedValue {    
/**
 * Function definition for UpdateDatedValues
 */
return &cdm.DatedValue{}
}

func UpdateIndexTransitionPriceAndRateOption(priceQuantity cdm.PriceQuantity, instruction *cdm.PriceQuantity ) cdm.PriceQuantity {    
/**
 * Function definition for UpdateIndexTransitionPriceAndRateOption
 */
return cdm.PriceQuantity{}
}

func UpdatePriceAmountForEachMatchingQuantity(price *cdm.PriceSchedule, change *cdm.PriceSchedule, direction cdm.QuantityChangeDirectionEnum ) *cdm.PriceSchedule {    
/**
 * Function definition for UpdatePriceAmountForEachMatchingQuantity
 */
return &cdm.PriceSchedule{}
}

func UpdateQuantityAmountForEachMatchingQuantity(quantity *cdm.NonNegativeQuantitySchedule, change *cdm.PriceQuantity, direction cdm.QuantityChangeDirectionEnum ) *cdm.NonNegativeQuantitySchedule {    
/**
 * Function definition for UpdateQuantityAmountForEachMatchingQuantity
 */
return &cdm.NonNegativeQuantitySchedule{}
}

func UpdateSpreadAdjustmentAndRateOptions(tradeState cdm.TradeState, instructions cdm.PriceQuantity ) cdm.TradeState {    
/**
 * Function definition for UpdateSpreadAdjustmentAndRateOptions
 */
return cdm.TradeState{}
}

func Update_ProductDirection(before cdm.NonTransferableProduct, originalPayer cdm.CounterpartyRoleEnum, originalReceiver cdm.CounterpartyRoleEnum ) cdm.NonTransferableProduct {    
/**
 * Function definition for Update_ProductDirection
 */
return cdm.NonTransferableProduct{}
}

func ValidateFloatingRateIndexName(floatingRateIndexName cdm.FloatingRateIndexEnum, contractualDefs *cdm.ContractualDefinitionsEnum ) bool {    
/**
 * Function definition for ValidateFloatingRateIndexName
 */
return false
}

func ValidateFpMLCodingSchemeDomain(code cdm.FpMLCodingScheme, domain string ) bool {    
/**
 * Function definition for ValidateFpMLCodingSchemeDomain
 */
return false
}

func VectorGrowthOperation(baseValue float64, factor *float64 ) *float64 {    
/**
 * Function definition for VectorGrowthOperation
 */
return nil
}

func VectorOperation(arithmeticOp cdm.ArithmeticOperationEnum, left *float64, right *float64 ) *float64 {    
/**
 * Function definition for VectorOperation
 */
return nil
}

func VectorScalarOperation(arithmeticOp cdm.ArithmeticOperationEnum, left *float64, right *float64 ) *float64 {    
/**
 * Function definition for VectorScalarOperation
 */
return nil
}

func YearFraction(dayCountFractionEnum cdm.DayCountFractionEnum, startDate time.Time, endDate time.Time, terminationDate *time.Time, periodsInYear *int ) float64 {    
/**
 * Function definition for YearFraction
 */
return 0
}

func YearFractionForOneDay(dcf cdm.DayCountFractionEnum ) float64 {    
/**
 * Function definition for YearFractionForOneDay
 */
return 0
}

