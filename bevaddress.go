package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

// Address struct
// TODO: add json struct fields for lowercase serialisation
type Address struct {
	PLZ, Gemeindename, Ortsname, Strassenname, Hausnr *string
	LatlongX, LatlongY                                *float64
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
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

func (con *connection) fulltextSearch(w http.ResponseWriter, r *http.Request) {

	// TODO: do not use query but pattern
	param := r.URL.Query().Get("q")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		info(err.Error())
		return
	}

	rows, err := con.Query(fulltextSearchSQL, param)
	if err != nil {
		info(err.Error())
		return
	}
	defer rows.Close()
	var plz, gemeindename, ortsname, strassenname, hausnrzahl1 *string
	var latlongy, latlongx *float64

	var addresses []Address

	for rows.Next() {
		if err := rows.Scan(&plz, &gemeindename, &ortsname, &strassenname, &hausnrzahl1, &latlongy, &latlongx); err != nil {
			info(err.Error())
			return
		}

		addr := Address{PLZ: plz, Gemeindename: gemeindename, Ortsname: ortsname, Strassenname: strassenname, Hausnr: hausnrzahl1, LatlongY: latlongy, LatlongX: latlongx}

		addresses = append(addresses, addr)
	}
	conn.WriteJSON(addresses)
	conn.Close()
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

func main() {
	info("starting up")
	conn, err := getDatabaseConnection()
	if err != nil {
		fatal(err.Error())
	}
	connection := &connection{DB: conn}

	r := mux.NewRouter()
	s := r.PathPrefix("/ws/").Subrouter()
	// "/ws/"
	s.HandleFunc("/address/fts/", connection.fulltextSearch)

	http.ListenAndServe(":8080", r)
}
