package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	rtr := mux.NewRouter()
	rtr.HandleFunc("/topics/{topic:[a-z.]+}", postOnTopic).Methods("POST")
	http.ListenAndServe(":8080", rtr)
}

func postOnTopic(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	topic := params["topic"]
	println("Handle POST on topic: " + topic)
}
