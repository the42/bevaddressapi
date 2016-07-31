package main

import (
	"database/sql"
	"fmt"
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
const nearbymeters = 50 // default distance to search nearby addresses in meter
const autocomplete = `search @@ to_tsquery(plainto_tsquery('german', $1)::text || ':*')`
const noautocomplete = `search @@ plainto_tsquery('german', $1)`

const fulltextSearchSQL = `select addritems.plz, addritems.gemeindename, addritems.ortsname, addritems.strassenname, addritems.hausnrzahl1, ST_Y(adresse.latlong), ST_X(adresse.latlong)
from adresse
inner join addritems
on addritems.adrcd = adresse.adrcd
and %s
and addritems.plz like COALESCE(NULLIF($2, ''), addritems.plz)
and addritems.gkz like COALESCE(NULLIF($3, ''), addritems.gkz)
and addritems.bld = COALESCE(CAST(NULLIF($4, '') AS smallint), addritems.bld)
and CASE ($5 = '' AND $6='') WHEN NOT FALSE THEN TRUE ELSE ST_DWithin(latlong_g, ST_GeomFromText('POINT(' || $5 || ' ' || $6 || ')', 4326)::geography, $7, false) END
limit $8`

func (con *connection) fulltextSearch(w http.ResponseWriter, r *http.Request) {

	var n uint64
	if nrows := r.URL.Query().Get("n"); nrows != "" {
		var err error
		if n, err = strconv.ParseUint(nrows, 10, 8); err != nil {
			s := "error when parsing parameter n: " + err.Error()
			info(s)
			http.Error(w, s, http.StatusBadRequest)
			return
		}
		if n > maxrowsFTS {
			s := "paramter out of range"
			info(s)
			http.Error(w, s, http.StatusBadRequest)
			return
		}
	} else {
		n = defaultrowsFTS
	}

	lat := r.URL.Query().Get("lat")
	lon := r.URL.Query().Get("lon")

	if (len(lat) > 0) != (len(lon) > 0) { // Latitude/Longitude: either both parameters are set or none of the two is set
		s := "lat/lon: either both parameters are set to a value or both have to be empty"
		info(s)
		http.Error(w, s, http.StatusBadRequest)
		return
	}

	q := r.URL.Query().Get("q")
	postcode := r.URL.Query().Get("postcode")
	citycode := r.URL.Query().Get("citycode")
	province := r.URL.Query().Get("province")

	var querystring string
	if acparam := r.URL.Query().Get("autocomplete"); acparam == "0" {
		querystring = fmt.Sprintf(fulltextSearchSQL, noautocomplete)
	} else {
		querystring = fmt.Sprintf(fulltextSearchSQL, noautocomplete)
	}

	rows, err := con.Query(querystring, q, postcode, citycode, province, lat, lon, nearbymeters, n)
	if err != nil {
		s := "database query failed: " + err.Error()
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
			s := "reading from database failed: " + err.Error()
			info(s)
			http.Error(w, s, http.StatusInternalServerError)
			return
		}

		addr := Address{PLZ: plz, Gemeindename: gemeindename, Ortsname: ortsname, Strassenname: strassenname, Hausnr: hausnrzahl1, LatlongY: latlongy, LatlongX: latlongx}
		addresses = append(addresses, addr)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s := "connection upgrade to websocket failed: " + err.Error()
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
				fatal("secure serving failed: " + err.Error())
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

// Log wrappers
func info(template string, values ...interface{}) {
	log.Printf("[bevaddress][info] "+template+"\n", values...)
}

func fatal(template string, values ...interface{}) {
	log.Fatalf("[bevaddress][fatal] "+template+"\n", values...)
}
