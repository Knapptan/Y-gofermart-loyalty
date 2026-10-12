package service

import "testing"

func TestLuhnValid(t *testing.T) {
	cases := []struct {
		number string
		want   bool
	}{
		{"12345678903", true}, // из ТЗ
		{"9278923470", true},  // из ТЗ
		{"2377225624", true},  // из ТЗ
		{"0", true},           // 0 — валиден по Луну
		{"", false},           // пусто
		{"abc", false},        // не цифры
		{"1234567890", false}, // не проходит
		{"1234567890123456", false},
		{"1234567890123452", true},
	}
	for _, c := range cases {
		if got := LuhnValid(c.number); got != c.want {
			t.Errorf("LuhnValid(%q) = %v, want %v", c.number, got, c.want)
		}
	}
}
