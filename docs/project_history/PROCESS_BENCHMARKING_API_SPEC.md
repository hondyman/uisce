# Process Performance Benchmarking - Backend API Specification

## Overview

This document specifies the backend API endpoints required to support the Process Performance Benchmarking system. The system provides industry benchmarks, peer comparisons, performance scoring, and best practice recommendations.

## Table of Contents

1. [Architecture](#architecture)
2. [Data Models](#data-models)
3. [API Endpoints](#api-endpoints)
4. [Scoring Algorithm](#scoring-algorithm)
5. [Implementation Guide](#implementation-guide)

---

## Architecture

### Technology Stack

- **Go** - Backend language
- **PostgreSQL** - Primary database for metrics storage
- **Chi Router** - HTTP routing
- **ML Service** (optional) - Python/scikit-learn for predictive analytics

### Database Schema

```sql
-- Industry benchmark data table
CREATE TABLE bp_industry_benchmarks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    industry VARCHAR(100) NOT NULL,
    process_type VARCHAR(100) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    median_value DECIMAL(10,2),
    top_quartile_value DECIMAL(10,2),
    bottom_quartile_value DECIMAL(10,2),
    sample_size INTEGER,
    last_updated TIMESTAMP DEFAULT NOW(),
    UNIQUE(industry, process_type, metric_name)
);

-- Performance scores table
CREATE TABLE bp_performance_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    process_type VARCHAR(100) NOT NULL,
    overall_score INTEGER CHECK (overall_score >= 0 AND overall_score <= 100),
    efficiency_score INTEGER CHECK (efficiency_score >= 0 AND efficiency_score <= 100),
    quality_score INTEGER CHECK (quality_score >= 0 AND quality_score <= 100),
    speed_score INTEGER CHECK (speed_score >= 0 AND speed_score <= 100),
    automation_score INTEGER CHECK (automation_score >= 0 AND automation_score <= 100),
    compliance_score INTEGER CHECK (compliance_score >= 0 AND compliance_score <= 100),
    percentile INTEGER,
    grade VARCHAR(3),
    calculated_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (tenant_id) REFERENCES alpha_tenants(id)
);

-- Best practices library
CREATE TABLE bp_best_practices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    industry VARCHAR(100),
    process_type VARCHAR(100),
    impact VARCHAR(20) CHECK (impact IN ('high', 'medium', 'low')),
    effort VARCHAR(20) CHECK (effort IN ('high', 'medium', 'low')),
    industry_adoption DECIMAL(5,2) CHECK (industry_adoption >= 0 AND industry_adoption <= 1),
    expected_improvement DECIMAL(5,2),
    implementation_steps JSONB,
    case_studies JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Peer group definitions
CREATE TABLE bp_peer_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_name VARCHAR(255) NOT NULL,
    industry VARCHAR(100) NOT NULL,
    size_category VARCHAR(50),
    region VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Peer group memberships
CREATE TABLE bp_peer_group_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    peer_group_id UUID NOT NULL REFERENCES bp_peer_groups(id),
    tenant_id UUID NOT NULL REFERENCES alpha_tenants(id),
    joined_at TIMESTAMP DEFAULT NOW(),
    is_anonymous BOOLEAN DEFAULT TRUE,
    UNIQUE(peer_group_id, tenant_id)
);

-- Gap analysis results
CREATE TABLE bp_gap_analysis (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES alpha_tenants(id),
    process_type VARCHAR(100) NOT NULL,
    dimension VARCHAR(100) NOT NULL,
    current_score INTEGER,
    target_score INTEGER,
    gap INTEGER,
    priority VARCHAR(20) CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    recommendations JSONB,
    analyzed_at TIMESTAMP DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_industry_benchmarks_industry ON bp_industry_benchmarks(industry, process_type);
CREATE INDEX idx_performance_scores_tenant ON bp_performance_scores(tenant_id, process_type);
CREATE INDEX idx_best_practices_industry ON bp_best_practices(industry, process_type);
CREATE INDEX idx_peer_members_tenant ON bp_peer_group_members(tenant_id);
CREATE INDEX idx_gap_analysis_tenant ON bp_gap_analysis(tenant_id, process_type);
```

---

## Data Models

### BenchmarkScore

```go
type BenchmarkScore struct {
    OverallScore     int                   `json:"overall_score"`
    DimensionScores  DimensionScores       `json:"dimension_scores"`
    Percentile       int                   `json:"percentile"`
    Grade            string                `json:"grade"` // A+, A, B+, B, C+, C, D, F
}

type DimensionScores struct {
    Efficiency   int `json:"efficiency"`
    Quality      int `json:"quality"`
    Speed        int `json:"speed"`
    Automation   int `json:"automation"`
    Compliance   int `json:"compliance"`
}
```

### IndustryBenchmark

```go
type IndustryBenchmark struct {
    Industry     string                  `json:"industry"`
    ProcessType  string                  `json:"process_type"`
    Metrics      BenchmarkMetrics        `json:"metrics"`
    TopQuartile  BenchmarkMetrics        `json:"top_quartile"`
    Median       BenchmarkMetrics        `json:"median"`
    SampleSize   int                     `json:"sample_size"`
}

type BenchmarkMetrics struct {
    AvgDurationMinutes float64 `json:"avg_duration_minutes"`
    SuccessRate        float64 `json:"success_rate"`
    AutomationRate     float64 `json:"automation_rate"`
    ErrorRate          float64 `json:"error_rate"`
    CycleTimeMinutes   float64 `json:"cycle_time_minutes"`
}
```

### PeerComparison

```go
type PeerComparison struct {
    PeerGroup          string               `json:"peer_group"`
    YourRank           int                  `json:"your_rank"`
    TotalPeers         int                  `json:"total_peers"`
    Percentile         int                  `json:"percentile"`
    ComparisonMetrics  []MetricComparison   `json:"comparison_metrics"`
}

type MetricComparison struct {
    MetricName  string  `json:"metric_name"`
    YourValue   float64 `json:"your_value"`
    PeerAvg     float64 `json:"peer_avg"`
    PeerBest    float64 `json:"peer_best"`
    Variance    float64 `json:"variance"` // Percentage difference from peer avg
}
```

### BestPractice

```go
type BestPractice struct {
    ID                  string       `json:"id"`
    Category            string       `json:"category"`
    Title               string       `json:"title"`
    Description         string       `json:"description"`
    Impact              string       `json:"impact"` // high, medium, low
    Effort              string       `json:"effort"` // high, medium, low
    IndustryAdoption    float64      `json:"industry_adoption"` // 0.0 to 1.0
    ExpectedImprovement float64      `json:"expected_improvement"` // Percentage
    ImplementationSteps []string     `json:"implementation_steps"`
    CaseStudies         []CaseStudy  `json:"case_studies"`
}

type CaseStudy struct {
    Company     string `json:"company"`
    Industry    string `json:"industry"`
    Improvement string `json:"improvement"`
    Timeframe   string `json:"timeframe"`
}
```

### GapAnalysis

```go
type GapAnalysis struct {
    Dimension        string   `json:"dimension"`
    CurrentScore     int      `json:"current_score"`
    TargetScore      int      `json:"target_score"`
    Gap              int      `json:"gap"`
    Priority         string   `json:"priority"` // critical, high, medium, low
    Recommendations  []string `json:"recommendations"`
}
```

---

## API Endpoints

### 1. GET /api/process-benchmarking/score

Calculate and return the overall performance score for a tenant's processes.

**Query Parameters:**
- `tenant_id` (required): UUID of the tenant
- `process_type` (optional): Specific process type to score

**Response:**
```json
{
  "overall_score": 87,
  "dimension_scores": {
    "efficiency": 92,
    "quality": 89,
    "speed": 85,
    "automation": 78,
    "compliance": 91
  },
  "percentile": 85,
  "grade": "A"
}
```

**Implementation Notes:**
- Query `bp_process_metrics` table to calculate current performance
- Apply scoring algorithm (see [Scoring Algorithm](#scoring-algorithm))
- Cache results in `bp_performance_scores` table
- Recalculate daily or on-demand

---

### 2. GET /api/process-benchmarking/industry

Retrieve industry benchmark data for comparison.

**Query Parameters:**
- `industry` (required): Industry identifier
- `process_type` (required): Process type

**Response:**
```json
{
  "industry": "financial_services",
  "process_type": "investment_approval",
  "metrics": {
    "avg_duration_minutes": 65.5,
    "success_rate": 0.92,
    "automation_rate": 0.75,
    "error_rate": 0.08,
    "cycle_time_minutes": 120.0
  },
  "top_quartile": {
    "avg_duration_minutes": 45.0,
    "success_rate": 0.97,
    "automation_rate": 0.88
  },
  "median": {
    "avg_duration_minutes": 65.5,
    "success_rate": 0.92,
    "automation_rate": 0.75
  },
  "sample_size": 247
}
```

**Data Sources:**
- Pre-populated `bp_industry_benchmarks` table
- External benchmark APIs (Gartner, Forrester, McKinsey)
- Anonymized peer data aggregation

---

### 3. GET /api/process-benchmarking/peers

Compare performance against peer organizations.

**Query Parameters:**
- `tenant_id` (required): UUID of the tenant
- `industry` (required): Industry for peer selection

**Response:**
```json
{
  "peer_group": "Mid-sized Wealth Management Firms",
  "your_rank": 12,
  "total_peers": 45,
  "percentile": 73,
  "comparison_metrics": [
    {
      "metric_name": "Average Processing Time",
      "your_value": 45.5,
      "peer_avg": 55.2,
      "peer_best": 38.0,
      "variance": -17.6
    },
    {
      "metric_name": "Success Rate",
      "your_value": 0.95,
      "peer_avg": 0.92,
      "peer_best": 0.98,
      "variance": 3.3
    }
  ]
}
```

**Implementation Notes:**
- Query `bp_peer_group_members` to identify peer group
- Aggregate metrics from `bp_performance_scores` for peers
- Anonymize peer data (no company names)
- Rank tenant within peer group

---

### 4. GET /api/process-benchmarking/best-practices

Retrieve industry best practices and recommendations.

**Query Parameters:**
- `industry` (required): Industry identifier
- `process_type` (required): Process type
- `min_impact` (optional): Filter by minimum impact (high, medium, low)

**Response:**
```json
[
  {
    "id": "bp-001",
    "category": "Automation",
    "title": "Implement Intelligent Document Processing",
    "description": "Use AI/ML to automatically extract and validate data from documents, reducing manual data entry by 80%.",
    "impact": "high",
    "effort": "medium",
    "industry_adoption": 0.62,
    "expected_improvement": 0.35,
    "implementation_steps": [
      "Assess current document processing volume and types",
      "Select IDP vendor or build in-house solution",
      "Train ML models on representative document samples",
      "Implement parallel processing for validation",
      "Monitor accuracy and adjust thresholds"
    ],
    "case_studies": [
      {
        "company": "Large Investment Bank",
        "industry": "Financial Services",
        "improvement": "Reduced processing time by 65% and errors by 82%",
        "timeframe": "6 months"
      }
    ]
  }
]
```

**Data Sources:**
- Curated `bp_best_practices` table
- Industry research reports
- Customer success stories
- Expert recommendations

---

### 5. GET /api/process-benchmarking/gap-analysis

Analyze gaps between current and target performance.

**Query Parameters:**
- `tenant_id` (required): UUID of the tenant
- `process_type` (required): Process type

**Response:**
```json
[
  {
    "dimension": "Automation",
    "current_score": 78,
    "target_score": 90,
    "gap": 12,
    "priority": "high",
    "recommendations": [
      "Implement RPA for repetitive data entry tasks",
      "Integrate with third-party APIs to reduce manual lookups",
      "Add smart routing rules to reduce manual triage"
    ]
  },
  {
    "dimension": "Speed",
    "current_score": 85,
    "target_score": 92,
    "gap": 7,
    "priority": "medium",
    "recommendations": [
      "Optimize database queries in approval workflow",
      "Implement parallel processing for independent steps",
      "Add caching layer for frequently accessed data"
    ]
  }
]
```

**Implementation Notes:**
- Calculate target scores based on top quartile industry benchmarks
- Prioritize gaps: critical (>20), high (15-20), medium (10-15), low (<10)
- Generate recommendations using rule engine or ML model
- Cache analysis results for 24 hours

---

### 6. POST /api/process-benchmarking/calculate-score

Manually trigger score recalculation for a tenant.

**Request Body:**
```json
{
  "tenant_id": "123e4567-e89b-12d3-a456-426614174000",
  "process_type": "investment_approval",
  "datasource_id": "123e4567-e89b-12d3-a456-426614174001"
}
```

**Response:**
```json
{
  "success": true,
  "score_id": "123e4567-e89b-12d3-a456-426614174002",
  "calculated_at": "2026-01-01T10:30:00Z"
}
```

---

## Scoring Algorithm

### Overall Score Calculation

The overall performance score (0-100) is calculated as a weighted average of five dimension scores:

```
Overall Score = (
  Efficiency * 0.25 +
  Quality * 0.25 +
  Speed * 0.20 +
  Automation * 0.15 +
  Compliance * 0.15
)
```

### Dimension Score Calculations

#### 1. Efficiency Score (0-100)

Measures resource utilization and cost-effectiveness.

```go
func calculateEfficiencyScore(metrics ProcessMetrics, benchmark IndustryBenchmark) int {
    // Resource utilization (0-50 points)
    utilizationScore := calculateUtilization(metrics.ResourceUsage)
    
    // Cost per workflow vs benchmark (0-50 points)
    costRatio := metrics.AvgCost / benchmark.Median.AvgCost
    costScore := 50.0
    if costRatio > 1.0 {
        costScore = max(0, 50.0 * (2.0 - costRatio))
    } else {
        costScore = min(50.0, 50.0 + (50.0 * (1.0 - costRatio)))
    }
    
    return int(utilizationScore + costScore)
}
```

#### 2. Quality Score (0-100)

Measures accuracy, completeness, and error rates.

```go
func calculateQualityScore(metrics ProcessMetrics, benchmark IndustryBenchmark) int {
    // Success rate (0-50 points)
    successScore := metrics.SuccessRate * 50.0
    
    // Error rate vs benchmark (0-30 points)
    errorRatio := metrics.ErrorRate / benchmark.Median.ErrorRate
    errorScore := max(0, 30.0 * (2.0 - errorRatio))
    
    // Rework rate (0-20 points)
    reworkScore := max(0, 20.0 * (1.0 - metrics.ReworkRate))
    
    return int(successScore + errorScore + reworkScore)
}
```

#### 3. Speed Score (0-100)

Measures processing time and cycle time.

```go
func calculateSpeedScore(metrics ProcessMetrics, benchmark IndustryBenchmark) int {
    // Duration vs benchmark (0-60 points)
    durationRatio := metrics.AvgDuration / benchmark.Median.AvgDuration
    durationScore := max(0, 60.0 * (2.0 - durationRatio))
    
    // Cycle time (0-40 points)
    cycleRatio := metrics.CycleTime / benchmark.Median.CycleTime
    cycleScore := max(0, 40.0 * (2.0 - cycleRatio))
    
    return int(durationScore + cycleScore)
}
```

#### 4. Automation Score (0-100)

Measures degree of automation and manual intervention.

```go
func calculateAutomationScore(metrics ProcessMetrics, benchmark IndustryBenchmark) int {
    // Automation rate (0-60 points)
    automationScore := metrics.AutomationRate * 60.0
    
    // Manual touch points (0-40 points)
    manualRatio := float64(metrics.ManualSteps) / float64(metrics.TotalSteps)
    manualScore := max(0, 40.0 * (1.0 - manualRatio))
    
    return int(automationScore + manualScore)
}
```

#### 5. Compliance Score (0-100)

Measures adherence to regulations and policies.

```go
func calculateComplianceScore(metrics ProcessMetrics) int {
    // Audit trail completeness (0-40 points)
    auditScore := metrics.AuditCoverage * 40.0
    
    // Policy violations (0-30 points)
    violationScore := max(0, 30.0 * (1.0 - metrics.ViolationRate))
    
    // Documentation completeness (0-30 points)
    docScore := metrics.DocumentationRate * 30.0
    
    return int(auditScore + violationScore + docScore)
}
```

### Grade Assignment

```go
func assignGrade(overallScore int) string {
    switch {
    case overallScore >= 97:
        return "A+"
    case overallScore >= 93:
        return "A"
    case overallScore >= 90:
        return "A-"
    case overallScore >= 87:
        return "B+"
    case overallScore >= 83:
        return "B"
    case overallScore >= 80:
        return "B-"
    case overallScore >= 77:
        return "C+"
    case overallScore >= 73:
        return "C"
    case overallScore >= 70:
        return "C-"
    case overallScore >= 60:
        return "D"
    default:
        return "F"
    }
}
```

---

## Implementation Guide

### Phase 1: Database Setup (Week 1)

1. Create database tables and indexes
2. Seed industry benchmark data
3. Create initial peer groups
4. Populate best practices library

### Phase 2: Core API Endpoints (Week 2)

1. Implement scoring algorithm
2. Build benchmark comparison endpoints
3. Create peer analysis functionality
4. Test with sample data

### Phase 3: Best Practices & Recommendations (Week 3)

1. Build recommendation engine
2. Integrate gap analysis
3. Add case studies and implementation guides
4. Create admin interface for managing practices

### Phase 4: Data Collection & Refinement (Week 4)

1. Integrate with external benchmark sources
2. Implement anonymized peer data collection
3. Add ML-based recommendation improvements
4. Performance optimization

### Sample Go Handler Implementation

```go
// backend/internal/api/benchmarking_handlers.go
package api

import (
    "encoding/json"
    "net/http"
)

func (s *Server) handleGetBenchmarkScore(w http.ResponseWriter, r *http.Request) {
    tenantID := r.URL.Query().Get("tenant_id")
    processType := r.URL.Query().Get("process_type")
    
    // Fetch process metrics
    metrics, err := s.getProcessMetrics(r.Context(), tenantID, processType)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Get industry benchmark
    benchmark, err := s.getIndustryBenchmark(r.Context(), "financial_services", processType)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Calculate scores
    score := BenchmarkScore{
        DimensionScores: DimensionScores{
            Efficiency:   calculateEfficiencyScore(metrics, benchmark),
            Quality:      calculateQualityScore(metrics, benchmark),
            Speed:        calculateSpeedScore(metrics, benchmark),
            Automation:   calculateAutomationScore(metrics, benchmark),
            Compliance:   calculateComplianceScore(metrics),
        },
    }
    
    // Calculate overall score
    score.OverallScore = int(
        float64(score.DimensionScores.Efficiency)*0.25 +
        float64(score.DimensionScores.Quality)*0.25 +
        float64(score.DimensionScores.Speed)*0.20 +
        float64(score.DimensionScores.Automation)*0.15 +
        float64(score.DimensionScores.Compliance)*0.15,
    )
    
    score.Grade = assignGrade(score.OverallScore)
    score.Percentile = calculatePercentile(score.OverallScore, benchmark)
    
    // Cache result
    if err := s.cachePerformanceScore(r.Context(), tenantID, processType, score); err != nil {
        // Log but don't fail
        s.logger.Warn("Failed to cache score", "error", err)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(score)
}
```

---

## Testing Strategy

### Unit Tests

Test scoring algorithm components:
- Dimension score calculations
- Grade assignment logic
- Percentile calculations

### Integration Tests

Test API endpoints:
- Score calculation with various metrics
- Benchmark comparison accuracy
- Peer ranking consistency

### Performance Tests

- Score calculation under load (1000+ concurrent requests)
- Database query optimization
- Cache hit rates

### Data Quality Tests

- Benchmark data completeness
- Peer group consistency
- Best practice relevance

---

## Monitoring & Observability

### Key Metrics

- **Score Calculation Time**: p50, p95, p99 latency
- **Cache Hit Rate**: Percentage of cached score retrievals
- **API Error Rate**: Failed requests per endpoint
- **Data Freshness**: Time since last benchmark update

### Alerts

- Score calculation failures
- Stale benchmark data (>7 days)
- Peer group membership changes
- Unusual score fluctuations

---

## Future Enhancements

1. **Machine Learning Integration**
   - Predictive scoring based on trends
   - Personalized recommendations
   - Anomaly detection

2. **Real-time Benchmarking**
   - Live peer comparisons
   - Dynamic target setting
   - Continuous improvement tracking

3. **Industry Reports**
   - Quarterly benchmark updates
   - Trend analysis
   - Market positioning reports

4. **Competitive Intelligence**
   - Anonymous peer insights
   - Market share analysis
   - Feature adoption rates

---

**Last Updated**: January 2026  
**Version**: 1.0.0  
**Maintainer**: Semlayer Backend Team
