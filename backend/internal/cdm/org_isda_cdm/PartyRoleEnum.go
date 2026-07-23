/**
 * This file is auto-generated from the ISDA Common Domain Model, do not edit.
 * Version: 7.0.0-dev.78
 */
package org_isda_cdm
  /**
   * The enumerated values for the party role. The enumerated values go beyond the FpML partyRoleScheme as they also include elements that are part of the FpML Trade, such as the Barrier Determination Agent and the Hedging Party.
   */
  
  const (
  /**
   * Organization responsible for preparing the accounting for the trade.
   */
  PartyRoleEnum_ACCOUNTANT PartyRoleEnum = iota + 1
  /**
   * An agent who lends securities of its principals under stock lending arrangements.
   */
  PartyRoleEnum_AGENT_LENDER PartyRoleEnum = iota + 1
  /**
   * The organization responsible for supplying the allocations for a trade to be allocated to multiple accounts/organizations.
   */
  PartyRoleEnum_ALLOCATION_AGENT PartyRoleEnum = iota + 1
  /**
   * The organization that arranged the trade, i.e. brought together the counterparties.  Synonyms/Alternatives: Inter-dealer broker, agent.
   */
  PartyRoleEnum_ARRANGING_BROKER PartyRoleEnum = iota + 1
  /**
   * The party specified in the related confirmation as Barrier Determination Agent.
   */
  PartyRoleEnum_BARRIER_DETERMINATION_AGENT PartyRoleEnum = iota + 1
  /**
   * The party that lends out securities under stock lending arrangements via an Agent Lender.
   */
  PartyRoleEnum_BENEFICIAL_OWNER PartyRoleEnum = iota + 1
  /**
   * Organization that suffers the economic benefit of the trade.  The beneficiary may be distinct from the principal/counterparty - an example occurs when a hedge fund trades via a prime broker; in this case the principal is the prime broker, but the beneficiary is the hedge fund.  This can be represented as a payer/receiver account in the name of the hedge fund, but it is also possible to add the party role of 'Beneficiary' at the partyTradeInformation level.
   */
  PartyRoleEnum_BENEFICIARY PartyRoleEnum = iota + 1
  /**
   * The entity for which the organization supporting the trade's processing has booked/recorded the trade. This is used in non-reporting workflows situations in which the trade doesn't need to be reported but a firm still wants to specify their own side.
   */
  PartyRoleEnum_BOOKING_PARTY PartyRoleEnum = iota + 1
  /**
   * The party that borrows securities under stock lending arrangements.
   */
  PartyRoleEnum_BORROWER PartyRoleEnum = iota + 1
  /**
   * Acquirer of the legal title to the financial instrument. In the case of an option, the buyer is the holder of the option. In the case of a swap or forward, the buyer will be determined by industry best practice.  This does not refer to an investor or investment manager or other organization on what is typically called  the 'Buy side'; for that, see the 'Client' role. Corresponds to 'Buyer' as defined in certain regulations such as ESMA MiFID II/MIFIR RTS 22 field 9.
   */
  PartyRoleEnum_BUYER PartyRoleEnum = iota + 1
  /**
   * The party or person who, having legal authority to act on behalf of the trade counterparty acting as Buyer as defined in this coding scheme, made the decision to acquire the financial instrument. Corresponds to 'buyer decision maker' as defined in ESMA's MIFIR RTS 23 report. This does not refer to the decision maker for what is traditionally called the 'Buy side'; for that, see the 'Client Decision Maker' role.
   */
  PartyRoleEnum_BUYER_DECISION_MAKER PartyRoleEnum = iota + 1
  /**
   * The party that provides credit support under English Law.
   */
  PartyRoleEnum_CHARGOR PartyRoleEnum = iota + 1
  /**
   * An organization that clears trades through a clearing house, via a clearing broker (member of the clearing house) who acts as an agent on its behalf. The term 'client' refers to the organization's role in the clearing process in relation to its clearing broker, and not whether it is a price maker or taker in the execution process.
   */
  PartyRoleEnum_CLEARING_CLIENT PartyRoleEnum = iota + 1
  /**
   * A party to the trade that claims a clearing exception, such as an end-user exception under Dodd-Frank Act provisions.
   */
  PartyRoleEnum_CLEARING_EXCEPTION_PARTY PartyRoleEnum = iota + 1
  /**
   * Organization that submits the trade to a clearing house on behalf of the principal. Synonyms/alternates: Futures Commission Merchant (FCM), Clearing Broker, Clearing Member Firm. Some implementations use 'Clearing Broker' as synonym.
   */
  PartyRoleEnum_CLEARING_FIRM PartyRoleEnum = iota + 1
  /**
   * The organization that acts as a central counterparty to clear a derivatives contract.  This is used to represent the role of Central Counterparties (CCPs) or Derivative Clearing Organizations (DCOs).  Sometimes called 'ClearingService'. Some implementations also use the term 'Clearer'.
   */
  PartyRoleEnum_CLEARING_ORGANIZATION PartyRoleEnum = iota + 1
  /**
   * Client as defined under ESMA MIFIR. This is generally the investor or other client of an investment firm, and is synonymous with the Beneficiary in many circumstances.
   */
  PartyRoleEnum_CLIENT PartyRoleEnum = iota + 1
  /**
   * The party or person who, having legal authority to act on behalf of a trade counterparty, made the decision to acquire or sell the financial instrument.
   */
  PartyRoleEnum_CLIENT_DECISION_MAKER PartyRoleEnum = iota + 1
  /**
   * Organization serving as a financial intermediary for the purposes of electronic confirmation or providing services for post-processing of transactional data.
   */
  PartyRoleEnum_CONFIRMATION_PLATFORM PartyRoleEnum = iota + 1
  /**
   * A party to a contractual document.  If the intended usage relates to the context of the trade lifecycle, more specific annotations have been defined which might be more appropriate.
   */
  PartyRoleEnum_CONTRACTUAL_PARTY PartyRoleEnum = iota + 1
  /**
   * Organization officially attached to the counterparty. e.g. partner, branch, subsidiary.
   */
  PartyRoleEnum_COUNTER_PARTY_AFFILIATE PartyRoleEnum = iota + 1
  /**
   * The topmost entity or organization, within the corporate hierarchy, responsible for the reporting party.
   */
  PartyRoleEnum_COUNTER_PARTY_ULTIMATE_PARENT PartyRoleEnum = iota + 1
  /**
   * An economic counterparty to the trade. Synonym: principal.
   */
  PartyRoleEnum_COUNTERPARTY PartyRoleEnum = iota + 1
  /**
   * Organization that enhances the credit of another organization (similar to guarantor, but may not fully guarantee the obligation).
   */
  PartyRoleEnum_CREDIT_SUPPORT_PROVIDER PartyRoleEnum = iota + 1
  /**
   * Organization that maintains custody of the asset represented by the trade on behalf of the owner/principal.
   */
  PartyRoleEnum_CUSTODIAN PartyRoleEnum = iota + 1
  /**
   * Entity submitting the transaction report to the competent authority.
   */
  PartyRoleEnum_DATA_SUBMITTER PartyRoleEnum = iota + 1
  /**
   * The party referenced is specified in the contract confirmation as Determination Party.
   */
  PartyRoleEnum_DETERMINING_PARTY PartyRoleEnum = iota + 1
  /**
   * Organization that is disputing the trade or transaction.
   */
  PartyRoleEnum_DISPUTING_PARTY PartyRoleEnum = iota + 1
  /**
   * A marketplace organization which purpose is to maintain document records.  If the intended usage relates to the context of the trade lifecycle, more specific annotations have been defined which might be more appropriate.
   */
  PartyRoleEnum_DOCUMENT_REPOSITORY PartyRoleEnum = iota + 1
  /**
   * The (generally sell-side) organization that executed the trade; the price-making party.
   */
  PartyRoleEnum_EXECUTING_BROKER PartyRoleEnum = iota + 1
  /**
   * Entity executing the transaction.  If the transaction is executed directly by the reporting party, it will be the reporting party.  If it is executed by an execution agent or an affiliated party on behalf of the reporting party, it will be that affiliate or agent.
   */
  PartyRoleEnum_EXECUTING_ENTITY PartyRoleEnum = iota + 1
  /**
   * The (generally buy-side) organization that acts to execute trades on behalf of an investor. Typically this is an investment manager or asset manager, and also makes the investment decisions for the investor. If required, a separate InvestmentDecision role can be specified to distinguish that the party making the investment decision is different.
   */
  PartyRoleEnum_EXECUTION_AGENT PartyRoleEnum = iota + 1
  /**
   * The facility, exchange, or market where the trade was executed. Synonym: Swap Execution Facility, Designated Contract Market, Execution Venue.
   */
  PartyRoleEnum_EXECUTION_FACILITY PartyRoleEnum = iota + 1
  /**
   * Organization that backs (guarantees) the credit risk of the trade.
   */
  PartyRoleEnum_GUARANTOR PartyRoleEnum = iota + 1
  /**
   * The ISDA Hedging Party that is specified in the related confirmation as Hedging, or if no Hedging Party is specified, either party to the contract.
   */
  PartyRoleEnum_HEDGING_PARTY PartyRoleEnum = iota + 1
  /**
   * The party that lends out securities under stock lending arrangements as principal.
   */
  PartyRoleEnum_LENDER PartyRoleEnum = iota + 1
  /**
   * Margin affiliate as defined by U.S. margin and capital rules §23.151.
   */
  PartyRoleEnum_MARGIN_AFFILIATE PartyRoleEnum = iota + 1
  /**
   * The entity transmitting the order to the reporting firm. Synonym: Transmitting Firm.
   */
  PartyRoleEnum_ORDER_TRANSMITTER PartyRoleEnum = iota + 1
  /**
   * The party that acts as either a portfolio compression or portfolio rebalancing service provider, as defined under ESMA MIFIR
   */
  PartyRoleEnum_PTRR_SERVICE_PROVIDER PartyRoleEnum = iota + 1
  /**
   * The party that provides credit support under New York Law.
   */
  PartyRoleEnum_PLEDGOR PartyRoleEnum = iota + 1
  /**
   * The organization that takes on or took on the credit risk for this trade by stepping in between the two economic parties (without a central counterparty clearing mechanism).
   */
  PartyRoleEnum_PRIME_BROKER PartyRoleEnum = iota + 1
  /**
   * The trade repository at which the trade was reported previous to the current trade repository.
   */
  PartyRoleEnum_PRIOR_TRADE_REPOSITORY PartyRoleEnum = iota + 1
  /**
   * The reporting service (whether trade repository, market data service, or exchange/facility/venue data distribution service) that published the report of this trade.
   */
  PartyRoleEnum_PUBLICATION_VENUE PartyRoleEnum = iota + 1
  /**
   * The party with the regulatory responsibility to report this trade.
   */
  PartyRoleEnum_REPORTING_PARTY PartyRoleEnum = iota + 1
  /**
   * Organization officially attached to the reporting party  e.g. partner, branch, subsidiary.
   */
  PartyRoleEnum_REPORTING_PARTY_AFFILIATE PartyRoleEnum = iota + 1
  /**
   * The topmost entity or organization, within the corporate hierarchy, responsible for the reporting party.
   */
  PartyRoleEnum_REPORTING_PARTY_ULTIMATE_PARENT PartyRoleEnum = iota + 1
  /**
   * The party that receives credit support under New York or English Law.
   */
  PartyRoleEnum_SECURED_PARTY PartyRoleEnum = iota + 1
  /**
   * A counterparty in a trade, which performs in one of the following capacities: 1) it transfers or agrees to transfer in the future an instrument or title to that instrument in exchange for payment, 2) it writes a derivatives instrument such as an option or a swap in which it provides risk protection to the buyer. This does not refer to the broker/dealer or other organization on what is typically called  the 'Sell side'; for that, see the 'Executing Broker' role. Corresponds to 'Seller' as defined in certain regulations such as ESMA MiFID II/MIFIR RTS 22 field 16.
   */
  PartyRoleEnum_SELLER PartyRoleEnum = iota + 1
  /**
   * The party or person who, having legal authority to act on behalf of the trade counterparty acting as Seller as defined in this coding scheme, made the decision to sell the financial instrument. Corresponds to 'seller decision maker' as defined in ESMA's MIFIR RTS 23 report. This does not refer to the decision maker for what is traditionally called the 'Sell side'; for that, see the 'Trader' person role.
   */
  PartyRoleEnum_SELLER_DECISION_MAKER PartyRoleEnum = iota + 1
  /**
   * The organization that makes or receives payments on behalf of the given principal party.
   */
  PartyRoleEnum_SETTLEMENT_AGENT PartyRoleEnum = iota + 1
  /**
   * A third-party providing custody, settlement, segregation and reporting services.
   */
  PartyRoleEnum_THIRD_PARTY_CUSTODIAN PartyRoleEnum = iota + 1
  /**
   * An organization that maintains records of the trade for regulatory reporting purposes.
   */
  PartyRoleEnum_TRADE_REPOSITORY PartyRoleEnum = iota + 1
  /**
   * The organization that originally supplied the record of the trade. In the context of regulatory reporting, it is the submitter of the trade record to a regulator or TR.
   */
  PartyRoleEnum_TRADE_SOURCE PartyRoleEnum = iota + 1
  /**
   * The entity responsible for managing the assets/investments of this party.  Synonym:  Asset Manager, Investment Manager, Trading Advisory.
   */
  PartyRoleEnum_TRADING_MANAGER PartyRoleEnum = iota + 1
  /**
   * An entity with which this party trades from time to time, ie. with which it acts as a counterparty on some transactions.   This role is used for static reference data, not individual transactions.
   */
  PartyRoleEnum_TRADING_PARTNER PartyRoleEnum = iota + 1
  /**
   * A third party entity that acts as an intermediary and provides automated clearing, settlement, allocation, valuation and reporting services.
   */
  PartyRoleEnum_TRIPARTY_AGENT PartyRoleEnum = iota + 1
  )    
