package vectorindex

import "fmt"

var ErrVectorDimensionMismatch = fmt.Errorf("vectorindex: vector dimension mismatch")

func newDimensionMismatchError(expected, actual int) error {
	return fmt.Errorf("%w: expected %d, got %d", ErrVectorDimensionMismatch, expected, actual)
}
