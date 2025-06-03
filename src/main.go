package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

var serverAddr = "ws://localhost:4000/api/tunnel/host"
var testServerAddr = "http://localhost:4000"
var hostServerAddr = "http://localhost:3000"
var tunnel_code uint8
var routepath string

func CreateBinaryPacket(parts ...any) ([]byte, error) {
	responseBfr := new(bytes.Buffer)
	for _, part := range parts {
		if err := binary.Write(responseBfr, binary.BigEndian, part); err != nil {
			return nil, err
		}
	}

	return responseBfr.Bytes(), nil
}

type TunnelRequestPayload struct {
	Method  string
	Path    string
	Headers map[string]string
	Body    []byte
}

type TunnelResponsePayload struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

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
		// message -> [1 1 3]
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading from connection", err.Error())
			return
		}

		switch message[1] {
		// receiving code from multi-mux
		case 1:
			tunnel_code = uint8(message[2])
			fmt.Println("Tunneling Code:", tunnel_code)

		// multi-mux requests html-data. message-data here is the routepath
		case 2:
			requestBuf := message[2:]
			var requestPayload TunnelRequestPayload

			dec := gob.NewDecoder(bytes.NewReader(requestBuf))
			if err := dec.Decode(&requestPayload); err != nil {
				log.Println("E:Decoding serialised request buffer.")
				return
			}

			endpoint := ""
			if requestPayload.Path == "" {
				endpoint = hostServerAddr

			} else {
				endpoint = hostServerAddr + "/" + requestPayload.Path
			}

			fmt.Println("Endpoint was:", endpoint)

			// Send http request data
			res, err := http.Get(endpoint)
			if err != nil {
				log.Println("Error sending request to localhost")
				continue
			}

			defer res.Body.Close()

			resHeaders := map[string]string{}
			for k, v := range res.Header {
				resHeaders[k] = strings.Join(v, ",")
			}

			resBody, err := io.ReadAll(res.Body)
			if err != nil {
				log.Println("Error reading response data", err.Error())
				continue
			}

			resPayload := TunnelResponsePayload{
				StatusCode: res.StatusCode,
				Headers:    resHeaders,
				Body:       resBody,
			}

			var responseBuf bytes.Buffer
			enc := gob.NewEncoder(&responseBuf)
			if err = enc.Encode(resPayload); err != nil {
				log.Println("E: Serializing to response buffer.")
				return
			}

			pkt, _ := CreateBinaryPacket(byte(1), byte(3), responseBuf.Bytes())
			if err := conn.WriteMessage(websocket.BinaryMessage, pkt); err != nil {
				log.Println("E:Writing response buffer to socket.", err.Error())
				return
			} else {
				fmt.Println("Res written")
			}
		}

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
