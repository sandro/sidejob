package sidecar

import (
	"encoding/gob"
	"log"
	"testing"
	"time"
)

type TimeRecorder struct {
	BasicJob
	Name string
}

func (o TimeRecorder) Run() error {
	log.Println("JOB RUNNING")
	// panic("IH")
	_, err := db.Exec(`
	  create table if not exists time_records(name text, time timestamp);
    insert into time_records (name, time) values (?,?);
	`, o.Name, time.Now().UTC())
	return err
}

// func (o TimeRecorder) CanRetry(n int) bool {
// 	return false
// }

func removeTimeRecords() {
	db.Exec("drop table time_records;")
}

func TestEnqueue(t *testing.T) {
	// defer removeTimeRecords()
	log.Println("testing a thing")
	gob.Register(TimeRecorder{})
	Config.DBConnectURI = ":memory:"
	// Config.DBConnectURI = "test.sqlite3"
	InitDB()
	job := TimeRecorder{Name: "test1"}
	err := Enqueue(job)
	OrPanic(err)
	time.Sleep(time.Millisecond * 500)
	startJobs()
	time.Sleep(time.Millisecond * 500)
	row := db.QueryRowx("select * from time_records where name = ?", job.Name)
	result := make(map[string]interface{})
	log.Println(row)
	row.MapScan(result)
	if result["name"] != "test1" || result["time"] == nil {
		t.Error("job not successful")
	}
}
