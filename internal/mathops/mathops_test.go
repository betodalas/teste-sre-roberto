package mathops

import "testing"

func TestSum(t *testing.T) {
	if got := Sum(4, 1); got != 5 {
		t.Errorf("Sum(4, 1) = %d, want 5", got)
	}
	if got := Sum(-4, 1); got != -3 {
		t.Errorf("Sum(-4, 1) = %d, want -3", got)
	}
}

func TestSub(t *testing.T) {
	if got := Sub(4, 1); got != 3 {
		t.Errorf("Sub(4, 1) = %d, want 3", got)
	}
	if got := Sub(1, 4); got != -3 {
		t.Errorf("Sub(1, 4) = %d, want -3", got)
	}
}

func TestMul(t *testing.T) {
	if got := Mul(4, 3); got != 12 {
		t.Errorf("Mul(4, 3) = %d, want 12", got)
	}
	if got := Mul(-4, 3); got != -12 {
		t.Errorf("Mul(-4, 3) = %d, want -12", got)
	}
}

func TestDiv(t *testing.T) {
	got, err := Div(10, 2)
	if err != nil {
		t.Fatalf("Div(10, 2) returned unexpected error: %v", err)
	}
	if got != 5 {
		t.Errorf("Div(10, 2) = %d, want 5", got)
	}

	if _, err := Div(10, 0); err != ErrDivisionByZero {
		t.Errorf("Div(10, 0) error = %v, want ErrDivisionByZero", err)
	}
}
