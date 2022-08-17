package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog"

	"gophermart/internal/config"
	"gophermart/internal/storage"
)

func Register(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg string
		log := zerolog.New(os.Stdout)
		registerUser := storage.RegisterUserStruct{}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusBadRequest)
		}
		if err := json.Unmarshal(body, &registerUser); err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusBadRequest)
		}
		userInfo, err := storage.ReturnIdByLogin(configRun, &registerUser.Login)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
		}
		if userInfo.IdUser != 0 {
			msg = "the login exists"
			fmt.Println(msg)
			return
		} else {
			err = storage.InsertUser(configRun, &registerUser)
			if err != nil {
				log.Warn().Msg(err.Error())
			}
			w.WriteHeader(http.StatusOK)
			w.Write(body)
			fmt.Sprintf(registerUser.Login, registerUser.Password)
		}
	}
}

func UploadOrder(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg string
		log := zerolog.New(os.Stdout)
		order := storage.UsingOrderStruct{}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusBadRequest)
		}
		orderId, err := strconv.Atoi(string(body))
		if err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusBadRequest)
		}
		order.IdOrder = orderId
		order.State = "NEW"
		order.UploadedAt = time.Now()
		//добавить UserId
		err = storage.InsertOrder(configRun, &order)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}
}

func GetOrdersList(configRun *config.Config, userId *int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg string
		log := zerolog.New(os.Stdout)
		arrOrders, err := storage.ReturnOrdersInfoByUserId(configRun, userId)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
		}
		ordersJSON, err := json.Marshal(arrOrders)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(ordersJSON)
	}
}

func GetBalance(configRun *config.Config, userId *int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg string
		log := zerolog.New(os.Stdout)
		balanceInfo, err := storage.ReturnBalanceByUserId(configRun, userId)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
		}
		balanceJSON, err := json.Marshal(balanceInfo)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(balanceJSON)
	}
}

func WithdrawBalance(configRun *config.Config, userId *int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg string
		log := zerolog.New(os.Stdout)
		orderToWithdraw := storage.OrderToWithdrawStruct{}
		orderToWithdrawFull := storage.UsingWithdrawStruct{}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusBadRequest)
		}
		if err := json.Unmarshal(body, &orderToWithdraw); err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusBadRequest)
		}
		currentBalance, err := storage.ReturnBalanceByUserId(configRun, userId)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
		}
		if currentBalance.Current < orderToWithdraw.Withdraw {
			msg = "balance is less then sum"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
			return
		}

		orderToWithdrawFull.ProcessedAt = time.Now()
		orderToWithdrawFull.Withdraw = orderToWithdraw.Withdraw
		orderToWithdrawFull.IdOrder = orderToWithdraw.IdOrder
		//добавить UserId
		err = storage.InsertWithdraw(configRun, &orderToWithdrawFull, userId)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
		}
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}
}

func GetWithdrawalsList(configRun *config.Config, userId *int) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg string
		log := zerolog.New(os.Stdout)
		arrWithdraws, err := storage.ReturnWithdrawsInfoByUserId(configRun, userId)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
		}
		ordersJSON, err := json.Marshal(arrWithdraws)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(ordersJSON)
	}
}
