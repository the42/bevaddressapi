A Websocket API to provide search functionality in Austria's Address register
as published under http://www.bev.gv.at/portal/page?_pageid=713,2601271&_dad=portal&_schema=PORTAL

This package relies on the workings of these other components:

* bevdockerdb, a PostGIS powered PostgreSQL installation with abbreviations and
  thesaurus dictionary for improved full text search;  
  [Github Project](https://github.com/the42/bevdockerdb)  
  [Docker Hub](https://hub.docker.com/r/the42/bevdockerdb/)
* [bevaddress-dataload](https://github.com/the42/bevaddress-dataload), a set of scripts to load data into the aforementioned PostGIS database.

# Installation

Install locally using

    go get github.com/the42/bevaddressapi

Local installation requires a working [Golang environment](https://golang.org/dl/).

# Configuration and running
bevaddressapi accepts two environment variables for configuration:

`PORT` - the tcp port on which the websocket API will listen for incoming connections;  
`DATABASE_URL` - a url defining the connection parameters to the database.

Currently only PostGIS is supported as the database backend. See the
[documentation](https://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters) on how to set this environment variable.

# Usage
* `/ws/address/fts`: A websocket endpoint for full text search. Paramters:  
`q` url-encoded string for full text search  
`postfix` either `true` or `false` (default), controlling if the search term(s) will also postfix match. Set to true, the search term `Krems` will match the city
Krems but also Kremsmünster.
