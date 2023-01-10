package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"sync"
	_ "sync"
	"time"
)

// var msgChan chan string
var msgChanMap = make(map[string]chan string)
var mut = &sync.Mutex{}

func addMsg() {

}

func sseHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic, okk := vars["topic"]
	fmt.Println(`topic := `, topic)

	if !okk {
		fmt.Println("id is missing in parameters")
	}
	fmt.Println("Client connected")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	//msgChan = make(chan string)
	//msgChanMap := make(map[string]chan string)

	//msgChanMap[topic] = make(chan string, 2)
	//msgChanMap[topic] <- topic

	defer func() {
		close(msgChanMap[topic])
		msgChanMap[topic] = nil
		fmt.Println("Client closed connection")
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		fmt.Println("Could not init http.Flusher")
	}

	for {
		select {
		case message := <-msgChanMap[topic]:
			fmt.Println("case message... sending message")
			fmt.Println(message)
			fmt.Fprintf(w, "data: %s\n\n", message)
			flusher.Flush()
		case <-r.Context().Done():
			fmt.Println("Client closed connection")
			return
		}
		//time.Sleep(300 * time.Millisecond)
		//fmt.Println("WAITING")
		//msgChanMap[topic] <- topic
	}

}

type Topic struct {
	Name     string
	Messages []Message
}
type Message struct {
	Id       int
	Contents string
	Date     time.Time
}

func main() {
	msgChanMap["general"] = make(chan string, 5)
	r := mux.NewRouter()
	r.Handle("/", http.Handler(http.FileServer(http.Dir("./"))))
	//r.HandleFunc("/infocenter/sa", sseHandler)
	r.HandleFunc("/infocenter/{topic}", Provisions)

	go func() {
		fmt.Println("serving on 8080")
		err := http.ListenAndServe(":8080", r)
		if err != nil {
			panic("ListenAndServe: " + err.Error())
		}
	}()

	/*go func() {
		for {
			//time.Sleep(300 * time.Millisecond)
			//msgChanMap := make(map[string]chan string)
			msgChanMap["general"] = make(chan string, 2)
			//mut.Lock()
			msgChanMap["general"] <- "GENERALmessage"
			//mut.Unlock()
			fmt.Println("id is missing in parameters")
		}
	}()*/
	go func() {
		fmt.Println("a")
	}()
	select {}
}

func Provisions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	topic, ok := vars["topic"]
	if !ok {
		fmt.Println("id is missing in parameters")
	}
	fmt.Println(`topic := `, topic)

	/*fmt.Println(`topic cvfvsv:= `, msgChanMap["topic"])
	if msgChanMap["topic"] != nil {
		msg := "topic"
		msgChanMap["topic"] <- msg
	}*/
	switch r.Method {
	case "GET":

		sseHandler(w, r)
	case "POST":
		fmt.Println("aaaaaaaaaaaaaaaaaaaaaaaa")
		mut.Lock()
		//time.Sleep(1000 * time.Millisecond)

		msgChanMap["general"] <- "Gdassadasdasdssage"
		mut.Unlock()
		fmt.Println("id is missing in parameters")

	default:
		fmt.Fprintf(w, "Sorry, only GET and POST methods are supported.")
	}

	//call http://localhost:8080/provisions/someId in your browser
	//Output : id := someId
}
