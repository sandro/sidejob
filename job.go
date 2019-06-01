package sidejob

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"time"
)

type JobRunner struct {
	ID           int64
	Name         string
	Payload      []byte
	RunAt        time.Time `db:"run_at"`
	FinishedAt   NullTime  `db:"finished_at"`
	FailureCount int       `db:"failure_count"`
	CreatedAt    time.Time `db:"created_at"`
	Processing   bool
	Message      string
	Terminal     bool
	Trace        string
	runnable     Runnable
}

func (o JobRunner) HandleError(jobError error) {
	log.Println("Handling error", jobError)
	o.FailureCount++
	tx, err := db.Begin()
	OrPanic(err)
	if o.runnable.CanRetry(o.FailureCount) {
		log.Println("Can retry", o.ID, o.Name, jobError.Error())
		o.RunAt = o.runnable.RetryAt(o.FailureCount).UTC()
		tx.Exec("update jobs set failure_count=?, run_at=?, processing=0 where id=?", o.FailureCount, DBTime(o.RunAt), o.ID)
		tx.Exec("insert into failed_jobs (job_id, name, message, trace) values(?,?,?,?)", o.ID, o.Name, jobError.Error(), string(debug.Stack()))
	} else {
		log.Println("Cannot retry")
		tx.Exec("update jobs set failure_count=? where id=?", o.FailureCount, o.ID)
		tx.Exec("insert into failed_jobs (job_id, name, message, terminal) values(?,?,?,1)", o.ID, o.Name, jobError.Error())
	}
	err = tx.Commit()
	OrPanic(err)
}

func (o JobRunner) HandleSuccess() {
	tx, err := db.Begin()
	OrPanic(err)
	tx.Exec("delete from jobs where id=?", o.ID)
	tx.Exec("insert into completed_jobs (id, name, payload, failure_count, job_id) values(?,?,?,?,?)", o.ID, o.Name, o.Payload, o.FailureCount, o.ID)
	err = tx.Commit()
	OrPanic(err)
}

func (o JobRunner) Start() (err error) {
	o.Name = fmt.Sprintf("%s", o.Payload)
	log.Println("started job", o.Name)
	defer func() {
		r := recover()
		if r != nil {
			log.Println("recovered", r)
			o.HandleError(errors.New(fmt.Sprintf("Panic recovery: %#v", r)))
		}
	}()
	buf := bytes.NewBuffer(o.Payload)
	dec := gob.NewDecoder(buf)
	err = dec.Decode(&o.runnable)
	if err != nil {
		log.Println("decode error", err)
		o.HandleError(err)
		return
	}
	o.Name = fmt.Sprintf("%T", o.runnable)
	log.Println("real name", o.Name)
	err = o.runnable.Run()
	if err != nil {
		o.HandleError(err)
		return
	}
	o.HandleSuccess()
	return
}

type GetJobsOption struct {
	Cursor string
	Limit  int64
}

func GetProcessingJobs() (jobs []JobRunner, err error) {
	err = db.Select(&jobs, "select * from jobs where processing=1 order by id desc")
	return jobs, err
}

func GetUnprocessedJobs() (jobs []JobRunner, err error) {
	err = db.Select(&jobs, "select * from jobs where processing=0 order by id desc")
	return jobs, err
}

func GetCompletedJobs(options GetJobsOption) (jobs []JobRunner, err error) {
	var args []interface{}
	sql := "select * from completed_jobs"
	if options.Cursor != "" {
		sql += " where id < ?"
		args = append(args, options.Cursor)
	}
	sql += " order by id desc"
	if options.Limit > 0 {
		sql += " limit ?"
		args = append(args, options.Limit)
	}
	return getJobsSql(sql, args...)
}

func GetFailedJobs() (jobs []JobRunner, err error) {
	err = db.Select(&jobs, "select * from failed_jobs")
	return jobs, err
}

func getJobsSql(sql string, args ...interface{}) (jobs []JobRunner, err error) {
	log.Println("getJobsSql", sql, args)
	err = db.Select(&jobs, sql, args...)
	return jobs, err
}

func GetJobStatus(id int) (job JobRunner, err error) {
	err = db.Get(&job, "select * from jobs where id=?", id)
	return
}

func GetCompletedJob(id int) (job JobRunner, err error) {
	err = db.Get(&job, "select * from completed_jobs where id=?", id)
	return
}

func WaitOnJobComplete(id int, timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	complete := make(chan bool)
	go func(c chan bool) {
		for {
			job, err := GetCompletedJob(id)
			if err != nil && err != sql.ErrNoRows {
				log.Panic(err)
			} else if job.ID != 0 {
				c <- true
			}
			time.Sleep(time.Millisecond * 100)
		}
	}(complete)
	select {
	case val := <-complete:
		return val
	case <-ctx.Done():
		return false
	}
}
