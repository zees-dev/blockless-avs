package operator

import (
	"testing"
)

// TestGetPriceByID function to test getPrice
func TestGetPriceByID(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"bitcoin", 67148},
	}

	for _, test := range tests {
		price, err := getPriceByID(test.input)
		if err != nil {
			t.Errorf("Failed to get price: %v", err)
		}
		if price != test.expected {
			t.Errorf("Expected price: %v, got: %v", test.expected, price)
		}
	}
}

func TestFormatPriceToSixDecimals(t *testing.T) {
	tests := []struct {
		input    float64
		expected uint
	}{
		{67148.123456789, 67148123457},
	}

	for _, test := range tests {
		price := formatPriceToSixDecimals(test.input)
		if price != test.expected {
			t.Errorf("Expected price: %v, got: %v", test.expected, price)
		}
	}
}
