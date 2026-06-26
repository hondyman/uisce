"""
Phase 3.21: Drift Detection Service
Complete production-grade drift detection with multiple algorithms,
Prometheus metrics, PostgreSQL persistence, and Temporal workflow integration.
"""

import uvicorn
from fastapi import FastAPI, HTTPException
from contextlib import asynccontextmanager
import logging

from app.api import router as api_router
from app.config import settings
from app.metrics.prometheus import setup_metrics

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Lifespan context
@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("Drift Detection Service starting...")
    setup_metrics()
    yield
    logger.info("Drift Detection Service shutdown.")

# Create FastAPI app
app = FastAPI(
    title="Drift Detection Service",
    description="Multi-algorithm drift detection for feature engineering",
    version="3.21.0",
    lifespan=lifespan
)

# Include routers
app.include_router(api_router, prefix="/api/v1", tags=["drift"])

# Health check endpoint
@app.get("/health/live")
async def health_live():
    return {"status": "live"}

@app.get("/health/ready")
async def health_ready():
    return {"status": "ready"}

@app.get("/")
async def root():
    return {
        "service": "drift-detection",
        "version": "3.21.0",
        "status": "operational"
    }

if __name__ == "__main__":
    uvicorn.run(
        "main:app",
        host=settings.HOST,
        port=settings.PORT,
        workers=settings.WORKERS,
        reload=settings.DEBUG
    )
