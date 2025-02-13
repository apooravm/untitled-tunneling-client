package main

import (
	"flag"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var serverAddr = "ws://localhost:4000/api/tunnel"

func main() {
	localAddr := flag.String("local", "localhost:8081", "Local service to expose.")
	flag.Parse()

	conn, _, err := websocket.DefaultDialer.Dial(serverAddr, nil)
	if err != nil {
		log.Fatal("Failed to connect to server. ", err.Error())
		return
	}

	defer conn.Close()

	log.Println("Connection live")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		reqData, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read req", http.StatusInternalServerError)
		}

		defer r.Body.Close()

		err = conn.WriteMessage(websocket.BinaryMessage, reqData)
		if err != nil {
			http.Error(w, "Failed to send request", http.StatusInternalServerError)
			return
		}

		_, respData, err := conn.ReadMessage()
		if err != nil {
			http.Error(w, "Failed to read request", http.StatusInternalServerError)
			return
		}

		w.Write(respData)
	})

	log.Println("Client proxy running at:", *localAddr)
	log.Fatal(http.ListenAndServe(*localAddr, nil))
}

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}
