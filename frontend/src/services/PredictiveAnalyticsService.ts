import { PoPComputation, TrendAnalysis, PredictiveModel } from '../types/dynamic';

export class PredictiveAnalyticsService {
  private static instance: PredictiveAnalyticsService;
  private models: Map<string, PredictiveModel> = new Map();

  static getInstance(): PredictiveAnalyticsService {
    if (!PredictiveAnalyticsService.instance) {
      PredictiveAnalyticsService.instance = new PredictiveAnalyticsService();
    }
    return PredictiveAnalyticsService.instance;
  }

  /**
   * Analyze trends in metric data
   */
  analyzeTrend(
    metricId: string,
    computations: PoPComputation[],
    period: string = '30d'
  ): TrendAnalysis {
    const sortedData = computations
      .sort((a, b) => new Date(a.periodStart).getTime() - new Date(b.periodStart).getTime())
      .slice(-30); // Use last 30 data points

    if (sortedData.length < 3) {
      return this.createEmptyTrendAnalysis(metricId, period);
    }

    const values = sortedData.map(comp => comp.currentValue);
    const dates = sortedData.map(comp => new Date(comp.periodStart).getTime());

    // Calculate linear regression
    const { slope, intercept, rSquared } = this.linearRegression(dates, values);

    // Determine trend direction
    let trend: 'increasing' | 'decreasing' | 'stable' | 'volatile' = 'stable';
    const slopeThreshold = Math.abs(slope) * dates.length / (dates[dates.length - 1] - dates[0]);

    if (slopeThreshold > 0.1) {
      trend = slope > 0 ? 'increasing' : 'decreasing';
    } else if (this.calculateVolatility(values) > 0.15) {
      trend = 'volatile';
    }

    // Check for seasonality
    const seasonality = this.detectSeasonality(values, 7); // Weekly seasonality

    // Generate forecast
    const forecast = this.generateForecast(dates, values, slope, intercept, 7); // 7-day forecast

    return {
      metricId,
      period,
      trend,
      slope,
      rSquared,
      confidence: Math.max(0, Math.min(1, rSquared)),
      seasonality,
      forecast
    };
  }

  /**
   * Generate predictive model for a metric
   */
  createPredictiveModel(
    metricId: string,
    computations: PoPComputation[],
    modelType: 'linear' | 'exponential' | 'arima' | 'prophet' = 'linear'
  ): PredictiveModel {
    const sortedData = computations
      .sort((a, b) => new Date(a.periodStart).getTime() - new Date(b.periodStart).getTime());

    if (sortedData.length < 5) {
      throw new Error('Insufficient data for model training');
    }

    const values = sortedData.map(comp => comp.currentValue);
    const accuracy = this.calculateModelAccuracy(values, modelType);

    const model: PredictiveModel = {
      id: `${metricId}-${modelType}-${Date.now()}`,
      metricId,
      modelType,
      parameters: this.trainModel(values, modelType),
      accuracy,
      lastTrained: new Date().toISOString(),
      nextPrediction: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString() // Next prediction in 24 hours
    };

    this.models.set(model.id, model);
    return model;
  }

  /**
   * Generate predictions using trained model
   */
  generatePredictions(
    modelId: string,
    futurePeriods: number = 7
  ): Array<{ date: string; predictedValue: number; confidence: number }> {
    const model = this.models.get(modelId);
    if (!model) {
      throw new Error('Model not found');
    }

    const predictions: Array<{ date: string; predictedValue: number; confidence: number }> = [];
    const baseDate = new Date();

    for (let i = 1; i <= futurePeriods; i++) {
      const futureDate = new Date(baseDate.getTime() + i * 24 * 60 * 60 * 1000);
      const predictedValue = this.predictValue(model, i);
      const confidence = this.calculatePredictionConfidence(model, i);

      predictions.push({
        date: futureDate.toISOString().split('T')[0],
        predictedValue,
        confidence
      });
    }

    return predictions;
  }

  /**
   * Detect anomalies using statistical methods
   */
  detectAnomalies(
    computations: PoPComputation[],
    threshold: number = 2.5
  ): Array<{
    computation: PoPComputation;
    zScore: number;
    isAnomaly: boolean;
    severity: 'low' | 'medium' | 'high' | 'critical';
  }> {
    const values = computations.map(comp => comp.currentValue);
    const mean = values.reduce((sum, val) => sum + val, 0) / values.length;
    const stdDev = Math.sqrt(
      values.reduce((sum, val) => sum + Math.pow(val - mean, 2), 0) / values.length
    );

    return computations.map(comp => {
      const zScore = stdDev === 0 ? 0 : (comp.currentValue - mean) / stdDev;
      const isAnomaly = Math.abs(zScore) > threshold;

      let severity: 'low' | 'medium' | 'high' | 'critical' = 'low';
      if (Math.abs(zScore) > 3) severity = 'critical';
      else if (Math.abs(zScore) > 2.5) severity = 'high';
      else if (Math.abs(zScore) > 2) severity = 'medium';

      return {
        computation: comp,
        zScore,
        isAnomaly,
        severity
      };
    });
  }

  /**
   * Calculate volatility of a time series
   */
  private calculateVolatility(values: number[]): number {
    if (values.length < 2) return 0;

    const returns: number[] = [];
    for (let i = 1; i < values.length; i++) {
      const return_pct = (values[i] - values[i - 1]) / values[i - 1];
      returns.push(return_pct);
    }

    const mean = returns.reduce((sum, ret) => sum + ret, 0) / returns.length;
    const variance = returns.reduce((sum, ret) => sum + Math.pow(ret - mean, 2), 0) / returns.length;

    return Math.sqrt(variance);
  }

  /**
   * Perform linear regression
   */
  private linearRegression(x: number[], y: number[]): { slope: number; intercept: number; rSquared: number } {
    const n = x.length;
    const sumX = x.reduce((sum, val) => sum + val, 0);
    const sumY = y.reduce((sum, val) => sum + val, 0);
    const sumXY = x.reduce((sum, val, i) => sum + val * y[i], 0);
    const sumXX = x.reduce((sum, val) => sum + val * val, 0);
  // sumYY removed: not used in calculations but kept calculation intentionally omitted to avoid unused var

    const slope = (n * sumXY - sumX * sumY) / (n * sumXX - sumX * sumX);
    const intercept = (sumY - slope * sumX) / n;

    // Calculate R-squared
    const yMean = sumY / n;
    const ssRes = y.reduce((sum, val, i) => {
      const predicted = slope * x[i] + intercept;
      return sum + Math.pow(val - predicted, 2);
    }, 0);
    const ssTot = y.reduce((sum, val) => sum + Math.pow(val - yMean, 2), 0);
    const rSquared = 1 - (ssRes / ssTot);

    return { slope, intercept, rSquared: isNaN(rSquared) ? 0 : rSquared };
  }

  /**
   * Detect seasonality in time series
   */
  private detectSeasonality(values: number[], period: number): boolean {
    if (values.length < period * 2) return false;

    // Simple autocorrelation check
    const autocorr: number[] = [];
    for (let lag = 1; lag <= period; lag++) {
      let sum = 0;
      let count = 0;
      for (let i = lag; i < values.length; i++) {
        sum += values[i] * values[i - lag];
        count++;
      }
      autocorr.push(sum / count);
    }

    // Check if there's significant autocorrelation at the seasonal period
    const seasonalCorr = autocorr[period - 1];
    const meanCorr = autocorr.reduce((sum, val) => sum + val, 0) / autocorr.length;

    return seasonalCorr > meanCorr * 1.5; // 50% higher than average
  }

  /**
   * Generate forecast using linear regression
   */
  private generateForecast(
    dates: number[],
    values: number[],
    slope: number,
    intercept: number,
    periods: number
  ): Array<{ date: string; predictedValue: number; upperBound: number; lowerBound: number }> {
    const lastDate = Math.max(...dates);
    const dateInterval = dates.length > 1 ?
      (dates[dates.length - 1] - dates[0]) / (dates.length - 1) : 24 * 60 * 60 * 1000;

    // Calculate standard error for confidence intervals
    const predictions: Array<{ date: string; predictedValue: number; upperBound: number; lowerBound: number }> = [];
    const stdError = this.calculateStandardError(dates, values, slope, intercept);

    for (let i = 1; i <= periods; i++) {
      const futureDate = lastDate + i * dateInterval;
      const predictedValue = slope * futureDate + intercept;

      // 95% confidence interval
      const margin = 1.96 * stdError * Math.sqrt(1 + 1/dates.length + Math.pow(futureDate - (dates.reduce((sum, val) => sum + val, 0) / dates.length), 2) / dates.reduce((sum, val) => sum + Math.pow(val - (dates.reduce((s, v) => s + v, 0) / dates.length), 2), 0));

      predictions.push({
        date: new Date(futureDate).toISOString().split('T')[0],
        predictedValue,
        upperBound: predictedValue + margin,
        lowerBound: predictedValue - margin
      });
    }

    return predictions;
  }

  /**
   * Calculate standard error of regression
   */
  private calculateStandardError(x: number[], y: number[], slope: number, intercept: number): number {
    const n = x.length;
    const residuals = y.map((val, i) => val - (slope * x[i] + intercept));
    const ssRes = residuals.reduce((sum, res) => sum + res * res, 0);
    return Math.sqrt(ssRes / (n - 2));
  }

  /**
   * Train predictive model
   */
  private trainModel(values: number[], modelType: string): Record<string, any> {
    switch (modelType) {
      case 'linear':
        // Simple linear trend
        const x = Array.from({ length: values.length }, (_, i) => i);
        const { slope, intercept } = this.linearRegression(x, values);
        return { slope, intercept };

      case 'exponential':
        // Exponential growth model
        const logValues = values.map(val => Math.log(Math.max(val, 0.001)));
        const x_exp = Array.from({ length: logValues.length }, (_, i) => i);
        const { slope: expSlope, intercept: expIntercept } = this.linearRegression(x_exp, logValues);
        return { slope: expSlope, intercept: expIntercept };

      default:
        return { slope: 0, intercept: values[values.length - 1] || 0 };
    }
  }

  /**
   * Calculate model accuracy
   */
  private calculateModelAccuracy(values: number[], modelType: string): number {
    if (values.length < 3) return 0;

    // Simple holdout validation
    const trainSize = Math.floor(values.length * 0.8);
    const trainData = values.slice(0, trainSize);
    const testData = values.slice(trainSize);

    const model = this.trainModel(trainData, modelType);
    let totalError = 0;

    testData.forEach((actual, i) => {
      const predicted = this.predictValueFromParams(model, trainSize + i, modelType);
      totalError += Math.abs(actual - predicted) / actual;
    });

    return Math.max(0, 1 - totalError / testData.length);
  }

  /**
   * Predict value using model parameters
   */
  private predictValue(model: PredictiveModel, periodsAhead: number): number {
    return this.predictValueFromParams(model.parameters, model.parameters.lastIndex + periodsAhead, model.modelType);
  }

  private predictValueFromParams(params: Record<string, any>, index: number, modelType: string): number {
    switch (modelType) {
      case 'linear':
        return params.slope * index + params.intercept;
      case 'exponential':
        return Math.exp(params.slope * index + params.intercept);
      default:
        return params.intercept;
    }
  }

  /**
   * Calculate prediction confidence
   */
  private calculatePredictionConfidence(model: PredictiveModel, periodsAhead: number): number {
    // Confidence decreases with distance from training data
    const baseConfidence = model.accuracy;
    const decayFactor = Math.exp(-periodsAhead * 0.1); // Exponential decay
    return Math.max(0, Math.min(1, baseConfidence * decayFactor));
  }

  /**
   * Create empty trend analysis
   */
  private createEmptyTrendAnalysis(metricId: string, period: string): TrendAnalysis {
    return {
      metricId,
      period,
      trend: 'stable',
      slope: 0,
      rSquared: 0,
      confidence: 0,
      seasonality: false,
      forecast: []
    };
  }
}

export const predictiveAnalytics = PredictiveAnalyticsService.getInstance();
