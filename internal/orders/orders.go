package orders

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"gophermart/internal/config"
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
func AccrualUpdate(configRun *config.Config) (err error) {
	req, err := http.NewRequest("GET", configRun.AccrualAddress, nil)
	if err != nil {
		log.Println(err)
	}
	fmt.Println(req)
	req.Header.Set("Content-Type", "Content-Type: text/plain")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println(err)
	} else if res.StatusCode != 200 {
		fmt.Println(err)
	}
	return
}
