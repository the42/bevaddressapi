package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/emicklei/go-restful"
	_ "github.com/lib/pq"
)

type Address struct {
	PLZ, Gemeindename, Ortsname, Strassenname, Hausnr *string
	LatlongX, LatlongY                                *float64
}

func main() {
	info("starting up")
	conn, err := getDatabaseConnection()
	if err != nil {
		fatal("Error %s", err)
	}
	connection := &connection{DB: conn}

	restful.DefaultRequestContentType(restful.MIME_JSON)
	restful.DefaultResponseContentType(restful.MIME_JSON)

	ws := new(restful.WebService)
	ws.
		Path("/search").
		Produces(restful.MIME_JSON)

	ws.Route(ws.GET("/fulltext/{pattern}").To(connection.fulltextSearch).
		Doc("").
		Param(ws.PathParameter("pattern", "search term").DataType("string")).
		Writes([]Address{}))

	cors := restful.CrossOriginResourceSharing{
		ExposeHeaders:  []string{"X-My-Header"},
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST"},
		CookiesAllowed: false}

	ws.Filter(cors.Filter)
	restful.Add(ws)
	http.ListenAndServe(":8080", nil)
}

type connection struct {
	*sql.DB
}

const fulltextSearchSQL = `select addritems.plz, addritems.gemeindename, addritems.ortsname, addritems.strassenname, adresse.hausnrzahl1, ST_Y(adresse.latlong), ST_X(adresse.latlong)
from addritems
inner join adresse
on adresse.adrcd = addritems.adrcd
inner join strasse
on adresse.skz = strasse.skz
and adresse.gkz = strasse.gkz
and adresse.gkz = strasse.gkz
inner join ortschaft
on adresse.okz = ortschaft.okz
and adresse.gkz = ortschaft.gkz
inner join gemeinde
on adresse.gkz = gemeinde.gkz
where search @@ to_tsquery(plainto_tsquery('german', $1)::text || ':*')
limit 25;`

func (con *connection) fulltextSearch(request *restful.Request, response *restful.Response) {
	param := request.PathParameter("pattern")
	fmt.Println(param)
	rows, err := con.Query(fulltextSearchSQL, param)
	if err != nil {
		fatal(err.Error())
	}
	defer rows.Close()

	var plz, gemeindename, ortsname, strassenname, hausnrzahl1 *string
	var latlongy, latlongx *float64

	var addresses []Address

	for rows.Next() {
		if err := rows.Scan(&plz, &gemeindename, &ortsname, &strassenname, &hausnrzahl1, &latlongy, &latlongx); err != nil {
			fatal(err.Error())
		}

		addr := Address{PLZ: plz, Gemeindename: gemeindename, Ortsname: ortsname, Strassenname: strassenname, Hausnr: hausnrzahl1, LatlongY: latlongy, LatlongX: latlongx}

		addresses = append(addresses, addr)
	}
	response.WriteEntity(addresses)
}

func getDatabaseConnection() (*sql.DB, error) {

	var dburl string

	if dburl = os.Getenv("DATABASE_URL"); dburl == "" {
		dburl = "postgres://"
	}

	db, err := sql.Open("postgres", dburl)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Log wrappers
func info(template string, values ...interface{}) {
	log.Printf("[bevaddress][info] "+template+"\n", values...)
}

func fatal(template string, values ...interface{}) {
	log.Fatalf("[bevaddress][fatal] "+template+"\n", values...)
}
