import torch
import torch.nn as nn
from transformers import BertModel

class NextBestActionModel(nn.Module):
    """
    Multi-task neural network that:
    1. Predicts optimal action type
    2. Estimates urgency score
    3. Calculates expected value (revenue impact)
    """
    
    def __init__(self, num_actions=50, client_embedding_dim=128):
        super().__init__()
        
        # Client context encoder (BERT for text features)
        self.bert = BertModel.from_pretrained('bert-base-uncased')
        
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
            nn.Linear(768 + 128 + 64, 256),  # BERT(768) + numeric(128) + signal(64)
            nn.ReLU(),
            nn.Dropout(0.4),
            nn.Linear(256, 128)
        )
        
        # Multi-task heads
        self.action_classifier = nn.Linear(128, num_actions)  # Which action?
        self.urgency_regressor = nn.Linear(128, 1)  # How urgent? (0-1)
        self.value_regressor = nn.Linear(128, 1)  # Expected revenue impact
        self.success_probability = nn.Linear(128, 1)  # Likelihood client responds
        
    def forward(self, text_features, numeric_features, signal_features):
        # Encode text (CRM notes, recent emails)
        bert_output = self.bert(**text_features).last_hidden_state[:, 0, :]  # CLS token
        
        # Encode numeric features
        numeric_encoded = self.numeric_encoder(numeric_features)
        
        # Encode signal features
        signal_encoded = self.signal_encoder(signal_features)
        
        # Fuse all representations
        fused = self.fusion(torch.cat([bert_output, numeric_encoded, signal_encoded], dim=1))
        
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
