package service

// LuhnValid проверяет номер по алгоритму Луна.
// Возвращает false, если номер пуст или содержит нецифры.
func LuhnValid(number string) bool {
	if number == "" {
		return false
	}
	sum := 0
	parity := len(number) % 2
	for i := 0; i < len(number); i++ {
		ch := number[i]
		if ch < '0' || ch > '9' {
			return false
		}
		d := int(ch - '0')
		if i%2 == parity {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}
