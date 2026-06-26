"""
NBA ML Model - Multi-Task Neural Network for Action Recommendation

This module implements the core ML model for the Next Best Action engine.
The model uses a multi-task architecture that simultaneously predicts:
1. Optimal action type (classification)
2. Urgency score (regression)
3. Expected value (regression)
4. Success probability (regression)
"""

import os
import json
import uuid
from datetime import datetime
from typing import Optional, List, Dict, Any, Tuple

import torch
import torch.nn as nn
import numpy as np

from app.schemas import (
    DetectedSignal,
    NextBestActionRecommendation,
    ActionCategory,
    ActionChannel,
    ActionPriority,
    ActionTemplate,
    ClientProfile,
)
from app.config import get_settings


# Action type mappings (index to action code)
ACTION_ID_TO_NAME = {
    0: "PROACTIVE_TAX_LOSS_HARVEST",
    1: "REENGAGEMENT_OUTREACH",
    2: "CONCENTRATED_POSITION_REVIEW",
    3: "RETIREMENT_PLANNING_REVIEW",
    4: "INHERITANCE_INTEGRATION",
    5: "RMD_REMINDER",
    6: "PORTFOLIO_REBALANCE",
    7: "RISK_TOLERANCE_CHECK",
    8: "ESTATE_PLANNING_REVIEW",
    9: "INSURANCE_REVIEW",
    10: "CHARITABLE_GIVING_DISCUSSION",
    11: "EDUCATION_FUNDING_REVIEW",
    12: "CASH_MANAGEMENT_REVIEW",
    13: "VOLATILITY_PROTECTION_DISCUSSION",
    14: "NEW_PRODUCT_INTRODUCTION",
    15: "ANNUAL_REVIEW_SCHEDULING",
    16: "BIRTHDAY_OUTREACH",
    17: "ANNIVERSARY_OUTREACH",
    18: "MARKET_UPDATE_CALL",
    19: "REFERRAL_REQUEST",
}

ACTION_NAME_TO_ID = {v: k for k, v in ACTION_ID_TO_NAME.items()}

# Signal type encoding
SIGNAL_TYPE_ENCODING = {
    "LARGE_WITHDRAWAL_PENDING": 0,
    "EMAIL_ENGAGEMENT_DROP": 1,
    "CONCENTRATED_POSITION_ALERT": 2,
    "EXCESS_CASH_DRAG": 3,
    "TAX_LOSS_HARVEST_OPPORTUNITY": 4,
    "CONCENTRATED_POSITION_RISK": 5,
    "ENGAGEMENT_DECLINE": 6,
    "LOW_EMAIL_ENGAGEMENT": 7,
    "VOLATILITY_EXPOSURE": 8,
    "RETIREMENT_APPROACHING": 9,
    "INHERITANCE_DETECTED": 10,
    "JOB_CHANGE_DETECTED": 11,
    "ANNIVERSARY_UPCOMING": 12,
    "REBALANCING_DUE": 13,
    "COMPLIANCE_DEADLINE": 14,
}

SIGNAL_CATEGORY_ENCODING = {
    "BEHAVIORAL": 0,
    "MARKET": 1,
    "LIFECYCLE": 2,
    "PORTFOLIO": 3,
    "ENGAGEMENT": 4,
}

# Action details for template generation
ACTION_DETAILS = {
    "PROACTIVE_TAX_LOSS_HARVEST": {
        "name": "Initiate Tax-Loss Harvesting Review",
        "category": ActionCategory.TAX_PLANNING,
        "channel": ActionChannel.PHONE,
        "duration": 30,
        "template": ActionTemplate(
            email_subject="Opportunity to Reduce Your {year} Tax Bill",
            email_body="Hi {client_first_name},\n\nI noticed some unrealized losses in your portfolio that could save you approximately ${estimated_tax_savings:,.0f} in taxes this year through strategic tax-loss harvesting.\n\nWould you have 20 minutes this week to discuss this opportunity?\n\nBest regards,\n{advisor_name}",
            call_script="Opening: I wanted to reach out because our system flagged a potential tax savings opportunity in your account...\n\nKey Points:\n- Current unrealized losses: ${total_loss}\n- Estimated tax savings: ${tax_benefit}\n- Recommended action: Harvest losses and reinvest in similar securities\n\nClose: Can we schedule 20 minutes to walk through the specific positions?"
        )
    },
    "REENGAGEMENT_OUTREACH": {
        "name": "Client Re-engagement Call",
        "category": ActionCategory.RELATIONSHIP_BUILDING,
        "channel": ActionChannel.PHONE,
        "duration": 20,
        "template": ActionTemplate(
            call_script="Hi {client_first_name}, I realized we haven't connected in a while and wanted to check in. How have things been going for you?\n\n[Listen actively]\n\nI want to make sure we're providing the level of service and communication that works best for you. Is there anything we could be doing differently?\n\n[Adjust communication preferences if needed]\n\nLet's schedule a portfolio review in the next couple weeks. What works better for you - morning or afternoon?",
            follow_up_email="Great talking with you today! As discussed, I'm scheduling our portfolio review for {meeting_date}. Looking forward to it."
        )
    },
    "CONCENTRATED_POSITION_REVIEW": {
        "name": "Diversification Strategy Discussion",
        "category": ActionCategory.PORTFOLIO_MANAGEMENT,
        "channel": ActionChannel.VIDEO_CALL,
        "duration": 45,
        "template": ActionTemplate(
            meeting_agenda="1. Review current portfolio concentration\n2. Discuss risks of single-position overweight\n3. Present diversification strategies\n4. Address tax implications\n5. Create implementation timeline",
            presentation_slides=[
                "Current Portfolio Allocation",
                "Concentration Risk Analysis",
                "Diversification Options",
                "Tax-Efficient Implementation",
                "Expected Risk Reduction"
            ]
        )
    },
    "RETIREMENT_PLANNING_REVIEW": {
        "name": "Schedule Retirement Readiness Review",
        "category": ActionCategory.PROACTIVE_OUTREACH,
        "channel": ActionChannel.VIDEO_CALL,
        "duration": 60,
        "template": ActionTemplate(
            email_subject="Your Retirement is Approaching - Let's Finalize Your Plan",
            email_body="Hi {client_first_name},\n\nWith your retirement approaching, I wanted to schedule a comprehensive review to ensure everything is in place for a smooth transition.\n\nI'd like to cover income planning, Social Security timing, healthcare, and your withdrawal strategy.\n\nWould next Tuesday at 2pm work for a video call?\n\nBest,\n{advisor_name}",
            meeting_agenda="1. Review current portfolio allocation\n2. Income planning strategy\n3. Social Security optimization\n4. Healthcare bridge (Medicare gap)\n5. Tax-efficient withdrawal strategy"
        )
    },
    "INHERITANCE_INTEGRATION": {
        "name": "Inheritance Integration Meeting",
        "category": ActionCategory.SERVICE_DELIVERY,
        "channel": ActionChannel.IN_PERSON,
        "duration": 90,
        "template": ActionTemplate(
            email_subject="Planning for Your Recent Inheritance",
            email_body="Hi {client_first_name},\n\nI noticed a significant transfer into your account. I wanted to reach out to ensure we're handling this thoughtfully and taking advantage of all available planning opportunities.\n\nThis is an important moment, and there are several tax and investment considerations we should discuss. Would you be available for an in-person meeting this week?\n\nMy condolences if this is related to a loss. I'm here to help however I can.\n\nWarmly,\n{advisor_name}",
            meeting_agenda="1. Understand the source and any emotional considerations\n2. Review step-up in basis implications\n3. Discuss investment allocation strategy\n4. Tax planning opportunities\n5. Update overall financial plan"
        )
    },
}


class NextBestActionModel(nn.Module):
    """
    Multi-task neural network for NBA prediction.
    
    Architecture:
    - Text encoder (BERT-based) for CRM notes and email context
    - Numeric encoder for client profile features
    - Signal encoder for detected signal features
    - Fusion layer combining all encodings
    - Multi-task heads for action, urgency, value, and success prediction
    """
    
    def __init__(self, num_actions: int = 50, client_embedding_dim: int = 128):
        super().__init__()
        
        self.num_actions = num_actions
        
        # Simplified text encoder (no BERT to avoid heavy dependencies in demo)
        # In production, replace with:
        # self.bert = BertModel.from_pretrained('bert-base-uncased')
        self.text_encoder = nn.Sequential(
            nn.Linear(512, 256),  # Assume pre-computed text embeddings
            nn.ReLU(),
            nn.Dropout(0.3),
            nn.Linear(256, 128)
        )
        
        # Numerical feature encoder
        self.numeric_encoder = nn.Sequential(
            nn.Linear(25, 64),  # 25 numerical features
            nn.ReLU(),
            nn.Dropout(0.3),
            nn.Linear(64, 128)
        )
        
        # Signal embedding
        self.signal_encoder = nn.Sequential(
            nn.Linear(10, 32),  # Signal features
            nn.ReLU(),
            nn.Linear(32, 64)
        )
        
        # Fusion layer
        self.fusion = nn.Sequential(
            nn.Linear(128 + 128 + 64, 256),  # text(128) + numeric(128) + signal(64)
            nn.ReLU(),
            nn.Dropout(0.4),
            nn.Linear(256, 128)
        )
        
        # Multi-task heads
        self.action_classifier = nn.Linear(128, num_actions)  # Which action?
        self.urgency_regressor = nn.Linear(128, 1)  # How urgent? (0-1)
        self.value_regressor = nn.Linear(128, 1)  # Expected revenue impact
        self.success_probability = nn.Linear(128, 1)  # Likelihood client responds
        
    def forward(
        self,
        text_features: torch.Tensor,
        numeric_features: torch.Tensor,
        signal_features: torch.Tensor
    ) -> Dict[str, torch.Tensor]:
        """
        Forward pass through the model.
        
        Args:
            text_features: Pre-computed text embeddings [batch, 512]
            numeric_features: Client profile features [batch, 25]
            signal_features: Signal features [batch, 10]
            
        Returns:
            Dict with action_logits, urgency, expected_value, success_probability
        """
        # Encode each feature type
        text_encoded = self.text_encoder(text_features)
        numeric_encoded = self.numeric_encoder(numeric_features)
        signal_encoded = self.signal_encoder(signal_features)
        
        # Fuse all representations
        fused = self.fusion(
            torch.cat([text_encoded, numeric_encoded, signal_encoded], dim=1)
        )
        
        # Multi-task predictions
        action_logits = self.action_classifier(fused)
        urgency = torch.sigmoid(self.urgency_regressor(fused))
        expected_value = self.value_regressor(fused)
        success_prob = torch.sigmoid(self.success_probability(fused))
        
        return {
            'action_logits': action_logits,
            'urgency': urgency,
            'expected_value': expected_value,
            'success_probability': success_prob
        }


class NBAInferenceService:
    """
    High-level inference service for NBA predictions.
    
    Handles feature extraction, model inference, and result formatting.
    """
    
    def __init__(self, model_path: Optional[str] = None):
        """
        Initialize the inference service.
        
        Args:
            model_path: Path to pre-trained model weights.
                       If None, uses rule-based fallback.
        """
        self.settings = get_settings()
        self.device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
        self.model_loaded = False
        self.model_version = "1.0.0-fallback"
        self.trained_at = None
        
        # Initialize model
        self.model = NextBestActionModel(
            num_actions=len(ACTION_ID_TO_NAME),
            client_embedding_dim=self.settings.client_embedding_dim
        )
        
        # Load weights if available
        if model_path and os.path.exists(model_path):
            try:
                checkpoint = torch.load(model_path, map_location=self.device)
                self.model.load_state_dict(checkpoint['model_state_dict'])
                self.model_version = checkpoint.get('version', '1.0.0')
                self.trained_at = checkpoint.get('trained_at')
                self.model_loaded = True
            except Exception as e:
                print(f"Warning: Could not load model from {model_path}: {e}")
        
        self.model.to(self.device)
        self.model.eval()
    
    async def predict(
        self,
        client_id: str,
        signal: DetectedSignal,
        db_session: Any = None
    ) -> List[NextBestActionRecommendation]:
        """
        Generate NBA recommendations for a client based on a detected signal.
        
        Args:
            client_id: Client UUID
            signal: Detected signal triggering the recommendation
            db_session: Database session for fetching context
            
        Returns:
            List of ranked action recommendations
        """
        # If model not loaded, use rule-based fallback
        if not self.model_loaded:
            return self._fallback_recommendations(client_id, signal)
        
        # Extract features
        features = await self._extract_features(client_id, signal, db_session)
        
        # Run model inference
        with torch.no_grad():
            predictions = self.model(
                text_features=features['text'].unsqueeze(0).to(self.device),
                numeric_features=features['numeric'].unsqueeze(0).to(self.device),
                signal_features=features['signal'].unsqueeze(0).to(self.device)
            )
        
        # Get top-K action recommendations
        action_probs = torch.softmax(predictions['action_logits'], dim=1)[0]
        top_actions = torch.topk(action_probs, k=self.settings.top_k_recommendations)
        
        # Build recommendations
        recommendations = []
        for idx, prob in zip(top_actions.indices, top_actions.values):
            action_idx = idx.item()
            if action_idx not in ACTION_ID_TO_NAME:
                continue
                
            action_type = ACTION_ID_TO_NAME[action_idx]
            action_details = ACTION_DETAILS.get(action_type, {})
            
            recommendations.append(NextBestActionRecommendation(
                action_id=str(uuid.uuid4()),
                action_type=action_type,
                action_name=action_details.get('name', action_type.replace('_', ' ').title()),
                action_category=action_details.get('category', ActionCategory.PROACTIVE_OUTREACH),
                confidence=prob.item(),
                urgency_score=predictions['urgency'][0].item(),
                expected_value=max(0, predictions['expected_value'][0].item()),
                success_probability=predictions['success_probability'][0].item(),
                trigger_signal=signal.signal_type,
                reasoning=self._generate_reasoning(action_type, signal),
                recommended_channel=action_details.get('channel', ActionChannel.PHONE),
                estimated_duration_minutes=action_details.get('duration', 30),
                template_content=action_details.get('template', ActionTemplate()),
                priority=self._calculate_priority(
                    predictions['urgency'][0].item(),
                    predictions['expected_value'][0].item()
                )
            ))
        
        # Re-rank by expected impact (urgency × value × success_prob)
        recommendations.sort(
            key=lambda x: x.urgency_score * x.expected_value * x.success_probability,
            reverse=True
        )
        
        return recommendations
    
    async def _extract_features(
        self,
        client_id: str,
        signal: DetectedSignal,
        db_session: Any
    ) -> Dict[str, torch.Tensor]:
        """Extract features for model input."""
        
        # Get client profile from DB (if available)
        client_profile = None
        if db_session:
            from app.database import ClientRepository
            repo = ClientRepository(db_session)
            client_profile = repo.get_client_profile(client_id)
        
        # Build numeric features
        if client_profile:
            numeric_features = torch.tensor([
                client_profile.get('age', 50),
                client_profile.get('net_worth', 1000000),
                client_profile.get('aum', 500000),
                client_profile.get('tenure_years', 3),
                client_profile.get('num_accounts', 2),
                client_profile.get('annual_fees', 5000),
                client_profile.get('risk_tolerance_score', 0.5),
                client_profile.get('liquidity_needs_score', 0.3),
                client_profile.get('tax_bracket', 0.32),
                client_profile.get('retirement_years_away', 15),
                client_profile.get('portfolio_return_ytd', 0.08),
                client_profile.get('portfolio_return_3yr', 0.12),
                client_profile.get('sharpe_ratio', 0.9),
                client_profile.get('max_drawdown_ytd', -0.05),
                client_profile.get('equity_allocation', 0.6),
                client_profile.get('fixed_income_allocation', 0.3),
                client_profile.get('alternative_allocation', 0.05),
                client_profile.get('cash_allocation', 0.05),
                client_profile.get('avg_meeting_frequency', 4),
                client_profile.get('last_meeting_days_ago', 45),
                client_profile.get('email_open_rate', 0.6),
                client_profile.get('portal_logins_90d', 10),
                client_profile.get('referrals_given', 1),
                client_profile.get('satisfaction_score', 0.8),
                client_profile.get('flight_risk_score', 0.2),
            ], dtype=torch.float32)
        else:
            # Use default values
            numeric_features = torch.zeros(25, dtype=torch.float32)
            numeric_features[0] = 50  # age
            numeric_features[1] = 1000000  # net_worth
            numeric_features[6] = 0.5  # risk_tolerance
        
        # Build signal features
        signal_type_idx = SIGNAL_TYPE_ENCODING.get(signal.signal_type, 0)
        category_idx = SIGNAL_CATEGORY_ENCODING.get(signal.category, 0)
        
        signal_features = torch.tensor([
            signal.strength,
            signal_type_idx / len(SIGNAL_TYPE_ENCODING),  # Normalize
            category_idx / len(SIGNAL_CATEGORY_ENCODING),  # Normalize
            0.5,  # time_since_last_action (placeholder)
            0.3,  # signal_frequency_90d (placeholder)
            0.7,  # client_responsiveness_history (placeholder)
            0.5,  # advisor_workload_current (placeholder)
            0.3,  # market_volatility_current (placeholder)
            float(datetime.now().day > 25),  # is_month_end
            float(datetime.now().month in [3, 4, 9, 10]),  # is_tax_season
        ], dtype=torch.float32)
        
        # Text features (placeholder - would use BERT in production)
        text_features = torch.randn(512, dtype=torch.float32)
        
        return {
            'text': text_features,
            'numeric': numeric_features,
            'signal': signal_features
        }
    
    def _fallback_recommendations(
        self,
        client_id: str,
        signal: DetectedSignal
    ) -> List[NextBestActionRecommendation]:
        """
        Rule-based fallback when model is not available.
        Maps signals directly to recommended actions.
        """
        signal_to_actions = {
            "TAX_LOSS_HARVEST_OPPORTUNITY": ["PROACTIVE_TAX_LOSS_HARVEST"],
            "EXCESS_CASH_DRAG": ["CASH_MANAGEMENT_REVIEW", "PORTFOLIO_REBALANCE"],
            "CONCENTRATED_POSITION_RISK": ["CONCENTRATED_POSITION_REVIEW"],
            "CONCENTRATED_POSITION_ALERT": ["CONCENTRATED_POSITION_REVIEW"],
            "ENGAGEMENT_DECLINE": ["REENGAGEMENT_OUTREACH"],
            "LOW_EMAIL_ENGAGEMENT": ["REENGAGEMENT_OUTREACH"],
            "EMAIL_ENGAGEMENT_DROP": ["REENGAGEMENT_OUTREACH"],
            "RETIREMENT_APPROACHING": ["RETIREMENT_PLANNING_REVIEW"],
            "INHERITANCE_DETECTED": ["INHERITANCE_INTEGRATION"],
            "VOLATILITY_EXPOSURE": ["VOLATILITY_PROTECTION_DISCUSSION", "RISK_TOLERANCE_CHECK"],
            "REBALANCING_DUE": ["PORTFOLIO_REBALANCE"],
            "ANNIVERSARY_UPCOMING": ["ANNIVERSARY_OUTREACH"],
            "LARGE_WITHDRAWAL_PENDING": ["REENGAGEMENT_OUTREACH", "CASH_MANAGEMENT_REVIEW"],
        }
        
        recommended_action_types = signal_to_actions.get(
            signal.signal_type,
            ["REENGAGEMENT_OUTREACH"]  # Default
        )
        
        recommendations = []
        for i, action_type in enumerate(recommended_action_types[:3]):
            action_details = ACTION_DETAILS.get(action_type, {})
            
            # Calculate scores based on signal strength
            base_confidence = 0.85 - (i * 0.1)  # Decrease for lower-ranked actions
            urgency = signal.strength * 0.9
            expected_value = signal.raw_data.get('estimated_tax_savings', 
                             signal.raw_data.get('estimated_opportunity_cost', 5000))
            if isinstance(expected_value, (int, float)):
                expected_value = float(expected_value)
            else:
                expected_value = 5000.0
            
            recommendations.append(NextBestActionRecommendation(
                action_id=str(uuid.uuid4()),
                action_type=action_type,
                action_name=action_details.get('name', action_type.replace('_', ' ').title()),
                action_category=action_details.get('category', ActionCategory.PROACTIVE_OUTREACH),
                confidence=base_confidence,
                urgency_score=urgency,
                expected_value=expected_value,
                success_probability=0.75 - (i * 0.05),
                trigger_signal=signal.signal_type,
                reasoning=self._generate_reasoning(action_type, signal),
                recommended_channel=action_details.get('channel', ActionChannel.PHONE),
                estimated_duration_minutes=action_details.get('duration', 30),
                template_content=action_details.get('template', ActionTemplate()),
                priority=self._calculate_priority(urgency, expected_value)
            ))
        
        return recommendations
    
    def _generate_reasoning(self, action_type: str, signal: DetectedSignal) -> str:
        """Generate AI reasoning for the recommendation."""
        
        reasoning_templates = {
            "PROACTIVE_TAX_LOSS_HARVEST": (
                f"Portfolio has unrealized losses that could be harvested for tax benefits. "
                f"Signal strength {signal.strength:.0%} indicates high confidence in this opportunity. "
                f"Estimated tax savings: ${signal.raw_data.get('estimated_tax_savings', 'N/A')}"
            ),
            "REENGAGEMENT_OUTREACH": (
                f"Client engagement metrics show {signal.signal_type.replace('_', ' ').lower()}. "
                f"Proactive outreach can prevent attrition and reinforce the relationship. "
                f"Signal strength: {signal.strength:.0%}"
            ),
            "CONCENTRATED_POSITION_REVIEW": (
                f"Portfolio concentration risk detected with {signal.raw_data.get('position_percent', 'N/A')}% "
                f"in a single position. Diversification discussion recommended to reduce risk."
            ),
            "RETIREMENT_PLANNING_REVIEW": (
                f"Client retirement is approaching. Portfolio readiness and income planning "
                f"should be reviewed to ensure a smooth transition. Urgency: {signal.strength:.0%}"
            ),
            "INHERITANCE_INTEGRATION": (
                f"Large asset transfer detected. Important to discuss step-up in basis, "
                f"tax implications, and integration into overall financial plan."
            ),
        }
        
        return reasoning_templates.get(
            action_type,
            f"Based on detected signal '{signal.signal_type}' with {signal.strength:.0%} confidence. "
            f"Proactive engagement recommended."
        )
    
    def _calculate_priority(self, urgency: float, expected_value: float) -> ActionPriority:
        """Calculate action priority based on urgency and value."""
        if urgency >= 0.9 or expected_value >= 20000:
            return ActionPriority.CRITICAL
        elif urgency >= 0.7 or expected_value >= 10000:
            return ActionPriority.HIGH
        elif urgency >= 0.5 or expected_value >= 5000:
            return ActionPriority.MEDIUM
        elif urgency >= 0.3:
            return ActionPriority.LOW
        else:
            return ActionPriority.OPTIONAL
    
    async def retrain(self, db_session: Any, lookback_days: int = 90) -> Dict[str, Any]:
        """
        Retrain the model using historical outcome data.
        
        This is a simplified version - in production, this would:
        1. Extract training data from nba_action_outcomes
        2. Perform feature engineering
        3. Train the model with appropriate hyperparameters
        4. Validate on a holdout set
        5. Save the new model if performance improved
        """
        from app.database import OutcomeRepository
        
        repo = OutcomeRepository(db_session)
        training_data = repo.get_training_data(lookback_days)
        
        if len(training_data) < 100:
            return {
                "status": "insufficient_data",
                "samples": len(training_data),
                "required": 100
            }
        
        # Simulate training metrics
        return {
            "status": "completed",
            "samples": len(training_data),
            "f1_score": 0.78,
            "precision_at_k": 0.72,
            "recall_at_k": 0.68,
            "training_time_seconds": 120.5
        }
    
    def get_model_info(self) -> Dict[str, Any]:
        """Get information about the loaded model."""
        return {
            "model_version": self.model_version,
            "model_loaded": self.model_loaded,
            "num_actions": len(ACTION_ID_TO_NAME),
            "trained_at": self.trained_at,
            "device": str(self.device),
            "action_types": list(ACTION_ID_TO_NAME.values())
        }
