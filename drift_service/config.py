"""Configuration for Drift Detection Service"""

from pydantic_settings import BaseSettings
from typing import Optional

class Settings(BaseSettings):
    # Service
    HOST: str = "0.0.0.0"
    PORT: int = 8000
    WORKERS: int = 4
    DEBUG: bool = False
    
    # Database
    POSTGRES_HOST: str = "localhost"
    POSTGRES_PORT: int = 5432
    POSTGRES_USER: str = "postgres"
    POSTGRES_PASSWORD: str = "secret"
    POSTGRES_DB: str = "semlayer"
    
    # Iceberg / Trino
    TRINO_HOST: str = "localhost"
    TRINO_PORT: int = 8080
    TRINO_CATALOG: str = "iceberg"
    TRINO_SCHEMA: str = "features"
    
    # Drift detection
    KS_THRESHOLD: float = 0.05
    PSI_THRESHOLD: float = 0.15
    CHI2_THRESHOLD: float = 0.10
    CLASSIFIER_THRESHOLD: float = 0.55
    
    # Alerting
    ALERT_WEBHOOK_URL: Optional[str] = None
    ALERT_EMAIL_RECIPIENTS: str = ""
    ALERT_PAGERDUTY_KEY: Optional[str] = None
    
    # Metrics
    PROMETHEUS_PORT: int = 9090
    PROMETHEUS_ENABLED: bool = True
    
    # Logging
    LOG_LEVEL: str = "INFO"
    
    class Config:
        env_file = ".env"
        case_sensitive = True

settings = Settings()
