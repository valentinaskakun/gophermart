package storage

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/valentinaskakun/gophermart/internal/config"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type CredUserStruct struct {
	Login    string `json:"login" ,db:"login"`
	Password string `json:"password" ,db:"password"`
}
type UsingUserStruct struct {
	IDUser int    `json:"id_user" ,db:"id_user"`
	Login  string `json:"login" ,db:"login"`
	jwt.StandardClaims
}
type UsingUserBalanceStruct struct {
	IDUser    int     `json:"id_user" ,db:"id_user"`
	Current   float64 `json:"current" ,db:"current"`
	Accrual   float64 `db:"accruals"`
	Withdrawn float64 `json:"withdrawn" ,db:"withdrawn"`
}
type OrderToWithdrawStruct struct {
	IDOrder string  `json:"order,omitempty" ,db:"id_order"`
	Sum     float64 `json:"sum,omitempty" ,db:"sum"`
}
type UsingOrderStruct struct {
	IDOrder    int       `json:"id_order,omitempty"`
	Number     string    `json:"number,omitempty" ,db:"id_order"`
	IDUser     int       `json:"id_user,omitempty" ,db:"id_user"`
	State      string    `json:"status,omitempty" ,db:"state"`
	Accrual    float64   `json:"accrual,omitempty" ,db:"accrual"`
	UploadedAt time.Time `json:"uploaded_at,omitempty" ,db:"uploaded_at"`
}
type UsingAccrualStruct struct {
	Order   string  `json:"order" ,db:"id_order"`
	Status  string  `json:"status" ,db:"state"`
	Accrual float64 `json:"accrual,omitempty" ,db:"accrual"`
}
type UsingWithdrawStruct struct {
	IDOrder     string    `json:"order" ,db:"id_order"`
	Withdraw    float64   `json:"sum" ,db:"withdraw"`
	ProcessedAt time.Time `json:"processed_at,omitempty" ,db:"processed_at"`
}

func InitTables(config *config.Config) (err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "InitTables.sql.Open",
		}).Error(err)
		return err
	} else {
		defer db.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_, err := db.ExecContext(ctx, PostgresDBRun.queryInitUsers)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "InitTables.ExecContext.PostgresDBRun.queryInitUsers",
			}).Error(err)
			return err
		}
		_, err = db.ExecContext(ctx, PostgresDBRun.queryInitOrders)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "InitTables.ExecContext.PostgresDBRun.queryInitOrders",
			}).Error(err)
			return err
		}
		_, err = db.ExecContext(ctx, PostgresDBRun.queryInitBalance)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "InitTables.ExecContext.PostgresDBRun.queryInitBalance",
			}).Error(err)
			return err
		}
		_, err = db.ExecContext(ctx, PostgresDBRun.queryInitWithdraws)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "InitTables.ExecContext.PostgresDBRun.queryInitWithdraws",
			}).Error(err)
			return err
		}
	}
	return
}

func InsertUser(config *config.Config, userAuthInfo *CredUserStruct) (userID int, err error) {
	var maxID int
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "InsertUser.db.sql.Open()",
		}).Error(err)
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectMaxIDUsers).Scan(&maxID)
	//todo: выпилить костыль NULL
	if err != nil {
		var count int
		err2 := db.QueryRowContext(ctx, PostgresDBRun.querySelectCountUsers).Scan(&count)
		if err2 == nil {
			if count == 0 {
				maxID = 0
				err = nil
			}
		} else {
			return
		}
	}
	newID := maxID + 1
	txn, err := db.Begin()
	if err != nil {
		log.WithFields(log.Fields{
			"func": "InsertUser.db.Begin()",
		}).Error(err)
		return userID, errors.Wrap(err, "could not start a new transaction")
	}
	defer txn.Rollback()
	_, err = txn.Exec(PostgresDBRun.queryInsertUser, newID, userAuthInfo.Login, userAuthInfo.Password)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "InsertUser.txn.Exec(PostgresDBRun.queryInsertUser)" + userAuthInfo.Login,
		}).Error(err)
		return userID, errors.Wrap(err, "failed to insert multiple records at once")
	}
	_, err = txn.Exec(PostgresDBRun.queryInsertUserBalance, newID)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "InsertUser.txn.Exec(PostgresDBRun.queryInsertUserBalance, newID)",
		}).Error(err)
		return userID, errors.Wrap(err, "failed to insert multiple records at once")
	}
	if err := txn.Commit(); err != nil {
		log.WithFields(log.Fields{
			"func": "InsertUser.txn.Commit()",
		}).Error(err)
		return userID, errors.Wrap(err, "failed to commit transaction")
	}
	userID = newID
	return

}

func CheckUserPass(config *config.Config, userAuthInfo *CredUserStruct) (result bool, err error) {
	var pass string
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "CheckUserPass.db.sql.Open()",
		}).Error(err)
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = db.QueryRowContext(ctx, PostgresDBRun.queryCheckPassword, userAuthInfo.Login).Scan(&pass)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "CheckUserPass.db.QueryRowContext(ctx, PostgresDBRun.queryCheckPassword, userAuthInfo.Login)" + userAuthInfo.Login,
		}).Error(err)
		return
	}
	if userAuthInfo.Password == pass {
		result = true
		return
	}
	result = false
	log.WithFields(log.Fields{
		"func": "CheckUserPass passwords don't match for" + userAuthInfo.Login,
	}).Info()
	return
}

func ReturnIDByLogin(config *config.Config, login *string) (userAuthInfo UsingUserStruct, err error) {
	userAuthInfo.Login = *login
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "ReturnIDByLogin.db.sql.Open()",
		}).Error(err)
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	var countByLogin int
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectCountByLogin, login).Scan(&countByLogin)
	if err != nil || countByLogin == 0 {
		userAuthInfo.IDUser = 0
		log.WithFields(log.Fields{
			"func": "ReturnIDByLogin.PostgresDBRun.querySelectCountByLogin" + *login,
		}).Error(err)
		return
	}
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectIDByLogin, login).Scan(&userAuthInfo.IDUser)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "ReturnIDByLogin.PostgresDBRun.querySelectIDByLogin" + *login,
		}).Error(err)
		return
	}
	return
}

func InsertOrder(config *config.Config, order *UsingOrderStruct) (err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "InsertOrder.sql.Open()",
		}).Error(err)
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = db.ExecContext(ctx, PostgresDBRun.queryInsertOrder, order.IDOrder, order.IDUser, order.State, 0, order.UploadedAt)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "InsertOrder.PostgresDBRun.queryInsertOrder ",
		}).Error(err)
		return err
	}
	return
}

func NewWithdraw(config *config.Config, order *OrderToWithdrawStruct, userID *int) (isBalance bool, result bool, err error) {
	var userBalanceInfo UsingUserBalanceStruct
	orderParsed, err := strconv.Atoi(order.IDOrder)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "NewWithdraw.strconv.Atoi(order.IDOrder)",
		}).Error(err)
		return
	}
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "NewWithdraw.sql.Open()",
		}).Error(err)
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	txn, err := db.Begin()
	if err != nil {
		log.WithFields(log.Fields{
			"func": "NewWithdraw.db.Begin()",
		}).Error(err)
		return
	}
	defer txn.Rollback()
	err = txn.QueryRowContext(ctx, PostgresDBRun.querySelectBalance, userID).Scan(&userBalanceInfo.Current, &userBalanceInfo.Accrual, &userBalanceInfo.Withdrawn)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "NewWithdraw.PostgresDBRun.querySelectBalance",
		}).Error(err)
		return
	}
	if userBalanceInfo.Current < order.Sum {
		isBalance = false
		result = true
		log.WithFields(log.Fields{
			"func": "NewWithdraw.userBalanceInfo balance < sum",
		}).Info()
		return
	}
	isBalance = true
	_, err = txn.ExecContext(ctx, PostgresDBRun.queryUpdateDecreaseBalance, userID, order.Sum)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "NewWithdraw.queryUpdateDecreaseBalance failed",
		}).Error(err)
		return
	}
	_, err = txn.ExecContext(ctx, PostgresDBRun.queryInsertWithdraw, orderParsed, userID, order.Sum, time.Now())
	if err != nil {
		log.WithFields(log.Fields{
			"func": "NewWithdraw.queryInsertWithdraw",
		}).Error(err)
		return
	}
	if err = txn.Commit(); err != nil {
		log.WithFields(log.Fields{
			"func": "NewWithdraw.txn.Commit()",
		}).Error(err)
		return
	}
	result = true
	return
}

func ReturnOrdersInfoByUserID(config *config.Config, userID int) (isOrders bool, arrOrders []UsingOrderStruct, err error) {
	var orderInfo UsingOrderStruct
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "ReturnOrdersInfoByUserID.sql.Open()",
		}).Error(err)
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rows, err := db.QueryContext(ctx, PostgresDBRun.querySelectOrderByUserID, userID)
	if err != nil || rows.Err() != nil {
		log.WithFields(log.Fields{
			"func": "ReturnOrdersInfoByUserID.PostgresDBRun.querySelectOrderByUserID",
		}).Error(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&orderInfo.Number, &orderInfo.State, &orderInfo.Accrual, &orderInfo.UploadedAt)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "ReturnOrdersInfoByUserID.ScanRow failed",
			}).Error(err)
			return
		}
		arrOrders = append(arrOrders, orderInfo)
	}
	isOrders = true
	return
}

func ReturnBalanceByUserID(config *config.Config, IDUser *int) (userBalanceInfo UsingUserBalanceStruct, err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "ReturnBalanceByUserID.sql.Open()",
		}).Error(err)
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectBalance, IDUser).Scan(&userBalanceInfo.Current, &userBalanceInfo.Accrual, &userBalanceInfo.Withdrawn)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "ReturnBalanceByUserID.PostgresDBRun.querySelectBalance ",
		}).Error(err)
		return
	}
	return
}

func ReturnOrderInfoByID(config *config.Config, orderID *int) (orderInfo UsingOrderStruct, err error) {
	var count int
	orderInfo.IDOrder = *orderID
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "ReturnOrderInfoByID.sql.Open()",
		}).Error(err)
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectCountOrdersByID, orderID).Scan(&count)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "ReturnOrderInfoByID.PostgresDBRun.querySelectCountOrdersByID ",
		}).Error(err)
	}
	if count != 0 {
		err = db.QueryRowContext(ctx, PostgresDBRun.querySelectOrderInfoByID, orderID).Scan(&orderInfo.IDOrder, &orderInfo.IDUser, &orderInfo.State, &orderInfo.Accrual, &orderInfo.UploadedAt)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "ReturnOrderInfoByID.PostgresDBRun.querySelectOrderInfoByID ",
			}).Error(err)
		}
		return

	}
	return
}

func ReturnWithdrawsInfoByUserID(config *config.Config, userID *int) (isWithdraws bool, arrWithdraws []UsingWithdrawStruct, err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "ReturnWithdrawsInfoByUserID.sql.Open()",
		}).Error(err)
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rows, err := db.QueryContext(ctx, PostgresDBRun.querySelectWithdrawsByUserID, userID)
	if err != nil || rows.Err() != nil {
		log.WithFields(log.Fields{
			"func": "ReturnWithdrawsInfoByUserID.PostgresDBRun.querySelectWithdrawsByUserID ",
		}).Error(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var withdrawInfo UsingWithdrawStruct
		err = rows.Scan(&withdrawInfo.IDOrder, &withdrawInfo.Withdraw, &withdrawInfo.ProcessedAt)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "ReturnWithdrawsInfoByUserID.PostgresDBRun.querySelectWithdrawsByUserID.Scan ",
			}).Error(err)
			return
		}
		arrWithdraws = append(arrWithdraws, withdrawInfo)
	}
	isWithdraws = true
	if err != nil {
		log.WithFields(log.Fields{
			"func": "ReturnWithdrawsInfoByUserID",
		}).Error(err)
	}
	return
}
func ReturnOrdersToProcess(config *config.Config) (isOrders bool, arrOrders []int, err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.WithFields(log.Fields{
			"func": "ReturnOrdersToProcess.sql.Open()",
		}).Error(err)
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rows, err := db.QueryContext(ctx, PostgresDBRun.querySelectOrdersToProcess)
	if err != nil || rows.Err() != nil {
		log.WithFields(log.Fields{
			"func": "ReturnOrdersToProcess.PostgresDBRun.querySelectOrdersToProcess",
		}).Error(err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var orderNum int
		err = rows.Scan(&orderNum)
		if err != nil {
			log.WithFields(log.Fields{
				"func": "ReturnOrdersToProcess.Scan(&orderNum)",
			}).Error(err)
			return
		}
		arrOrders = append(arrOrders, orderNum)
	}
	if err != nil {
		log.WithFields(log.Fields{
			"func": "ReturnOrdersToProcess failed",
		}).Error(err)
		return
	}
	isOrders = true
	return
}
