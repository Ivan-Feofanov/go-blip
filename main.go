package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// PingResult holds the timestamp and the latencies for two endpoints.
type PingResult struct {
	Timestamp       time.Time `json:"timestamp"`
	GstaticLatency  int64     `json:"gstatic_latency"`  // in milliseconds
	ApenwarrLatency int64     `json:"apenwarr_latency"` // in milliseconds
}

// WebSocket upgrader.
var upgrader = websocket.Upgrader{
	// Allow any origin (for testing purposes)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade connection to WebSocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	// Define the two endpoints.
	gstaticURL := "https://www.gstatic.com/generate_204" // lightweight target
	apenwarrURL := "https://apenwarr.ca"                 // middleweight target

	for {
		select {
		case tick := <-ticker.C:
			// Measure latency for gstatic.
			start := time.Now()
			resp, err := http.Get(gstaticURL)
			gLatency := time.Since(start).Milliseconds()
			if err != nil {
				log.Println("Gstatic ping error:", err)
				gLatency = -1 // indicate error
			} else {
				resp.Body.Close()
			}

			// Measure latency for apenwarr.
			start = time.Now()
			resp, err = http.Get(apenwarrURL)
			aLatency := time.Since(start).Milliseconds()
			if err != nil {
				log.Println("Apennwarr ping error:", err)
				aLatency = -1
			} else {
				resp.Body.Close()
			}

			// Package both measurements.
			result := PingResult{
				Timestamp:       tick.UTC(), // using UTC; adjust if needed
				GstaticLatency:  gLatency,
				ApenwarrLatency: aLatency,
			}
			data, err := json.Marshal(result)
			if err != nil {
				log.Println("JSON Marshal error:", err)
				continue
			}

			// Send the JSON record over WebSocket.
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Println("WebSocket Write error:", err)
				return
			}
		}
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	// Serve the index.html file.
	http.ServeFile(w, r, "index.html")
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/ws", wsHandler)

	port := ":8080"
	log.Printf("Server started on http://localhost%s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
