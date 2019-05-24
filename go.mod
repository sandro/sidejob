module github.com/sandro/sidejob

go 1.12

require (
	github.com/frankban/quicktest v1.3.0
	github.com/gin-contrib/multitemplate v0.0.0-20190301062633-f9896279eead // indirect
	github.com/gin-gonic/gin v1.4.0 // indirect
	github.com/jmoiron/sqlx v1.2.0
	github.com/mattn/go-sqlite3 v1.10.0
	github.com/sandro/sidejob/web v0.0.0-20190524130525-ca9f75412868 // indirect
)

replace github.com/sandro/sidejob/web => ./web

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
