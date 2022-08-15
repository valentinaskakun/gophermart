package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi"

	"gophermart/internal/config"
	"gophermart/internal/handlers"
	"gophermart/internal/orders"
	"gophermart/internal/storage"
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
	err = storage.InitTables(&configRun)
	if err != nil {
		log.Fatal(err)
	}
	res := orders.CheckOrderId(9278923470)
	userId := 3
	fmt.Println(res)

	r := chi.NewRouter()
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", handlers.Register(&configRun))
		//r.Post("/login", handlers.Login(&metricsRun, saveConfigRun))
		r.Post("/orders", handlers.UploadOrder(&configRun))
		r.Get("/orders", handlers.GetOrdersList(&configRun, &userId))
		r.Get("/balance", handlers.GetBalance(&configRun, &userId))
		r.Post("/balance/withdraw", handlers.WithdrawBalance(&configRun, &userId))
		//r.Get("/balance/withdrawals", handlers.GetScore(&metricsRun, saveConfigRun))
	})
	log.Fatal(http.ListenAndServe(configRun.Address, r))
}
