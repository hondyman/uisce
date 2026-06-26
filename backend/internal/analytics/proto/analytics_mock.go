package proto

import (
	"context"

	"google.golang.org/grpc"
)

type AnalyticsEngineClient interface {
	CalculateRegression(ctx context.Context, in *RegressionRequest, opts ...grpc.CallOption) (*RegressionResponse, error)
}

type analyticsEngineClient struct {
	cc grpc.ClientConnInterface
}

func NewAnalyticsEngineClient(cc grpc.ClientConnInterface) AnalyticsEngineClient {
	return &analyticsEngineClient{cc}
}

func (c *analyticsEngineClient) CalculateRegression(ctx context.Context, in *RegressionRequest, opts ...grpc.CallOption) (*RegressionResponse, error) {
	out := new(RegressionResponse)
	err := c.cc.Invoke(ctx, "/analytics.AnalyticsEngine/CalculateRegression", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type RegressionRequest struct {
	IndependentData []*VerificationData
	DependentData   []*VerificationData
}

type VerificationData struct {
	Date  string
	Value float64
}

type RegressionResponse struct {
	Alpha    float64
	Beta     float64
	RSquared float64
	Status   string
}
