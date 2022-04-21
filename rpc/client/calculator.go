package client

import (
	"context"

	"github.com/go-kit/log"
	"github.com/josephmbassey/calculator-service/internals/monitoring/depstatus"
	"github.com/josephmbassey/calculator-service/internals/monitoring/depstatus/grpcdep"

	"github.com/josephmbassey/calculator-service/rpc/proto/calculatorpb"
	"google.golang.org/grpc"
)

// CalculatorClient is the calculator API gRPC client
type CalculatorClient struct {
	l    log.Logger
	conn *grpc.ClientConn
	c    calculatorpb.CalculatorServiceClient
}

// New creates a new Calculator gRPC client
func NewCalculatorServiceClient(l log.Logger, conn *grpc.ClientConn) *CalculatorClient {
	if l == nil {
		l = log.NewNopLogger()
	}
	c := &CalculatorClient{
		l:    l,
		conn: conn,
		c:    calculatorpb.NewCalculatorServiceClient(conn),
	}
	depstatus.Register(grpcdep.Wrap("calculator", conn))
	return c
}

// Close closes the connection of the gRPC client
func (c *CalculatorClient) Close() {
	c.conn.Close()
}

// Calculator takes in the operand and operators and return result base on the operand
func (c *CalculatorClient) Calculator(ctx context.Context, in *calculatorpb.CalculateRequest) (*calculatorpb.CalculateResponse, error) {
	resp, err := c.c.Calculator(ctx, in)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
