package module

import (
	"context"
	"database/sql"
	"myblog"
)

func InitPDB() *sql.DB {
	db, err := sql.Open("postgres", myblog.Config["psql"])
	if err != nil {
		panic(err)
	}
	return db
}

type UserInfo struct {
	name      string
	password  string
	timestamp int
	login     bool
}

func User(user string, db *sql.DB) *UserInfo {
	statement, err := db.Prepare("SELECT * FROM users WHERE name=$1")
	if err != nil {
		panic(err)
	}
	resultSet, err := statement.Query(context.Background(), user)
	if err != nil {
		panic(err)
	}
	defer statement.Close()
	defer resultSet.Close()
	if resultSet.Next() {
		userInfo := &UserInfo{}
		resultSet.Scan(&userInfo.name, &userInfo.password, &userInfo.timestamp, &userInfo.login)
		return userInfo
	} else {
		return &UserInfo{name: ""}
	}
}

func CreateUser(userInfo *UserInfo, db *sql.DB) {
	_, err := db.Exec("INSERT INTO users(name, password, timestamp) VALUES($1, $2, $3, $4)", userInfo.name, userInfo.password, userInfo.timestamp, userInfo.login)
	if err != nil {
		panic(err)
	}
}

func DeleteUser(userInfo *UserInfo, db *sql.DB) {
	_, err := db.Exec("DELETE FROM users WHERE name=$1 AND password=$2", userInfo.name, userInfo.password)
	if err != nil {
		panic(err)
	}
}

func UpdateUser(oldInfo *UserInfo, newInfo *UserInfo, db *sql.DB) {
	_, err := db.Exec("UPDATE users SET name=$1, password=$2, timestamp=$3 WHERE user=$4 AND password=$5", newInfo.name, newInfo.password, newInfo.timestamp, oldInfo.name, oldInfo.password)
	if err != nil {
		panic(err)
	}
}
