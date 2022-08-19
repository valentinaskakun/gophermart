package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"

	"github.com/zamibaru89/gophermart/internal/config"
)

type PostgresStorage struct {
	Connection *pgx.Conn
}

func NewPostgresStorage(c config.ServerConfig) (Repo, *pgx.Conn, error) {
	conn, err := pgx.Connect(context.Background(), c.DSN)
	if err != nil {
		log.Println(err)
	}

	query := `CREATE TABLE IF NOT EXISTS  users(
    id serial PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    password VARCHAR(50)
);
CREATE TABLE IF NOT EXISTS orders (
				  orderID           VARCHAR(64) UNIQUE,
				  userID           INT NOT NULL,
				  state 	  TEXT NOT NULL,
				  accrual	double precision ,
					withdraw double precision,
					uploaded_at TIMESTAMP );

CREATE TABLE IF NOT EXISTS balance (
				userID           INT UNIQUE NOT NULL,
				balance double precision);

CREATE TABLE IF NOT EXISTS withdrawals  (
				userID				INT NOT NULL,
  				orderID				VARCHAR(64) UNIQUE,
				sum					double precision,
    			processed_at		TIMESTAMP);

`
	_, err = conn.Exec(context.Background(), query)
	if err != nil {
		log.Println(err)
		return &PostgresStorage{Connection: conn}, conn, err
	}

	return &PostgresStorage{Connection: conn}, conn, nil
}

func (p *PostgresStorage) RegisterUser(credentials Credentials) error {

	query := `
		INSERT INTO users(
		username,
		password
		)
		VALUES($1, $2);
	`
	queryReturnID := `
		SELECT id FROM users where username = $1;
	`
	_, err := p.Connection.Exec(context.Background(), query, credentials.Username, credentials.Password)
	if err != nil {
		log.Println(err)
		return err
	}
	err = p.Connection.QueryRow(context.Background(), queryReturnID, credentials.Username).Scan(&credentials.ID)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (p *PostgresStorage) SelectUser(user Credentials) (Credentials, error) {

	query := `
		SELECT ID FROM users
		WHERE username like $1;
	`
	_, err := p.Connection.Exec(context.Background(), query, user.Username)
	if err != nil {
		log.Println(err)
		return user, err

	}
	return user, nil
}
func (p *PostgresStorage) SignIn(user Credentials) (Credentials, error) {
	var id int
	query := `
		SELECT id FROM users WHERE username=$1 and password=$2;
	`
	result, err := p.Connection.Query(context.Background(), query, user.Username, user.Password)

	if err != nil {
		return user, err
	}
	defer result.Close()
	for result.Next() {
		err = result.Scan(&id)

	}
	user.ID = id
	if err != nil {
		return user, errors.New("not auth")
	} else {
		return user, nil
	}
}
func (p *PostgresStorage) PostOrder(order Order) error {

	query := `INSERT INTO orders(
					orderID, userID, state, accrual, uploaded_at
					)
					VALUES($1, $2, $3, 0, $5)
ON CONFLICT (orderID) DO 
		    UPDATE SET 	state=$3,
		            	accrual=$4
		            	;`
	res, err := p.Connection.Exec(context.Background(), query, order.OrderID, order.UserID, order.State, order.Accrual, order.UploadedAt)
	if err != nil {
		log.Println(err)
		return err

	}
	log.Println(res)
	return nil
}

func (p *PostgresStorage) GetOrderByOrderID(id string) (Order, error) {
	var order Order
	order.OrderID = id
	query := `
		SELECT orderID, userID, state, accrual, uploaded_at FROM orders  WHERE orderID=$1;
	`
	result, err := p.Connection.Query(context.Background(), query, id)

	if err != nil {
		return order, err
	}
	defer result.Close()
	fmt.Println("result rows =", result)
	for result.Next() {
		err = result.Scan(&order.OrderID, &order.UserID, &order.State, &order.Accrual, &order.UploadedAt)

	}

	if err != nil {
		return order, err
	} else {
		return order, nil
	}
}
func (p *PostgresStorage) GetOrdersByUserID(userid int64) ([]Order, error) {
	var orders []Order

	query := `
		SELECT orderID, userID, state, accrual, uploaded_at FROM orders 
		WHERE userID=$1 ORDER BY uploaded_at ASC;
	`
	result, err := p.Connection.Query(context.Background(), query, userid)
	if err != nil {
		return orders, err
	}
	defer result.Close()
	for result.Next() {
		var order Order

		err = result.Scan(&order.OrderID, &order.UserID, &order.State, &order.Accrual, &order.UploadedAt)

		orders = append(orders, order)
	}
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return orders, nil
}

func (p *PostgresStorage) GetOrdersForUpdate() ([]Order, error) {
	var orders []Order

	query := `
		SELECT orderID, userID, state, accrual, uploaded_at FROM orders 
		WHERE state in ('NEW', 'REGISTERED', 'PROCESSING');
	`
	result, err := p.Connection.Query(context.Background(), query)
	if err != nil {
		return orders, err
	}
	defer result.Close()
	for result.Next() {
		var order Order
		err = result.Scan(&order.OrderID, &order.UserID, &order.State, &order.Accrual, &order.UploadedAt)

		orders = append(orders, order)
	}
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return orders, nil
}

func (p *PostgresStorage) GetBalanceByUserID(userid int64) (float64, error) {

	query := `
		SELECT balance FROM balance
		WHERE userID=$1;
	`
	result, err := p.Connection.Query(context.Background(), query, userid)
	if err != nil {
		return 0, err
	}
	defer result.Close()
	var current float64
	for result.Next() {

		err = result.Scan(&current)

	}
	if err != nil {
		log.Println(err)
		return 0, err
	}

	return current, nil
}

func (p *PostgresStorage) GetWithdrawalHistoryForUser(userid int64) (float64, error) {

	query := `
		SELECT sum(sum)  FROM withdrawals
		WHERE userID=$1;
	`
	result, err := p.Connection.Query(context.Background(), query, userid)
	if err != nil {
		return 0, err
	}
	defer result.Close()
	var withdrawn float64
	for result.Next() {

		err = result.Scan(&withdrawn)

	}
	if err != nil {
		log.Println(err)
		return 0, err
	}

	return withdrawn, nil
}

func (p *PostgresStorage) SetBalanceByUserID(userid int64, current float64) error {

	query := `INSERT INTO balance(
					userid, balance
					)
					VALUES($1, $2)
ON CONFLICT (userid) DO UPDATE
		    SET balance=$2;`
	_, err := p.Connection.Exec(context.Background(), query, userid, current)
	if err != nil {
		log.Println(err)
		return err

	}
	return nil
}

func (p *PostgresStorage) PostWithdrawal(withdrawal Withdrawal) error {

	query := `INSERT INTO withdrawals(
					orderID, userID, sum, processed_at
					)
					VALUES($1, $2, $3, $4);`
	_, err := p.Connection.Exec(context.Background(), query, withdrawal.OrderID, withdrawal.UserID, withdrawal.Sum, withdrawal.ProcessedAt)
	if err != nil {
		log.Println(err)
		return err

	}
	return nil
}

func (p *PostgresStorage) GetWithdrawals(userid int64) ([]Withdrawal, error) {
	var withdrawals []Withdrawal

	query := `
		SELECT orderID, userID, sum, processed_at FROM withdrawals 
		WHERE userID=$1 ORDER BY processed_at ASC;
	`
	result, err := p.Connection.Query(context.Background(), query, userid)
	if err != nil {
		return withdrawals, err
	}
	defer result.Close()
	for result.Next() {
		var withdrawal Withdrawal
		err = result.Scan(&withdrawal.OrderID, &withdrawal.UserID, &withdrawal.Sum, &withdrawal.ProcessedAt)

		withdrawals = append(withdrawals, withdrawal)
	}
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return withdrawals, nil
}
