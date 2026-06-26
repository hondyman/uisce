"""
Configuration settings for NBA ML Service.
"""

import os
from typing import Optional
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    """Application settings loaded from environment variables."""
    
    # Service settings
    service_name: str = "nba-ml-service"
    debug: bool = False
    log_level: str = "INFO"
    
    # Model settings
    model_path: str = "models/nba_model.pt"
    bert_model_name: str = "bert-base-uncased"
    num_actions: int = 50
    client_embedding_dim: int = 128
    
    # Database settings
    database_url: str = "postgresql://postgres:postgres@localhost:5432/alpha"
    db_pool_size: int = 5
    db_max_overflow: int = 10
    
    # Feature settings
    num_numeric_features: int = 25
    num_signal_features: int = 10
    max_text_length: int = 512
    
    # Inference settings
    top_k_recommendations: int = 5
    confidence_threshold: float = 0.7
    batch_size: int = 32
    
    # Redis settings (for caching)
    redis_url: Optional[str] = None
    cache_ttl_seconds: int = 300
    
    # Server settings
    host: str = "0.0.0.0"
    port: int = 5001
    
    class Config:
        env_prefix = "NBA_ML_"
        env_file = ".env"
        case_sensitive = False


# Singleton settings instance
_settings: Optional[Settings] = None


def get_settings() -> Settings:
    """Get settings singleton."""
    global _settings
    if _settings is None:
        _settings = Settings()
    return _settings
