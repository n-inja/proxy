package utils

import (
	"database/sql"
	"time"

	"errors"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func Open(userName, password, address, databaseName string) error {
	var err error
	db, err = sql.Open("mysql", userName+":"+password+"@"+address+"/"+databaseName)
	if err != nil {
		return err
	}
	db.SetMaxIdleConns(0)
	return initDB()
}

func Close() {
	db.Close()
}

func initDB() error {
	rows, err := db.Query("show tables like 'sessions'")
	if err != nil {
		return err
	}
	if !rows.Next() {
		return errors.New("session table not found")
	}
	rows.Close()
	return nil
}

func CheckSession(session string) (string, error) {
	now := time.Now()
	rows, err := db.Query("select id from sessions where session = ? and expiration_date > ?", session, now.Format("2006-01-02 15:04:05"))
	if err != nil {
		return "", err
	}
	defer rows.Close()
	if !rows.Next() {
		return "", nil
	}
	var ID string
	rows.Scan(&ID)
	return ID, nil
}
