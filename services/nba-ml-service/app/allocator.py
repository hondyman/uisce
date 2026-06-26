"""
Alternative Investment Allocation Engine
Institutional-grade portfolio construction for alternative investments
"""

from dataclasses import dataclass, field
from typing import List, Dict, Optional, Tuple
from enum import Enum
from datetime import datetime, date
import numpy as np
from uuid import UUID
import json


class InvestmentType(str, Enum):
    PRIVATE_EQUITY = "PRIVATE_EQUITY"
    VENTURE_CAPITAL = "VENTURE_CAPITAL"
    REAL_ESTATE = "REAL_ESTATE"
    HEDGE_FUND = "HEDGE_FUND"
    PRIVATE_CREDIT = "PRIVATE_CREDIT"
    INFRASTRUCTURE = "INFRASTRUCTURE"
    NATURAL_RESOURCES = "NATURAL_RESOURCES"
    SECONDARIES = "SECONDARIES"
    CO_INVESTMENT = "CO_INVESTMENT"
    DIRECT_INVESTMENT = "DIRECT_INVESTMENT"


class RiskLevel(str, Enum):
    LOW = "LOW"
    MEDIUM = "MEDIUM"
    HIGH = "HIGH"
    CRITICAL = "CRITICAL"


@dataclass
class AlternativePosition:
    """Represents a single alternative investment position"""
    investment_id: str
    investment_type: InvestmentType
    fund_name: str
    vintage_year: Optional[int]
    commitment_amount: float
    unfunded_commitment: float
    current_nav: float
    irr: Optional[float] = None
    tvpi: Optional[float] = None
    dpi: Optional[float] = None
    general_partner: Optional[str] = None


@dataclass
class Portfolio:
    """Represents a client's complete portfolio"""
    client_id: str
    total_value: float
    liquid_assets: float
    alternatives: List[AlternativePosition]
    risk_tolerance: float  # 1-10 scale
    time_horizon_years: int = 10
    tax_bracket: float = 0.37
    
    @property
    def total_alt_value(self) -> float:
        return sum(pos.current_nav for pos in self.alternatives)
    
    @property
    def total_unfunded(self) -> float:
        return sum(pos.unfunded_commitment for pos in self.alternatives)
    
    @property
    def alt_allocation_pct(self) -> float:
        if self.total_value == 0:
            return 0
        return (self.total_alt_value / self.total_value) * 100


@dataclass
class InvestmentOpportunity:
    """Represents an investment opportunity being evaluated"""
    opportunity_id: str
    opportunity_type: InvestmentType
    fund_name: str
    general_partner: str
    vintage_year: int
    minimum_commitment: float
    target_irr: float
    target_tvpi: float
    management_fee: float = 0.02
    carried_interest: float = 0.20
    fund_size: Optional[float] = None
    strategy: Optional[str] = None


@dataclass
class AllocationRecommendation:
    """Allocation recommendation output"""
    opportunity_id: str
    recommended_amount: float
    recommended_pct_of_portfolio: float
    current_alt_exposure_pct: float
    post_allocation_alt_exposure_pct: float
    diversification_benefit_score: float
    liquidity_impact_score: float
    risk_budget_utilization_pct: float
    overall_score: float
    fit_score: float
    timing_score: float
    rationale: str
    pros: List[str]
    cons: List[str]
    risks: List[str]
    constraints_violated: List[str]
    model_confidence: float


@dataclass
class AllocationConstraints:
    """Investment policy constraints"""
    max_alternatives_pct: float = 20.0
    max_single_fund_pct: float = 5.0
    max_single_manager_pct: float = 10.0
    max_single_strategy_pct: float = 15.0
    max_vintage_concentration_pct: float = 25.0
    min_liquid_assets_pct: float = 20.0
    max_unfunded_to_liquid_ratio: float = 0.5
    min_manager_track_record_years: int = 5
    max_fund_size_for_emerging: float = 500_000_000


class AlternativeInvestmentAllocator:
    """
    Institutional-grade portfolio construction for alternative investments.
    
    This allocator provides risk-adjusted recommendations considering:
    - Portfolio concentration limits
    - Liquidity constraints
    - Diversification benefits
    - Vintage year distribution
    - Manager and strategy concentration
    - Risk budget utilization
    """
    
    # Default target allocations by asset class
    DEFAULT_TARGET_ALLOCATIONS = {
        InvestmentType.PRIVATE_EQUITY: 0.08,
        InvestmentType.VENTURE_CAPITAL: 0.03,
        InvestmentType.REAL_ESTATE: 0.05,
        InvestmentType.HEDGE_FUND: 0.05,
        InvestmentType.PRIVATE_CREDIT: 0.03,
        InvestmentType.INFRASTRUCTURE: 0.02,
        InvestmentType.NATURAL_RESOURCES: 0.01,
        InvestmentType.SECONDARIES: 0.01,
    }
    
    # Correlation assumptions between asset classes (simplified)
    CORRELATION_MATRIX = {
        InvestmentType.PRIVATE_EQUITY: {
            InvestmentType.PRIVATE_EQUITY: 1.0,
            InvestmentType.VENTURE_CAPITAL: 0.6,
            InvestmentType.REAL_ESTATE: 0.3,
            InvestmentType.HEDGE_FUND: 0.5,
            InvestmentType.PRIVATE_CREDIT: 0.4,
        },
        InvestmentType.VENTURE_CAPITAL: {
            InvestmentType.PRIVATE_EQUITY: 0.6,
            InvestmentType.VENTURE_CAPITAL: 1.0,
            InvestmentType.REAL_ESTATE: 0.2,
            InvestmentType.HEDGE_FUND: 0.4,
            InvestmentType.PRIVATE_CREDIT: 0.3,
        },
        InvestmentType.REAL_ESTATE: {
            InvestmentType.PRIVATE_EQUITY: 0.3,
            InvestmentType.VENTURE_CAPITAL: 0.2,
            InvestmentType.REAL_ESTATE: 1.0,
            InvestmentType.HEDGE_FUND: 0.3,
            InvestmentType.PRIVATE_CREDIT: 0.5,
        },
        InvestmentType.HEDGE_FUND: {
            InvestmentType.PRIVATE_EQUITY: 0.5,
            InvestmentType.VENTURE_CAPITAL: 0.4,
            InvestmentType.REAL_ESTATE: 0.3,
            InvestmentType.HEDGE_FUND: 1.0,
            InvestmentType.PRIVATE_CREDIT: 0.4,
        },
        InvestmentType.PRIVATE_CREDIT: {
            InvestmentType.PRIVATE_EQUITY: 0.4,
            InvestmentType.VENTURE_CAPITAL: 0.3,
            InvestmentType.REAL_ESTATE: 0.5,
            InvestmentType.HEDGE_FUND: 0.4,
            InvestmentType.PRIVATE_CREDIT: 1.0,
        },
    }
    
    def __init__(
        self, 
        constraints: Optional[AllocationConstraints] = None,
        target_allocations: Optional[Dict[InvestmentType, float]] = None
    ):
        self.constraints = constraints or AllocationConstraints()
        self.target_allocations = target_allocations or self.DEFAULT_TARGET_ALLOCATIONS
    
    def recommend_allocation(
        self, 
        portfolio: Portfolio, 
        opportunity: InvestmentOpportunity
    ) -> AllocationRecommendation:
        """
        Generate an allocation recommendation for an investment opportunity.
        
        Args:
            portfolio: Client's current portfolio
            opportunity: Investment opportunity being evaluated
            
        Returns:
            AllocationRecommendation with sizing and analysis
        """
        # Initialize tracking
        pros = []
        cons = []
        risks = []
        constraints_violated = []
        
        # 1. Calculate current allocations
        current_alt_exposure = portfolio.alt_allocation_pct
        
        # 2. Calculate risk budget
        risk_budget_remaining = self._calculate_risk_budget(portfolio)
        
        # 3. Diversification analysis
        diversification_score = self._calculate_diversification_benefit(
            portfolio, opportunity
        )
        
        # 4. Liquidity stress test
        liquidity_score = self._simulate_liquidity_scenario(
            portfolio, opportunity.minimum_commitment
        )
        
        # 5. Constraint checking and sizing
        max_by_portfolio_pct = portfolio.total_value * (self.constraints.max_single_fund_pct / 100)
        max_by_risk_budget = risk_budget_remaining * 0.25  # Max 25% of remaining risk budget per deal
        max_by_liquidity = self._max_by_liquidity(portfolio, opportunity)
        max_by_concentration = self._max_by_concentration(portfolio, opportunity)
        
        # Calculate recommended amount
        sizing_limits = [
            opportunity.minimum_commitment,
            max_by_portfolio_pct,
            max_by_risk_budget,
            max_by_liquidity,
            max_by_concentration,
        ]
        
        recommended_amount = min(sizing_limits)
        
        # Check if we can't meet minimum
        if recommended_amount < opportunity.minimum_commitment:
            constraints_violated.append(
                f"Cannot meet minimum commitment of ${opportunity.minimum_commitment:,.0f}"
            )
            recommended_amount = 0
        
        # 6. Check all constraints
        self._check_constraints(
            portfolio, opportunity, recommended_amount, constraints_violated
        )
        
        # 7. Calculate post-allocation metrics
        post_alt_exposure = self._calculate_post_allocation_exposure(
            portfolio, recommended_amount
        )
        
        # 8. Score the opportunity
        fit_score = self._calculate_fit_score(portfolio, opportunity)
        timing_score = self._calculate_timing_score(portfolio, opportunity)
        
        overall_score = (
            fit_score * 0.3 +
            timing_score * 0.2 +
            diversification_score * 0.25 +
            liquidity_score * 0.25
        ) * 100
        
        # 9. Build pros/cons
        self._analyze_pros_cons(
            portfolio, opportunity, diversification_score, 
            liquidity_score, pros, cons, risks
        )
        
        # 10. Generate rationale
        rationale = self._generate_rationale(
            portfolio, opportunity, recommended_amount, 
            diversification_score, liquidity_score
        )
        
        # 11. Calculate confidence
        confidence = self._calculate_confidence(
            portfolio, constraints_violated, fit_score
        )
        
        return AllocationRecommendation(
            opportunity_id=opportunity.opportunity_id,
            recommended_amount=recommended_amount,
            recommended_pct_of_portfolio=(recommended_amount / portfolio.total_value * 100) if portfolio.total_value > 0 else 0,
            current_alt_exposure_pct=current_alt_exposure,
            post_allocation_alt_exposure_pct=post_alt_exposure,
            diversification_benefit_score=diversification_score * 100,
            liquidity_impact_score=liquidity_score * 100,
            risk_budget_utilization_pct=(recommended_amount / risk_budget_remaining * 100) if risk_budget_remaining > 0 else 0,
            overall_score=overall_score,
            fit_score=fit_score * 100,
            timing_score=timing_score * 100,
            rationale=rationale,
            pros=pros,
            cons=cons,
            risks=risks,
            constraints_violated=constraints_violated,
            model_confidence=confidence,
        )
    
    def _calculate_risk_budget(self, portfolio: Portfolio) -> float:
        """Calculate remaining risk budget for alternatives"""
        target_alt_pct = self.constraints.max_alternatives_pct
        current_alt_pct = portfolio.alt_allocation_pct
        remaining_pct = max(0, target_alt_pct - current_alt_pct)
        return portfolio.total_value * (remaining_pct / 100)
    
    def _calculate_diversification_benefit(
        self, 
        portfolio: Portfolio, 
        opportunity: InvestmentOpportunity
    ) -> float:
        """
        Calculate diversification benefit of adding this opportunity.
        Returns score from 0 to 1 (higher = more diversification benefit)
        """
        if not portfolio.alternatives:
            return 0.9  # First alternative investment gets high diversification score
        
        # Get existing asset classes
        existing_types = set(pos.investment_type for pos in portfolio.alternatives)
        
        # New asset class = high diversification
        if opportunity.opportunity_type not in existing_types:
            return 0.85
        
        # Calculate average correlation with existing positions
        correlations = []
        for pos in portfolio.alternatives:
            try:
                corr = self.CORRELATION_MATRIX.get(
                    opportunity.opportunity_type, {}
                ).get(pos.investment_type, 0.5)
                correlations.append(corr)
            except (KeyError, TypeError):
                correlations.append(0.5)
        
        avg_correlation = np.mean(correlations) if correlations else 0.5
        
        # Lower correlation = higher diversification benefit
        diversification_score = 1 - avg_correlation
        
        # Adjust for vintage year diversification
        existing_vintages = [
            pos.vintage_year for pos in portfolio.alternatives 
            if pos.vintage_year is not None
        ]
        if opportunity.vintage_year not in existing_vintages:
            diversification_score += 0.1
        
        return min(diversification_score, 1.0)
    
    def _simulate_liquidity_scenario(
        self, 
        portfolio: Portfolio, 
        commitment_amount: float
    ) -> float:
        """
        Simulate liquidity impact of new commitment.
        Returns score from 0 to 1 (higher = less liquidity impact)
        """
        # Current liquidity metrics
        current_liquidity_ratio = (
            portfolio.liquid_assets / portfolio.total_value 
            if portfolio.total_value > 0 else 0
        )
        
        # Post-commitment liquidity (assuming immediate capital call)
        post_liquid = portfolio.liquid_assets - commitment_amount
        post_liquidity_ratio = (
            post_liquid / portfolio.total_value 
            if portfolio.total_value > 0 else 0
        )
        
        # Unfunded commitment impact
        total_unfunded_post = portfolio.total_unfunded + commitment_amount
        unfunded_to_liquid_ratio = (
            total_unfunded_post / portfolio.liquid_assets 
            if portfolio.liquid_assets > 0 else float('inf')
        )
        
        # Score based on:
        # 1. Maintaining minimum liquid assets
        if post_liquidity_ratio < self.constraints.min_liquid_assets_pct / 100:
            liquidity_score = 0.3
        elif post_liquidity_ratio < (self.constraints.min_liquid_assets_pct * 1.5) / 100:
            liquidity_score = 0.6
        else:
            liquidity_score = 0.9
        
        # 2. Unfunded commitment ratio
        if unfunded_to_liquid_ratio > self.constraints.max_unfunded_to_liquid_ratio:
            liquidity_score *= 0.7
        
        return min(max(liquidity_score, 0), 1.0)
    
    def _max_by_liquidity(
        self, 
        portfolio: Portfolio, 
        opportunity: InvestmentOpportunity
    ) -> float:
        """Calculate maximum allocation based on liquidity constraints"""
        # Maximum that maintains min liquid assets
        min_liquid_required = portfolio.total_value * (self.constraints.min_liquid_assets_pct / 100)
        max_from_liquid = portfolio.liquid_assets - min_liquid_required
        
        # Maximum based on unfunded commitment ratio
        max_unfunded = portfolio.liquid_assets * self.constraints.max_unfunded_to_liquid_ratio
        remaining_unfunded_capacity = max(0, max_unfunded - portfolio.total_unfunded)
        
        return max(0, min(max_from_liquid, remaining_unfunded_capacity))
    
    def _max_by_concentration(
        self, 
        portfolio: Portfolio, 
        opportunity: InvestmentOpportunity
    ) -> float:
        """Calculate maximum allocation based on concentration limits"""
        # Manager concentration
        manager_exposure = sum(
            pos.current_nav for pos in portfolio.alternatives
            if pos.general_partner == opportunity.general_partner
        )
        max_manager = (
            portfolio.total_value * (self.constraints.max_single_manager_pct / 100) 
            - manager_exposure
        )
        
        # Strategy concentration
        strategy_exposure = sum(
            pos.current_nav for pos in portfolio.alternatives
            if pos.investment_type == opportunity.opportunity_type
        )
        max_strategy = (
            portfolio.total_value * (self.constraints.max_single_strategy_pct / 100) 
            - strategy_exposure
        )
        
        # Vintage concentration
        vintage_exposure = sum(
            pos.current_nav for pos in portfolio.alternatives
            if pos.vintage_year == opportunity.vintage_year
        )
        max_vintage = (
            portfolio.total_value * (self.constraints.max_vintage_concentration_pct / 100) 
            - vintage_exposure
        )
        
        return max(0, min(max_manager, max_strategy, max_vintage))
    
    def _check_constraints(
        self,
        portfolio: Portfolio,
        opportunity: InvestmentOpportunity,
        amount: float,
        violations: List[str]
    ) -> None:
        """Check all investment constraints and add violations"""
        # Overall alternatives limit
        post_alt_pct = self._calculate_post_allocation_exposure(portfolio, amount)
        if post_alt_pct > self.constraints.max_alternatives_pct:
            violations.append(
                f"Would exceed max alternatives allocation ({post_alt_pct:.1f}% > {self.constraints.max_alternatives_pct}%)"
            )
        
        # Single position limit
        position_pct = (amount / portfolio.total_value * 100) if portfolio.total_value > 0 else 0
        if position_pct > self.constraints.max_single_fund_pct:
            violations.append(
                f"Position size ({position_pct:.1f}%) exceeds single fund limit ({self.constraints.max_single_fund_pct}%)"
            )
        
        # Liquidity constraint
        post_liquid = portfolio.liquid_assets - amount
        post_liquid_pct = (post_liquid / portfolio.total_value * 100) if portfolio.total_value > 0 else 0
        if post_liquid_pct < self.constraints.min_liquid_assets_pct:
            violations.append(
                f"Would breach minimum liquidity ({post_liquid_pct:.1f}% < {self.constraints.min_liquid_assets_pct}%)"
            )
    
    def _calculate_post_allocation_exposure(
        self, 
        portfolio: Portfolio, 
        amount: float
    ) -> float:
        """Calculate post-allocation alternatives exposure percentage"""
        post_alt_value = portfolio.total_alt_value + amount
        return (post_alt_value / portfolio.total_value * 100) if portfolio.total_value > 0 else 0
    
    def _calculate_fit_score(
        self, 
        portfolio: Portfolio, 
        opportunity: InvestmentOpportunity
    ) -> float:
        """Calculate how well opportunity fits portfolio needs"""
        score = 0.5  # Base score
        
        # Target allocation fit
        target = self.target_allocations.get(opportunity.opportunity_type, 0.05)
        current_type_allocation = sum(
            pos.current_nav for pos in portfolio.alternatives
            if pos.investment_type == opportunity.opportunity_type
        ) / portfolio.total_value if portfolio.total_value > 0 else 0
        
        if current_type_allocation < target:
            score += 0.2  # Under-allocated to this type
        
        # Risk tolerance alignment
        high_risk_types = {InvestmentType.VENTURE_CAPITAL, InvestmentType.PRIVATE_EQUITY}
        if opportunity.opportunity_type in high_risk_types:
            if portfolio.risk_tolerance >= 7:
                score += 0.15
            elif portfolio.risk_tolerance <= 4:
                score -= 0.15
        
        # Time horizon alignment
        if portfolio.time_horizon_years >= 10:
            score += 0.1  # Long horizon suits illiquid investments
        elif portfolio.time_horizon_years <= 5:
            score -= 0.2  # Short horizon is concerning
        
        # Target return alignment
        if opportunity.target_irr >= 15 and portfolio.risk_tolerance >= 6:
            score += 0.1
        
        return min(max(score, 0), 1.0)
    
    def _calculate_timing_score(
        self, 
        portfolio: Portfolio, 
        opportunity: InvestmentOpportunity
    ) -> float:
        """Calculate timing score based on vintage year and market conditions"""
        score = 0.6  # Base score
        
        current_year = datetime.now().year
        
        # Vintage year analysis
        if opportunity.vintage_year == current_year or opportunity.vintage_year == current_year + 1:
            score += 0.15  # Current/next year vintage is timely
        
        # Check vintage year distribution in portfolio
        existing_vintages = {}
        for pos in portfolio.alternatives:
            if pos.vintage_year:
                existing_vintages[pos.vintage_year] = existing_vintages.get(pos.vintage_year, 0) + 1
        
        # Bonus for underrepresented vintage years
        if opportunity.vintage_year not in existing_vintages:
            score += 0.15
        elif existing_vintages.get(opportunity.vintage_year, 0) < 2:
            score += 0.05
        
        # Fee analysis (lower fees = better timing)
        if opportunity.management_fee <= 0.015:
            score += 0.1
        elif opportunity.management_fee >= 0.025:
            score -= 0.1
        
        return min(max(score, 0), 1.0)
    
    def _analyze_pros_cons(
        self,
        portfolio: Portfolio,
        opportunity: InvestmentOpportunity,
        diversification_score: float,
        liquidity_score: float,
        pros: List[str],
        cons: List[str],
        risks: List[str]
    ) -> None:
        """Analyze and populate pros, cons, and risks"""
        # Diversification
        if diversification_score >= 0.7:
            pros.append("Strong diversification benefit to portfolio")
        elif diversification_score <= 0.4:
            cons.append("Limited diversification benefit due to existing exposure")
        
        # Liquidity
        if liquidity_score >= 0.8:
            pros.append("Minimal impact on portfolio liquidity")
        elif liquidity_score <= 0.5:
            cons.append("Significant liquidity impact - monitor cash reserves")
            risks.append("LIQUIDITY_CONSTRAINT")
        
        # Return profile
        if opportunity.target_irr >= 20:
            pros.append(f"Attractive target return ({opportunity.target_irr}% IRR)")
        
        # Fee structure
        if opportunity.management_fee <= 0.015:
            pros.append("Competitive fee structure")
        elif opportunity.management_fee >= 0.025:
            cons.append("Above-market management fees")
        
        # Vintage year
        existing_vintages = [p.vintage_year for p in portfolio.alternatives if p.vintage_year]
        if opportunity.vintage_year not in existing_vintages:
            pros.append(f"Adds {opportunity.vintage_year} vintage exposure")
        
        # Standard risks
        risks.append("ILLIQUIDITY_5YR+")
        if opportunity.opportunity_type in {InvestmentType.VENTURE_CAPITAL, InvestmentType.PRIVATE_EQUITY}:
            risks.append("J_CURVE_EFFECT")
        
        # Concentration
        manager_exposure = sum(
            pos.current_nav for pos in portfolio.alternatives
            if pos.general_partner == opportunity.general_partner
        )
        if manager_exposure > 0:
            risks.append("MANAGER_CONCENTRATION")
    
    def _generate_rationale(
        self,
        portfolio: Portfolio,
        opportunity: InvestmentOpportunity,
        amount: float,
        diversification_score: float,
        liquidity_score: float
    ) -> str:
        """Generate human-readable rationale for the recommendation"""
        if amount == 0:
            return (
                f"A commitment to {opportunity.fund_name} is not recommended at this time "
                f"due to constraint violations. Current alternatives exposure is "
                f"{portfolio.alt_allocation_pct:.1f}% and liquidity constraints "
                f"would be stressed."
            )
        
        pct_of_portfolio = (amount / portfolio.total_value * 100) if portfolio.total_value > 0 else 0
        
        return (
            f"Recommend a ${amount:,.0f} commitment ({pct_of_portfolio:.1f}% of portfolio) "
            f"to {opportunity.fund_name}. This {opportunity.opportunity_type.value} investment "
            f"offers a target {opportunity.target_irr}% IRR and provides "
            f"{'strong' if diversification_score > 0.7 else 'moderate'} diversification benefits "
            f"(score: {diversification_score*100:.0f}). Portfolio liquidity remains "
            f"{'healthy' if liquidity_score > 0.7 else 'adequate'} post-commitment "
            f"(score: {liquidity_score*100:.0f})."
        )
    
    def _calculate_confidence(
        self,
        portfolio: Portfolio,
        violations: List[str],
        fit_score: float
    ) -> float:
        """Calculate model confidence in the recommendation"""
        confidence = 0.85  # Base confidence
        
        # Reduce for violations
        confidence -= len(violations) * 0.1
        
        # Reduce for sparse portfolio data
        if len(portfolio.alternatives) < 3:
            confidence -= 0.1  # Less data to analyze
        
        # Reduce for extreme fit scores
        if fit_score < 0.3 or fit_score > 0.9:
            confidence -= 0.05
        
        return min(max(confidence, 0.3), 0.95)
    
    def generate_recommendations(
        self,
        portfolio: Portfolio,
        opportunities: List[InvestmentOpportunity]
    ) -> List[AllocationRecommendation]:
        """
        Generate recommendations for multiple opportunities.
        Ranked by overall score.
        """
        recommendations = []
        
        for opp in opportunities:
            rec = self.recommend_allocation(portfolio, opp)
            recommendations.append(rec)
        
        # Sort by overall score descending
        recommendations.sort(key=lambda r: r.overall_score, reverse=True)
        
        return recommendations
    
    def suggest_diversification_opportunities(
        self,
        portfolio: Portfolio
    ) -> List[Dict]:
        """
        Analyze portfolio and suggest types of investments to consider.
        """
        suggestions = []
        
        # Calculate current allocation by type
        type_allocations = {}
        for pos in portfolio.alternatives:
            type_allocations[pos.investment_type] = (
                type_allocations.get(pos.investment_type, 0) + pos.current_nav
            )
        
        # Compare to targets
        for inv_type, target in self.target_allocations.items():
            current = type_allocations.get(inv_type, 0) / portfolio.total_value if portfolio.total_value > 0 else 0
            gap = target - current
            
            if gap > 0.01:  # More than 1% underweight
                suggestions.append({
                    'strategy': inv_type.value,
                    'current_allocation_pct': current * 100,
                    'target_allocation_pct': target * 100,
                    'gap_pct': gap * 100,
                    'recommended_amount': portfolio.total_value * gap,
                    'priority': 'HIGH' if gap > 0.03 else 'MEDIUM'
                })
        
        # Sort by gap size
        suggestions.sort(key=lambda s: s['gap_pct'], reverse=True)
        
        return suggestions
    
    def vintage_year_analysis(
        self,
        portfolio: Portfolio
    ) -> Dict:
        """
        Analyze vintage year distribution and identify gaps.
        """
        current_year = datetime.now().year
        vintage_distribution = {}
        
        for pos in portfolio.alternatives:
            if pos.vintage_year:
                vintage_distribution[pos.vintage_year] = (
                    vintage_distribution.get(pos.vintage_year, 0) + pos.current_nav
                )
        
        # Calculate percentages
        total_alt = portfolio.total_alt_value
        vintage_pcts = {
            year: (value / total_alt * 100) if total_alt > 0 else 0
            for year, value in vintage_distribution.items()
        }
        
        # Identify gaps in recent years
        missing_vintages = [
            year for year in range(current_year - 5, current_year + 2)
            if year not in vintage_distribution
        ]
        
        return {
            'distribution': vintage_pcts,
            'missing_vintages': missing_vintages,
            'recommendation': f"Consider adding {missing_vintages[0]} vintage exposure" if missing_vintages else "Vintage year coverage is adequate"
        }


class AIInvestmentRecommender:
    """
    AI-powered investment recommender that provides proactive suggestions
    based on portfolio analysis.
    """
    
    def __init__(self, allocator: AlternativeInvestmentAllocator):
        self.allocator = allocator
    
    def generate_recommendations(
        self,
        portfolio: Portfolio
    ) -> List[Dict]:
        """
        Generate proactive investment recommendations based on portfolio analysis.
        """
        recommendations = []
        
        # 1. Diversification opportunities
        diversification_suggestions = self.allocator.suggest_diversification_opportunities(portfolio)
        for suggestion in diversification_suggestions[:3]:  # Top 3
            recommendations.append({
                'strategy': suggestion['strategy'],
                'rationale': f"Portfolio underweight in {suggestion['strategy']}. "
                           f"Current: {suggestion['current_allocation_pct']:.1f}%, "
                           f"Target: {suggestion['target_allocation_pct']:.1f}%",
                'target_allocation_pct': suggestion['target_allocation_pct'],
                'recommended_amount': suggestion['recommended_amount'],
                'priority': suggestion['priority'],
                'type': 'DIVERSIFICATION'
            })
        
        # 2. Vintage year balancing
        vintage_analysis = self.allocator.vintage_year_analysis(portfolio)
        if vintage_analysis['missing_vintages']:
            for vintage in vintage_analysis['missing_vintages'][:2]:
                recommendations.append({
                    'strategy': 'PRIVATE_EQUITY',  # Default
                    'vintage_year': vintage,
                    'rationale': f"Portfolio missing {vintage} vintage. "
                               f"Current cycle favorable for new commitments.",
                    'target_allocation_pct': 5.0,
                    'priority': 'MEDIUM',
                    'type': 'VINTAGE_BALANCING'
                })
        
        # 3. Liquidity optimization
        if portfolio.total_unfunded > portfolio.liquid_assets * 0.4:
            recommendations.append({
                'strategy': 'SECONDARY_MARKET',
                'rationale': 'Upcoming unfunded commitments approach liquidity limits. '
                           'Consider secondary market sales for liquidity optimization.',
                'urgency': 'HIGH',
                'type': 'LIQUIDITY_OPTIMIZATION'
            })
        
        # Sort by priority
        priority_order = {'HIGH': 0, 'MEDIUM': 1, 'LOW': 2}
        recommendations.sort(
            key=lambda r: (priority_order.get(r.get('priority', 'LOW'), 2), -r.get('recommended_amount', 0))
        )
        
        return recommendations
    
    def predict_capital_calls(
        self,
        portfolio: Portfolio,
        horizon_days: int = 180
    ) -> Dict:
        """
        Predict expected capital calls based on unfunded commitments.
        Uses simplified model based on typical drawdown curves.
        """
        # Typical drawdown rates by year from commitment
        drawdown_rates = {
            0: 0.15,  # Year 0: 15% of commitment
            1: 0.25,  # Year 1: 25%
            2: 0.20,  # Year 2: 20%
            3: 0.15,  # Year 3: 15%
            4: 0.10,  # Year 4: 10%
            5: 0.10,  # Year 5: 10%
        }
        
        current_year = datetime.now().year
        expected_calls = 0
        
        for pos in portfolio.alternatives:
            if pos.unfunded_commitment > 0 and pos.vintage_year:
                years_since_vintage = current_year - pos.vintage_year
                # Estimate remaining drawdown
                remaining_rate = sum(
                    rate for year, rate in drawdown_rates.items()
                    if year > years_since_vintage
                )
                # Pro-rate for horizon
                horizon_rate = remaining_rate * (horizon_days / 365) * 0.5  # Conservative
                expected_calls += pos.unfunded_commitment * horizon_rate
        
        return {
            'expected_calls': expected_calls,
            'horizon_days': horizon_days,
            'coverage_ratio': portfolio.liquid_assets / expected_calls if expected_calls > 0 else float('inf'),
            'alert': expected_calls > portfolio.liquid_assets * 0.5
        }


# Convenience function for API usage
def analyze_allocation(
    portfolio_data: Dict,
    opportunity_data: Dict,
    constraints_data: Optional[Dict] = None
) -> Dict:
    """
    Main entry point for allocation analysis.
    
    Args:
        portfolio_data: Dictionary with portfolio information
        opportunity_data: Dictionary with opportunity information  
        constraints_data: Optional constraints overrides
        
    Returns:
        Dictionary with recommendation and analysis
    """
    # Build portfolio object
    alternatives = [
        AlternativePosition(
            investment_id=pos['investment_id'],
            investment_type=InvestmentType(pos['investment_type']),
            fund_name=pos['fund_name'],
            vintage_year=pos.get('vintage_year'),
            commitment_amount=pos['commitment_amount'],
            unfunded_commitment=pos.get('unfunded_commitment', 0),
            current_nav=pos['current_nav'],
            irr=pos.get('irr'),
            tvpi=pos.get('tvpi'),
            dpi=pos.get('dpi'),
            general_partner=pos.get('general_partner'),
        )
        for pos in portfolio_data.get('alternatives', [])
    ]
    
    portfolio = Portfolio(
        client_id=portfolio_data['client_id'],
        total_value=portfolio_data['total_value'],
        liquid_assets=portfolio_data['liquid_assets'],
        alternatives=alternatives,
        risk_tolerance=portfolio_data.get('risk_tolerance', 5),
        time_horizon_years=portfolio_data.get('time_horizon_years', 10),
        tax_bracket=portfolio_data.get('tax_bracket', 0.37),
    )
    
    opportunity = InvestmentOpportunity(
        opportunity_id=opportunity_data['opportunity_id'],
        opportunity_type=InvestmentType(opportunity_data['opportunity_type']),
        fund_name=opportunity_data['fund_name'],
        general_partner=opportunity_data['general_partner'],
        vintage_year=opportunity_data['vintage_year'],
        minimum_commitment=opportunity_data['minimum_commitment'],
        target_irr=opportunity_data['target_irr'],
        target_tvpi=opportunity_data.get('target_tvpi', 2.0),
        management_fee=opportunity_data.get('management_fee', 0.02),
        carried_interest=opportunity_data.get('carried_interest', 0.20),
        fund_size=opportunity_data.get('fund_size'),
        strategy=opportunity_data.get('strategy'),
    )
    
    constraints = AllocationConstraints(**(constraints_data or {}))
    
    allocator = AlternativeInvestmentAllocator(constraints=constraints)
    recommendation = allocator.recommend_allocation(portfolio, opportunity)
    
    return {
        'recommendation': {
            'opportunity_id': recommendation.opportunity_id,
            'recommended_amount': recommendation.recommended_amount,
            'recommended_pct_of_portfolio': recommendation.recommended_pct_of_portfolio,
            'current_alt_exposure_pct': recommendation.current_alt_exposure_pct,
            'post_allocation_alt_exposure_pct': recommendation.post_allocation_alt_exposure_pct,
            'diversification_benefit_score': recommendation.diversification_benefit_score,
            'liquidity_impact_score': recommendation.liquidity_impact_score,
            'risk_budget_utilization_pct': recommendation.risk_budget_utilization_pct,
            'overall_score': recommendation.overall_score,
            'fit_score': recommendation.fit_score,
            'timing_score': recommendation.timing_score,
            'rationale': recommendation.rationale,
            'pros': recommendation.pros,
            'cons': recommendation.cons,
            'risks': recommendation.risks,
            'constraints_violated': recommendation.constraints_violated,
            'model_confidence': recommendation.model_confidence,
        },
        'portfolio_summary': {
            'total_value': portfolio.total_value,
            'total_alt_value': portfolio.total_alt_value,
            'total_unfunded': portfolio.total_unfunded,
            'alt_allocation_pct': portfolio.alt_allocation_pct,
            'position_count': len(portfolio.alternatives),
        }
    }
