// Package format provides shared display formatting helpers for the CLI.
package format

import (
	"fmt"
	"strings"
)

// FormatAmount formats a monetary amount with thousands separators and two decimal places.
// Example: 12345678.9 -> "12,345,678.90"
func FormatAmount(amount float64) string {
	return formatFixed(amount, 2)
}

// FormatAmountWhole formats a monetary amount with thousands separators and no fractional part.
// Example: 125000 -> "125,000"
func FormatAmountWhole(amount float64) string {
	return formatFixed(amount, 0)
}

// FormatMoney formats currency code and amount for human-readable display.
// Example: FormatMoney(1234.5, "USD") -> "USD 1,234.50"
func FormatMoney(amount float64, currency string) string {
	cur := strings.TrimSpace(currency)
	if cur == "" {
		cur = "USD"
	}
	return fmt.Sprintf("%s %s", cur, FormatAmount(amount))
}

// FormatChartMoney formats currency and amount for chart axis labels (whole numbers when >= 1000).
func FormatChartMoney(amount float64, currency string) string {
	cur := strings.TrimSpace(currency)
	if cur == "" {
		cur = "USD"
	}
	if amount >= 1000 {
		return fmt.Sprintf("%s %s", cur, FormatAmountWhole(amount))
	}
	return fmt.Sprintf("%s %s", cur, FormatAmount(amount))
}

func formatFixed(amount float64, decimals int) string {
	s := fmt.Sprintf("%.*f", decimals, amount)
	if decimals == 0 {
		return addThousandsSeparators(s)
	}
	parts := strings.SplitN(s, ".", 2)
	intPart := addThousandsSeparators(parts[0])
	if len(parts) < 2 {
		return intPart + ".00"
	}
	return intPart + "." + parts[1]
}

func addThousandsSeparators(intPart string) string {
	neg := strings.HasPrefix(intPart, "-")
	if neg {
		intPart = intPart[1:]
	}
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	return b.String()
}
