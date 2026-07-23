"""
NBA Model Training Pipeline

Trains the multi-task neural network on historical advisor-client interactions
"""

import os
import json
import torch
import torch.nn as nn
import torch.optim as optim
from torch.utils.data import DataLoader, random_split
from transformers import BertTokenizer
import mlflow
import mlflow.pytorch
from tqdm import tqdm
import numpy as np
from typing import Dict, List, Tuple
import psycopg2
from datetime import datetime

from model import NextBestActionModel, NBADataset, NBALoss

class NBATrainer:
    """
    Training pipeline for NBA ML model
    """
    
    def __init__(
        self,
        db_config: Dict[str, str],
        model_config: Dict = None,
        training_config: Dict = None
    ):
        self.db_config = db_config
        self.model_config = model_config or {}
        self.training_config = training_config or {}
        
        # Device
        self.device = torch.device('cuda' if torch.cuda.is_available() else 'cpu')
        print(f"Using device: {self.device}")
        
        # Tokenizer
        self.tokenizer = BertTokenizer.from_pretrained('bert-base-uncased')
        
        # Model
        self.model = NextBestActionModel(**self.model_config).to(self.device)
        
        # Loss function
        self.criterion = NBALoss()
        
        # Optimizer
        self.optimizer = optim.AdamW(
            self.model.parameters(),
            lr=self.training_config.get('learning_rate', 2e-5),
            weight_decay=self.training_config.get('weight_decay', 0.01)
        )
        
        # Learning rate scheduler
        self.scheduler = optim.lr_scheduler.ReduceLROnPlateau(
            self.optimizer,
            mode='min',
            factor=0.5,
            patience=3,
            verbose=True
        )
        
    def load_training_data(self, tenant_id: str = None) -> Tuple[NBADataset, NBADataset]:
        """
        Load training data from database
        
        Query historical advisor actions and outcomes for training
        """
        print("Loading training data from database...")
        
        conn = psycopg2.connect(**self.db_config)
        cursor = conn.cursor()
        
        # Query historical recommendations and outcomes
        query = """
            SELECT 
                r.recommendation_id,
                r.client_id,
                r.action_id,
                a.action_code,
                r.confidence_score,
                r.urgency_score,
                r.expected_value,
                r.success_probability,
                o.action_successful,
                o.revenue_generated,
                o.client_responded,
                
                -- Client context (we'll build this from multiple tables)
                c.age,
                c.net_worth,
                c.tenure_years,
                -- ... 25 total numeric features
                
                -- Text features (CRM notes, emails)
                (SELECT string_agg(note_text, ' ') FROM crm_notes WHERE client_id = r.client_id LIMIT 10) as crm_notes,
                
                -- Signal features
                s.signal_type,
                s.signal_strength,
                s.signal_data
                
            FROM nba_recommendations r
            JOIN nba_action_outcomes o ON r.recommendation_id = o.recommendation_id
            JOIN nba_action_catalog a ON r.action_id = a.action_id
            JOIN clients c ON r.client_id = c.client_id
            LEFT JOIN nba_signals s ON r.trigger_signal_id = s.signal_id
            WHERE o.action_successful IS NOT NULL  -- Only completed actions
            ORDER BY r.recommended_at DESC
            LIMIT 10000  -- Limit for initial training
        """
        
        cursor.execute(query)
        rows = cursor.fetchall()
        
        print(f"Loaded {len(rows)} training examples")
        
        # Process data into model format
        text_data = []
        numeric_data = []
        signal_data = []
        labels = {
            'action': [],
            'urgency': [],
            'value': [],
            'success': []
        }
        
        # Action code to ID mapping (for classification)
        action_to_id = self._get_action_mapping(cursor)
        
        for row in rows:
            # Text data (CRM notes)
            text_data.append(row[16] if row[16] else "No recent notes")
            
            # Numeric features (client profile)
            numeric_features = torch.tensor([
                row[11],  # age
                row[12],  # net_worth
                row[13],  # tenure_years
                # ... extract all 25 features
                # For now, padding with zeros
                *([0.0] * 22)
            ], dtype=torch.float32)
            numeric_data.append(numeric_features)
            
            # Signal features
            signal_features = torch.tensor([
                row[18] or 0.5,  # signal_strength
                # ... extract 10 signal features
                *([0.0] * 9)
            ], dtype=torch.float32)
            signal_data.append(signal_features)
            
            # Labels
            labels['action'].append(action_to_id.get(row[3], 0))
            labels['urgency'].append(row[5])
            labels['value'].append(row[6])
            labels['success'].append(1.0 if row[8] else 0.0)
        
        # Convert to tensors
        numeric_data = torch.stack(numeric_data)
        signal_data = torch.stack(signal_data)
        labels = {
            'action': torch.tensor(labels['action'], dtype=torch.long),
            'urgency': torch.tensor(labels['urgency'], dtype=torch.float32),
            'value': torch.tensor(labels['value'], dtype=torch.float32),
            'success': torch.tensor(labels['success'], dtype=torch.float32)
        }
        
        # Create dataset
        dataset = NBADataset(
            text_data=text_data,
            numeric_data=numeric_data,
            signal_data=signal_data,
            labels=labels,
            tokenizer=self.tokenizer
        )
        
        # Train/val split (80/20)
        train_size = int(0.8 * len(dataset))
        val_size = len(dataset) - train_size
        train_dataset, val_dataset = random_split(dataset, [train_size, val_size])
        
        cursor.close()
        conn.close()
        
        return train_dataset, val_dataset
    
    def _get_action_mapping(self, cursor) -> Dict[str, int]:
        """Get action code to integer ID mapping"""
        cursor.execute("SELECT action_code FROM nba_action_catalog ORDER BY action_code")
        actions = [row[0] for row in cursor.fetchall()]
        return {action: idx for idx, action in enumerate(actions)}
    
    def train(
        self,
        train_dataset: NBADataset,
        val_dataset: NBADataset,
        epochs: int = 10,
        batch_size: int = 16
    ):
        """
        Train the model
        """
        # Data loaders
        train_loader = DataLoader(
            train_dataset,
            batch_size=batch_size,
            shuffle=True,
            num_workers=4
        )
        val_loader = DataLoader(
            val_dataset,
            batch_size=batch_size,
            shuffle=False,
            num_workers=4
        )
        
        # MLflow tracking
        mlflow.set_experiment("NBA_Model_Training")
        
        with mlflow.start_run():
            # Log parameters
            mlflow.log_params({
                'epochs': epochs,
                'batch_size': batch_size,
                'learning_rate': self.training_config.get('learning_rate', 2e-5),
                'model_type': 'BERT-MultiTask'
            })
            
            best_val_loss = float('inf')
            
            for epoch in range(epochs):
                print(f"\nEpoch {epoch+1}/{epochs}")
                print("-" * 50)
                
                # Training phase
                train_loss, train_metrics = self._train_epoch(train_loader)
                
                # Validation phase
                val_loss, val_metrics = self._validate_epoch(val_loader)
                
                # Learning rate scheduling
                self.scheduler.step(val_loss)
                
                # Logging
                print(f"Train Loss: {train_loss:.4f} | Val Loss: {val_loss:.4f}")
                print(f"Val Top-5 Accuracy: {val_metrics['top5_accuracy']:.4f}")
                
                mlflow.log_metrics({
                    'train_loss': train_loss,
                    'val_loss': val_loss,
                    'val_top5_accuracy': val_metrics['top5_accuracy']
                }, step=epoch)
                
                # Save best model
                if val_loss < best_val_loss:
                    best_val_loss = val_loss
                    self.save_model(f'nba_model_epoch_{epoch+1}.pt')
                    print(f"✓ Saved best model (val_loss: {val_loss:.4f})")
            
            # Log final model
            mlflow.pytorch.log_model(self.model, "nba_model")
            
        print("\nTraining complete!")
    
    def _train_epoch(self, train_loader: DataLoader) -> Tuple[float, Dict]:
        """Train for one epoch"""
        self.model.train()
        total_loss = 0
        
        progress_bar = tqdm(train_loader, desc="Training")
        for batch in progress_bar:
            # Move to device
            text_features = {
                'input_ids': batch['input_ids'].to(self.device),
                'attention_mask': batch['attention_mask'].to(self.device)
            }
            numeric_features = batch['numeric_features'].to(self.device)
            signal_features = batch['signal_features'].to(self.device)
            labels = {
                'action': batch['action_label'].to(self.device),
                'urgency': batch['urgency_label'].to(self.device),
                'value': batch['value_label'].to(self.device),
                'success': batch['success_label'].to(self.device)
            }
            
            # Forward pass
            predictions = self.model(text_features, numeric_features, signal_features)
            loss, loss_components = self.criterion(predictions, labels)
            
            # Backward pass
            self.optimizer.zero_grad()
            loss.backward()
            torch.nn.utils.clip_grad_norm_(self.model.parameters(), max_norm=1.0)
            self.optimizer.step()
            
            total_loss += loss.item()
            progress_bar.set_postfix({'loss': loss.item()})
        
        avg_loss = total_loss / len(train_loader)
        return avg_loss, {}
    
    def _validate_epoch(self, val_loader: DataLoader) -> Tuple[float, Dict]:
        """Validate for one epoch"""
        self.model.eval()
        total_loss = 0
        correct_top5 = 0
        total_samples = 0
        
        with torch.no_grad():
            for batch in tqdm(val_loader, desc="Validation"):
                # Move to device
                text_features = {
                    'input_ids': batch['input_ids'].to(self.device),
                    'attention_mask': batch['attention_mask'].to(self.device)
                }
                numeric_features = batch['numeric_features'].to(self.device)
                signal_features = batch['signal_features'].to(self.device)
                labels = {
                    'action': batch['action_label'].to(self.device),
                    'urgency': batch['urgency_label'].to(self.device),
                    'value': batch['value_label'].to(self.device),
                    'success': batch['success_label'].to(self.device)
                }
                
                # Forward pass
                predictions = self.model(text_features, numeric_features, signal_features)
                loss, _ = self.criterion(predictions, labels)
                
                total_loss += loss.item()
                
                # Top-5 accuracy
                _, top5_pred = torch.topk(predictions['action_logits'], k=5, dim=1)
                correct_top5 += (top5_pred == labels['action'].unsqueeze(1)).any(dim=1).sum().item()
                total_samples += labels['action'].size(0)
        
        avg_loss = total_loss / len(val_loader)
        top5_accuracy = correct_top5 / total_samples
        
        return avg_loss, {'top5_accuracy': top5_accuracy}
    
    def save_model(self, path: str):
        """Save model checkpoint"""
        torch.save({
            'model_state_dict': self.model.state_dict(),
            'optimizer_state_dict': self.optimizer.state_dict(),
            'model_config': self.model_config,
            'training_config': self.training_config
        }, path)

# Main training script
if __name__ == '__main__':
    # Database configuration
    db_config = {
        'host': os.getenv('DB_HOST', 'localhost'),
        'port': os.getenv('DB_PORT', 5432),
        'database': os.getenv('DB_NAME', 'semlayer'),
        'user': os.getenv('DB_USER', 'postgres'),
        'password': os.getenv('DB_PASSWORD', 'password')
    }
    
    # Model configuration
    model_config = {
        'num_actions': 50,
        'client_embedding_dim': 128,
        'freeze_bert': False
    }
    
    # Training configuration
    training_config = {
        'learning_rate': 2e-5,
        'weight_decay': 0.01
    }
    
    # Initialize trainer
    trainer = NBATrainer(db_config, model_config, training_config)
    
    # Load data
    train_dataset, val_dataset = trainer.load_training_data()
    
    # Train
    trainer.train(train_dataset, val_dataset, epochs=10, batch_size=16)
