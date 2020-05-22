package main

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

func createHandler(w http.ResponseWriter, r *http.Request) {

	var recipe recipe
	err := json.NewDecoder(r.Body).Decode(&recipe)
	if err != nil {
		handleError(w, err)
		return
	}

	err = recipe.save(rdb)
	if err != nil {
		handleError(w, err)
		return
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request) {

	id := mux.Vars(r)["id"]
	idInt, err := strconv.Atoi(id)
	if err != nil {
		handleError(w, err)
		return
	}

	var recipe recipe
	err = json.NewDecoder(r.Body).Decode(&recipe)
	if err != nil {
		handleError(w, err)
		return
	}

	recipe.ID = int64(idInt)
	err = recipe.save(rdb)
	if err != nil {
		handleError(w, err)
		return
	}
}

func getHandler(w http.ResponseWriter, r *http.Request) {

	id := mux.Vars(r)["id"]
	idInt, err := strconv.Atoi(id)
	if err != nil {
		handleError(w, err)
		return
	}

	var recipe = &recipe{}

	err = recipe.load(int64(idInt), rdb)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(recipe)
	if err != nil {
		handleError(w, err)
		return
	}
}

func listHandler(w http.ResponseWriter, r *http.Request) {

	pageParam, ok := r.URL.Query()["page"]
	if !ok {
		handleError(w, errors.New("missing page parameter"))
		return
	}

	page, err := strconv.Atoi(pageParam[0])
	if err != nil {
		handleError(w, err)
		return
	}

	l, err := list(page, rdb)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(l)
	if err != nil {
		handleError(w, err)
		return
	}
}

func handleError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err.Error()))
}
