package storage

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
