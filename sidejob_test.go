package sidejob

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"
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

type failJob struct{ BasicJob }

func (o failJob) Run() (err error) {
	err = errors.New("failure")
	return
}

func init() {
	gob.Register(TimeRecorder{})
	gob.Register(failJob{})
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

func TestEnqueueFailure(t *testing.T) {
	c := qt.New(t)
	InitDB()
	job := failJob{}
	_, err := Enqueue(job)
	time.Sleep(time.Millisecond * 500)
	startJobs()
	time.Sleep(time.Millisecond * 500)
	failures, err := GetFailedJobs()
	OrPanic(err)
	failure := failures[0]
	c.Assert(failure.Name, qt.Equals, "sidejob.failJob")
}
