"""
Database connection and session management for NBA ML Service.
"""

import os
from typing import Optional, Generator
from contextlib import contextmanager

from sqlalchemy import create_engine, text
from sqlalchemy.orm import sessionmaker, Session
from sqlalchemy.pool import QueuePool

from app.config import get_settings

# Type alias for clarity
DatabaseSession = Session

# Global engine and session factory
_engine = None
_SessionLocal = None


def get_engine():
    """Get or create the SQLAlchemy engine."""
    global _engine
    if _engine is None:
        settings = get_settings()
        _engine = create_engine(
            settings.database_url,
            poolclass=QueuePool,
            pool_size=settings.db_pool_size,
            max_overflow=settings.db_max_overflow,
            pool_pre_ping=True,  # Verify connections before use
        )
    return _engine


def get_session_factory():
    """Get or create the session factory."""
    global _SessionLocal
    if _SessionLocal is None:
        _SessionLocal = sessionmaker(
            autocommit=False,
            autoflush=False,
            bind=get_engine()
        )
    return _SessionLocal


def get_db_session() -> Generator[Session, None, None]:
    """
    Dependency that provides a database session.
    Automatically handles commit/rollback and closing.
    """
    SessionLocal = get_session_factory()
    session = SessionLocal()
    try:
        yield session
        session.commit()
    except Exception:
        session.rollback()
        raise
    finally:
        session.close()


@contextmanager
def get_db_context() -> Generator[Session, None, None]:
    """Context manager for database sessions (for non-FastAPI usage)."""
    SessionLocal = get_session_factory()
    session = SessionLocal()
    try:
        yield session
        session.commit()
    except Exception:
        session.rollback()
        raise
    finally:
        session.close()


class ClientRepository:
    """Repository for client-related database queries."""
    
    def __init__(self, session: Session):
        self.session = session
    
    def get_client_profile(self, client_id: str) -> Optional[dict]:
        """Fetch client profile features for ML model."""
        query = text("""
            SELECT 
                c.id,
                c.first_name,
                c.last_name,
                EXTRACT(YEAR FROM AGE(c.date_of_birth)) as age,
                c.risk_tolerance_score,
                c.liquidity_needs_score,
                c.tax_bracket,
                ps.total_portfolio_value as net_worth,
                ps.total_portfolio_value as aum,
                ps.cash_balance / NULLIF(ps.total_portfolio_value, 0) as cash_allocation,
                -- Calculate other allocations from holdings
                COALESCE(eq.equity_pct, 0) as equity_allocation,
                COALESCE(fi.fixed_income_pct, 0) as fixed_income_allocation,
                COALESCE(alt.alternative_pct, 0) as alternative_allocation,
                -- Engagement metrics
                COALESCE(eng.portal_logins_90d, 0) as portal_logins_90d,
                COALESCE(eng.email_open_rate, 0.5) as email_open_rate,
                COALESCE(eng.last_meeting_days_ago, 90) as last_meeting_days_ago,
                -- Calculated scores
                COALESCE(eng.satisfaction_score, 0.7) as satisfaction_score,
                COALESCE(eng.flight_risk_score, 0.3) as flight_risk_score
            FROM clients c
            LEFT JOIN portfolio_summary ps ON c.id = ps.client_id
            LEFT JOIN (
                SELECT client_id, 
                       SUM(CASE WHEN asset_class = 'EQUITY' THEN position_value ELSE 0 END) / 
                       NULLIF(SUM(position_value), 0) as equity_pct
                FROM holdings GROUP BY client_id
            ) eq ON c.id = eq.client_id
            LEFT JOIN (
                SELECT client_id,
                       SUM(CASE WHEN asset_class = 'FIXED_INCOME' THEN position_value ELSE 0 END) /
                       NULLIF(SUM(position_value), 0) as fixed_income_pct
                FROM holdings GROUP BY client_id
            ) fi ON c.id = fi.client_id
            LEFT JOIN (
                SELECT client_id,
                       SUM(CASE WHEN asset_class = 'ALTERNATIVE' THEN position_value ELSE 0 END) /
                       NULLIF(SUM(position_value), 0) as alternative_pct
                FROM holdings GROUP BY client_id
            ) alt ON c.id = alt.client_id
            LEFT JOIN (
                SELECT 
                    client_id,
                    COUNT(*) FILTER (WHERE login_at > NOW() - INTERVAL '90 days') as portal_logins_90d,
                    AVG(CASE WHEN opened_at IS NOT NULL THEN 1.0 ELSE 0.0 END) as email_open_rate,
                    EXTRACT(DAY FROM NOW() - MAX(meeting_date)) as last_meeting_days_ago,
                    AVG(satisfaction_rating) as satisfaction_score,
                    -- Simple flight risk based on engagement
                    CASE 
                        WHEN COUNT(*) FILTER (WHERE login_at > NOW() - INTERVAL '30 days') = 0 THEN 0.8
                        WHEN COUNT(*) FILTER (WHERE login_at > NOW() - INTERVAL '30 days') < 2 THEN 0.5
                        ELSE 0.2
                    END as flight_risk_score
                FROM client_engagement_metrics
                GROUP BY client_id
            ) eng ON c.id = eng.client_id
            WHERE c.id = :client_id
        """)
        
        try:
            result = self.session.execute(query, {"client_id": client_id})
            row = result.fetchone()
            if row:
                return dict(row._mapping)
            return None
        except Exception:
            # Tables might not exist, return None
            return None
    
    def get_client_name(self, client_id: str) -> str:
        """Fetch client name."""
        query = text("""
            SELECT CONCAT(first_name, ' ', last_name) as name
            FROM clients
            WHERE id = :client_id
        """)
        
        try:
            result = self.session.execute(query, {"client_id": client_id})
            row = result.fetchone()
            if row:
                return row.name
        except Exception:
            pass
        return "Client"
    
    def get_recent_crm_notes(self, client_id: str, days: int = 90) -> str:
        """Fetch recent CRM notes for text context."""
        query = text("""
            SELECT note_text
            FROM crm_notes
            WHERE client_id = :client_id
                            AND created_at > NOW() - make_interval(days => :days)
            ORDER BY created_at DESC
            LIMIT 10
        """)
        
        try:
            result = self.session.execute(query, {"client_id": client_id, "days": days})
            notes = [row.note_text for row in result]
            return "\n".join(notes) if notes else ""
        except Exception:
            return ""


class OutcomeRepository:
    """Repository for action outcome data (for training)."""
    
    def __init__(self, session: Session):
        self.session = session
    
    def get_training_data(self, lookback_days: int = 90) -> list:
        """Fetch completed action outcomes for model training."""
        query = text("""
            SELECT 
                o.outcome_id,
                o.action_id,
                o.client_id,
                o.advisor_id,
                o.trigger_signal_type,
                o.client_responded,
                o.action_successful,
                o.revenue_generated,
                o.client_satisfaction_change,
                o.aum_change,
                o.advisor_rating,
                c.action_code,
                c.action_name,
                c.action_category,
                c.estimated_revenue_impact,
                c.estimated_duration_minutes
            FROM nba_action_outcomes o
            JOIN nba_action_catalog c ON o.action_id = c.action_id
                        WHERE o.completed_at > NOW() - make_interval(days => :days)
              AND o.executed_at IS NOT NULL
            ORDER BY o.completed_at DESC
        """)
        
        try:
            result = self.session.execute(query, {"days": lookback_days})
            return [dict(row._mapping) for row in result]
        except Exception:
            return []
    
    def get_action_catalog(self) -> list:
        """Fetch all actions from the catalog."""
        query = text("""
            SELECT 
                action_id,
                action_code,
                action_name,
                action_category,
                description,
                default_channel,
                estimated_duration_minutes,
                estimated_revenue_impact,
                client_value_impact,
                automation_eligible,
                template_content,
                required_advisor_skills,
                compliance_review_required,
                success_metrics
            FROM nba_action_catalog
            ORDER BY action_name
        """)
        
        try:
            result = self.session.execute(query)
            return [dict(row._mapping) for row in result]
        except Exception:
            return []
