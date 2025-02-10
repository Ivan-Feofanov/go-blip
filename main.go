package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// PingResult represents one measurement.
type PingResult struct {
	Timestamp time.Time `json:"timestamp"`
	Latency   int64     `json:"latency"` // in milliseconds
}

// Upgrader is used for upgrading HTTP connections to WebSocket.
var upgrader = websocket.Upgrader{
	// Allow any origin (for testing)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade the connection to WebSocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	// ticker for periodic ping (e.g., every second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	targetURL := "https://www.gstatic.com/generate_204" // lightweight target

	for {
		select {
		case t := <-ticker.C:
			// measure the latency by doing a simple GET request
			start := time.Now()
			resp, err := http.Get(targetURL)
			latency := time.Since(start).Milliseconds()
			if err != nil {
				log.Println("Ping error:", err)
				// if there’s an error, you can choose to mark it with a high latency value
				latency = -1
			} else {
				resp.Body.Close()
			}

			// prepare the JSON record with current time and latency
			result := PingResult{
				Timestamp: t.UTC(), // send UTC timestamp (or use local time as desired)
				Latency:   latency,
			}
			data, err := json.Marshal(result)
			if err != nil {
				log.Println("JSON Marshal error:", err)
				continue
			}

			// write JSON message to the WebSocket
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("WebSocket Write error:", err)
				return
			}
		}
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// serve the index.html file (assumed to be in the same directory)
	http.ServeFile(w, r, "index.html")
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/ws", wsHandler)

	port := ":8080"
	log.Printf("Starting server on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
