package main

import (
	"github.com/go-redis/redis/v7"
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

var rdb *redis.Client

func main() {

	rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})

	r := mux.NewRouter()

	r.Path("/recipe").Methods("POST").HandlerFunc(createHandler)
	r.Path("/recipe/{id}").Methods("PUT").HandlerFunc(updateHandler)
	r.Path("/recipe/{id}").Methods("GET").HandlerFunc(getHandler)
	r.Path("/recipes").Methods("GET").HandlerFunc(listHandler)

	log.Fatal(http.ListenAndServe(":8080", r))
}
