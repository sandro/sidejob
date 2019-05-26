package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sandro/sidejob"
	"github.com/sandro/sidejob/web"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n", os.Args[0])
		fmt.Fprintln(flag.CommandLine.Output(), `
  $sidejob [-db <file.sqlite3>] [-poll <1s>]

Running this command with no arguments starts the job runner.

Use the subcommand 'web' to start the web interface:
  $ sidejob web
		`)
		flag.PrintDefaults()
	}

	dbConnectURI := flag.String("db", "", "The db connection URI")
	pollDuration := flag.Duration("poll", time.Duration(0), "The polling duration, i.e. 1s")

	flag.Parse()

	if *dbConnectURI != "" {
		log.Println("setting dbConnectURI", dbConnectURI)
		sidejob.Config.DBConnectURI = *dbConnectURI
	}
	if *pollDuration != time.Duration(0) {
		log.Println("setting pollDuration", pollDuration)
		sidejob.Config.PollDuration = *pollDuration
	}

	fmt.Println(os.Args)
	if len(flag.Args()) == 1 {
		switch flag.Arg(0) {
		case "web":
			sidejob.InitDB()
			web.Start()
		default:
			flag.Usage()
			os.Exit(1)
		}
	} else {
		sidejob.InitDB()
		sidejob.Start()
	}
}
