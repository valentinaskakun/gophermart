package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth/v5"
	"github.com/rs/zerolog"

	"github.com/valentinaskakun/gophermart/internal/config"
	"github.com/valentinaskakun/gophermart/internal/orders"
	"github.com/valentinaskakun/gophermart/internal/storage"
)

func Register(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg string
		expirationTime := time.Now().Add(360 * time.Minute)
		log := zerolog.New(os.Stdout)
		registerUser := storage.CredUserStruct{}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &registerUser); err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		userInfo, err := storage.ReturnIDByLogin(configRun, &registerUser.Login)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
			return
		}
		if userInfo.IDUser != 0 {
			msg = "the login exists"
			fmt.Println(msg)
			w.WriteHeader(http.StatusConflict)
			return
		}
		registerUserID, err := storage.InsertUser(configRun, &registerUser)
		if err != nil {
			log.Warn().Msg(err.Error())
			return
		}
		userAuthInfo := storage.UsingUserStruct{
			Login:  registerUser.Login,
			IDUser: registerUserID,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: expirationTime.Unix(),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, userAuthInfo)
		tokenString, err := token.SignedString([]byte(configRun.KeyToken))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:    "jwt",
			Value:   tokenString,
			Expires: expirationTime,
		})
		w.WriteHeader(http.StatusOK)
	}
}

func Login(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg string
		expirationTime := time.Now().Add(5 * time.Minute)
		log := zerolog.New(os.Stdout)
		var userCred storage.CredUserStruct
		err := json.NewDecoder(r.Body).Decode(&userCred)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		userInfo, err := storage.ReturnIDByLogin(configRun, &userCred.Login)
		if err != nil {
			msg = "something went wrong"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
		}
		if userInfo.IDUser == 0 {
			msg = "the login doesn't exist"
			fmt.Println(msg)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		result, err := storage.CheckUserPass(configRun, &userCred)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusBadRequest)
		}
		if !result {
			msg = "the pass doesn't match"
			fmt.Println(msg)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		userAuthInfo := storage.UsingUserStruct{
			Login:  userCred.Login,
			IDUser: userInfo.IDUser,
			StandardClaims: jwt.StandardClaims{
				ExpiresAt: expirationTime.Unix(),
			},
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, userAuthInfo)
		tokenString, err := token.SignedString([]byte(configRun.KeyToken))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:    "jwt",
			Value:   tokenString,
			Expires: expirationTime,
		})
		w.WriteHeader(http.StatusOK)
	}

}
func Welcome(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	w.Write([]byte(fmt.Sprintf("Hello %v ", claims)))
	fmt.Println("welcome")
	//fmt.Println(userID)
}

func UploadOrder(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var msg string
		log := zerolog.New(os.Stdout)
		fmt.Println("im uploadorder")
		_, claims, _ := jwtauth.FromContext(r.Context())
		userID := int((claims["id_user"]).(float64))
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		orderID, err := strconv.Atoi(string(body))
		if err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		if !orders.CheckOrderID(orderID) {
			log.Warn().Msg("CRC failed")
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		orderInfo, err := storage.ReturnOrderInfoByID(configRun, &orderID)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if orderInfo.IDOrder != 0 && orderInfo.State != "" {
			if orderInfo.IDUser == userID {
				fmt.Println("номер заказа загружен этим пользователем")
				w.WriteHeader(http.StatusOK)
				return
			} else {
				fmt.Println("номер заказа загружен другим пользователем")
				w.WriteHeader(http.StatusConflict)
				return
			}

		}

		orderInfo.IDUser = userID
		orderInfo.State = "NEW"
		orderInfo.UploadedAt = time.Now()

		err = storage.InsertOrder(configRun, &orderInfo)
		if err != nil {
			msg = "error while inserting order"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func GetOrdersList(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var msg string
		_, claims, _ := jwtauth.FromContext(r.Context())
		userID := int((claims["id_user"]).(float64))
		isOrders, arrOrders, err := storage.ReturnOrdersInfoByUserID(configRun, userID)
		if err != nil {
			msg = "something went wrong while returning orders"
			fmt.Println(msg, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !isOrders {
			fmt.Println("нет данных для ответа")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		ordersJSON, err := json.Marshal(arrOrders)
		if err != nil {
			msg = "something went wrong while marshaling orders"
			fmt.Println(msg)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Println("GetOrdersList.im orders in JSON", ordersJSON)
		w.Write(ordersJSON)
	}
}

func GetBalance(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg string
		_, claims, _ := jwtauth.FromContext(r.Context())
		userID := int((claims["id_user"]).(float64))
		balanceInfo, err := storage.ReturnBalanceByUserID(configRun, &userID)
		if err != nil {
			msg = "something went wrong returning balance"
			fmt.Println(msg)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		balanceJSON, err := json.Marshal(balanceInfo)
		if err != nil {
			msg = "something went wrong while marshaling balance"
			fmt.Println(msg)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(balanceJSON)
	}
}

func NewWithdraw(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var msg string
		log := zerolog.New(os.Stdout)
		_, claims, _ := jwtauth.FromContext(r.Context())
		userID := int((claims["id_user"]).(float64))
		orderToWithdrawReq := storage.OrderToWithdrawStruct{}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(body, &orderToWithdrawReq); err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
		orderParsed, err := strconv.Atoi(orderToWithdrawReq.IDOrder)
		if err != nil {
			return
		}
		if !orders.CheckOrderID(orderParsed) {
			log.Warn().Msg("CRC failed")
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
		isBalance, result, err := storage.NewWithdraw(configRun, &orderToWithdrawReq, &userID)
		if err != nil || !result {
			msg = "something went wrong while new withdraw"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !isBalance {
			msg = "sum > balance"
			fmt.Println(msg)
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func GetWithdrawalsList(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, claims, _ := jwtauth.FromContext(r.Context())
		userID := int((claims["id_user"]).(float64))
		var msg string
		log := zerolog.New(os.Stdout)
		isWithdraws, arrWithdraws, err := storage.ReturnWithdrawsInfoByUserID(configRun, &userID)
		if err != nil {
			msg = "something went wrong while returning withdraws"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !isWithdraws {
			fmt.Println("нет списаний")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		withdrawsJSON, err := json.Marshal(arrWithdraws)
		if err != nil {
			msg = "something went wrong while marshalling withdraws"
			fmt.Println(msg)
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(withdrawsJSON)
	}
}
