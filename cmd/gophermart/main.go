package main

import (
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/valentinaskakun/gophermart/internal/config"
	"github.com/valentinaskakun/gophermart/internal/handlers"
	"github.com/valentinaskakun/gophermart/internal/orders"
	"github.com/valentinaskakun/gophermart/internal/storage"

	"github.com/go-chi/chi"
	"github.com/go-chi/jwtauth/v5"
	log "github.com/sirupsen/logrus"
)

func handleSignal(signal os.Signal) {
	log.Println("* Got:", signal)
	os.Exit(-1)
}

func main() {
	config.InitLog()
	var tokenAuth *jwtauth.JWTAuth
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
		log.WithFields(log.Fields{
			"func": "config.LoadConfigServer()",
		}).Error(err)
		log.Fatal(err)
	}
	tokenAuth = jwtauth.New("HS256", []byte(configRun.KeyToken), nil)
	err = storage.InitTables(&configRun)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "storage.InitTables(&configRun)",
		}).Error(err)
	}
	tickerUpdateAccrual := time.NewTicker(2 * time.Second)
	go func() {
		for range tickerUpdateAccrual.C {
			err := orders.AccrualUpdate(&configRun)
			if err != nil {
				log.WithFields(log.Fields{
					"func": "orders.AccrualUpdate(&configRun)",
				}).Error(err)
			}
		}
	}()
	r := chi.NewRouter()
	r.Route("/api/user", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(jwtauth.Verifier(tokenAuth))
			r.Use(jwtauth.Authenticator)
			r.Post("/orders", handlers.UploadOrder(&configRun))
			r.Get("/orders", handlers.GetOrdersList(&configRun))
			r.Get("/balance", handlers.GetBalance(&configRun))
			r.Post("/balance/withdraw", handlers.NewWithdraw(&configRun))
			r.Get("/withdrawals", handlers.GetWithdrawalsList(&configRun))
		})
		r.Group(func(r chi.Router) {
			r.Post("/register", handlers.Register(&configRun))
			r.Post("/login", handlers.Login(&configRun))
		})
	})
	log.Fatal(http.ListenAndServe(configRun.Address, r))
}
