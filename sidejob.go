package sidejob

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

type Configuration struct {
	DBConnectURI string
	PollDuration time.Duration
	MaxRetries   int
}

var Config Configuration

type Runnable interface {
	Run() error
	CanRetry(int) bool
	RetryAt(int) time.Time
}

type BasicJob struct{}

func (o BasicJob) Run() error {
	return nil
}

func (o BasicJob) CanRetry(n int) bool {
	if n < Config.MaxRetries {
		return true
	}
	return false
}

func (o BasicJob) RetryAt(n int) time.Time {
	at := time.Now().Add(time.Duration(n) * time.Minute)
	return at
}

func Enqueue(runnable Runnable) (int, error) {
	return EnqueueAt(runnable, time.Now())
}

func EnqueueAt(runnable Runnable, at time.Time) (ID int, err error) {
	at = at.UTC()
	var payload bytes.Buffer
	enc := gob.NewEncoder(&payload)
	err = enc.Encode(&runnable)
	if err != nil {
		return
	}
	res, err := db.Exec("insert or ignore into jobs (name, payload, run_at) values(?, ?, ?);", fmt.Sprintf("%T", runnable), payload.Bytes(), DBTime(at))
	if err != nil {
		return
	}
	id, err := res.LastInsertId()
	ID = int(id)
	return
}

func init() {
	log.Println("Sidecar INIT")
	Config = Configuration{
		DBConnectURI: "sidecar.sqlite3",
		PollDuration: time.Second,
		MaxRetries:   5,
	}
	gob.Register(BasicJob{})
}

func reserveJobs() (jobs []JobRunner, err error) {
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
		go func(job JobRunner) {
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

func OrPanic(err error) {
	if err != nil {
		log.Panic(err)
	}
}
