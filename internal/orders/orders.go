package orders

import (
	"strconv"
)

func CheckOrderId(orderToCheck int) (result bool) {
	orderToCheckString := strconv.Itoa(orderToCheck)
	sum := 0
	for i := len(orderToCheckString) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(orderToCheckString[i]))
		if i%2 == 0 {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	result = sum%10 == 0
	return result
}
