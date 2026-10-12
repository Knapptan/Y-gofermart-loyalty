package service

import "testing"

func BenchmarkLuhnValid(b *testing.B) {
	const number = "12345678903"
	for b.Loop() {
		_ = LuhnValid(number)
	}
}
