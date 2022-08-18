package orders

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/valentinaskakun/gophermart/internal/config"
	"github.com/valentinaskakun/gophermart/internal/storage"

	"github.com/go-resty/resty/v2"
)

var QueryUpdateIncreaseBalance = `UPDATE balance set current = current + $2, accrual = accrual + $2 
					where user_id = (SELECT user_id from orders where id_order = $1);`
var QueryUpdateOrdersAccrual = `UPDATE orders SET state = $2, accrual = $3 WHERE id_order = $1`

// Valid check number is valid or not based on Luhn algorithm
func CheckOrderId(number int) bool {
	return (number%10+checksum(number/10))%10 == 0
}

func checksum(number int) int {
	var luhn int

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

//
//func CheckOrderId(orderToCheck int) (result bool) {
//	orderToCheckString := strconv.Itoa(orderToCheck)
//	sum := 0
//	for i := len(orderToCheckString) - 1; i >= 0; i-- {
//		digit, _ := strconv.Atoi(string(orderToCheckString[i]))
//		if i%2 == 0 {
//			digit *= 2
//			if digit > 9 {
//				digit -= 9
//			}
//		}
//		sum += digit
//	}
//	result = sum%10 == 0
//	return result
//}
func AccrualUpdate(configRun *config.Config) (err error) {
	isOrders, arrOrders, err := storage.ReturnOrdersToProcess(configRun)
	if isOrders == false {
		fmt.Println("nothing to accrual")
		return
	}
	req := resty.New().
		SetBaseURL(configRun.AccrualAddress).
		R().
		SetHeader("Content-Type", "application/json")
	for _, order := range arrOrders {
		orderNum := strconv.Itoa(order)
		resp, errResp := req.Get("/api/orders/" + orderNum)
		if errResp != nil {
			fmt.Println("something went wrong while GET accrual for " + orderNum)
			return errResp
		}
		reqStatus := resp.StatusCode()
		if reqStatus == http.StatusInternalServerError {
			fmt.Println("StatusCode StatusInternalServerError 500 for " + orderNum)
			return
		} else if reqStatus == http.StatusTooManyRequests {
			fmt.Println("StatusCode StatusTooManyRequests 429 for " + orderNum)
			time.Sleep(60 * time.Second)
			return
		} else if reqStatus == http.StatusOK {
			var orderToAccrual storage.UsingAccrualStruct
			if err = json.Unmarshal(resp.Body(), &orderToAccrual); err != nil {
				fmt.Println("error while unmarshalling Accrual " + orderNum)
				return
			}
			orderToAccrualInt, errConv := strconv.Atoi(orderToAccrual.Order)
			if errConv != nil {
				return errConv
			}
			db, errSql := sql.Open("pgx", configRun.Database)
			if errSql != nil {
				return errSql
			}
			defer db.Close()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			txn, errSql := db.Begin()
			if errSql != nil {
				fmt.Println("could not start a new transaction")
				return errSql
			}
			defer txn.Rollback()
			_, err = txn.ExecContext(ctx, QueryUpdateOrdersAccrual, orderToAccrualInt, orderToAccrual.Status, orderToAccrual.Accrual)
			if err != nil {
				fmt.Println("failed to Update orders accrual")
				return
			}
			if orderToAccrual.Accrual == 0 {
				fmt.Println("accrual value is 0")
				return
			}
			_, err = txn.ExecContext(ctx, QueryUpdateIncreaseBalance, orderToAccrual.Order, orderToAccrual.Status)
			if err != nil {
				fmt.Println("failed to increase balance")
				return
			}
			if err = txn.Commit(); err != nil {
				fmt.Println("failed to commit transaction")
				return
			}
			return
		}
		return
	}
	return
}
