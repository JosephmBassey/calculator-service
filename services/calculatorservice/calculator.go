package calculatorservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/josephmbassey/calculator-service/rpc/proto/calculatorpb"
)

func (c *Calculator) Calculator(ctx context.Context, operator calculatorpb.OPERATOR, operands *calculatorpb.OPERANDS) (result float64, err error) {

	switch operator {
	case calculatorpb.OPERATOR_OPERATOR_ADD:
		return add(operands.Number_1, operands.Number_2)
	case calculatorpb.OPERATOR_OPERATOR_MULTIPLY:
		return multiply(operands.Number_1, operands.Number_2)
	case calculatorpb.OPERATOR_OPERATOR_SUBTRACT:
		return subtract(operands.Number_1, operands.Number_2)
	case calculatorpb.OPERATOR_OPERATOR_DIVIDE:
		return divide(operands.Number_1, operands.Number_2)
	default:
		return 0.0, errors.New("error: some arguments are not supplied")
	}
}

func add(number1, number2 float64) (float64, error) {
	return number1 + number2, nil
}

func multiply(number1, number2 float64) (float64, error) {
	return number1 * number2, nil
}

func divide(number1, number2 float64) (float64, error) {
	if number2 == 0.0 {
		return 0, fmt.Errorf("you can not divide %F by %F", number1, number2)
	}
	return number1 / number2, nil
}

func subtract(number1, number2 float64) (float64, error) {
	return number1 - number2, nil
}
