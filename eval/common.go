package eval

import "fmt"

func FormatRoughSize(value int64) string {
	if value < 100000 {
		return "~0"
	}

	if value >= 1000000000 { // 1 billion or more
		billions := float64(value) / 1000000000.0
		return fmt.Sprintf("%.1fG", billions)
	}

	if value >= 100000 { // 1 million or more
		millions := float64(value) / 1000000.0
		return fmt.Sprintf("%.1fM", millions)
	}

	// Between 100,000 and 1,000,000
	return fmt.Sprintf("%d", value)
}
