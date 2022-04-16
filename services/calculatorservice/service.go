package calculatorservice

import (
	"context"

	"github.com/go-kit/log"
	"github.com/josephmbassey/calculator-service/rpc/proto/calculatorpb"
)

// Service ...
type Service interface {
	Calculator(ctx context.Context, operator calculatorpb.OPERATOR, operands *calculatorpb.OPERANDS) (result float64, err error)
}

type Calculator struct {
	logger log.Logger
}

// NewService ...
func NewService(logger log.Logger) (Service, error) {
	return &Calculator{
		logger: logger,
	}, nil
}
