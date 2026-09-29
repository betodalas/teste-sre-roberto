// Package mathops implements the four basic arithmetic operations used by
// the API: addition, subtraction, multiplication and division.
package mathops

import "errors"

// ErrDivisionByZero is returned when a division by zero is attempted.
var ErrDivisionByZero = errors.New("division by zero")

// Sum returns termOne + termTwo.
func Sum(termOne, termTwo int64) int64 {
	return termOne + termTwo
}

// Sub returns termOne - termTwo.
func Sub(termOne, termTwo int64) int64 {
	return termOne - termTwo
}

// Mul returns termOne * termTwo.
func Mul(termOne, termTwo int64) int64 {
	return termOne * termTwo
}

// Div returns the integer division of termOne by termTwo.
// It returns ErrDivisionByZero when termTwo is zero.
func Div(termOne, termTwo int64) (int64, error) {
	if termTwo == 0 {
		return 0, ErrDivisionByZero
	}
	return termOne / termTwo, nil
}
