package storage

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/stdlib"
	"github.com/rs/zerolog"

	"gophermart/internal/config"
)

type RegisterUserStruct struct {
	Login    string `json:"login" ,db:"login"`
	Password string `json:"password" ,db:"password"`
}
type UsingUserStruct struct {
	IdUser int    `json:"id_user" ,db:"id_user"`
	Login  string `json:"login" ,db:"login"`
}
type UsingUserBalanceStruct struct {
	Current   float64 `json:"current" ,db:"current"`
	Withdrawn float64 `json:"withdrawn" ,db:"withdrawn"`
}
type OrderToWithdrawStruct struct {
	IdOrder  int     `json:"id_order" ,db:"id_order"`
	Withdraw float64 `json:"withdraw" ,db:"withdraw"`
}
type UsingOrderStruct struct {
	IdOrder    int       `json:"id_order" ,db:"id_order"`
	IdUser     int       `json:"id_user,omitempty" ,db:"id_user"`
	State      string    `json:"state,omitempty" ,db:"state"`
	Accrual    float64   `json:"accrual,omitempty" ,db:"accrual"`
	Withdraw   float64   `json:"withdraw,omitempty" ,db:"withdraw"`
	UploadedAt time.Time `json:"uploaded_at,omitempty" ,db:"uploaded_at"`
}
type PostgresDB struct {
	queryInitUsers             string
	querySelectMaxIdUsers      string
	querySelectCountUsers      string
	querySelectIdByLogin       string
	querySelectCountByLogin    string
	queryInsertUser            string
	queryInitOrders            string
	querySelectCountOrdersById string
	querySelectOrderByUserId   string
	querySelectOrderStateById  string
	querySelectCountByOrder    string
	queryInsertOrder           string
	querySelectBalance         string
}

var PostgresDBRun = PostgresDB{
	queryInitUsers: `CREATE TABLE IF NOT EXISTS users (
				  id_user           INT UNIQUE PRIMARY KEY,
				  login 	  TEXT UNIQUE NOT NULL,
				  password		   TEXT NOT NULL);`,
	//todo: creation date, is deleted
	querySelectMaxIdUsers:   `SELECT MAX(id_user) FROM users;`,
	querySelectCountUsers:   `SELECT count(id_user) FROM users;`,
	querySelectIdByLogin:    `SELECT id_user FROM users WHERE login = $1;`,
	querySelectCountByLogin: `SELECT count(id_user) FROM users WHERE login = $1;`,
	queryInsertUser: `INSERT INTO users(
					id_user, login, password
					)
					VALUES($1, $2, $3);`,
	queryInitOrders: `CREATE TABLE IF NOT EXISTS orders (
				  id_order           bigint UNIQUE PRIMARY KEY NOT NULL,
				  id_user           INT NOT NULL,
				  state 	  TEXT NOT NULL,
				  accrual	double precision ,
					withdraw double precision,
					uploaded_at TIMESTAMP );`,
	querySelectCountOrdersById: `SELECT COUNT(id_order) FROM orders WHERE id_order = $1;`,
	querySelectOrderByUserId:   `SELECT id_order, id_user, state, accrual, withdraw, uploaded_at FROM orders WHERE id_user = $1`,
	queryInsertOrder: `INSERT INTO orders(
					id_order, id_user, state, accrual, uploaded_at
					)
					VALUES($1, $2, $3, $4, $5);`,
	querySelectBalance: `SELECT SUM(accrual)-SUM(withdrawn), SUM(withdrawn) FROM orders WHERE id_user = $1;`,
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
	}
	return
}

func InsertUser(config *config.Config, registerUser *RegisterUserStruct) (err error) {
	var maxId int
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return err
	} else {
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
				return err
			}
		}
		_, err = db.ExecContext(ctx, PostgresDBRun.queryInsertUser, maxId+1, registerUser.Login, registerUser.Password)
		if err != nil {
			return err
		}
	}
	return
}
func ReturnIdByLogin(config *config.Config, login *string) (userAuthInfo UsingUserStruct, err error) {
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
	_, err = db.ExecContext(ctx, PostgresDBRun.queryInsertOrder, order.IdOrder, order.IdUser, order.State, order.Accrual, order.UploadedAt)
	if err != nil {
		return err
	}
	fmt.Println(msg)
	return
}

func ReturnOrdersInfoByUserId(config *config.Config, userId *int) (arrOrders []UsingOrderStruct, err error) {
	var msg string
	var count int
	db, err := sql.Open("pgx", config.Database)
	if err != nil {
		return arrOrders, err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	rows, err := db.QueryContext(ctx, PostgresDBRun.querySelectOrderByUserId, userId)
	defer rows.Close()
	fmt.Println(rows)
	for rows.Next() {
		var orderInfo UsingOrderStruct
		err := rows.Scan(&orderInfo.IdOrder, &orderInfo.IdUser, &orderInfo.State, &orderInfo.Accrual, &orderInfo.UploadedAt)
		if err != nil {
			return arrOrders, err
		}
		arrOrders = append(arrOrders, orderInfo)
	}
	fmt.Println(arrOrders)
	if err != nil {
		return arrOrders, err
	}
	if count != 0 {
		msg = "order exists"
		return arrOrders, err
	}
	fmt.Println(msg)
	return arrOrders, err
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
		return orderInfo, err
	}
	fmt.Println(msg)
	return orderInfo, err
}
