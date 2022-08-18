package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth/v5"

	"github.com/valentinaskakun/gophermart/internal/config"
	"github.com/valentinaskakun/gophermart/internal/handlers"
	"github.com/valentinaskakun/gophermart/internal/orders"
	"github.com/valentinaskakun/gophermart/internal/storage"
)

var tokenAuth *jwtauth.JWTAuth

func handleSignal(signal os.Signal) {
	log.Println("* Got:", signal)
	os.Exit(-1)
}
func main() {
	//обработка сигналов
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		for {
			sig := <-sigs
			handleSignal(sig)
		}
	}()
	configRun, err := config.LoadConfigServer()
	if err != nil {
		log.Fatal(err)
	}
	tokenAuth = jwtauth.New("HS256", []byte(configRun.KeyToken), nil)
	err = storage.InitTables(&configRun)
	if err != nil {
		log.Fatal(err)
	}
	tickerUpdateAccrual := time.NewTicker(2 * time.Second)
	go func() {
		for range tickerUpdateAccrual.C {
			err := orders.AccrualUpdate(&configRun)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(tokenAuth))
		r.Use(jwtauth.Authenticator)
		r.Get("/api/user/welcome", handlers.Welcome)
		r.Post("/api/user/orders", handlers.UploadOrder(&configRun))
		r.Get("/api/user/orders", handlers.GetOrdersList(&configRun))
		r.Get("/api/user/balance", handlers.GetBalance(&configRun))
		r.Post("/api/user/balance/withdraw", handlers.NewWithdraw(&configRun))
		r.Get("/api/user/withdrawals", handlers.GetWithdrawalsList(&configRun))
	})
	r.Group(func(r chi.Router) {
		r.Post("/api/user/register", handlers.Register(&configRun))
		r.Post("/api/user/login", handlers.Login(&configRun))
	})
	log.Fatal(http.ListenAndServe(configRun.Address, r))
}
