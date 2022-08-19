package handlers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-chi/jwtauth/v5"
	"github.com/go-chi/render"

	"github.com/zamibaru89/gophermart/internal/config"
	"github.com/zamibaru89/gophermart/internal/functions"
	"github.com/zamibaru89/gophermart/internal/storage"
)

type Claims struct {
	Username string `json:"username"`
	ID       int    `json:"ID"`
	jwt.StandardClaims
}

func SignUp(config config.ServerConfig, st storage.Repo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var user storage.Credentials
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			//render.JSON(w, r, user)

			return

		}
		err = st.RegisterUser(user)
		if err != nil {
			log.Println(err)
		}
		expirationTime := time.Now().Add(5 * time.Minute)

		claims := &Claims{
			Username: user.Username,
			ID:       user.ID,
			StandardClaims: jwt.StandardClaims{
				// In JWT, the expiry time is expressed as unix milliseconds
				ExpiresAt: expirationTime.Unix(),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

		tokenString, err := token.SignedString([]byte(config.SecretKey))
		if err != nil {
			// If there is an error in creating the JWT return an internal server error
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:    "jwt",
			Value:   tokenString,
			Expires: expirationTime,
		})
	}
}
func Welcome(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	w.Write([]byte(fmt.Sprintf("Hello %v ID %v", claims["username"], claims["ID"])))
}

func PostOrder(config config.ServerConfig, st storage.Repo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, claims, _ := jwtauth.FromContext(r.Context())
		var order storage.Order
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
		}

		order.OrderID = string(body)
		ID := claims["ID"]

		order.UserID = int(ID.(float64))
		fmt.Println("order.UserID", order.UserID)
		checkOrder, err := st.GetOrderByOrderID(order.OrderID)
		if err != nil {
			return
		}
		fmt.Println("checkOrder", checkOrder)
		luhn, err := functions.CheckOrderID(order.OrderID)
		if err != nil {
			log.Println(err)
		}
		if !luhn {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		fmt.Println("checkOrder.UserID=", checkOrder.UserID)
		fmt.Println(" order.UserID=", order.UserID)
		if checkOrder.UserID != 0 {
			if checkOrder.UserID == order.UserID {
				log.Println("the same order for same user")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				return
			} else {
				w.Header().Set("Content-Type", "application/json")
				log.Println("the same order for different user")
				w.WriteHeader(http.StatusConflict)
				return
			}

		} else {

			order.State = "NEW"
			order.UploadedAt = time.Now()
			err = st.PostOrder(order)
			if err != nil {
				return
			}
			log.Println("order was created")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
		}
	}
}

func GetOrders(config config.ServerConfig, st storage.Repo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var order storage.Order
		_, claims, _ := jwtauth.FromContext(r.Context())
		ID := claims["ID"]

		order.UserID = int(ID.(float64))

		listOrders, err := st.GetOrdersByUserID(int64(order.UserID))

		if err != nil {

			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if len(listOrders) == 0 {
			w.WriteHeader(http.StatusNoContent)
		}
		render.JSON(w, r, listOrders)

	}
}
func GetBalance(config config.ServerConfig, st storage.Repo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		var balance storage.Balance
		_, claims, _ := jwtauth.FromContext(r.Context())
		ID := claims["ID"]

		balance.UserID = int(ID.(float64))
		log.Println(ID)
		current, err := st.GetBalanceByUserID(int64(balance.UserID))

		if err != nil {

			balance.Current = 0
		}

		balance.Current = current
		withdrawn, err := st.GetWithdrawalHistoryForUser(int64(balance.UserID))

		if err != nil {
			balance.Withdrawn = 0

		}
		balance.Withdrawn = withdrawn
		render.JSON(w, r, balance)

	}
}

func PostWithdrawal(config config.ServerConfig, st storage.Repo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		_, claims, _ := jwtauth.FromContext(r.Context())
		var withdrawal storage.Withdrawal
		err := json.NewDecoder(r.Body).Decode(&withdrawal)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)

			return

		}

		ID := claims["ID"]

		withdrawal.UserID = int(ID.(float64))

		luhn, err := functions.CheckOrderID(withdrawal.OrderID)
		if err != nil {
			return
		}
		if !luhn {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		currentBalance, err := st.GetBalanceByUserID(int64(withdrawal.UserID))
		if err != nil {
			return
		}
		balanceAfter := currentBalance - withdrawal.Sum
		if balanceAfter >= 0 {
			withdrawal.ProcessedAt = time.Now()
			err = st.PostWithdrawal(withdrawal)
			if err != nil {
				return
			}
			err = st.SetBalanceByUserID(int64(withdrawal.UserID), balanceAfter)
			if err != nil {
				return
			}
		} else {
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}

	}
}
func GetWithdrawals(config config.ServerConfig, st storage.Repo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var withdrawal storage.Withdrawal
		_, claims, _ := jwtauth.FromContext(r.Context())
		ID := claims["ID"]

		withdrawal.UserID = int(ID.(float64))

		listWithdrawals, err := st.GetWithdrawals(int64(withdrawal.UserID))
		if len(listWithdrawals) == 0 {
			w.WriteHeader(http.StatusNoContent)
		}
		if err != nil {

			w.WriteHeader(http.StatusInternalServerError)
			fmt.Println(err)
		}

		render.JSON(w, r, listWithdrawals)

	}
}
func SignIn(config config.ServerConfig, st storage.Repo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var user storage.Credentials
		err := json.NewDecoder(r.Body).Decode(&user)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			//render.JSON(w, r, user)

			return

		}
		user, err = st.SignIn(user)
		if err != nil {
			log.Println(err)
		}
		fmt.Println(user.ID)
		if user.ID != 0 {
			expirationTime := time.Now().Add(60 * time.Minute)
			// Create the JWT claims, which includes the username and expiry time
			claims := &Claims{
				Username: user.Username,
				ID:       user.ID,
				StandardClaims: jwt.StandardClaims{
					// In JWT, the expiry time is expressed as unix milliseconds
					ExpiresAt: expirationTime.Unix(),
				},
			}

			// Declare the token with the algorithm used for signing, and the claims
			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
			// Create the JWT string
			tokenString, err := token.SignedString([]byte(config.SecretKey))
			if err != nil {
				// If there is an error in creating the JWT return an internal server error
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")

			http.SetCookie(w, &http.Cookie{
				Name:    "jwt",
				Value:   tokenString,
				Expires: expirationTime,
			})
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
		}

	}
}
