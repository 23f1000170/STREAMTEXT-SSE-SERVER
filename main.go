package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type RequestBody struct {
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

func streamHandler(w http.ResponseWriter, r *http.Request) {

	// Enable SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	var body RequestBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		sendError(w, flusher, "Invalid JSON input")
		return
	}

	if body.Prompt == "" {
		sendError(w, flusher, "Prompt cannot be empty")
		return
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		sendError(w, flusher, "Missing OpenAI API key")
		return
	}

	// Base URL
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://aipipe.org/openai/v1"
	}

	endpoint := baseURL + "/chat/completions"

	// Prepare payload
	payload := map[string]interface{}{
		"model":  "gpt-4o-mini",
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": body.Prompt},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		sendError(w, flusher, "Failed to encode JSON")
		return
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonPayload))
	if err != nil {
		sendError(w, flusher, "Failed to create API request")
		return
	}

	req.Header.Set("Authorization", "Bearer "+openaiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		sendError(w, flusher, "API request failed")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		sendError(w, flusher, string(bodyBytes))
		return
	}

	reader := bufio.NewReader(resp.Body)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				break
			}
			sendError(w, flusher, "Error reading stream")
			return
		}

		if len(line) > 0 {
			w.Write(line)
			flusher.Flush()
		}
	}
}

func sendError(w http.ResponseWriter, flusher http.Flusher, message string) {
	errorResponse := map[string]string{
		"error": message,
	}
	jsonErr, _ := json.Marshal(errorResponse)
	fmt.Fprintf(w, "data: %s\n\n", jsonErr)
	flusher.Flush()
}

func main() {
	http.HandleFunc("/stream", streamHandler)

	fmt.Println("Server running at http://localhost:9090")
	log.Fatal(http.ListenAndServe(":9090", nil))

}
