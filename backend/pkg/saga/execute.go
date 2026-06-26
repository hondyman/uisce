package saga

import (
	"context"
	"errors"

	"github.com/hondyman/semlayer/backend/pkg/custodian"
	"github.com/hondyman/semlayer/backend/pkg/optimizer"
)

// ResultStep represents a single step in the saga execution
type ResultStep struct {
	Leg  int
	Req  custodian.OrderRequest
	Resp custodian.OrderResponse
	Err  error
}

// Result represents the final outcome of the saga
type Result struct {
	Steps         []ResultStep
	Status        string // completed | compensated | failed
	Compensations []ResultStep
}

// Executor handles the execution of a rebalancing plan
type Executor struct {
	Adapter custodian.Adapter
}

// Execute runs the rebalancing plan as a saga
func (e Executor) Execute(ctx context.Context, workflowID, proposalID string, plan optimizer.Plan) (Result, error) {
	res := Result{Steps: make([]ResultStep, 0)}
	
	// Execute SELLs first then BUYs (usually)
	// The plan.Trades order should already be sorted by dependency if needed
	leg := 0
	for _, tr := range plan.Trades {
		req := custodian.OrderRequest{
			ClientOrderID: custodian.ClientOrderID(workflowID, proposalID, leg),
			Symbol:        tr.Symbol,
			Side:          tr.Side,
			Qty:           tr.Qty,
			Type:          "MARKET", // Default to market for rebalancing
		}
		leg++
		
		resp, err := e.Adapter.PlaceOrder(ctx, req)
		res.Steps = append(res.Steps, ResultStep{Leg: leg, Req: req, Resp: resp, Err: err})
		
		if err != nil || resp.Status == "REJECTED" {
			// Compensation: reverse any filled prior legs
			comp := e.compensate(ctx, workflowID, proposalID, res.Steps)
			res.Compensations = append(res.Compensations, comp...)
			res.Status = "compensated"
			return res, errors.New("leg failure; compensated")
		}
	}
	
	res.Status = "completed"
	return res, nil
}

// compensate reverses successful trades to restore the portfolio state
func (e Executor) compensate(ctx context.Context, workflowID, proposalID string, steps []ResultStep) []ResultStep {
	var comps []ResultStep
	leg := 1000 // Start compensation legs at a high number
	
	// Iterate backwards through steps to reverse them in LIFO order
	for i := len(steps) - 1; i >= 0; i-- {
		s := steps[i]
		if s.Err == nil && s.Resp.Status == "FILLED" {
			reverseSide := "BUY"
			if s.Req.Side == "BUY" {
				reverseSide = "SELL"
			}
			
			req := custodian.OrderRequest{
				ClientOrderID: custodian.ClientOrderID(workflowID, proposalID, leg),
				Symbol:        s.Req.Symbol,
				Side:          reverseSide,
				Qty:           s.Resp.FilledQty,
				Type:          "MARKET",
			}
			leg++
			
			resp, err := e.Adapter.PlaceOrder(ctx, req)
			comps = append(comps, ResultStep{Leg: leg, Req: req, Resp: resp, Err: err})
		}
	}
	return comps
}
