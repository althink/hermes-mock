package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/patrickmn/go-cache"
)

var subscriptions map[string]map[string]string
var httpClient = &http.Client{
	Timeout: time.Minute,
}
var eventsCache = cache.New(5*time.Minute, 10*time.Minute)
var correlationHeader = "X-Correlation-Id"

type CorrelationInfo struct {
	Events []EventInfo
}

type EventInfo struct {
	CorrelationID string      `json:"correlationId"`
	TopicName     string      `json:"topic"`
	Headers       http.Header `json:"headers"`
	Body          []byte      `json:"body"`
}

var eventsChan chan EventInfo

func main() {
	eventsChan = make(chan EventInfo)
	subscriptions = make(map[string]map[string]string)
	rtr := mux.NewRouter()
	rtr.HandleFunc("/topics/{topic:[a-zA-Z0-9._-]+}", postEventOnTopic).Methods("POST")
	rtr.HandleFunc("/groups", registerGroup).Methods("POST")
	rtr.HandleFunc("/topics", registerTopic).Methods("POST")
	rtr.HandleFunc("/topics/{topic:[a-zA-Z._-]+}/subscriptions", registerSubscription).Methods("POST")
	rtr.HandleFunc("/events/{correlationID:[a-zA-Z0-9_-]+}", queryEvents).Methods("GET")

	go startEventsCacheJob()

	http.ListenAndServe(":8080", rtr)
}

func postEventOnTopic(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	topicName := params["topic"]
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	headersBuilder := strings.Builder{}
	for key, value := range req.Header {
		headersBuilder.WriteString(fmt.Sprintf(" - %v: %v\n", key, value))
	}
	fmt.Printf("Handle event on topic: %v\nHeaders: \n%vBody:\n%v\n", topicName, headersBuilder.String(), string(body))

	if cidHeader, ok := req.Header[correlationHeader]; ok && len(cidHeader) > 0 {
		eventsChan <- EventInfo{

			CorrelationID: cidHeader[0],
			TopicName:     topicName,
			Headers:       req.Header,
			Body:          body,
		}
	}

	if topicSubs, ok := subscriptions[topicName]; ok {
		headers := filterHeaders(req.Header)
		for _, endpoint := range topicSubs {
			go sendEventToEndpoint(endpoint, body, headers)
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

func startEventsCacheJob() {
	for event := range eventsChan {
		var info *CorrelationInfo
		if fromCache, ok := eventsCache.Get(event.CorrelationID); ok {
			info = fromCache.(*CorrelationInfo)
		} else {
			info = &CorrelationInfo{Events: []EventInfo{}}
			err := eventsCache.Add(event.CorrelationID, info, cache.DefaultExpiration)
			if err != nil {
				fmt.Printf("Failed to add to cache: %v\n", err)
			}
		}
		info.Events = append(info.Events, event)
	}
}

func queryEvents(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	correlationID := params["correlationID"]
	if info, ok := eventsCache.Get(correlationID); ok {
		json.NewEncoder(w).Encode(info.(*CorrelationInfo).Events)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}
