package calculatorservice_test

import (
	"context"
	"os"
	"testing"

	"github.com/go-kit/log"
	"github.com/josephmbassey/calculator-service/rpc/proto/calculatorpb"
	"github.com/josephmbassey/calculator-service/services/calculatorservice"
	"github.com/stretchr/testify/assert"
)

func Test_Calculator(t *testing.T) {
	calculatorSvc, _ := calculatorservice.NewService(log.NewLogfmtLogger(os.Stdout))
	tests := []struct {
		name           string
		operands       *calculatorpb.OPERANDS
		operator       calculatorpb.OPERATOR
		expectedResult float64
	}{
		{
			name:     "AdditionCalculation",
			operator: calculatorpb.OPERATOR_OPERATOR_ADD,
			operands: &calculatorpb.OPERANDS{
				Number_1: 2,
				Number_2: 5,
			},
			expectedResult: 7,
		},
		{
			name:     "SubstractionCalculation",
			operator: calculatorpb.OPERATOR_OPERATOR_SUBTRACT,
			operands: &calculatorpb.OPERANDS{
				Number_1: 10,
				Number_2: 5,
			},
			expectedResult: 5,
		},
		{
			name:     "DivideCalculation",
			operator: calculatorpb.OPERATOR_OPERATOR_DIVIDE,
			operands: &calculatorpb.OPERANDS{
				Number_1: 17,
				Number_2: 4,
			},
			expectedResult: 4.25,
		},
		{
			name:     "MultiplicationCalculation",
			operator: calculatorpb.OPERATOR_OPERATOR_MULTIPLY,
			operands: &calculatorpb.OPERANDS{
				Number_1: 2.0,
				Number_2: 9,
			},
			expectedResult: 18,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := calculatorSvc.Calculator(context.Background(), tt.operator, tt.operands)
			assert.Nil(t, err)
			assert.Equal(t, tt.expectedResult, res)
		})
	}
}
