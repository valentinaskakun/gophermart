package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/valentinaskakun/gophermart/internal/config"
	"github.com/valentinaskakun/gophermart/internal/orders"
	"github.com/valentinaskakun/gophermart/internal/storage"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth/v5"
)

func Register(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		expirationTime := time.Now().Add(360 * time.Minute)
		registerUser := storage.CredUserStruct{}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "Register ioutil.ReadAll(r.Body)",
			}).Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if err := json.Unmarshal(body, &registerUser); err != nil {
			log.WithFields(log.Fields{
				"func": "Register json.Unmarshal(body, &registerUser)",
			}).Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		userInfo, err := storage.ReturnIDByLogin(configRun, &registerUser.Login)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "Register.ReturnIDByLogin)",
			}).Error(err)
			return
		}
		if userInfo.IDUser != 0 {
			log.WithFields(log.Fields{
				"func": "Register The login exists",
			}).Info()
			w.WriteHeader(http.StatusConflict)
			return
		}
		registerUserID, err := storage.InsertUser(configRun, &registerUser)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "Register.storage.InsertUser",
			}).Error(err)
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
			log.WithFields(log.Fields{
				"func": "Register.tokenPreparing",
			}).Error(err)
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
		w.Header().Set("Content-Type", "application/json")
		expirationTime := time.Now().Add(5 * time.Minute)
		var userCred storage.CredUserStruct
		err := json.NewDecoder(r.Body).Decode(&userCred)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "Login.json.NewDecoder",
			}).Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		userInfo, err := storage.ReturnIDByLogin(configRun, &userCred.Login)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "Login.ReturnIDByLogin",
			}).Error(err)
		}
		if userInfo.IDUser == 0 {
			log.WithFields(log.Fields{
				"func": "Login.the login doesn't exist",
			}).Info()
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		result, err := storage.CheckUserPass(configRun, &userCred)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "Login.CheckUserPass can't check the pass",
			}).Info()
			w.WriteHeader(http.StatusBadRequest)
		}
		if !result {
			log.WithFields(log.Fields{
				"func": "Login.the pass doesn't match",
			}).Info()
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
			log.WithFields(log.Fields{
				"func": "Login.tokenPreparing",
			}).Error(err)
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

func UploadOrder(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, claims, _ := jwtauth.FromContext(r.Context())
		userID := int((claims["id_user"]).(float64))
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "UploadOrder.ioutil.ReadAll(r.Body)",
			}).Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		orderID, err := strconv.Atoi(string(body))
		if err != nil {
			log.WithFields(log.Fields{
				"func": "UploadOrder.strconv.Atoi(string(body)",
			}).Error(err)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		if !orders.CheckOrderID(orderID) {
			log.WithFields(log.Fields{
				"func": "UploadOrder CRC failed",
			}).Info()
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		orderInfo, err := storage.ReturnOrderInfoByID(configRun, &orderID)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "UploadOrder.ReturnOrderInfoByID",
			}).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if orderInfo.IDOrder != 0 && orderInfo.State != "" {
			if orderInfo.IDUser == userID {
				log.WithFields(log.Fields{
					"func": "UploadOrder.номер заказа загружен этим пользователем",
				}).Info()
				w.WriteHeader(http.StatusOK)
				return
			} else {
				log.WithFields(log.Fields{
					"func": "UploadOrder.номер заказа загружен не этим пользователем",
				}).Info()
				w.WriteHeader(http.StatusConflict)
				return
			}

		}

		orderInfo.IDUser = userID
		orderInfo.State = "NEW"
		orderInfo.UploadedAt = time.Now()

		err = storage.InsertOrder(configRun, &orderInfo)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "UploadOrder.InsertOrder",
			}).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

func GetOrdersList(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, claims, _ := jwtauth.FromContext(r.Context())
		userID := int((claims["id_user"]).(float64))
		isOrders, arrOrders, err := storage.ReturnOrdersInfoByUserID(configRun, userID)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "GetOrdersList.ReturnOrdersInfoByUserID",
			}).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !isOrders {
			log.WithFields(log.Fields{
				"func": "UploadOrder.нет данных для ответа",
			}).Info()
			w.WriteHeader(http.StatusNoContent)
			return
		}
		ordersJSON, err := json.Marshal(arrOrders)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "GetOrdersList.Marshal(arrOrders)",
			}).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(ordersJSON)
	}
}

func GetBalance(configRun *config.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, claims, _ := jwtauth.FromContext(r.Context())
		userID := int((claims["id_user"]).(float64))
		balanceInfo, err := storage.ReturnBalanceByUserID(configRun, &userID)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "GetBalance.ReturnBalanceByUserID",
			}).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		balanceJSON, err := json.Marshal(balanceInfo)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "GetBalance.json.Marshal(balanceInfo)",
			}).Error(err)
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
		_, claims, _ := jwtauth.FromContext(r.Context())
		userID := int((claims["id_user"]).(float64))
		orderToWithdrawReq := storage.OrderToWithdrawStruct{}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "NewWithdraw.ioutil.ReadAll(r.Body)",
			}).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(body, &orderToWithdrawReq); err != nil {
			log.WithFields(log.Fields{
				"func": "NewWithdraw.json.Unmarshal(body, &orderToWithdrawReq)",
			}).Error(err)
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
		orderParsed, err := strconv.Atoi(orderToWithdrawReq.IDOrder)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "NewWithdraw.strconv.Atoi(orderToWithdrawReq.IDOrder)",
			}).Error(err)
			return
		}
		if !orders.CheckOrderID(orderParsed) {
			log.WithFields(log.Fields{
				"func": "NewWithdraw.CRC failed",
			}).Error(err)
			w.WriteHeader(http.StatusUnprocessableEntity)
		}
		isBalance, result, err := storage.NewWithdraw(configRun, &orderToWithdrawReq, &userID)
		if err != nil || !result {
			log.WithFields(log.Fields{
				"func": "NewWithdraw.storage.NewWithdraw",
			}).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !isBalance {
			log.WithFields(log.Fields{
				"func": "NewWithdraw.balance",
			}).Info()
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
		isWithdraws, arrWithdraws, err := storage.ReturnWithdrawsInfoByUserID(configRun, &userID)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "GetWithdrawalsList.ReturnWithdrawsInfoByUserID",
			}).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !isWithdraws {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		withdrawsJSON, err := json.Marshal(arrWithdraws)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "GetWithdrawalsList.json.Marshal(arrWithdraws)",
			}).Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write(withdrawsJSON)
	}
}
