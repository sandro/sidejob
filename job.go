package sidecar

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"time"
)

type Job struct {
	ID           int
	Name         string
	Payload      []byte
	RunAt        time.Time `db:"run_at"`
	FailureCount int       `db:"failure_count"`
	CreatedAt    time.Time `db:"created_at"`
	Processing   bool
	runnable     Runnable
}

func (o Job) HandleError(jobError error) {
	log.Println("Handling error", jobError)
	o.FailureCount++
	tx, err := db.Begin()
	OrPanic(err)
	if o.runnable.CanRetry(o.FailureCount) {
		log.Println("Can retry", o.ID, o.Name, jobError.Error())
		o.RunAt = o.runnable.RetryAt(o.FailureCount).UTC()
		tx.Exec("update jobs set failure_count=?, run_at=?, processing=0 where id=?", o.FailureCount, DBTime(o.RunAt), o.ID)
		tx.Exec("insert into failed_jobs (job_id, name, message) values(?,?,?)", o.ID, o.Name, jobError.Error())
	} else {
		log.Println("Cannot retry")
		tx.Exec("update jobs set failure_count=? where id=?", o.FailureCount, o.ID)
		tx.Exec("insert into failed_jobs (job_id, name, message, terminal) values(?,?,?,1)", o.ID, o.Name, jobError.Error())
	}
	err = tx.Commit()
	OrPanic(err)
}

func (o Job) HandleSuccess() {
	tx, err := db.Begin()
	OrPanic(err)
	tx.Exec("delete from jobs where id=?", o.ID)
	tx.Exec("insert into completed_jobs (name, payload, failure_count, job_id) values(?,?,?,?)", o.Name, o.Payload, o.FailureCount, o.ID)
	err = tx.Commit()
	OrPanic(err)
}

func (o Job) Start() (err error) {
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
