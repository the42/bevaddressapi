package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/emicklei/go-restful"
	"github.com/lib/pq"
)

func getDatabaseConnection() (*sql.DB, error) {

	var dburl, dbconnstring string

	if dburl = os.Getenv("DATABASE_URL"); dburl == "" {
		dburl = "postgres://"
	}

	dbconnstring, err := pq.ParseURL(dburl)
	if err != nil {
		return nil, fmt.Errorf("Invalid Database Url: %s (%s)\n", dburl, err)
	}

	db, err := sql.Open("postgres", dbconnstring)
	if err != nil {
		return nil, err
	}
	return db, nil
}

type User struct {
	Name string
}

func main() {
	conn, err := getDatabaseConnection()
	if err != nil {
		fmt.Printf("Error %s", err)
	}
	row := conn.QueryRow("SELECT localtime")

	var t pq.NullTime
	if err := row.Scan(&t); err != nil {
		fmt.Printf("Error %s", err)
		return
	}
	fmt.Println(t)

	ws := new(restful.WebService)

	ws.
		Path("/users").
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/{user-id}").To(findUser).
		Doc("get a user").
		Param(ws.PathParameter("user-id", "identifier of the user").DataType("string")).
		Writes(User{}))

	restful.Add(ws)
	http.ListenAndServe(":8080", nil)
}

func findUser(request *restful.Request, response *restful.Response) {
	id := request.PathParameter("user-id")
	response.WriteAsJson(User{Name: id + "world"})
}
