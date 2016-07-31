Working Demos can be found here:
* [Official Website](https://www.offene-adressen.at)
* [GitHub Preview](http://htmlpreview.github.io/?https://github.com/the42/bevaddressapi/blob/master/bevaddressftssearch.html)

A Websocket API to provide search functionality in Austria's Address register
as published under http://www.bev.gv.at/portal/page?_pageid=713,2601271&_dad=portal&_schema=PORTAL

This package relies on the workings of these other components:

* bevdockerdb, a PostGIS powered PostgreSQL installation with abbreviations and
  thesaurus dictionary for improved full text search;  
  [Github Project](https://github.com/the42/bevdockerdb)  
  [Docker Hub](https://hub.docker.com/r/the42/bevdockerdb/)
* [bevaddress-dataload](https://github.com/the42/bevaddress-dataload), a set of scripts to load data into the aforementioned PostGIS database.

# Installation

## Install locally using

    go get github.com/the42/bevaddressapi

Local installation requires a working [Golang environment](https://golang.org/dl/).

## Configuration and running
bevaddressapi accepts two environment variables for configuration:

`PORT` - the tcp port on which the websocket API will listen for incoming connections, defaults to 5000 if not set;
`SECPORT` - the tcp port on which the websocket API will listen for incoming TLS connections; If not set or empty, the service is only served unencrypted;
`DATABASE_URL` - a url defining the connection parameters to the database.

Currently only PostGIS is supported as the database backend. See the
[documentation](https://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters) on how to set this environment variable.

## Install using Docker
    docker pull the42/bevaddressapi

Running

    docker run -it --name bevadr -P -p 5000:5000 -e DATABASE_URL=postgres://<DB_username>:<DB_password>@<DB_host>:<DB_port>/bevaddress the42/bevaddressapi

Replace DB_.... wit the appropriate values to the [Address-database](https://hub.docker.com/r/the42/bevdockerdb/)

In case your database server does not support SSl-encryption use

    docker run -it --name bevadr -P -p 5000:5000 -e DATABASE_URL=postgres://<DB_username>:<DB_password>@<DB_host>:<DB_port>/bevaddress?sslmode=disable the42/bevaddressapi


# Usage

The fulltext search endpoint is exposed as a websocket and listens as following:

`/ws/address/fts`: A websocket endpoint for full text search. Paramters:
* `q`: url-encoded string for full text search
* `autocomplete`: when set to `0`, entered queries have to match exact, otherwise a prefix search is performed.

Optional Filters:
* `postcode`: filter by zip-code (Postleitzahl)
* `citycode`: filter by [Gemeindekennzahl](http://www.statistik.at/web_de/klassifikationen/regionale_gliederungen/gemeinden/index.html)
* `province`: filter by province (Bundesland). The coding is according to https://de.wikipedia.org/wiki/ISO_3166-2:AT
* `lat`, `lon`: filter by latitude and longitude using [WGS84 coordinates](https://de.wikipedia.org/wiki/World_Geodetic_System_1984). When used, both parameters have to be set.

* `n` (optional), `25` (default); return up to n results. A hard limit is implemented which prevents bulk download and to bring down the server.
