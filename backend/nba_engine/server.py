from flask import Flask, request, jsonify
import torch
from transformers import BertTokenizer
from model import NextBestActionModel
import os

app = Flask(__name__)

# Global variables
model = None
tokenizer = None
action_id_to_name = {
    0: "PROACTIVE_TAX_LOSS_HARVEST",
    1: "REENGAGEMENT_OUTREACH",
    2: "CONCENTRATED_POSITION_REVIEW",
    # Add more mappings as needed
}

def load_model():
    global model, tokenizer
    model_path = os.environ.get("NBA_MODEL_PATH", "nba_model.pth")
    
    model = NextBestActionModel(num_actions=50)
    if os.path.exists(model_path):
        model.load_state_dict(torch.load(model_path, map_location=torch.device('cpu')))
        print(f"Loaded model from {model_path}")
    else:
        print(f"Model file {model_path} not found, using initialized weights")
    
    model.eval()
    tokenizer = BertTokenizer.from_pretrained('bert-base-uncased')

@app.route('/predict', methods=['POST'])
def predict():
    data = request.json
    client_id = data.get('client_id')
    signal = data.get('signal')
    
    # In a real app, we would fetch features from DB or receive them in request
    # For now, we expect features in the request or mock them
    
    text_input = data.get('text', "Client is interested in tax optimization.")
    numeric_features = torch.randn(1, 25) # Mock
    signal_features = torch.randn(1, 10) # Mock
    
    text_tokens = tokenizer(
        text_input,
        return_tensors='pt',
        truncation=True,
        max_length=512,
        padding='max_length'
    )
    
    with torch.no_grad():
        predictions = model(
            text_features=text_tokens,
            numeric_features=numeric_features,
            signal_features=signal_features
        )
    
    action_probs = torch.softmax(predictions['action_logits'], dim=1)[0]
    top_actions = torch.topk(action_probs, k=5)
    
    recommendations = []
    for idx, prob in zip(top_actions.indices, top_actions.values):
        action_type = action_id_to_name.get(idx.item(), f"ACTION_{idx.item()}")
        
        recommendations.append({
            'action_type': action_type,
            'confidence': prob.item(),
            'urgency_score': predictions['urgency'][0].item(),
            'expected_value': predictions['expected_value'][0].item(),
            'success_probability': predictions['success_probability'][0].item(),
            'trigger_signal': signal.get('SignalType') if signal else "UNKNOWN",
            'reasoning': "AI generated recommendation based on signal and client profile."
        })
    
    return jsonify(recommendations)

@app.route('/health', methods=['GET'])
def health():
    return jsonify({"status": "ok"})

if __name__ == '__main__':
    load_model()
    app.run(host='0.0.0.0', port=5001)
