package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"sync"
	"time"
)

var IDs = make(map[int]bool)

// var Topics = make(map[string]bool)
var Brokers = make(map[string]*Broker)

var idCounter uint64
var id sync.Mutex
var ChanLock sync.Mutex

type TopicsList struct {
	topics     map[string]bool
	topicsLock sync.Mutex
}

var Topics = TopicsList{topics: map[string]bool{}}

type Broker struct {
	Topic          string
	Notifier       chan []byte
	newClients     chan chan []byte
	closingClients chan chan []byte
	clients        map[chan []byte]bool
}

type Message struct {
	Message string `json:"msg"`
	ID      int    `json:"id"`
}

func NewServer(Topic string) (broker *Broker) {
	// Instantiate a broker
	broker = &Broker{
		Notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
		Topic:          Topic,
	}

	go broker.listen()

	return
}

func (broker *Broker) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)

	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	messageChan := make(chan []byte)

	connectionDate := time.Now()

	broker.newClients <- messageChan

	defer func() {
		broker.closingClients <- messageChan

	}()

	go func() {
		time.Sleep(30 * time.Second)
		fmt.Fprintf(w, "event: timeout\ndata: %s\n\n", fmt.Stringer(time.Since(connectionDate)))
		flusher.Flush()
		broker.closingClients <- messageChan
	}()

	notify := w.(http.CloseNotifier).CloseNotify()

	go func() {
		<-notify
		broker.closingClients <- messageChan
	}()

	for {
		_, err := fmt.Fprintf(w, "data: %s\n\n", <-messageChan)
		if err != nil {
			// Handle error
			return
		}
		flusher.Flush()
	}

}

func (broker *Broker) listen() {
	for {
		select {
		case s := <-broker.newClients:

			broker.clients[s] = true
			log.Printf("Client added. %d registered clients", len(broker.clients))
		case s := <-broker.closingClients:

			delete(broker.clients, s)
			log.Printf("Removed client. %d registered clients", len(broker.clients))
			if len(broker.clients) == 0 {
				delete(Topics.topics, broker.Topic)
				return
			}
		case event := <-broker.Notifier:

			for clientMessageChan := range broker.clients {
				clientMessageChan <- event
			}
		}
	}

}

func (broker *Broker) BroadcastMessage(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var msg Message
	ChanLock.Lock()
	defer ChanLock.Unlock()
	msg.ID = GenerateId()

	body := json.NewDecoder(r.Body)

	if err := body.Decode(&msg); err != nil {
		http.Error(w, "Bad Request: Invalid JSON format", http.StatusBadRequest)
		return
	}

	j, _ := json.Marshal(msg)
	broker.Notifier <- []byte(j)
}

func GenerateId() int {
	id.Lock()
	defer id.Unlock()
	idCounter++
	return int(idCounter)
}

func RequestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic, ok := vars["topic"]
	if !ok {
		fmt.Println("id is missing in parameters")
	}

	switch r.Method {
	case "GET":
		if r.Header.Get("Accept") == "text/event-stream" { //Checks if get request is event-stream type
			if Topics.topics[topic] == false {
				fmt.Println("Created new server")
				Topics.topics[topic] = true
				broker := NewServer(topic)
				Brokers[topic] = broker
				broker.Stream(w, r)
			} else {
				Brokers[topic].Stream(w, r)
			}
		} else {
			w.WriteHeader(400)
		}

	case "POST":
		if Topics.topics[topic] == false {
			fmt.Println("Created new server")
			Topics.topics[topic] = true
			broker := NewServer(topic)
			Brokers[topic] = broker
			Brokers[topic].BroadcastMessage(w, r)
		} else {
			Brokers[topic].BroadcastMessage(w, r)

		}
	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}
}

func main() {

	router := mux.NewRouter()

	router.Handle("/", http.Handler(http.FileServer(http.Dir("./"))))

	router.HandleFunc("/infocenter/{topic}", RequestHandler)

	log.Fatal(http.ListenAndServe(":8080", router))

}
