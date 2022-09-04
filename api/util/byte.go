package util

import "fmt"

const base = uint64(1 << 10)

// HumanReadableBytes formatting byte size to human readable format
func HumanReadableBytes(size uint64) string {
	if size < base {
		return fmt.Sprintf("%dB", size)
	}

	exp := 0
	for n := size; n >= base; n /= base {
		exp++
	}

	return fmt.Sprintf("%.2f%cB",
		float64(size)/float64(uint64(1<<(exp*10))), " KMGTPE"[exp])
}
