package sidecar

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

type Configuration struct {
	DBConnectURI string
	PollDuration time.Duration
}

var Config Configuration

type Runnable interface {
	Run() error
	CanRetry(int) bool
	RetryAt(int) time.Time
}

type BasicJob struct {
}

func (o BasicJob) Run() error {
	return nil
}

func (o BasicJob) CanRetry(n int) bool {
	if n < 5 {
		return true
	}
	return false
}

func (o BasicJob) RetryAt(n int) time.Time {
	at := time.Now().Add(time.Duration(n) * time.Minute)
	return at
}

func Enqueue(runnable Runnable) error {
	return EnqueueAt(runnable, time.Now())
}

func EnqueueAt(runnable Runnable, at time.Time) (err error) {
	at = at.UTC()
	var payload bytes.Buffer
	enc := gob.NewEncoder(&payload)
	err = enc.Encode(&runnable)
	if err != nil {
		return
	}
	_, err = db.Exec("insert or ignore into jobs (payload, run_at) values(?, ?);", payload.Bytes(), DBTime(at))
	return
}

func init() {
	Config = Configuration{
		DBConnectURI: "sidecar.sqlite3",
		PollDuration: time.Second,
	}
}

func reserveJobs() (jobs []Job, err error) {
	err = db.Select(&jobs, "select * from jobs where run_at <= ? and processing=0", time.Now().UTC().String())
	if err != nil && err != sql.ErrNoRows {
		return
	}
	if len(jobs) > 0 {
		var ids []int
		for _, j := range jobs {
			ids = append(ids, j.ID)
		}
		query, args, err := sqlx.In("update jobs set processing=1 where id in (?)", ids)
		OrPanic(err)
		_, err = db.Exec(query, args...)
	}
	return jobs, err
}

func startJobs() {
	jobs, err := reserveJobs()
	OrPanic(err)

	for _, j := range jobs {
		log.Println("have a job", j)
		go func(job Job) {
			job.Start()
		}(j)
	}
}

func Start() {
	for {
		<-time.After(Config.PollDuration)
		go func() {
			startJobs()
		}()
	}
}
