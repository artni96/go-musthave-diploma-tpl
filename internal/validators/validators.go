package validators

func LuhnValidator(number int) bool {
	strNumber := string(rune(number))
	sum := 0
	strLength := len(strNumber)
	parity := strLength % 2

	for i := 0; i < strLength-1; i++ {
		digit := int(strNumber[i] - '0')
		if parity == (i+1)%2 {
			sum += digit
		} else if digit > 4 {
			sum += 2*digit - 9
		} else {
			sum += 2 * digit
		}
	}
	controlSum := (10 - (sum % 10)) % 10
	return int(strNumber[strLength-1]-'0') == controlSum
}
