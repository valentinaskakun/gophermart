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
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusBadRequest)
		}
		if err := json.Unmarshal(body, &orderToWithdraw); err != nil {
			log.Warn().Msg(err.Error())
			w.WriteHeader(http.StatusBadRequest)
		}
		userInfo, err := storage.ReturnIdByLogin(configRun, userId)
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

//func ListMetric(metricsRun *storage.Metrics, saveConfig *storage.SaveConfig) func(w http.ResponseWriter, r *http.Request) {
//	return func(w http.ResponseWriter, r *http.Request) {
//		metricsRun.GetMetrics(saveConfig)
//		metricType := chi.URLParam(r, "metricType")
//		metricName := chi.URLParam(r, "metricName")
//		if metricType == "gauge" {
//			if val, ok := metricsRun.GaugeMetric[metricName]; ok {
//				fmt.Fprintln(w, val)
//			} else {
//				w.WriteHeader(http.StatusNotFound)
//			}
//		} else if metricType == "counter" {
//			if val, ok := metricsRun.CounterMetric[metricName]; ok {
//				fmt.Fprintln(w, val)
//			} else {
//				w.WriteHeader(http.StatusNotFound)
//			}
//		} else {
//			w.WriteHeader(http.StatusNotImplemented)
//		}
//	}
//}
//
//func ListMetricJSON(metricsRun *storage.Metrics, saveConfig *storage.SaveConfig, useHash string) func(w http.ResponseWriter, r *http.Request) {
//	return func(w http.ResponseWriter, r *http.Request) {
//		log := zerolog.New(os.Stdout)
//		metricsRun.GetMetrics(saveConfig)
//		w.Header().Set("Content-Type", "application/json")
//		metricReq, metricRes := storage.MetricsJSON{}, storage.MetricsJSON{}
//		body, err := ioutil.ReadAll(r.Body)
//		if err != nil {
//			log.Warn().Msg(err.Error())
//			w.WriteHeader(http.StatusBadRequest)
//		}
//		if err := json.Unmarshal(body, &metricReq); err != nil {
//			log.Warn().Msg(err.Error())
//			w.WriteHeader(http.StatusBadRequest)
//		}
//		if metricReq.MType == "gauge" {
//			if _, ok := metricsRun.GaugeMetric[metricReq.ID]; ok {
//				metricRes.ID, metricRes.MType, metricRes.Delta = metricReq.ID, metricReq.MType, metricReq.Delta
//				valueRes := metricsRun.GaugeMetric[metricReq.ID]
//				metricRes.Value = &valueRes
//				if len(useHash) > 0 {
//					metricRes.Hash = config.Hash(fmt.Sprintf("%s:gauge:%f", metricRes.ID, *metricRes.Value), useHash)
//				}
//			} else {
//				w.WriteHeader(http.StatusNotFound)
//				return
//			}
//		} else if metricReq.MType == "counter" {
//			if _, ok := metricsRun.CounterMetric[metricReq.ID]; ok {
//				metricRes.ID, metricRes.MType, metricRes.Value = metricReq.ID, metricReq.MType, metricReq.Value
//				valueRes := metricsRun.CounterMetric[metricReq.ID]
//				metricRes.Delta = &valueRes
//				if len(useHash) > 0 {
//					metricRes.Hash = config.Hash(fmt.Sprintf("%s:counter:%d", metricRes.ID, *metricRes.Delta), useHash)
//				}
//			} else {
//				w.WriteHeader(http.StatusNotFound)
//				return
//			}
//		} else {
//			w.WriteHeader(http.StatusNotFound)
//			return
//		}
//		if resBody, err := json.Marshal(metricRes); err != nil {
//			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
//			return
//		} else {
//			w.WriteHeader(http.StatusOK)
//			w.Write(resBody)
//		}
//	}
//}
//
//func UpdateMetric(metricsRun *storage.Metrics, saveConfig *storage.SaveConfig) func(w http.ResponseWriter, r *http.Request) {
//	return func(w http.ResponseWriter, r *http.Request) {
//		log := zerolog.New(os.Stdout)
//		metricsRun.GetMetrics(saveConfig)
//		metricType := chi.URLParam(r, "metricType")
//		metricName := chi.URLParam(r, "metricName")
//		metricValue := chi.URLParam(r, "metricValue")
//		if metricType == "gauge" {
//			valParsed, err := strconv.ParseFloat(metricValue, 64)
//			if err != nil {
//				log.Warn().Msg(err.Error())
//				w.WriteHeader(http.StatusBadRequest)
//			} else {
//				metricsRun.MuGauge.Lock()
//				metricsRun.GaugeMetric[metricName] = valParsed
//				metricsRun.MuGauge.Unlock()
//			}
//		} else if metricType == "counter" {
//			valParsed, err := strconv.ParseInt(metricValue, 10, 64)
//			if err != nil {
//				log.Warn().Msg(err.Error())
//				w.WriteHeader(http.StatusBadRequest)
//			} else {
//				metricsRun.MuCounter.Lock()
//				metricsRun.CounterMetric[metricName] += valParsed
//				metricsRun.MuCounter.Unlock()
//			}
//		} else {
//			w.WriteHeader(http.StatusNotImplemented)
//		}
//		metricsRun.SaveMetrics(saveConfig)
//	}
//}
//
//func UpdateMetricJSON(metricsRun *storage.Metrics, saveConfig *storage.SaveConfig, useHash string) func(w http.ResponseWriter, r *http.Request) {
//	return func(w http.ResponseWriter, r *http.Request) {
//		log := zerolog.New(os.Stdout)
//		metricsRun.GetMetrics(saveConfig)
//		metricReq := storage.MetricsJSON{}
//		body, err := ioutil.ReadAll(r.Body)
//		if err != nil {
//			log.Warn().Msg(err.Error())
//			w.WriteHeader(http.StatusBadRequest)
//		}
//		if err := json.Unmarshal(body, &metricReq); err != nil {
//			log.Warn().Msg(err.Error())
//			w.WriteHeader(http.StatusBadRequest)
//		}
//		if metricReq.MType == "gauge" {
//			if (len(useHash) > 0) && (metricReq.Hash != config.Hash(fmt.Sprintf("%s:gauge:%f", metricReq.ID, *metricReq.Value), useHash)) {
//				w.WriteHeader(http.StatusBadRequest)
//			} else {
//				metricsRun.MuGauge.Lock()
//				metricsRun.GaugeMetric[metricReq.ID] = *metricReq.Value
//				metricsRun.MuGauge.Unlock()
//			}
//		} else if metricReq.MType == "counter" {
//			if (len(useHash) > 0) && (metricReq.Hash != config.Hash(fmt.Sprintf("%s:counter:%d", metricReq.ID, *metricReq.Delta), useHash)) {
//				w.WriteHeader(http.StatusBadRequest)
//			} else {
//				metricsRun.MuCounter.Lock()
//				metricsRun.CounterMetric[metricReq.ID] += *metricReq.Delta
//				metricsRun.MuCounter.Unlock()
//			}
//		} else {
//			w.WriteHeader(http.StatusNotImplemented)
//		}
//		metricsRun.SaveMetrics(saveConfig)
//		//какой-то непонятный костыль, только для 11-го инкремента?
//		if saveConfig.ToDatabase {
//			err = storage.UpdateRow(saveConfig, &metricReq)
//			if err != nil {
//				log.Warn().Msg(err.Error())
//			}
//		}
//		w.WriteHeader(http.StatusOK)
//		resBody, err := json.Marshal("{}")
//		if err != nil {
//			log.Warn().Msg(err.Error())
//		}
//		w.Write(resBody)
//	}
//}
//
//func UpdateMetrics(metricsRun *storage.Metrics, saveConfig *storage.SaveConfig) func(w http.ResponseWriter, r *http.Request) {
//	return func(w http.ResponseWriter, r *http.Request) {
//		log := zerolog.New(os.Stdout)
//		metricsRun.GetMetrics(saveConfig)
//		var metricsBatch []storage.MetricsJSON
//		body, err := ioutil.ReadAll(r.Body)
//		if err != nil {
//			log.Warn().Msg(err.Error())
//			w.WriteHeader(http.StatusBadRequest)
//		}
//		if err := json.Unmarshal(body, &metricsBatch); err != nil {
//			log.Warn().Msg(err.Error())
//			w.WriteHeader(http.StatusBadRequest)
//		}
//		//todo: переделать на интерфейс хранения
//		for _, metricReq := range metricsBatch {
//			if metricReq.MType == "gauge" {
//				metricsRun.MuGauge.Lock()
//				metricsRun.GaugeMetric[metricReq.ID] = *metricReq.Value
//				metricsRun.MuGauge.Unlock()
//			} else if metricReq.MType == "counter" {
//				metricsRun.MuCounter.Lock()
//				metricsRun.CounterMetric[metricReq.ID] += *metricReq.Delta
//				metricsRun.MuCounter.Unlock()
//			}
//		}
//		metricsRun.SaveMetrics(saveConfig)
//		if saveConfig.ToDatabase {
//			err = storage.UpdateBatch(saveConfig, metricsBatch)
//			if err != nil {
//				log.Warn().Msg(err.Error())
//			}
//		}
//	}
//}
//
//func Ping(saveConfig *storage.SaveConfig) func(w http.ResponseWriter, r *http.Request) {
//	return func(w http.ResponseWriter, r *http.Request) {
//		log := zerolog.New(os.Stdout)
//		if saveConfig.ToDatabase {
//			err := storage.PingDatabase(saveConfig)
//			if err != nil {
//				w.WriteHeader(http.StatusInternalServerError)
//				log.Print("err", err)
//			} else {
//				w.WriteHeader(http.StatusOK)
//			}
//		} else {
//			w.WriteHeader(http.StatusInternalServerError)
//			log.Print(w, "Database DSN isn't set")
//		}
//	}
//}
