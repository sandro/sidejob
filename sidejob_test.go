package sidejob

import (
	"encoding/gob"
	"fmt"
	"log"
	"testing"
	"time"
)

type TimeRecorder struct {
	BasicJob
	Name string
	Time time.Time
}

func (o TimeRecorder) Run() error {
	log.Println("JOB RUNNING")
	// panic("IH")
	_, err := db.Exec(`
	  create table if not exists time_records(name text, time timestamp);
    insert into time_records (name, time) values (?,?);
	`, o.Name, o.Time.UTC())
	return err
}

func init() {
	gob.Register(TimeRecorder{})
}

// func (o TimeRecorder) CanRetry(n int) bool {
// 	return false
// }

func removeTimeRecords() {
	db.Exec("drop table time_records;")
}

func TestEnqueue(t *testing.T) {
	defer removeTimeRecords()
	log.Println("testing a thing")
	// Config.DBConnectURI = ":memory:"
	// Config.DBConnectURI = "test.sqlite3"
	InitDB()
	job := TimeRecorder{Name: "test1", Time: time.Unix(1558062306, 0)}
	_, err := Enqueue(job)
	OrPanic(err)
	time.Sleep(time.Millisecond * 500)
	startJobs()
	time.Sleep(time.Millisecond * 500)
	row := db.QueryRowx("select * from time_records where name = ?", job.Name)
	result := make(map[string]interface{})
	log.Println(row)
	row.MapScan(result)
	fmt.Println(result)
	a := string(result["name"].([]byte)) == "test1"
	b := result["time"].(time.Time).Equal(job.Time)
	if !a || !b {
		t.Error("job not successful")
	}
}
