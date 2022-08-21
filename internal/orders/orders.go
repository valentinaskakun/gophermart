package orders

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/valentinaskakun/gophermart/internal/config"
	"github.com/valentinaskakun/gophermart/internal/storage"

	"github.com/go-resty/resty/v2"
)

var QueryUpdateIncreaseBalance = `UPDATE balance set current = current + $2, accruals = accruals + $2 
					where id_user in (SELECT id_user from orders where id_order = $1);`
var QueryUpdateOrdersAccrual = `UPDATE orders SET state = $2, accrual = $3 WHERE id_order = $1;`

// Valid check number is valid or not based on Luhn algorithm
func CheckOrderID(number int) bool {
	return number != 0 && (number%10+checksum(number/10))%10 == 0
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

func AccrualUpdate(configRun *config.Config) (err error) {
	isOrders, arrOrders, err := storage.ReturnOrdersToProcess(configRun)
	if !isOrders {
		log.WithFields(log.Fields{
			"func": "AccrualUpdate nothing to accrual",
		}).Info()
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
			log.WithFields(log.Fields{
				"func": "AccrualUpdate something went wrong while GET accrual for " + orderNum,
			}).Warn(errResp)
			return errResp
		}
		reqStatus := resp.StatusCode()
		if reqStatus == http.StatusInternalServerError {
			log.WithFields(log.Fields{
				"func": "AccrualUpdate StatusCode StatusInternalServerError 500 for " + orderNum,
			}).Warn()
			return
		} else if reqStatus == http.StatusTooManyRequests {
			log.WithFields(log.Fields{
				"func": "AccrualUpdate StatusCode StatusTooManyRequests 429 for " + orderNum,
			}).Warn()
			time.Sleep(60 * time.Second)
			return
		} else if reqStatus == http.StatusOK {
			var orderToAccrual storage.UsingAccrualStruct
			if err = json.Unmarshal(resp.Body(), &orderToAccrual); err != nil {
				log.WithFields(log.Fields{
					"func": "AccrualUpdate error while unmarshalling Accrual  " + orderNum,
				}).Error(err)
				return
			}
			orderToAccrualInt, errConv := strconv.Atoi(orderToAccrual.Order)
			if errConv != nil {
				log.WithFields(log.Fields{
					"func": "AccrualUpdate error while strconv.Atoi(orderToAccrual.Order)",
				}).Error(errConv)
				return errConv
			}
			db, errSQL := sql.Open("pgx", configRun.Database)
			if errSQL != nil {
				log.WithFields(log.Fields{
					"func": "AccrualUpdate.db.sql.Open()",
				}).Error(errSQL)
				return errSQL
			}
			defer db.Close()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
			defer cancel()
			txn, errSQL := db.Begin()
			if errSQL != nil {
				log.WithFields(log.Fields{
					"func": "AccrualUpdate.db.Begin()",
				}).Error(errSQL)
				return errSQL
			}
			defer txn.Rollback()
			_, err = txn.ExecContext(ctx, QueryUpdateOrdersAccrual, orderToAccrualInt, orderToAccrual.Status, orderToAccrual.Accrual)
			if err != nil {
				log.WithFields(log.Fields{
					"func": "AccrualUpdate.QueryUpdateOrdersAccrual",
				}).Error(err)
				return
			}
			if orderToAccrual.Accrual == 0 {
				log.WithFields(log.Fields{
					"func": "AccrualUpdate accrual value is 0",
				}).Warn(err)
				return
			}
			_, err = txn.ExecContext(ctx, QueryUpdateIncreaseBalance, orderToAccrual.Order, orderToAccrual.Accrual)
			if err != nil {
				log.WithFields(log.Fields{
					"func": "AccrualUpdate QueryUpdateIncreaseBalance",
				}).Error(err)
				return
			}
			if err = txn.Commit(); err != nil {
				log.WithFields(log.Fields{
					"func": "AccrualUpdate.txn.Commit()",
				}).Error(err)
				return
			}
			return
		}
	}
	return
}
