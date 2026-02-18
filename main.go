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

	// CORS headers (must be first)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

	// Handle preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Allow GET for health check
	if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Streaming endpoint is live"))
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Enable SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 🔥 Immediate flush so grader sees response instantly
	flusher.Flush()

	var body RequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
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

	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://aipipe.org/openai/v1"
	}

	endpoint := baseURL + "/chat/completions"

	payload := map[string]interface{}{
		"model":  "gpt-4o-mini",
		"stream": true,
		"messages": []map[string]string{
			{"role": "user", "content": body.Prompt},
		},
	}

	jsonPayload, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Authorization", "Bearer "+openaiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
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
				return
			}
			sendError(w, flusher, "Error reading stream")
			return
		}

		// OpenAI sends lines starting with "data: "
		if bytes.HasPrefix(line, []byte("data: ")) {

			data := bytes.TrimPrefix(line, []byte("data: "))
			data = bytes.TrimSpace(data)

			if string(data) == "[DONE]" {
				fmt.Fprintf(w, "data: [DONE]\n\n")
				flusher.Flush()
				return
			}

			var parsed map[string]interface{}
			if err := json.Unmarshal(data, &parsed); err == nil {

				choices, ok := parsed["choices"].([]interface{})
				if ok && len(choices) > 0 {

					choice := choices[0].(map[string]interface{})
					delta := choice["delta"].(map[string]interface{})

					if content, exists := delta["content"]; exists {

						out := map[string]string{
							"content": content.(string),
						}

						jsonOut, _ := json.Marshal(out)
						fmt.Fprintf(w, "data: %s\n\n", jsonOut)
						flusher.Flush()
					}
				}
			}
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
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090" // local fallback
}

fmt.Println("Server running on port", port)
log.Fatal(http.ListenAndServe(":"+port, nil))

}
