package calculatorservice

import (
	"context"

	"github.com/josephmbassey/calculator-service/rpc/proto/calculatorpb"
)

// GRPCHandler ...
type GRPCHandler struct {
	service Service
	calculatorpb.UnimplementedCalculatorServiceServer
}

// NewGRPCHandler ...
func NewGRPCHandler(service Service) *GRPCHandler {
	return &GRPCHandler{
		service: service,
	}
}

// Calculator ...
func (h *GRPCHandler) Calculator(ctx context.Context, req *calculatorpb.CalculateRequest) (*calculatorpb.CalculateResponse, error) {
	result, err := h.service.Calculator(ctx, req.Operator, req.Operands)
	if err != nil {
		return nil, err
	}

	return &calculatorpb.CalculateResponse{
		Result: result,
	}, nil
}
