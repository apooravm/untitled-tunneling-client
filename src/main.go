package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var serverAddr = "ws://localhost:4000/api/tunnel"

func main() {
	conn, _, err := websocket.DefaultDialer.Dial(serverAddr, nil)
	if err != nil {
		log.Println("E:Connecting ws server.", err.Error())
		return
	}

	defer conn.Close()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading from connection", err.Error())
			return
		}

		fmt.Println("Message:", string(message))

		// Send http request data
		res, err := http.Get("http://localhost:5000")
		if err != nil {
			log.Println("Error sending request")
		}

		defer res.Body.Close()

		resData, err := io.ReadAll(res.Body)
		if err != nil {
			log.Println("Error reading response data", err.Error())
		}

		conn.WriteMessage(websocket.BinaryMessage, resData)
		break
	}
}

func main2() {
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
