package functions

import (
	"encoding/json"
	"github.com/go-resty/resty/v2"
	"github.com/zamibaru89/gophermart/internal/config"
	"github.com/zamibaru89/gophermart/internal/storage"
	"net/http"
	"time"
)

func AccrualUpdate(repo storage.Repo, conf config.ServerConfig) error {
	updateList, err := repo.GetOrdersForUpdate()
	if err != nil {
		return err
	}

	if len(updateList) != 0 {
		req := resty.New().
			SetBaseURL(conf.AccrualAddress).
			R().
			SetHeader("Content-Type", "application/json")
		for _, order := range updateList {

			orderNum := order.OrderID
			resp, err := req.Get("/api/orders/" + orderNum)
			if err != nil {
				return err
			}

			status := resp.StatusCode()
			switch status {
			case http.StatusTooManyRequests:
				time.Sleep(10 * time.Second)
				return nil

			case http.StatusOK:
				var toUpdateOrder storage.Accrual
				err = json.Unmarshal(resp.Body(), &toUpdateOrder)
				if err != nil {
					return err
				}
				order.State = toUpdateOrder.State
				order.OrderID = toUpdateOrder.OrderID

				order.Accrual = toUpdateOrder.Accrual

				err = repo.PostOrder(order)
				if err != nil {
					return err
				}
				oldBalance, err := repo.GetBalanceByUserID(int64(order.UserID))
				if err != nil {
					return err
				}
				newBalance := oldBalance + toUpdateOrder.Accrual

				err = repo.SetBalanceByUserID(int64(order.UserID), newBalance)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
