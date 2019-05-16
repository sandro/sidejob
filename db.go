package sidecar

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func OrPanic(err error) {
	if err != nil {
		log.Panic(err)
	}
}

var db *sqlx.DB

func ConnectDB() {
	log.Println("connecting to database", Config.DBConnectURI)
	db = sqlx.MustConnect("sqlite3", Config.DBConnectURI)
}

func Disconnect() error {
	return db.Close()
}

func InitDB() {
	if db == nil {
		ConnectDB()
	}
	db.MustExec(structure)
}

func ClearDB() {
	tx, err := db.Begin()
	OrPanic(err)
	_, err = tx.Exec("delete from jobs; delete from completed_jobs; delete from job_stats;")
	OrPanic(err)
	err = tx.Commit()
	OrPanic(err)
}

func DBTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.000Z")
}

func makeResultString(result sql.Result) string {
	resultStr := "lastInsertId: %d and rowsAfected: %d"
	id, _ := result.LastInsertId()
	n, _ := result.RowsAffected()
	return fmt.Sprintf(resultStr, id, n)
}

var structure string = `
	PRAGMA foreign_keys=ON;
		-- (cast(CURRENT_TIMESTAMP as text)),
  create table if not exists jobs (
	  id integer primary key,
		payload blob not null,
		created_at datetime not null default CURRENT_TIMESTAMP,
		run_at datetime not null,
		processing integer not null default 0 check (processing in (0,1)),
		failure_count integer not null default 0,
		unique(payload, run_at)
	);

  create table if not exists completed_jobs (
	  id integer primary key,
		name text not null,
		payload blob not null,
		failure_count integer not null,
		job_id integer not null,
		created_at datetime not null default CURRENT_TIMESTAMP
	);

	create table if not exists failed_jobs (
	  id integer primary key,
		job_id integer not null,
		created_at datetime not null default CURRENT_TIMESTAMP,
		name text,
		message text,
		terminal integer not null default 0 check (terminal in (0,1))
	);

	create table if not exists job_stats (
	  id integer primary key,
		succeeded integer,
		failed integer
	);
`
