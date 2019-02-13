package hello

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Message struct {
	From    string `json:"from"`
	Message string `json:"message"`
}

// HelloWorld prints "Hello, World!" or Hello, $message.From"
func HelloWorldFn(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	message := &Message{}
	if err := decoder.Decode(&message); err == nil && message.From != "" {
		_, _ = fmt.Fprintf(w, fmt.Sprintf("Hello, %v!", message.From))
		return
	}
	_, _ = fmt.Fprintf(w, "Hello, World!")
}
