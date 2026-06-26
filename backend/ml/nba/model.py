"""
NBA (Next Best Action) ML Model
Multi-task neural network using BERT for advisor action recommendations

Architecture:
- Text encoder: BERT (pre-trained)
- Numerical encoder: Dense layers
- Signal encoder: Dense layers
- Fusion layer: Concatenate + Dense
- Multi-task heads: 
  1. Action classification (50 classes)
  2. Urgency regression (0-1)
  3. Expected value regression
  4. Success probability (0-1)
"""

import torch
import torch.nn as nn
from transformers import BertModel, BertTokenizer
from typing import Dict, List, Tuple

class NextBestActionModel(nn.Module):
    """
    Multi-task neural network for NBA predictions
    
    Inputs:
        - text_features: BERT tokenized text (CRM notes, emails)
        - numeric_features: 25 client/portfolio features
        - signal_features: 10 signal context features
    
    Outputs:
        - action_logits: Probabilities for 50 action types
        - urgency: Urgency score 0-1
        - expected_value: Expected revenue impact
        - success_probability: Likelihood of success 0-1
    """
    
    def __init__(
        self,
        num_actions: int = 50,
        client_embedding_dim: int = 128,
        bert_model_name: str = 'bert-base-uncased',
        freeze_bert: bool = False
    ):
        super().__init__()
        
        # BERT for text encoding (CRM notes, emails, meeting notes)
        self.bert = BertModel.from_pretrained(bert_model_name)
        self.bert_hidden_size = self.bert.config.hidden_size  # 768 for base
        
        if freeze_bert:
            # Freeze BERT parameters for faster training
            for param in self.bert.parameters():
                param.requires_grad = False
        
        # Numerical feature encoder (25 client/portfolio features)
        self.numeric_encoder = nn.Sequential(
            nn.Linear(25, 64),
            nn.ReLU(),
            nn.BatchNorm1d(64),
            nn.Dropout(0.3),
            nn.Linear(64, 128),
            nn.ReLU(),
            nn.BatchNorm1d(128)
        )
        
        # Signal feature encoder (10 signal context features)
        self.signal_encoder = nn.Sequential(
            nn.Linear(10, 32),
            nn.ReLU(),
            nn.BatchNorm1d(32),
            nn.Linear(32, 64),
            nn.ReLU(),
            nn.BatchNorm1d(64)
        )
        
        # Fusion layer - combines all modalities
        fusion_input_dim = self.bert_hidden_size + 128 + 64  # 768 + 128 + 64 = 960
        self.fusion = nn.Sequential(
            nn.Linear(fusion_input_dim, 512),
            nn.ReLU(),
            nn.BatchNorm1d(512),
            nn.Dropout(0.4),
            nn.Linear(512, 256),
            nn.ReLU(),
            nn.BatchNorm1d(256),
            nn.Dropout(0.3),
            nn.Linear(256, 128),
            nn.ReLU()
        )
        
        # Multi-task prediction heads
        self.action_classifier = nn.Linear(128, num_actions)  # Which action?
        self.urgency_regressor = nn.Linear(128, 1)  # How urgent? (0-1)
        self.value_regressor = nn.Linear(128, 1)  # Expected revenue
        self.success_predictor = nn.Linear(128, 1)  # Success probability (0-1)
        
    def forward(
        self,
        text_features: Dict[str, torch.Tensor],
        numeric_features: torch.Tensor,
        signal_features: torch.Tensor
    ) -> Dict[str, torch.Tensor]:
        """
        Forward pass
        
        Args:
            text_features: Dict with 'input_ids' and 'attention_mask' from BERT tokenizer
            numeric_features: (batch_size, 25) - client/portfolio features
            signal_features: (batch_size, 10) - signal context features
        
        Returns:
            Dict with action_logits, urgency, expected_value, success_probability
        """
        
        # 1. Encode text with BERT (CLS token embedding)
        bert_output = self.bert(
            input_ids=text_features['input_ids'],
            attention_mask=text_features['attention_mask']
        )
        text_encoded = bert_output.last_hidden_state[:, 0, :]  # CLS token (batch_size, 768)
        
        # 2. Encode numerical features
        numeric_encoded = self.numeric_encoder(numeric_features)  # (batch_size, 128)
        
        # 3. Encode signal features
        signal_encoded = self.signal_encoder(signal_features)  # (batch_size, 64)
        
        # 4. Fuse all representations
        fused = torch.cat([text_encoded, numeric_encoded, signal_encoded], dim=1)  # (batch_size, 960)
        fused_encoded = self.fusion(fused)  # (batch_size, 128)
        
        # 5. Multi-task predictions
        action_logits = self.action_classifier(fused_encoded)  # (batch_size, 50)
        urgency = torch.sigmoid(self.urgency_regressor(fused_encoded))  # (batch_size, 1) in [0, 1]
        expected_value = self.value_regressor(fused_encoded)  # (batch_size, 1)
        success_prob = torch.sigmoid(self.success_predictor(fused_encoded))  # (batch_size, 1) in [0, 1]
        
        return {
            'action_logits': action_logits,
            'urgency': urgency,
            'expected_value': expected_value,
            'success_probability': success_prob
        }

class NBADataset(torch.utils.data.Dataset):
    """
    PyTorch Dataset for NBA training data
    """
    
    def __init__(
        self,
        text_data: List[str],
        numeric_data: torch.Tensor,
        signal_data: torch.Tensor,
        labels: Dict[str, torch.Tensor],
        tokenizer: BertTokenizer,
        max_length: int = 512
    ):
        self.text_data = text_data
        self.numeric_data = numeric_data
        self.signal_data = signal_data
        self.labels = labels
        self.tokenizer = tokenizer
        self.max_length = max_length
        
    def __len__(self):
        return len(self.text_data)
    
    def __getitem__(self, idx):
        # Tokenize text
        text_encoding = self.tokenizer(
            self.text_data[idx],
            truncation=True,
            max_length=self.max_length,
            padding='max_length',
            return_tensors='pt'
        )
        
        return {
            'input_ids': text_encoding['input_ids'].squeeze(0),
            'attention_mask': text_encoding['attention_mask'].squeeze(0),
            'numeric_features': self.numeric_data[idx],
            'signal_features': self.signal_data[idx],
            'action_label': self.labels['action'][idx],
            'urgency_label': self.labels['urgency'][idx],
            'value_label': self.labels['value'][idx],
            'success_label': self.labels['success'][idx]
        }

class NBALoss(nn.Module):
    """
    Multi-task loss function for NBA model
    
    L_total = α * L_classification + β * L_urgency + γ * L_value + δ * L_success
    """
    
    def __init__(
        self,
        alpha: float = 1.0,  # Action classification weight
        beta: float = 0.5,   # Urgency regression weight
        gamma: float = 0.3,  # Value regression weight
        delta: float = 0.5   # Success probability weight
    ):
        super().__init__()
        self.alpha = alpha
        self.beta = beta
        self.gamma = gamma
        self.delta = delta
        
        self.classification_loss = nn.CrossEntropyLoss()
        self.mse_loss = nn.MSELoss()
        self.bce_loss = nn.BCELoss()
        
    def forward(
        self,
        predictions: Dict[str, torch.Tensor],
        labels: Dict[str, torch.Tensor]
    ) -> Tuple[torch.Tensor, Dict[str, float]]:
        """
        Calculate multi-task loss
        
        Args:
            predictions: Model outputs
            labels: Ground truth labels
        
        Returns:
            total_loss, loss_components_dict
        """
        
        # 1. Action classification loss (CrossEntropy)
        loss_action = self.classification_loss(
            predictions['action_logits'],
            labels['action']
        )
        
        # 2. Urgency regression loss (MSE)
        loss_urgency = self.mse_loss(
            predictions['urgency'],
            labels['urgency'].unsqueeze(1)
        )
        
        # 3. Expected value regression loss (MSE)
        loss_value = self.mse_loss(
            predictions['expected_value'],
            labels['value'].unsqueeze(1)
        )
        
        # 4. Success probability loss (BCE)
        loss_success = self.bce_loss(
            predictions['success_probability'],
            labels['success'].unsqueeze(1)
        )
        
        # Combined weighted loss
        total_loss = (
            self.alpha * loss_action +
            self.beta * loss_urgency +
            self.gamma * loss_value +
            self.delta * loss_success
        )
        
        # Return loss components for logging
        loss_components = {
            'total': total_loss.item(),
            'action': loss_action.item(),
            'urgency': loss_urgency.item(),
            'value': loss_value.item(),
            'success': loss_success.item()
        }
        
        return total_loss, loss_components

# Example usage:
if __name__ == '__main__':
    # Initialize model
    model = NextBestActionModel(num_actions=50)
    
    # Example forward pass
    batch_size = 8
    
    # Dummy inputs
    text_features = {
        'input_ids': torch.randint(0, 30522, (batch_size, 512)),
        'attention_mask': torch.ones(batch_size, 512)
    }
    numeric_features = torch.randn(batch_size, 25)
    signal_features = torch.randn(batch_size, 10)
    
    # Forward pass
    outputs = model(text_features, numeric_features, signal_features)
    
    print("Model outputs:")
    for key, value in outputs.items():
        print(f"  {key}: {value.shape}")
    
    # Total parameters
    total_params = sum(p.numel() for p in model.parameters())
    trainable_params = sum(p.numel() for p in model.parameters() if p.requires_grad)
    print(f"\nTotal parameters: {total_params:,}")
    print(f"Trainable parameters: {trainable_params:,}")
