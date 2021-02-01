package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

var subscriptions map[string]map[string]string
var httpClient = &http.Client{
	Timeout: time.Minute,
}

func main() {
	subscriptions = make(map[string]map[string]string)
	rtr := mux.NewRouter()
	rtr.HandleFunc("/topics/{topic:[a-zA-Z._-]+}", postEventOnTopic).Methods("POST")
	rtr.HandleFunc("/groups", registerGroup).Methods("POST")
	rtr.HandleFunc("/topics", registerTopic).Methods("POST")
	rtr.HandleFunc("/topics/{topic:[a-zA-Z._-]+}/subscriptions", registerSubscription).Methods("POST")
	http.ListenAndServe(":8080", rtr)
}

func postEventOnTopic(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	topicName := params["topic"]
	println("Handle event on topic: " + topicName)

	if topicSubs, ok := subscriptions[topicName]; ok {
		headers := filterHeaders(req.Header)
		if body, err := ioutil.ReadAll(req.Body); err == nil {
			for _, endpoint := range topicSubs {
				go sendEventToEndpoint(endpoint, body, headers)
			}
		}
	}
}

func registerGroup(w http.ResponseWriter, req *http.Request) {
	var reqJSON map[string]string
	json.NewDecoder(req.Body).Decode(&reqJSON)
	println("Register group:", reqJSON["groupName"])
}

func registerTopic(w http.ResponseWriter, req *http.Request) {
	var reqJSON map[string]string
	json.NewDecoder(req.Body).Decode(&reqJSON)
	println("Register topic:", reqJSON["name"])
}

func registerSubscription(w http.ResponseWriter, req *http.Request) {
	var reqJSON map[string]string
	json.NewDecoder(req.Body).Decode(&reqJSON)
	topicName := reqJSON["topicName"]
	subscriptionName := reqJSON["name"]
	endpoint := reqJSON["endpoint"]

	println("Register subscription on topic:", topicName, "["+subscriptionName+"]", "->", endpoint)

	if _, ok := subscriptions[topicName]; !ok {
		subscriptions[topicName] = make(map[string]string)
	}
	subscriptions[topicName][subscriptionName] = endpoint
}

func sendEventToEndpoint(endpoint string, body []byte, headers map[string][]string) {
	println("Send event to endpoint:", endpoint)
	req, err := http.NewRequest("POST", endpoint, bytes.NewReader(body))
	if err != nil {
		println("Failed to build request for endpoint:", endpoint)
	}
	req.Header = headers
	_, err = httpClient.Do(req)
	if err != nil {
		println("Error on calling endpoint:", endpoint, err)
	}
}

func filterHeaders(headers map[string][]string) map[string][]string {
	newHeaders := make(map[string][]string)
	for name, values := range headers {
		newHeaders[name] = values
	}
	return newHeaders
}
