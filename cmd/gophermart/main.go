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
	tokenAuth := jwtauth.New("HS256", configRun.KeyToken, nil)
	err = storage.InitTables(&configRun)
	if err != nil {
		log.Fatal(err)
	}
	tickerUpdateAccrual := time.NewTicker(10 * time.Second)
	go func() {
		for range tickerUpdateAccrual.C {
			err := orders.AccrualUpdate(&configRun)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()
	r := chi.NewRouter()
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", handlers.Register(&configRun))
		r.Post("/login", handlers.Login(&configRun))
		r.With(jwtauth.Verifier(tokenAuth)).Post("/orders", handlers.UploadOrder(&configRun))
		r.With(jwtauth.Verifier(tokenAuth)).Get("/orders", handlers.GetOrdersList(&configRun))
		r.With(jwtauth.Verifier(tokenAuth)).Get("/balance", handlers.GetBalance(&configRun))
		r.With(jwtauth.Verifier(tokenAuth)).Post("/balance/withdraw", handlers.NewWithdraw(&configRun))
		r.With(jwtauth.Verifier(tokenAuth)).Get("/balance/withdrawals", handlers.GetWithdrawalsList(&configRun))
	})
	log.Fatal(http.ListenAndServe(configRun.Address, r))
}
