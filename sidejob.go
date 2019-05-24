package sidejob

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"fmt"
	"log"
	"reflect"
	"time"

	"github.com/jmoiron/sqlx"
)

var registeredStructs = map[string]struct{}{}

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

type BasicJob struct {
	SidejobID int64
}

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

func registerForEncoding(runnable Runnable) {
	rt := reflect.TypeOf(runnable)
	name := rt.Name()
	_, ok := registeredStructs[name]
	log.Println("name is", name, ok)
	if !ok {
		gob.Register(runnable)
		registeredStructs[name] = struct{}{}
		log.Println("gob registered", registeredStructs)
	}
}

func Enqueue(runnable Runnable) (int64, error) {
	return EnqueueAt(runnable, time.Now())
}

func EnqueueAt(runnable Runnable, at time.Time) (ID int64, err error) {
	// registerForEncoding(runnable)
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
	ID, err = res.LastInsertId()
	return
}

func init() {
	log.Println("Sidejob INIT")
	Config = Configuration{
		DBConnectURI: "sidejob.sqlite3",
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
		var ids []int64
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
		go func() {
			startJobs()
		}()
		time.Sleep(Config.PollDuration)
	}
}

func OrPanic(err error) {
	if err != nil {
		log.Panic(err)
	}
}
