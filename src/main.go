package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var serverAddr = "ws://localhost:4000/api/tunnel/host"
var testServerAddr = "http://localhost:4000"
var hostServerAddr = "http://localhost:5000"

// Bit of an issue here regarding how i am handling routepaths
// So if the routepath is empty or "", it defaults to localhost:5000 NOT localhost:5000/
// Even though they are different. Dunno how to differentiate b/w them for now
func main() {
	fmt.Println("Online")
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

		routepath := string(message[2:])
		fmt.Println("Target route path:", routepath)

		endpoint := ""
		if routepath == "" {
			endpoint = hostServerAddr

		} else {
			endpoint = hostServerAddr + "/" + routepath
		}

		fmt.Println("Endpoint was:", endpoint)

		// Send http request data
		res, err := http.Get(endpoint)
		if err != nil {
			log.Println("Error sending request to localhost")
		}

		defer res.Body.Close()

		resData, err := io.ReadAll(res.Body)
		if err != nil {
			log.Println("Error reading response data", err.Error())
		}

		conn.WriteMessage(websocket.BinaryMessage, resData)
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
