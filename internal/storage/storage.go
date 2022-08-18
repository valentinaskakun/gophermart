package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/dgrijalva/jwt-go"

	"github.com/valentinaskakun/gophermart/internal/config"
)

type CredUserStruct struct {
	Login    string `json:"login" ,db:"login"`
	Password string `json:"password" ,db:"password"`
}
type UsingUserStruct struct {
	IdUser int    `json:"id_user" ,db:"id_user"`
	Login  string `json:"login" ,db:"login"`
	jwt.StandardClaims
}
type UsingUserBalanceStruct struct {
	Current   float64 `json:"current" ,db:"current"`
	Accrual   float64 `db:"accrual"`
	Withdrawn float64 `json:"withdrawn" ,db:"withdrawn"`
}
type OrderToWithdrawStruct struct {
	IdOrder int     `json:"id_order" ,db:"id_order"`
	Sum     float64 `json:"sum" ,db:"sum"`
}
type UsingOrderStruct struct {
	IdOrder    int       `json:"id_order" ,db:"id_order"`
	IdUser     int       `json:"id_user,omitempty" ,db:"id_user"`
	State      string    `json:"state,omitempty" ,db:"state"`
	Accrual    float64   `json:"accrual,omitempty" ,db:"accrual"`
	UploadedAt time.Time `json:"uploaded_at,omitempty" ,db:"uploaded_at"`
}
type UsingAccrualStruct struct {
	Order   int     `json:"order" ,db:"id_order"`
	Status  string  `json:"status,omitempty" ,db:"state"`
	Accrual float64 `json:"accrual,omitempty" ,db:"accrual"`
}
type UsingWithdrawStruct struct {
	IdOrder     int       `json:"id_order" ,db:"id_order"`
	Withdraw    float64   `json:"withdraw" ,db:"withdraw"`
	ProcessedAt time.Time `json:"processed_at,omitempty" ,db:"processed_at"`
}
type PostgresDB struct {
	queryInitUsers               string
	querySelectMaxIdUsers        string
	querySelectCountUsers        string
	querySelectIdByLogin         string
	querySelectCountByLogin      string
	queryInsertUser              string
	queryInitOrders              string
	queryInitWithdraws           string
	queryInitBalance             string
	querySelectCountOrdersById   string
	querySelectOrderByUserId     string
	querySelectWithdrawsByUserId string
	querySelectOrderInfoById     string
	querySelectCountByOrder      string
	queryInsertOrder             string
	querySelectBalance           string
	queryInsertWithdraw          string
	queryUpdateIncreaseBalance   string
	queryUpdateDecreaseBalance   string
	queryInsertUserBalance       string
	queryCheckPassword           string
	querySelectOrdersToProcess   string
	queryUpdateOrdersAccrual     string
}

var PostgresDBRun = PostgresDB{
	queryInitUsers: `CREATE TABLE IF NOT EXISTS users (
				  id_user           INT UNIQUE PRIMARY KEY,
				  login 	  TEXT UNIQUE NOT NULL,
				  password		   TEXT NOT NULL);`,
	//todo: creation date, is deleted
	queryInitBalance: `CREATE TABLE IF NOT EXISTS balance (
				  id_user           INT UNIQUE PRIMARY KEY,
				  	  accruals	double precision,
				  	  withdraws	double precision,
				  	  current	double precision);`,
	querySelectMaxIdUsers:   `SELECT MAX(id_user) FROM users;`,
	querySelectCountUsers:   `SELECT count(id_user) FROM users;`,
	querySelectIdByLogin:    `SELECT id_user FROM users WHERE login = $1;`,
	querySelectCountByLogin: `SELECT count(id_user) FROM users WHERE login = $1;`,
	queryInsertUser: `INSERT INTO users(
					id_user, login, password
					)
					VALUES($1, $2, $3);`,
	queryInsertUserBalance: `INSERT INTO balance(
					id_user, current, accruals, withdraws
					)
					VALUES($1, 0, 0, 0);`,
	queryInitOrders: `CREATE TABLE IF NOT EXISTS orders (
				  id_order           bigint UNIQUE PRIMARY KEY NOT NULL,
				  id_user           INT NOT NULL,
				  state 	  TEXT NOT NULL,
				  accrual	double precision ,
					uploaded_at TIMESTAMP );`,
	queryInitWithdraws: `CREATE TABLE IF NOT EXISTS withdraws (
				  id_order           bigint UNIQUE PRIMARY KEY NOT NULL,
				  id_user           INT NOT NULL,
					withdraw double precision,
					processed_at TIMESTAMP );`,
	querySelectOrderInfoById:     `SELECT id_order, id_user, state, accrual, uploaded_at FROM orders WHERE id_order = $1`,
	querySelectCountOrdersById:   `SELECT COUNT(id_order) FROM orders WHERE id_order = $1;`,
	querySelectOrderByUserId:     `SELECT id_order, id_user, state, accrual, uploaded_at FROM orders WHERE id_user = $1`,
	querySelectWithdrawsByUserId: `SELECT id_order, id_user, withdraw, processed_at FROM orders WHERE id_user = $1`,
	queryInsertOrder: `INSERT INTO orders(
					id_order, id_user, state, accrual, uploaded_at
					)
					VALUES($1, $2, $3, $4, $5);`,
	queryInsertWithdraw: `INSERT INTO orders(
					id_order, id_user, state, accrual, processed_at
					)
					VALUES($1, $2, $3, $4, $5);`,
	querySelectBalance: `SELECT current, accrual, withdrawn FROM balance WHERE id_user = $1;`,
	queryUpdateIncreaseBalance: `UPDATE balance set current = current + $2, accrual = accrual + $2 
					where id_user = $1;`,
	queryUpdateDecreaseBalance: `UPDATE balance set current = current - $2, withdrawn = withdrawn + $2 
					where id_user = $1;`,
	queryCheckPassword:         `SELECT password FROM users WHERE login = $1;`,
	querySelectOrdersToProcess: `SELECT id_order FROM orders WHERE state in ('NEW', 'REGISTERED', 'PROCESSING');`,
	queryUpdateOrdersAccrual:   `UPDATE orders SET state = $2 WHERE id_order = $1`,
}

func InitTables(config *config.Config) (err error) {
	log := zerolog.New(os.Stdout)
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		log.Warn().Msg(err.Error())
		return err
	} else {
		defer db.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_, err := db.ExecContext(ctx, PostgresDBRun.queryInitUsers)
		if err != nil {
			log.Warn().Msg(err.Error())
			return err
		}
		_, err = db.ExecContext(ctx, PostgresDBRun.queryInitOrders)
		if err != nil {
			log.Warn().Msg(err.Error())
			return err
		}
		_, err = db.ExecContext(ctx, PostgresDBRun.queryInitBalance)
		if err != nil {
			log.Warn().Msg(err.Error())
			return err
		}
		_, err = db.ExecContext(ctx, PostgresDBRun.queryInitWithdraws)
		if err != nil {
			log.Warn().Msg(err.Error())
			return err
		}
	}
	return
}

func InsertUser(config *config.Config, userAuthInfo *CredUserStruct) (userId int, err error) {
	var maxId int
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectMaxIdUsers).Scan(&maxId)
	if err != nil {
		var count int
		err2 := db.QueryRowContext(ctx, PostgresDBRun.querySelectCountUsers).Scan(&count)
		if err2 == nil {
			if count == 0 {
				maxId = 0
				err = nil
			}
		} else {
			return
		}
	}
	newID := maxId + 1
	txn, err := db.Begin()
	if err != nil {
		return userId, errors.Wrap(err, "could not start a new transaction")
	}
	defer txn.Rollback()
	_, err = txn.Exec(PostgresDBRun.queryInsertUser, newID, userAuthInfo.Login, userAuthInfo.Password)
	if err != nil {
		return userId, errors.Wrap(err, "failed to insert multiple records at once")
	}
	_, err = txn.Exec(PostgresDBRun.queryInsertUserBalance, newID)
	if err != nil {
		return userId, errors.Wrap(err, "failed to insert multiple records at once")
	}
	if err := txn.Commit(); err != nil {
		return userId, errors.Wrap(err, "failed to commit transaction")
	}
	userId = newID
	return

}

func CheckUserPass(config *config.Config, userAuthInfo *CredUserStruct) (result bool, err error) {
	var pass string
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = db.QueryRowContext(ctx, PostgresDBRun.queryCheckPassword, userAuthInfo.Login).Scan(&pass)
	if err != nil {
		fmt.Println("error while password checking")
		return
	}
	if userAuthInfo.Password == pass {
		result = true
		fmt.Println("passwords match")
		return
	}
	result = false
	fmt.Println("passwords don't match")
	return
}

func ReturnIdByLogin(config *config.Config, login *string) (userAuthInfo UsingUserStruct, err error) {
	userAuthInfo.Login = *login
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return userAuthInfo, err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	var countByLogin int
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectCountByLogin, login).Scan(&countByLogin)
	if err != nil || countByLogin == 0 {
		userAuthInfo.IdUser = 0
		return userAuthInfo, err
	}
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectIdByLogin, login).Scan(&userAuthInfo.IdUser)
	if err != nil {
		return userAuthInfo, err
	}
	return
}

func InsertOrder(config *config.Config, order *UsingOrderStruct) (err error) {
	var msg string
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	_, err = db.ExecContext(ctx, PostgresDBRun.queryInsertOrder, order.IdOrder, order.IdUser, order.State, 0, order.UploadedAt)
	if err != nil {
		return err
	}
	fmt.Println(msg)
	return
}

func NewWithdraw(config *config.Config, order *OrderToWithdrawStruct, userId *int) (isBalance bool, result bool, err error) {
	var msg string
	var userBalanceInfo UsingUserBalanceStruct
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	txn, err := db.Begin()
	if err != nil {
		fmt.Println("could not start a new transaction")
		return
	}
	defer txn.Rollback()
	err = txn.QueryRowContext(ctx, PostgresDBRun.querySelectBalance, userId).Scan(&userBalanceInfo)
	if err != nil {
		fmt.Println("failed to query balance")
		return
	}
	if userBalanceInfo.Current < order.Sum {
		isBalance = false
		result = true
		return
	}
	isBalance = true
	_, err = txn.ExecContext(ctx, PostgresDBRun.queryUpdateDecreaseBalance, userId, order.Sum)
	if err != nil {
		fmt.Println("failed to decrease balance")
		return
	}
	_, err = txn.ExecContext(ctx, PostgresDBRun.queryInsertWithdraw, order.IdOrder, userId, order.Sum, time.Now())
	if err != nil {
		fmt.Println("failed to insert withdraw")
		return
	}
	if err = txn.Commit(); err != nil {
		fmt.Println("failed to commit transaction")
		return
	}
	result = true
	fmt.Println(msg)
	return
}

func ReturnOrdersInfoByUserId(config *config.Config, userId *int) (isOrders bool, arrOrders []UsingOrderStruct, err error) {
	var msg string
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rows, err := db.QueryContext(ctx, PostgresDBRun.querySelectOrderByUserId, userId)
	if rows == nil {
		isOrders = false
		return
	} else {
		isOrders = true
	}
	defer rows.Close()
	fmt.Println(rows)
	for rows.Next() {
		var orderInfo UsingOrderStruct
		err = rows.Scan(&orderInfo.IdOrder, &orderInfo.IdUser, &orderInfo.State, &orderInfo.Accrual, &orderInfo.UploadedAt)
		if err != nil {
			return
		}
		arrOrders = append(arrOrders, orderInfo)
	}
	fmt.Println(arrOrders)
	if err != nil {
		return
	}
	fmt.Println(msg)
	return
}

func ReturnBalanceByUserId(config *config.Config, IdUser *int) (userBalanceInfo UsingUserBalanceStruct, err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return userBalanceInfo, err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectBalance, IdUser).Scan(&userBalanceInfo)
	if err != nil {
		return userBalanceInfo, err
	}
	return userBalanceInfo, err
}

func ReturnOrderInfoById(config *config.Config, orderId *int) (orderInfo UsingOrderStruct, err error) {
	var msg string
	var count int
	orderInfo.IdOrder = *orderId
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return orderInfo, err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectCountOrdersById).Scan(&count)
	if err != nil {
		return orderInfo, err
	}
	if count != 0 {
		msg = "order exists"
		err = db.QueryRowContext(ctx, PostgresDBRun.querySelectOrderInfoById, orderId).Scan(&orderInfo)
		if err != nil {
			return orderInfo, err
		}
		return orderInfo, err
	}
	fmt.Println(msg)
	return orderInfo, err
}

func ChangeBalanceByUserId(config *config.Config, IdUser *int, action string, sum *float64) (err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if action == "increase" {
		_, err = db.ExecContext(ctx, PostgresDBRun.queryUpdateIncreaseBalance, IdUser, sum)
		if err != nil {
			return
		}
	} else if action == "decrease" {
		_, err = db.ExecContext(ctx, PostgresDBRun.queryUpdateDecreaseBalance, IdUser, sum)
		if err != nil {
			return
		}
	}
	return
}
func ReturnWithdrawsInfoByUserId(config *config.Config, userId *int) (isWithdraws bool, arrWithdraws []UsingWithdrawStruct, err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rows, err := db.QueryContext(ctx, PostgresDBRun.querySelectWithdrawsByUserId, userId)
	if rows == nil {
		isWithdraws = false
		return
	} else {
		isWithdraws = true
	}
	defer rows.Close()
	fmt.Println(rows)
	for rows.Next() {
		var withdrawInfo UsingWithdrawStruct
		err = rows.Scan(&withdrawInfo.IdOrder, &withdrawInfo.Withdraw, &withdrawInfo.ProcessedAt)
		if err != nil {
			return
		}
		arrWithdraws = append(arrWithdraws, withdrawInfo)
	}
	fmt.Println(arrWithdraws)
	if err != nil {
		return
	}
	return
}
func ReturnOrdersToProcess(config *config.Config) (isOrders bool, arrOrders []int, err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rows, err := db.QueryContext(ctx, PostgresDBRun.querySelectOrdersToProcess)
	if rows == nil {
		isOrders = false
		return
	} else {
		isOrders = true
	}
	defer rows.Close()
	fmt.Println(rows)
	for rows.Next() {
		var orderNum int
		err = rows.Scan(&orderNum)
		if err != nil {
			return
		}
		arrOrders = append(arrOrders, orderNum)
	}
	fmt.Println(arrOrders)
	if err != nil {
		return
	}
	return
}
