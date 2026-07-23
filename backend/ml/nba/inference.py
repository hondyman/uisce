"""
NBA Model Inference API (gRPC)

High-performance inference service for NBA model predictions
"""

import grpc
from concurrent import futures
import torch
from transformers import BertTokenizer
import numpy as np
from typing import List, Dict
import json

# Import proto generated files (would be generated from proto definition)
# import nba_pb2
# import nba_pb2_grpc

from model import NextBestActionModel

class NBAInferenceService:
    """
    NBA model inference service
    """
    
    def __init__(self, model_path: str, device: str = 'cuda'):
        self.device = torch.device(device if torch.cuda.is_available() else 'cpu')
        print(f"Loading model on {self.device}...")
        
        # Load model checkpoint
        checkpoint = torch.load(model_path, map_location=self.device)
        
        # Initialize model
        self.model = NextBestActionModel(**checkpoint['model_config'])
        self.model.load_state_dict(checkpoint['model_state_dict'])
        self.model.to(self.device)
        self.model.eval()
        
        # Tokenizer
        self.tokenizer = BertTokenizer.from_pretrained('bert-base-uncased')
        
        print("Model loaded successfully!")
    
    def predict(
        self,
        client_id: str,
        text_features: str,
        numeric_features: List[float],
        signal_features: List[float]
    ) -> Dict:
        """
        Generate NBA predictions for a client
        
        Args:
            client_id: Client UUID
            text_features: Combined CRM notes, emails
            numeric_features: 25 client/portfolio features
            signal_features: 10 signal context features
        
        Returns:
            Dict with top 5 recommended actions and scores
        """
        
        with torch.no_grad():
            # Tokenize text
            text_encoding = self.tokenizer(
                text_features,
                truncation=True,
                max_length=512,
                padding='max_length',
                return_tensors='pt'
            ).to(self.device)
            
            # Convert features to tensors
            numeric_tensor = torch.tensor([numeric_features], dtype=torch.float32).to(self.device)
            signal_tensor = torch.tensor([signal_features], dtype=torch.float32).to(self.device)
            
            # Model inference
            predictions = self.model(
                text_features=text_encoding,
                numeric_features=numeric_tensor,
                signal_features=signal_tensor
            )
            
            # Get top-5 actions
            action_probs = torch.softmax(predictions['action_logits'], dim=1)[0]
            top5_values, top5_indices = torch.topk(action_probs, k=5)
            
            # Format results
            recommendations = []
            for idx, prob in zip(top5_indices.cpu().numpy(), top5_values.cpu().numpy()):
                recommendations.append({
                    'action_id': int(idx),
                    'confidence': float(prob),
                    'urgency_score': float(predictions['urgency'][0].cpu().numpy()[0]),
                    'expected_value': float(predictions['expected_value'][0].cpu().numpy()[0]),
                    'success_probability': float(predictions['success_probability'][0].cpu().numpy()[0])
                })
            
            return {
                'client_id': client_id,
                'recommendations': recommendations,
                'model_version': 'nba-v1.0.0'
            }
    
    def batch_predict(
        self,
        batch_data: List[Dict]
    ) -> List[Dict]:
        """
        Batch inference for multiple clients
        
        More efficient than individual predictions
        """
        # Prepare batch tensors
        text_list = [item['text_features'] for item in batch_data]
        numeric_list = [item['numeric_features'] for item in batch_data]
        signal_list = [item['signal_features'] for item in batch_data]
        
        # Batch tokenization
        text_encodings = self.tokenizer(
            text_list,
            truncation=True,
            max_length=512,
            padding='max_length',
            return_tensors='pt'
        ).to(self.device)
        
        numeric_tensor = torch.tensor(numeric_list, dtype=torch.float32).to(self.device)
        signal_tensor = torch.tensor(signal_list, dtype=torch.float32).to(self.device)
        
        with torch.no_grad():
            predictions = self.model(
                text_features=text_encodings,
                numeric_features=numeric_tensor,
                signal_features=signal_tensor
            )
        
        # Format results for each client
        results = []
        batch_size = len(batch_data)
        
        for i in range(batch_size):
            action_probs = torch.softmax(predictions['action_logits'][i], dim=0)
            top5_values, top5_indices = torch.topk(action_probs, k=5)
            
            recommendations = []
            for idx, prob in zip(top5_indices.cpu().numpy(), top5_values.cpu().numpy()):
                recommendations.append({
                    'action_id': int(idx),
                    'confidence': float(prob),
                    'urgency_score': float(predictions['urgency'][i].cpu().numpy()[0]),
                    'expected_value': float(predictions['expected_value'][i].cpu().numpy()[0]),
                    'success_probability': float(predictions['success_probability'][i].cpu().numpy()[0])
                })
            
            results.append({
                'client_id': batch_data[i]['client_id'],
                'recommendations': recommendations
            })
        
        return results

# gRPC Server (placeholder - would implement actual proto definitions)
class NBAInferenceServicer:
    """
    gRPC servicer for NBA model
    """
    
    def __init__(self, model_path: str):
        self.inference_service = NBAInferenceService(model_path)
    
    def Predict(self, request, context):
        """
        RPC method for single prediction
        """
        result = self.inference_service.predict(
            client_id=request.client_id,
            text_features=request.text_features,
            numeric_features=list(request.numeric_features),
            signal_features=list(request.signal_features)
        )
        
        # Convert to proto response
        # return nba_pb2.PredictionResponse(**result)
        return result
    
    def BatchPredict(self, request, context):
        """
        RPC method for batch prediction
        """
        batch_data = [
            {
                'client_id': item.client_id,
                'text_features': item.text_features,
                'numeric_features': list(item.numeric_features),
                'signal_features': list(item.signal_features)
            }
            for item in request.items
        ]
        
        results = self.inference_service.batch_predict(batch_data)
        
        # Convert to proto response
        # return nba_pb2.BatchPredictionResponse(results=results)
        return results

def serve(model_path: str, port: int = 50051):
    """
    Start gRPC server
    """
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    
    # Register servicer
    servicer = NBAInferenceServicer(model_path)
    # nba_pb2_grpc.add_NBAInferenceServicer_to_server(servicer, server)
    
    server.add_insecure_port(f'[::]:{port}')
    server.start()
    
    print(f"NBA Inference Server started on port {port}")
    server.wait_for_termination()

if __name__ == '__main__':
    import sys
    
    if len(sys.argv) < 2:
        print("Usage: python inference.py <model_path>")
        sys.exit(1)
    
    model_path = sys.argv[1]
    serve(model_path)
