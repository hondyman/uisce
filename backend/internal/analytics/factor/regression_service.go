package factor

import (
	"context"
	"fmt"
	"time"

	pb "github.com/hondyman/semlayer/backend/internal/analytics/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RegressionResult holds the output of a linear regression
type RegressionResult struct {
	Alpha     float64
	Betas     []float64
	RSquared  float64
	Residuals []float64
}

type RegressionService struct {
	client pb.AnalyticsEngineClient
	conn   *grpc.ClientConn
}

func NewRegressionService(address string) (*RegressionService, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("did not connect: %v", err)
	}
	client := pb.NewAnalyticsEngineClient(conn)
	return &RegressionService{
		client: client,
		conn:   conn,
	}, nil
}

func (s *RegressionService) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
}

// CalculateRollingBeta performs a rolling linear regression using the remote Analytics Engine
// y: dependent variable (portfolio returns)
// x: independent variables (factor returns matrix, where rows are time, cols are factors)
// window: rolling window size
func (s *RegressionService) CalculateRollingBeta(y []float64, x [][]float64, window int) ([]RegressionResult, error) {
	n := len(y)
	if n < window {
		return nil, fmt.Errorf("insufficient data points for window size %d", window)
	}

	results := make([]RegressionResult, 0, n-window+1)

	for i := 0; i <= n-window; i++ {
		// Slice the window
		yWindow := y[i : i+window]
		xWindow := x[i : i+window] // Assuming len(x) == len(y)

		// For now, assume a single factor in x[j][0] for the gRPC call
		// The Proto currently supports list of values, simplified for single variable regression
		// We will take the FIRST column of X as the market factor.
		xFlat := make([]float64, window)
		for j := 0; j < window; j++ {
			if len(xWindow[j]) > 0 {
				xFlat[j] = xWindow[j][0]
			}
		}

		res, err := s.RunRegression(context.Background(), xFlat, yWindow)
		if err != nil {
			// fallback or error?
			fmt.Printf("Error calculating rolling beta at index %d: %v\n", i, err)
			continue
		}

		results = append(results, RegressionResult{
			Alpha:    res.Alpha,
			Betas:    []float64{res.Beta}, // Proto returns single beta
			RSquared: res.RSquared,
		})
	}

	return results, nil
}

// PerformOLS performs Ordinary Least Squares regression via gRPC
func (s *RegressionService) PerformOLS(y []float64, x [][]float64) (*RegressionResult, error) {
	// Flatten X (take first column)
	xFlat := make([]float64, len(y))
	if len(x) == len(y) {
		for i := range y {
			if len(x[i]) > 0 {
				xFlat[i] = x[i][0]
			}
		}
	}

	res, err := s.RunRegression(context.Background(), xFlat, y)
	if err != nil {
		return nil, err
	}

	return &RegressionResult{
		Alpha:    res.Alpha,
		Betas:    []float64{res.Beta},
		RSquared: res.RSquared,
	}, nil
}

func (s *RegressionService) RunRegression(ctx context.Context, independent []float64, dependent []float64) (*pb.RegressionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Assuming dates are aligned by index for MVP.
	// In real system, we'd pass dates too.
	req := &pb.RegressionRequest{
		IndependentData: make([]*pb.VerificationData, len(independent)),
		DependentData:   make([]*pb.VerificationData, len(dependent)),
	}

	for i, v := range independent {
		req.IndependentData[i] = &pb.VerificationData{Value: v}
	}
	for i, v := range dependent {
		req.DependentData[i] = &pb.VerificationData{Value: v}
	}

	return s.client.CalculateRegression(ctx, req)
}
