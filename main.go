package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// Ollama API Request Structure
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Format string `json:"format"`
	Stream bool   `json:"stream"`
}

// Ollama API Response Structure
type OllamaResponse struct {
	Response string `json:"response"`
}

func classifyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the unstructured text from the frontend
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	
	accidentText := string(body)

	// Strict prompt engineering to enforce the exact structure needed
	systemPrompt := fmt.Sprintf(`Analyze the following German workplace accident report. 
Extract the data and return ONLY a valid JSON object with these exact keys. Do not include markdown formatting or extra text.
{
  "category": "(The type of accident, e.g., Sturz, Maschinenschaden, Ergonomie)",
  "body_part": "(The injured body part in German, e.g., Rechter Knöchel)",
  "severity": "(Niedrig, Mittel, or Hoch)",
  "summary_en": "(A 1-sentence technical summary in English)",
  "summary_zh": "(The exact same summary translated to simplified Mandarin)"
}

Report: %s`, accidentText)

	reqBody := OllamaRequest{
		Model:  "qwen2.5",
		Prompt: systemPrompt,
		Format: "json", // Instructs Ollama to guarantee JSON output
		Stream: false,
	}
	jsonData, _ := json.Marshal(reqBody)

	// Call the local Ollama API
	resp, err := http.Post("http://127.0.0.1:11434/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error connecting to Ollama: %v", err)
		http.Error(w, "Error calling local LLM", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Forward the JSON response to the client
	ollamaResult, _ := io.ReadAll(resp.Body)
	var finalResponse OllamaResponse
	if err := json.Unmarshal(ollamaResult, &finalResponse); err != nil {
		log.Printf("Error parsing Ollama response: %v", err)
		http.Error(w, "Invalid response from LLM", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(finalResponse.Response))
}

func main() {
	// Serve the frontend files from the /static directory
	http.Handle("/", http.FileServer(http.Dir("./static")))
	
	// API endpoint for the frontend to hit
	http.HandleFunc("/api/classify", classifyHandler)

	fmt.Println("Accident Classifier Prototype running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
