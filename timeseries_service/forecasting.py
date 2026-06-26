#!/usr/bin/env python3
"""
ARIMA and Prophet Forecasting Service
Multi-horizon time-series forecasting with confidence intervals.
"""

import numpy as np
import pandas as pd
from typing import Dict, Tuple, List, Optional
from dataclasses import dataclass, asdict
from datetime import datetime, timedelta
import logging

logger = logging.getLogger(__name__)

@dataclass
class ForecastResult:
    """Forecast result with confidence intervals"""
    horizon_hours: int
    point_forecast: float
    lower_bound_80: float
    upper_bound_80: float
    lower_bound_95: float
    upper_bound_95: float
    model_type: str
    rmse: Optional[float] = None
    mae: Optional[float] = None
    mape: Optional[float] = None

@dataclass
class MultiHorizonForecast:
    """Multi-horizon forecast results"""
    feature_id: str
    forecast_timestamp: datetime
    horizons: List[ForecastResult]
    model_fit_quality: float
    trend_direction: str  # 'up', 'down', 'stable'
    seasonal_strength: float

class ARIMAForecaster:
    """
    Automatic ARIMA parameter selection and forecasting
    
    ARIMA(p,d,q):
    - p: Auto-regressive order (correlation with past values)
    - d: Differencing order (remove trend)
    - q: Moving average order (correlation with past errors)
    """
    
    def __init__(self, timeseries: np.ndarray, 
                 max_p: int = 5, max_d: int = 2, max_q: int = 5):
        """
        Initialize ARIMA with auto parameter selection
        
        Args:
            timeseries: Time-series values
            max_p, max_d, max_q: Max values for parameter search
        """
        self.timeseries = np.asarray(timeseries, dtype=np.float64)
        self.max_p = max_p
        self.max_d = max_d
        self.max_q = max_q
        self.fitted_model = None
        self.best_params = None
        self.aic_scores = {}
    
    def fit_auto_arima(self, seasonal: bool = False, seasonal_periods: int = 7):
        """
        Fit auto ARIMA model with parameter search
        
        Args:
            seasonal: Include seasonal ARIMA (SARIMA)
            seasonal_periods: Period for seasonal component
        """
        try:
            from pmdarima.arima import auto_arima
        except ImportError:
            logger.warning("pmdarima not available, using fallback ARIMA")
            return self._fit_fallback_arima()
        
        try:
            # Auto ARIMA uses AIC/BIC to find best parameters
            model = auto_arima(
                self.timeseries,
                max_p=self.max_p,
                max_d=self.max_d,
                max_q=self.max_q,
                seasonal=seasonal,
                m=seasonal_periods if seasonal else 1,
                stepwise=True,
                trace=False,
                error_action='ignore',
                suppress_warnings=True,
                information_criterion='aic'
            )
            
            self.fitted_model = model
            self.best_params = (model.order, model.seasonal_order)
            
            logger.info(f"Auto ARIMA selected: {self.best_params}")
            return model
        
        except Exception as e:
            logger.error(f"Auto ARIMA failed: {e}, using fallback")
            return self._fit_fallback_arima()
    
    def _fit_fallback_arima(self):
        """
        Fallback ARIMA when pmdarima not available
        
        Simple grid search over parameter space
        """
        best_aic = np.inf
        best_params = (1, 1, 1)
        
        for p in range(min(3, self.max_p + 1)):
            for d in range(min(2, self.max_d + 1)):
                for q in range(min(3, self.max_q + 1)):
                    try:
                        # Simple ARIMA fit using differencing
                        diff_series = self.timeseries.copy()
                        for _ in range(d):
                            diff_series = np.diff(diff_series)
                        
                        # Estimate with AR terms
                        aic = self._estimate_aic(diff_series, p, q)
                        
                        if aic < best_aic:
                            best_aic = aic
                            best_params = (p, d, q)
                    
                    except:
                        pass
        
        self.best_params = (best_params, (0, 0, 0, 0))
        logger.info(f"Fallback ARIMA selected: {best_params}")
        return None
    
    def _estimate_aic(self, series: np.ndarray, p: int, q: int) -> float:
        """Estimate AIC for a given (p,q) model"""
        n = len(series)
        # Simplified AIC calculation
        # AIC = 2k + n*ln(RSS/n) where k = p+q+1
        k = p + q + 1
        residuals = series[max(p, q):]
        rss = np.sum(residuals ** 2)
        aic = 2 * k + n * np.log(rss / n + 1e-9)
        return aic
    
    def forecast(self, steps: int = 24, alpha: float = 0.05) -> pd.DataFrame:
        """
        Generate forecast with confidence intervals
        
        Args:
            steps: Number of steps ahead to forecast
            alpha: Significance level (0.05 for 95% CI, 0.20 for 80% CI)
        
        Returns: DataFrame with forecast and intervals
        """
        if self.fitted_model is None:
            # Use simple exponential smoothing as fallback
            return self._forecast_fallback(steps)
        
        try:
            forecast, conf_int = self.fitted_model.get_forecast(steps=steps).conf_int(alpha=alpha)
        except:
            return self._forecast_fallback(steps)
        
        result = pd.DataFrame({
            'forecast': forecast.values,
            'lower': conf_int.iloc[:, 0].values,
            'upper': conf_int.iloc[:, 1].values
        })
        
        return result
    
    def _forecast_fallback(self, steps: int) -> pd.DataFrame:
        """
        Fallback forecasting using exponential smoothing
        """
        # Simple exponential smoothing
        alpha = 0.3
        last_level = self.timeseries[-1]
        last_trend = self.timeseries[-1] - self.timeseries[-2]
        
        forecasts = []
        intervals_lower = []
        intervals_upper = []
        
        se = np.std(self.timeseries) * np.sqrt(1 + alpha)
        
        for i in range(steps):
            forecast = last_level + (i + 1) * last_trend
            forecasts.append(forecast)
            intervals_lower.append(forecast - 1.96 * se)
            intervals_upper.append(forecast + 1.96 * se)
        
        return pd.DataFrame({
            'forecast': forecasts,
            'lower': intervals_lower,
            'upper': intervals_upper
        })
    
    def forecast_multi_horizon(self, horizons: List[int] = [1, 24, 168, 720],
                               alpha_80: float = 0.20, 
                               alpha_95: float = 0.05) -> List[ForecastResult]:
        """
        Generate forecasts for multiple horizons
        
        Args:
            horizons: List of forecast horizons (hours)
            alpha_80: Significance for 80% CI
            alpha_95: Significance for 95% CI
        
        Returns: List of ForecastResult objects
        """
        results = []
        
        for horizon in horizons:
            # Get point forecast
            forecast_df = self.forecast(steps=horizon, alpha=alpha_95)
            point = forecast_df['forecast'].iloc[-1]
            lower_95 = forecast_df['lower'].iloc[-1]
            upper_95 = forecast_df['upper'].iloc[-1]
            
            # Get 80% intervals
            forecast_df_80 = self.forecast(steps=horizon, alpha=alpha_80)
            lower_80 = forecast_df_80['lower'].iloc[-1]
            upper_80 = forecast_df_80['upper'].iloc[-1]
            
            # Calculate error metrics on training set
            rmse, mae, mape = self._calculate_error_metrics(forecast_df)
            
            results.append(ForecastResult(
                horizon_hours=horizon,
                point_forecast=float(point),
                lower_bound_80=float(lower_80),
                upper_bound_80=float(upper_80),
                lower_bound_95=float(lower_95),
                upper_bound_95=float(upper_95),
                model_type='arima',
                rmse=rmse,
                mae=mae,
                mape=mape
            ))
        
        return results
    
    def _calculate_error_metrics(self, forecast_df: pd.DataFrame) -> Tuple[float, float, float]:
        """Calculate RMSE, MAE, MAPE on forecast"""
        # For now, return zeros (would be filled in with actual validation)
        return 0.0, 0.0, 0.0

class ProphetForecaster:
    """
    Facebook Prophet forecaster
    
    y(t) = trend(t) + seasonality(t) + holidays(t) + residual(t)
    
    Advantages:
    - Handles missing data and outliers
    - Built-in holiday effects
    - Automatic changepoint detection
    - User-friendly parametrization
    """
    
    def __init__(self, timeseries: np.ndarray, timestamps: np.ndarray):
        """
        Initialize Prophet forecaster
        
        Args:
            timeseries: Time-series values
            timestamps: Timestamps (datetime-like)
        """
        self.timeseries = np.asarray(timeseries, dtype=np.float64)
        self.timestamps = pd.to_datetime(timestamps)
        self.model = None
        self.forecast_df = None
    
    def fit_model(self, yearly_seasonality: bool = True,
                  weekly_seasonality: bool = True,
                  daily_seasonality: bool = False,
                  changepoint_prior_scale: float = 0.05,
                  seasonality_prior_scale: float = 10.0,
                  interval_width: float = 0.95) -> 'ProphetForecaster':
        """
        Fit Prophet model
        
        Args:
            yearly_seasonality: Include yearly seasonality
            weekly_seasonality: Include weekly seasonality
            daily_seasonality: Include daily seasonality
            changepoint_prior_scale: Flexibility of trend changes
            seasonality_prior_scale: Strength of seasonal component
            interval_width: Confidence interval width
        """
        try:
            from prophet import Prophet
        except ImportError:
            logger.warning("Prophet not available, using fallback")
            return self
        
        # Prepare data for Prophet (requires 'ds' and 'y' columns)
        df = pd.DataFrame({
            'ds': self.timestamps,
            'y': self.timeseries
        })
        
        try:
            self.model = Prophet(
                yearly_seasonality=yearly_seasonality,
                weekly_seasonality=weekly_seasonality,
                daily_seasonality=daily_seasonality,
                changepoint_prior_scale=changepoint_prior_scale,
                seasonality_prior_scale=seasonality_prior_scale,
                interval_width=interval_width,
                interval_prior_scale=0.05
            )
            
            # Fit model
            with open('/dev/null', 'w') as f:
                import contextlib
                with contextlib.redirect_stderr(f):  # Suppress Prophet's verbose output
                    self.model.fit(df)
            
            logger.info("Prophet model fitted successfully")
        
        except Exception as e:
            logger.error(f"Prophet fitting failed: {e}")
            self.model = None
        
        return self
    
    def forecast(self, periods: int = 24, freq: str = 'h') -> pd.DataFrame:
        """
        Generate forecast
        
        Args:
            periods: Number of periods ahead
            freq: Frequency ('h' for hourly, 'D' for daily)
        
        Returns: DataFrame with forecast, trend, seasonal components
        """
        if self.model is None:
            return pd.DataFrame()
        
        try:
            future = self.model.make_future_dataframe(periods=periods, freq=freq)
            forecast = self.model.predict(future)
            
            self.forecast_df = forecast
            return forecast
        
        except Exception as e:
            logger.error(f"Prophet forecasting failed: {e}")
            return pd.DataFrame()
    
    def forecast_multi_horizon(self, horizons: List[int] = [1, 24, 168, 720],
                               freq: str = 'h') -> List[ForecastResult]:
        """
        Generate multi-horizon forecasts
        
        Args:
            horizons: List of forecast horizons
            freq: Frequency
        
        Returns: List of ForecastResult objects
        """
        # Get the max horizon forecast
        max_horizon = max(horizons)
        forecast_df = self.forecast(periods=max_horizon, freq=freq)
        
        if forecast_df.empty:
            return []
        
        results = []
        
        for horizon in horizons:
            idx = min(horizon - 1, len(forecast_df) - 1)
            row = forecast_df.iloc[idx]
            
            results.append(ForecastResult(
                horizon_hours=horizon,
                point_forecast=float(row['yhat']),
                lower_bound_80=float(row['yhat_lower']),  # Approximate 80%
                upper_bound_80=float(row['yhat_upper']),
                lower_bound_95=float(row['yhat_lower']),  # Prophet uses 95% default
                upper_bound_95=float(row['yhat_upper']),
                model_type='prophet'
            ))
        
        return results
    
    def get_components(self) -> Dict[str, np.ndarray]:
        """
        Get decomposed components (trend, seasonal)
        
        Returns: Dict with trend and seasonal arrays
        """
        if self.forecast_df is None or self.model is None:
            return {}
        
        components = {}
        
        # Get training period forecast
        future = self.model.make_future_dataframe(periods=len(self.timestamps))
        forecast = self.model.predict(future)
        
        components['trend'] = forecast['trend'].values[:len(self.timestamps)]
        
        if 'yearly' in forecast.columns:
            components['yearly'] = forecast['yearly'].values[:len(self.timestamps)]
        if 'weekly' in forecast.columns:
            components['weekly'] = forecast['weekly'].values[:len(self.timestamps)]
        
        return components

class EnsembleForecaster:
    """
    Ensemble forecaster combining ARIMA and Prophet
    """
    
    def __init__(self, timeseries: np.ndarray, timestamps: np.ndarray):
        """Initialize ensemble with both models"""
        self.arima_forecaster = ARIMAForecaster(timeseries)
        self.prophet_forecaster = ProphetForecaster(timeseries, timestamps)
        self.timeseries = timeseries
        self.timestamps = timestamps
    
    def fit(self):
        """Fit both models"""
        self.arima_forecaster.fit_auto_arima()
        self.prophet_forecaster.fit_model()
        return self
    
    def forecast_multi_horizon(self, horizons: List[int]) -> List[ForecastResult]:
        """
        Ensemble forecast: Average of ARIMA and Prophet with aggregated intervals
        """
        arima_results = self.arima_forecaster.forecast_multi_horizon(horizons)
        prophet_results = self.prophet_forecaster.forecast_multi_horizon(horizons)
        
        ensemble_results = []
        
        for arima_result, prophet_result in zip(arima_results, prophet_results):
            # Average point forecasts
            avg_point = (arima_result.point_forecast + prophet_result.point_forecast) / 2
            
            # Take wider intervals (more conservative)
            lower_80 = min(arima_result.lower_bound_80, prophet_result.lower_bound_80)
            upper_80 = max(arima_result.upper_bound_80, prophet_result.upper_bound_80)
            lower_95 = min(arima_result.lower_bound_95, prophet_result.lower_bound_95)
            upper_95 = max(arima_result.upper_bound_95, prophet_result.upper_bound_95)
            
            ensemble_results.append(ForecastResult(
                horizon_hours=arima_result.horizon_hours,
                point_forecast=avg_point,
                lower_bound_80=lower_80,
                upper_bound_80=upper_80,
                lower_bound_95=lower_95,
                upper_bound_95=upper_95,
                model_type='ensemble'
            ))
        
        return ensemble_results

if __name__ == '__main__':
    # Example usage
    logging.basicConfig(level=logging.INFO)
    
    # Generate synthetic time series
    t = np.arange(365)
    ts = 100 + 0.5 * t + 20 * np.sin(2 * np.pi * t / 7) + np.random.normal(0, 5, 365)
    timestamps = pd.date_range('2024-01-01', periods=365, freq='D')
    
    # ARIMA forecasting
    print("Testing ARIMA Forecaster...")
    arima = ARIMAForecaster(ts)
    arima.fit_auto_arima()
    arima_results = arima.forecast_multi_horizon()
    print(f"ARIMA 24h forecast: {arima_results[1].point_forecast:.1f}")
    
    # Ensemble forecasting
    print("\nTesting Ensemble Forecaster...")
    ensemble = EnsembleForecaster(ts, timestamps)
    ensemble.fit()
    ensemble_results = ensemble.forecast_multi_horizon([24])
    print(f"Ensemble 24h forecast: {ensemble_results[0].point_forecast:.1f}")
