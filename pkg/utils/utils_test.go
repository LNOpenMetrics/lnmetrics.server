package utils

import (
	"testing"
)

func TestPercentageOne(t *testing.T) {
	result := Percentage(10, 200)

	if result != 5 {
		t.Errorf("Percentage expected is 5 but I got %d", result)
	}
}

func TestPercentageTwo(t *testing.T) {
	result := Percentage(200, 200)

	if result != 100 {
		t.Errorf("Percentage expected is 100 but I got %d", result)
	}
}
