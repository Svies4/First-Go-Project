package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"math/rand"
	"net/http"
	"time"
)

var IDs = make(map[int]bool)
var Topics = make(map[string]bool)
var Brokers = make(map[string]*Broker)

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
		fmt.Fprintf(w, "data: %s\n\n", <-messageChan)
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
				delete(Topics, broker.Topic)
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
	var msg Message

	msg.ID = GenerateId()
	_ = json.NewDecoder(r.Body).Decode(&msg)
	j, _ := json.Marshal(msg)

	broker.Notifier <- []byte(j)
	//json.NewEncoder(w).Encode(msg)

}

func GenerateId() int {
	for {
		var ran = rand.Intn(99999-9999) + 9999
		if IDs[ran] == false {
			IDs[ran] = true
			return ran
		}
	}
}
func RequestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic, ok := vars["topic"]
	if !ok {
		fmt.Println("id is missing in parameters")
	}

	switch r.Method {
	case "GET":
		if Topics[topic] == false {
			fmt.Println("Created new server")
			Topics[topic] = true
			broker := NewServer(topic)
			Brokers[topic] = broker
			broker.Stream(w, r)
		} else {
			Brokers[topic].Stream(w, r)
		}

	case "POST":
		if Topics[topic] == false {
			fmt.Println("Created new server")
			Topics[topic] = true
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
