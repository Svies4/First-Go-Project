package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"math/rand"
	"net/http"
)

var IDs = make(map[int]bool)

type Broker struct {
	Topic          string
	Notifier       chan []byte
	newClients     chan chan []byte
	closingClients chan chan []byte
	clients        map[chan []byte]bool
}

/*type Messagee struct {
	Id       int
	Contents string
	Date     time.Time
}*/

func NewServer(Topic string) (broker *Broker) {
	// Instantiate a broker
	broker = &Broker{
		Notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
		Topic:          Topic,
	}

	// Set it running - listening and broadcasting events
	go broker.listen()

	return
}

type Message struct {
	Message string `json:"msg"`
	ID      int    `json:"id"`
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

	// Each connection registers its own message channel with the Broker's connections registry
	messageChan := make(chan []byte)

	// Signal the broker that we have a new connection
	broker.newClients <- messageChan

	// Remove this client from the map of connected clients
	// when this handler exits.
	defer func() {
		broker.closingClients <- messageChan
	}()

	// Listen to connection close and un-register messageChan
	notify := w.(http.CloseNotifier).CloseNotify()

	go func() {
		<-notify
		broker.closingClients <- messageChan
	}()

	for {

		// Write to the ResponseWriter
		// Server Sent Events compatible
		fmt.Fprintf(w, "data: %s\n\n", <-messageChan)

		// Flush the data immediatly instead of buffering it for later.
		flusher.Flush()
	}

}

func (broker *Broker) listen() {
	for {
		select {
		case s := <-broker.newClients:

			// A new client has connected.
			// Register their message channel
			broker.clients[s] = true
			log.Printf("Client added. %d registered clients", len(broker.clients))
		case s := <-broker.closingClients:

			// A client has dettached and we want to
			// stop sending them messages.
			delete(broker.clients, s)
			log.Printf("Removed client. %d registered clients", len(broker.clients))
		case event := <-broker.Notifier:

			// We got a new event from the outside!
			// Send event to all connected clients
			for clientMessageChan, _ := range broker.clients {
				clientMessageChan <- event
			}
		}
	}

}

func (broker *Broker) BroadcastMessage(w http.ResponseWriter, r *http.Request) {
	// params := mux.Vars(r)

	var msg Message
	fmt.Println(json.NewDecoder(r.Body).Decode(&msg))
	msg.ID = GenerateId()
	_ = json.NewDecoder(r.Body).Decode(&msg)

	//broker.Notifier <- []byte(fmt.Sprintf("the time is %v", msg))
	j, _ := json.Marshal(msg)
	//fmt.Println(r)
	broker.Notifier <- []byte(j)
	json.NewEncoder(w).Encode(msg)
	//broker.Notifier <- []byte("aaaa")

}

func GenerateId() int {
	for {
		var ran = rand.Intn(99999999-9999999) + 9999999
		if IDs[ran] == false {
			IDs[ran] = true
			fmt.Println(ran)
			return ran
		}
	}
}
func main() {
	//broker := NewServer()
	router := mux.NewRouter()

	router.Handle("/", http.Handler(http.FileServer(http.Dir("./"))))

	//router.HandleFunc("/messages", broker.BroadcastMessage).Methods("POST")

	router.HandleFunc("/infocenter/{topic}", Provisions)

	log.Fatal(http.ListenAndServe(":8080", router))

}

var Topics = make(map[string]bool)
var Brokers = make(map[string]*Broker)

//var Servers = make(map[Broker]New)

func Provisions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic, ok := vars["topic"]
	if !ok {
		fmt.Println("id is missing in parameters")
	}
	fmt.Println(`topic := `, topic)

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
