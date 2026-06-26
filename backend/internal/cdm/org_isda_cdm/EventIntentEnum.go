/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumeration values to qualify the intent associated with a transaction event.
   */
  
  const (
  /**
   * The intent is to allocate one or more trades as part of an allocated block trade.
   */
  EventIntentEnum_ALLOCATION EventIntentEnum = iota + 1
  /**
   * The intent is to designate a stand-alone cash transfer as a result of Trade contracual terms e.g. incurred by payout for instance a Performance Amount or a Floating Rate Amount. The particular CashFlow at stake shall be further specified in priceTransferEnum or transferTypeEnum. For clarity, such intentEnum value shall not be used whenever a cash transfer is not stand-alone but is instead embedded in another Event as part of the composable modelling e.g. Decrease with Fees, Cross-Currency Notional Reset, etc. or any other Event whenever including a cash transfer with other features. For clarity, a principal payment related to a Principal Exhange is excluded as well, because a dedicated intentEnum value exists for this event i.e. PrincipalExchange value.
   */
  EventIntentEnum_CASH_FLOW EventIntentEnum = iota + 1
  /**
   * The intent is to clear the contract.
   */
  EventIntentEnum_CLEARING EventIntentEnum = iota + 1
  /**
   * The intent is to compress multiple trades as part of a netting or compression event.
   */
  EventIntentEnum_COMPRESSION EventIntentEnum = iota + 1
  /**
   * The intent is to form a contract from an execution.
   */
  EventIntentEnum_CONTRACT_FORMATION EventIntentEnum = iota + 1
  /**
   * The intent is to amend the terms of the contract through renegotiation.
   */
  EventIntentEnum_CONTRACT_TERMS_AMENDMENT EventIntentEnum = iota + 1
  /**
   * The intent is to take into effect the occurrence of a Corporate Action and the particular Corporate Action at stake shall be further specified in CorporateActionTypeEnum.
   */
  EventIntentEnum_CORPORATE_ACTION_ADJUSTMENT EventIntentEnum = iota + 1
  /**
   * The intent is to take into effect the occurrence of a Credit Event.
   */
  EventIntentEnum_CREDIT_EVENT EventIntentEnum = iota + 1
  /**
   * The intent is to Decrease the quantity or notional of the contract.
   */
  EventIntentEnum_DECREASE EventIntentEnum = iota + 1
  /**
   * The intent is to fully unwind the Trade, as a result of the application of Trade contractual terms (e.g. an obligation to do so before Termination Date as part of any kind of Early Termination terms) as defined within the CDM EarlyTerminationProvision data type. Accordingly, increase and decrease of positions which result from negotiation by the parties shall not be designated by such intentEnum. For clarity, partial exercise of an option before its expiration date is excluded as well, though related to Trade contract terms, because a dedicated intentEnum value exists for this event i.e. OptionExercise value.
   */
  EventIntentEnum_EARLY_TERMINATION_PROVISION EventIntentEnum = iota + 1
  /**
   * The intent is to Increase the quantity or notional of the contract.
   */
  EventIntentEnum_INCREASE EventIntentEnum = iota + 1
  /**
   * The intent is to replace an interest rate index by another one during the life of a trade and add a transition spread on top of this index (and on top of the spreads already defined in the trade, if any). 
   */
  EventIntentEnum_INDEX_TRANSITION EventIntentEnum = iota + 1
  /**
   * The intent is to increase or to decrease the notional of the Trade, in accordance with Notional Reset features e.g. could apply for Cross Currency Swaps, Equity Performance Swaps, etc.
   */
  EventIntentEnum_NOTIONAL_RESET EventIntentEnum = iota + 1
  /**
   * The intent is to increase or to decrease the notional of the Trade, in accordance with Step features attached to a Payout Quantity.
   */
  EventIntentEnum_NOTIONAL_STEP EventIntentEnum = iota + 1
  /**
   * The intent is to novate the contract.
   */
  EventIntentEnum_NOVATION EventIntentEnum = iota + 1
  /**
   * The intent is to record any kind of stand-alone obervervations e.g. internal data recording, usage of CDM for recording and/or exchanging data as part of pricing 'consensus' processing, etc. For clarity, such intentEnum value shall not be used whenever an observation is not stand-alone but is instead embedded in another Event as part of the composable modelling e.g. CashFlow to which an observation of prices is associated, etc.
   */
  EventIntentEnum_OBSERVATION_RECORD EventIntentEnum = iota + 1
  /**
   * The intent is to Exercise a contract that is made of one or several option payout legs. For clarity, such intentEnum value shall not be used whenever an optional right is exercised in relation with a Trade which composition includes other types of payout legs e.g. right to call or to cancel before Termination Date as part of any kind of Early Termination terms other than genuine bermuda or american style features described in option payout. 
   */
  EventIntentEnum_OPTION_EXERCISE EventIntentEnum = iota + 1
  /**
   * The intent is to cancel the trade through exercise of an optional right as defined within the CDM OptionProvision data type.
   */
  EventIntentEnum_OPTIONAL_CANCELLATION EventIntentEnum = iota + 1
  /**
   * The intent is to extend the trade through exercise of an optional right as defined within the CDM OptionProvision data type.
   */
  EventIntentEnum_OPTIONAL_EXTENSION EventIntentEnum = iota + 1
  /**
   * The intent is to rebalance a portfolio, by inserting new derivatives transactions into portfolios of participants to reduce risks linked to those trades. These are offsetting trades that rebalance relationships between different counterparties when it comes to exposure of portfolios to certain types of risk, such as interest rate risk.
   */
  EventIntentEnum_PORTFOLIO_REBALANCING EventIntentEnum = iota + 1
  /**
   * The intent is to pay or to receive a cash transfer, in accordance with Principal Exchange features.
   */
  EventIntentEnum_PRINCIPAL_EXCHANGE EventIntentEnum = iota + 1
  /**
   * The intent is to reallocate one or more trades as part of an allocated block trade.
   */
  EventIntentEnum_REALLOCATION EventIntentEnum = iota + 1
  /**
   * The intent is to close a repo transaction through repurchase.
   */
  EventIntentEnum_REPURCHASE EventIntentEnum = iota + 1
  )    
