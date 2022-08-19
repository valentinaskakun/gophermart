package functions

import "strconv"

func CheckOrderID(string string) (bool, error) {
	number, err := strconv.ParseInt(string, 10, 64)
	if err != nil {
		return false, err
	}
	return (number%10+checksum(number/10))%10 == 0, nil
}

func checksum(number int64) int64 {
	var luhn int64

	for i := 0; number > 0; i++ {
		cur := number % 10

		if i%2 == 0 { // even
			cur = cur * 2
			if cur > 9 {
				cur = cur%10 + cur/10
			}
		}

		luhn += cur
		number = number / 10
	}
	return luhn % 10
}
