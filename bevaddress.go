package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/lib/pq"
)

// Address struct is the response returned after a request for addresses
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

const maxrowsFTS = 200
const defaultrowsFTS = 25
const fulltextSearchSQL = `select adresse.plz, gemeinde.gemeindename, ortschaft.ortsname, strasse.strassenname, adresse.hausnrzahl1, ST_Y(adresse.latlong), ST_X(adresse.latlong)
from adresse
inner join (select addritems.adrcd
from addritems
where search @@ to_tsquery(plainto_tsquery('german', $1)::text || ':*')
limit $2) s
on s.adrcd = adresse.adrcd
inner join strasse
on adresse.skz = strasse.skz
and adresse.gkz = strasse.gkz
inner join ortschaft
on adresse.okz = ortschaft.okz
and adresse.gkz = ortschaft.gkz
inner join gemeinde
on adresse.gkz = gemeinde.gkz`

func (con *connection) fulltextSearch(w http.ResponseWriter, r *http.Request) {

	q := r.URL.Query().Get("q")

	var n uint64
	if nrows := r.URL.Query().Get("n"); nrows != "" {
		var err error
		if n, err = strconv.ParseUint(nrows, 10, 8); err != nil {
			s := "Error when parsing parameter n: " + err.Error()
			info(s)
			http.Error(w, s, http.StatusBadRequest)
			return
		}
		if n > maxrowsFTS {
			s := "Paramter out of range"
			info(s)
			http.Error(w, s, http.StatusBadRequest)
			return
		}
	} else {
		n = defaultrowsFTS
	}

	rows, err := con.Query(fulltextSearchSQL, q, n)
	if err != nil {
		s := "Database query failed: " + err.Error()
		info(s)
		http.Error(w, s, http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var plz, gemeindename, ortsname, strassenname, hausnrzahl1 *string
	var latlongy, latlongx *float64

	var addresses []Address

	for rows.Next() {
		if err = rows.Scan(&plz, &gemeindename, &ortsname, &strassenname, &hausnrzahl1, &latlongy, &latlongx); err != nil {
			s := "Reading from database failed: " + err.Error()
			info(s)
			http.Error(w, s, http.StatusInternalServerError)
			return
		}

		addr := Address{PLZ: plz, Gemeindename: gemeindename, Ortsname: ortsname, Strassenname: strassenname, Hausnr: hausnrzahl1, LatlongY: latlongy, LatlongX: latlongx}
		addresses = append(addresses, addr)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s := "Connection upgrade to Websocket failed: " + err.Error()
		info(s)
		http.Error(w, s, http.StatusInternalServerError)
		return
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
	currdir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	info("starting up in " + currdir)

	conn, err := getDatabaseConnection()
	if err != nil {
		fatal(err.Error())
	}
	connection := &connection{DB: conn}

	r := mux.NewRouter()
	s := r.PathPrefix("/ws/").Subrouter()
	s.HandleFunc("/address/fts", connection.fulltextSearch)

	var port, secport string
	if secport = os.Getenv("SECPORT"); secport != "" {
		go func() {
			if err := http.ListenAndServeTLS(":"+secport, "cert.pem", "key.pem", r); err != nil {
				fatal("Secure serving failed: " + err.Error())
			}
		}()
		info("serving securely on port " + secport)
	}

	if port = os.Getenv("PORT"); port == "" {
		port = "5000"
	}

	info("serving on port " + port)
	http.ListenAndServe(":"+port, r)
}
