"""
NBA ML Inference Service - Main Application

AI-Driven Next Best Action Engine for Wealth Management Advisors.
Provides real-time action recommendations using a multi-task neural network
that predicts optimal actions, urgency, expected value, and success probability.
"""

import os
import logging
from contextlib import asynccontextmanager
from typing import Optional

import structlog
from fastapi import FastAPI, HTTPException, Depends
from fastapi.middleware.cors import CORSMiddleware
from prometheus_client import make_asgi_app, Counter, Histogram, Gauge

from app.models.nba_model import NBAInferenceService
from app.schemas import (
    PredictionRequest,
    PredictionResponse,
    HealthResponse,
    TrainingRequest,
    BatchPredictionRequest,
    BatchPredictionResponse,
)
from app.database import get_db_session, DatabaseSession
from app.config import Settings

# Configure structured logging
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
        structlog.processors.JSONRenderer()
    ],
    wrapper_class=structlog.stdlib.BoundLogger,
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger(__name__)

# Prometheus metrics
PREDICTION_COUNTER = Counter(
    'nba_predictions_total',
    'Total number of NBA predictions made',
    ['signal_type', 'action_type']
)
PREDICTION_LATENCY = Histogram(
    'nba_prediction_latency_seconds',
    'Latency of NBA predictions',
    buckets=[0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0]
)
MODEL_LOADED = Gauge(
    'nba_model_loaded',
    'Whether the NBA model is loaded'
)
ACTIVE_REQUESTS = Gauge(
    'nba_active_requests',
    'Number of active prediction requests'
)

# Global inference service
inference_service: Optional[NBAInferenceService] = None
settings = Settings()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan handler for startup/shutdown."""
    global inference_service
    
    logger.info("Starting NBA ML Inference Service")
    
    # Load model on startup
    try:
        model_path = os.getenv("MODEL_PATH", "models/nba_model.pt")
        inference_service = NBAInferenceService(model_path=model_path)
        MODEL_LOADED.set(1)
        logger.info("NBA model loaded successfully", model_path=model_path)
    except Exception as e:
        logger.warning("Could not load pretrained model, using fallback", error=str(e))
        inference_service = NBAInferenceService(model_path=None)  # Fallback mode
        MODEL_LOADED.set(0)
    
    yield
    
    # Cleanup on shutdown
    logger.info("Shutting down NBA ML Inference Service")
    MODEL_LOADED.set(0)


# Create FastAPI application
app = FastAPI(
    title="NBA ML Inference Service",
    description="AI-Driven Next Best Action Engine for Wealth Management Advisors",
    version="1.0.0",
    lifespan=lifespan,
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Mount Prometheus metrics endpoint
metrics_app = make_asgi_app()
app.mount("/metrics", metrics_app)


def get_inference_service() -> NBAInferenceService:
    """Dependency to get the inference service."""
    if inference_service is None:
        raise HTTPException(
            status_code=503,
            detail="Model not loaded. Service is starting up."
        )
    return inference_service


@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    return HealthResponse(
        status="healthy",
        model_loaded=inference_service is not None,
        version="1.0.0"
    )


@app.post("/predict", response_model=PredictionResponse)
async def predict(
    request: PredictionRequest,
    service: NBAInferenceService = Depends(get_inference_service),
    db: DatabaseSession = Depends(get_db_session)
):
    """
    Generate Next Best Action recommendations for a client based on detected signals.
    
    This endpoint takes a detected signal and client context, then returns
    a ranked list of recommended actions with confidence scores.
    """
    ACTIVE_REQUESTS.inc()
    
    try:
        with PREDICTION_LATENCY.time():
            logger.info(
                "Processing prediction request",
                client_id=request.client_id,
                signal_type=request.signal.signal_type
            )
            
            # Get recommendations from model
            recommendations = await service.predict(
                client_id=request.client_id,
                signal=request.signal,
                db_session=db
            )
            
            # Track metrics
            for rec in recommendations:
                PREDICTION_COUNTER.labels(
                    signal_type=request.signal.signal_type,
                    action_type=rec.action_type
                ).inc()
            
            logger.info(
                "Prediction completed",
                client_id=request.client_id,
                num_recommendations=len(recommendations)
            )
            
            return PredictionResponse(
                client_id=request.client_id,
                recommendations=recommendations,
                generated_at=request.signal.detected_at
            )
            
    except Exception as e:
        logger.error(
            "Prediction failed",
            client_id=request.client_id,
            error=str(e)
        )
        raise HTTPException(status_code=500, detail=str(e))
    finally:
        ACTIVE_REQUESTS.dec()


@app.post("/predict/batch", response_model=BatchPredictionResponse)
async def predict_batch(
    request: BatchPredictionRequest,
    service: NBAInferenceService = Depends(get_inference_service),
    db: DatabaseSession = Depends(get_db_session)
):
    """
    Generate Next Best Action recommendations for multiple clients in batch.
    
    Useful for scheduled batch processing of all clients' signals.
    """
    ACTIVE_REQUESTS.inc()
    
    try:
        logger.info(
            "Processing batch prediction request",
            num_requests=len(request.requests)
        )
        
        results = []
        for pred_request in request.requests:
            with PREDICTION_LATENCY.time():
                recommendations = await service.predict(
                    client_id=pred_request.client_id,
                    signal=pred_request.signal,
                    db_session=db
                )
                
                results.append(PredictionResponse(
                    client_id=pred_request.client_id,
                    recommendations=recommendations,
                    generated_at=pred_request.signal.detected_at
                ))
        
        logger.info(
            "Batch prediction completed",
            num_processed=len(results)
        )
        
        return BatchPredictionResponse(results=results)
        
    except Exception as e:
        logger.error("Batch prediction failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))
    finally:
        ACTIVE_REQUESTS.dec()


@app.post("/train")
async def trigger_training(
    request: TrainingRequest,
    service: NBAInferenceService = Depends(get_inference_service),
    db: DatabaseSession = Depends(get_db_session)
):
    """
    Trigger model retraining with latest outcome data.
    
    This endpoint initiates the model retraining process using
    historical action outcomes from the database.
    """
    logger.info(
        "Training request received",
        lookback_days=request.lookback_days
    )
    
    try:
        # This would typically be async/background job
        result = await service.retrain(
            db_session=db,
            lookback_days=request.lookback_days
        )
        
        logger.info("Training completed", metrics=result)
        
        return {
            "status": "completed",
            "metrics": result
        }
        
    except Exception as e:
        logger.error("Training failed", error=str(e))
        raise HTTPException(status_code=500, detail=str(e))


@app.get("/model/info")
async def model_info(
    service: NBAInferenceService = Depends(get_inference_service)
):
    """Get information about the currently loaded model."""
    return service.get_model_info()


if __name__ == "__main__":
    import uvicorn
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=int(os.getenv("PORT", 5001)),
        reload=True
    )
