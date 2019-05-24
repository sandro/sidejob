package web

import (
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/sandro/sidejob"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
)

var PageLimit = 2

func Start() {
	sidejob.InitDB()
	router := gin.Default()
	// router.LoadHTMLGlob("./web/templates/*")
	router.HTMLRender = loadTemplates("./web/templates")

	router.GET("/", func(c *gin.Context) {
		processingJobs, err := sidejob.GetProcessingJobs()
		OrPanic(err)

		unprocessedJobs, err := sidejob.GetUnprocessedJobs()
		OrPanic(err)

		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"processingJobs":  processingJobs,
			"unprocessedJobs": unprocessedJobs,
		})
	})

	router.GET("/completed", func(c *gin.Context) {
		cursor := c.Query("cursor")
		options := sidejob.GetJobsOption{
			Cursor: cursor,
			Limit:  5,
		}
		completedJobs, err := sidejob.GetCompletedJobs(options)
		OrPanic(err)
		var nextCursor int
		var previousCursor int
		if len(completedJobs) == options.Limit {
			nextCursor = completedJobs[len(completedJobs)-1].ID
			if cursor != "" {
				ii, err := strconv.Atoi(cursor)
				OrPanic(err)
				previousCursor = ii + options.Limit
			}
		}

		c.HTML(http.StatusOK, "completed.tmpl", gin.H{
			"completedJobs":  completedJobs,
			"nextCursor":     nextCursor,
			"previousCursor": previousCursor,
		})
	})

	router.GET("/failed", func(c *gin.Context) {
		failedJobs, err := sidejob.GetFailedJobs()
		OrPanic(err)

		c.HTML(http.StatusOK, "failed.tmpl", gin.H{
			"failedJobs": failedJobs,
		})
	})

	router.StaticFS("/assets", gin.Dir("./web/public", false))
	router.Run(":8080")
}

func loadTemplates(templatesDir string) multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	layouts, err := filepath.Glob(templatesDir + "/layouts/*.tmpl")
	if err != nil {
		panic(err.Error())
	}

	includes, err := filepath.Glob(templatesDir + "/*.tmpl")
	if err != nil {
		panic(err.Error())
	}

	// Generate our templates map from our layouts/ and includes/ directories
	for _, include := range includes {
		layoutCopy := make([]string, len(layouts))
		copy(layoutCopy, layouts)
		files := append(layoutCopy, include)
		r.AddFromFiles(filepath.Base(include), files...)
	}
	return r
}

func OrPanic(err error) {
	if err != nil {
		log.Panic(err)
	}
}
