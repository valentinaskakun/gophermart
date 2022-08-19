package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/jwtauth/v5"
	"github.com/zamibaru89/gophermart/internal/config"
	"github.com/zamibaru89/gophermart/internal/functions"
	"github.com/zamibaru89/gophermart/internal/handlers"
	"github.com/zamibaru89/gophermart/internal/storage"
	"log"
	"net/http"
	"time"
)

var tokenAuth *jwtauth.JWTAuth

func main() {

	ServerConfig, err := config.LoadServerConfig()
	if err != nil {
		log.Fatal(err)
		return
	}
	tokenAuth = jwtauth.New("HS256", []byte(ServerConfig.SecretKey), nil)
	Server, Conn, err := storage.NewPostgresStorage(ServerConfig)
	fmt.Println(Server, Conn)
	if err != nil {
		log.Fatal(err)
		return

	}
	tickerUpdate := time.NewTicker(10 * time.Second)
	go func() {
		for range tickerUpdate.C {
			//log.Println("start AccrualUpdate")
			err := functions.AccrualUpdate(Server, ServerConfig)
			if err != nil {
				log.Println(err)
			}
		}
	}()
	r := chi.NewRouter()
	r.Route("/api/user", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(jwtauth.Verifier(tokenAuth))
			r.Use(jwtauth.Authenticator)
			r.Post("/orders", handlers.PostOrder(ServerConfig, Server))
			r.Get("/orders", handlers.GetOrders(ServerConfig, Server))
			r.Get("/balance", handlers.GetBalance(ServerConfig, Server))
			r.Post("/balance/withdraw", handlers.PostWithdrawal(ServerConfig, Server))
			r.Get("/withdrawals", handlers.GetWithdrawals(ServerConfig, Server))
		})
		r.Group(func(r chi.Router) {
			r.Post("/register", handlers.SignUp(ServerConfig, Server))
			r.Post("/login", handlers.SignIn(ServerConfig, Server))
		})
	})
	http.ListenAndServe(ServerConfig.Address, r)

}
