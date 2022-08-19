package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/jackc/pgx/stdlib"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"

	"github.com/valentinaskakun/gophermart/internal/config"
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
type PostgresDB struct {
	queryInitUsers               string
	querySelectMaxIDUsers        string
	querySelectCountUsers        string
	querySelectIDByLogin         string
	querySelectCountByLogin      string
	queryInsertUser              string
	queryInitOrders              string
	queryInitWithdraws           string
	queryInitBalance             string
	querySelectCountOrdersByID   string
	querySelectOrderByUserID     string
	querySelectWithdrawsByUserID string
	querySelectOrderInfoByID     string
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
				  	  withdrawn	double precision,
				  	  current	double precision);`,
	querySelectMaxIDUsers:   `SELECT MAX(id_user) FROM users;`,
	querySelectCountUsers:   `SELECT count(id_user) FROM users;`,
	querySelectIDByLogin:    `SELECT id_user FROM users WHERE login = $1;`,
	querySelectCountByLogin: `SELECT count(id_user) FROM users WHERE login = $1;`,
	queryInsertUser: `INSERT INTO users(
					id_user, login, password
					)
					VALUES($1, $2, $3);`,
	queryInsertUserBalance: `INSERT INTO balance(
					id_user, current, accruals, withdrawn
					)
					VALUES($1, 0, 0, 0);`,
	queryInitOrders: `CREATE TABLE IF NOT EXISTS orders (
				  id_order           bigint UNIQUE PRIMARY KEY NOT NULL,
				  id_user           INT NOT NULL,
				  state 	  TEXT,
				  accrual	double precision ,
					uploaded_at TIMESTAMP );`,
	queryInitWithdraws: `CREATE TABLE IF NOT EXISTS withdraws (
				  id_order           bigint UNIQUE PRIMARY KEY NOT NULL,
				  id_user           INT NOT NULL,
					withdraw double precision,
					processed_at TIMESTAMP );`,
	querySelectOrderInfoByID:     `SELECT id_order, id_user, state, accrual, uploaded_at FROM orders WHERE id_order = $1 ORDER BY uploaded_at ASC;`,
	querySelectCountOrdersByID:   `SELECT COUNT(id_order) FROM orders WHERE id_order = $1;`,
	querySelectOrderByUserID:     `SELECT id_order, state, accrual, uploaded_at FROM orders WHERE id_user = $1;`,
	querySelectWithdrawsByUserID: `SELECT id_order, withdraw, processed_at FROM withdraws WHERE id_user = $1 ORDER BY processed_at ASC;`,
	queryInsertOrder: `INSERT INTO orders(
					id_order, id_user, state, accrual, uploaded_at
					)
					VALUES($1, $2, $3, $4, $5);`,
	queryInsertWithdraw: `INSERT INTO withdraws(
					id_order, id_user, withdraw, processed_at
					)
					VALUES($1, $2, $3, $4);`,
	querySelectBalance: `SELECT current, accruals, withdrawn FROM balance WHERE id_user = $1;`,
	queryUpdateIncreaseBalance: `UPDATE balance set current = current + $2, accruals = accruals + $2 
					where id_user = $1;`,
	queryUpdateDecreaseBalance: `UPDATE balance set current = current - $2, withdrawn = withdrawn + $2 
					where id_user = $1;`,
	queryCheckPassword:         `SELECT password FROM users WHERE login = $1;`,
	querySelectOrdersToProcess: `SELECT id_order FROM orders WHERE state in ('NEW', 'REGISTERED', 'PROCESSING');`,
	queryUpdateOrdersAccrual:   `UPDATE orders SET state = $2 WHERE id_order = $1;`,
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

func InsertUser(config *config.Config, userAuthInfo *CredUserStruct) (userID int, err error) {
	var maxID int
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectMaxIDUsers).Scan(&maxID)
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
		return userID, errors.Wrap(err, "could not start a new transaction")
	}
	defer txn.Rollback()
	_, err = txn.Exec(PostgresDBRun.queryInsertUser, newID, userAuthInfo.Login, userAuthInfo.Password)
	if err != nil {
		return userID, errors.Wrap(err, "failed to insert multiple records at once")
	}
	_, err = txn.Exec(PostgresDBRun.queryInsertUserBalance, newID)
	if err != nil {
		return userID, errors.Wrap(err, "failed to insert multiple records at once")
	}
	if err := txn.Commit(); err != nil {
		return userID, errors.Wrap(err, "failed to commit transaction")
	}
	userID = newID
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

func ReturnIDByLogin(config *config.Config, login *string) (userAuthInfo UsingUserStruct, err error) {
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
		userAuthInfo.IDUser = 0
		return userAuthInfo, err
	}
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectIDByLogin, login).Scan(&userAuthInfo.IDUser)
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
	_, err = db.ExecContext(ctx, PostgresDBRun.queryInsertOrder, order.IDOrder, order.IDUser, order.State, 0, order.UploadedAt)
	if err != nil {
		return err
	}
	fmt.Println(msg)
	return
}

func NewWithdraw(config *config.Config, order *OrderToWithdrawStruct, userID *int) (isBalance bool, result bool, err error) {
	var msg string
	var userBalanceInfo UsingUserBalanceStruct
	orderParsed, err := strconv.Atoi(order.IDOrder)
	if err != nil {
		return
	}
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
	err = txn.QueryRowContext(ctx, PostgresDBRun.querySelectBalance, userID).Scan(&userBalanceInfo.Current, &userBalanceInfo.Accrual, &userBalanceInfo.Withdrawn)
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
	_, err = txn.ExecContext(ctx, PostgresDBRun.queryUpdateDecreaseBalance, userID, order.Sum)
	if err != nil {
		fmt.Println("failed to decrease balance")
		return
	}
	_, err = txn.ExecContext(ctx, PostgresDBRun.queryInsertWithdraw, orderParsed, userID, order.Sum, time.Now())
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

func ReturnOrdersInfoByUserID(config *config.Config, userID int) (isOrders bool, arrOrders []UsingOrderStruct, err error) {
	var orderInfo UsingOrderStruct
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rows, err := db.QueryContext(ctx, PostgresDBRun.querySelectOrderByUserID, userID)
	if err != nil || rows.Err() != nil {
		return
	}
	defer rows.Close()
	if err != nil {
		fmt.Println("querySelectOrderByUserID.Scan.orderInfo.Rows", err)
		return
	}
	for rows.Next() {
		err = rows.Scan(&orderInfo.Number, &orderInfo.State, &orderInfo.Accrual, &orderInfo.UploadedAt)
		if err != nil {
			fmt.Println("querySelectOrderByUserID.Scan.orderInfo.Rows", err)
			return
		}
		arrOrders = append(arrOrders, orderInfo)
	}
	if err != nil {
		return
	}
	isOrders = true
	fmt.Println("querySelectOrderByUserID", rows)
	return
}

func ReturnBalanceByUserID(config *config.Config, IDUser *int) (userBalanceInfo UsingUserBalanceStruct, err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return userBalanceInfo, err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectBalance, IDUser).Scan(&userBalanceInfo.Current, &userBalanceInfo.Accrual, &userBalanceInfo.Withdrawn)
	if err != nil {
		return userBalanceInfo, err
	}
	return userBalanceInfo, err
}

func ReturnOrderInfoByID(config *config.Config, orderID *int) (orderInfo UsingOrderStruct, err error) {
	var msg string
	var count int
	orderInfo.IDOrder = *orderID
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return orderInfo, err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	err = db.QueryRowContext(ctx, PostgresDBRun.querySelectCountOrdersByID, orderID).Scan(&count)
	if err != nil {
		return orderInfo, err
	}
	if count != 0 {
		fmt.Println("order exists")
		err = db.QueryRowContext(ctx, PostgresDBRun.querySelectOrderInfoByID, orderID).Scan(&orderInfo.IDOrder, &orderInfo.IDUser, &orderInfo.State, &orderInfo.Accrual, &orderInfo.UploadedAt)
		if err != nil {
			return orderInfo, err
		}
		return orderInfo, err

	}
	fmt.Println(msg)
	return orderInfo, err
}

func ChangeBalanceByUserID(config *config.Config, IDUser *int, action string, sum *float64) (err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	if action == "increase" {
		_, err = db.ExecContext(ctx, PostgresDBRun.queryUpdateIncreaseBalance, IDUser, sum)
		if err != nil {
			return
		}
	} else if action == "decrease" {
		_, err = db.ExecContext(ctx, PostgresDBRun.queryUpdateDecreaseBalance, IDUser, sum)
		if err != nil {
			return
		}
	}
	return
}
func ReturnWithdrawsInfoByUserID(config *config.Config, userID *int) (isWithdraws bool, arrWithdraws []UsingWithdrawStruct, err error) {
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rows, err := db.QueryContext(ctx, PostgresDBRun.querySelectWithdrawsByUserID, userID)
	if err != nil || rows.Err() != nil {
		return
	}
	defer rows.Close()
	fmt.Println(rows)
	for rows.Next() {
		var withdrawInfo UsingWithdrawStruct
		err = rows.Scan(&withdrawInfo.IDOrder, &withdrawInfo.Withdraw, &withdrawInfo.ProcessedAt)
		if err != nil {
			return
		}
		arrWithdraws = append(arrWithdraws, withdrawInfo)
	}
	fmt.Println(arrWithdraws)
	isWithdraws = true
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
	if err != nil || rows.Err() != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var orderNum int
		err = rows.Scan(&orderNum)
		if err != nil {
			return
		}
		arrOrders = append(arrOrders, orderNum)
		fmt.Println("arrOrdersToAcc", arrOrders)
	}
	if err != nil {
		return
	}
	isOrders = true
	return
}
