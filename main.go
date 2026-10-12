
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Block struct {
	Index      int      `json:"index"`
	Timestamp  string   `json:"timestamp"`
	Data       []string `json:"data"`
	PrevHash   string   `json:"prevHash"`
	Hash       string   `json:"hash"`
	Nonce      int      `json:"nonce"`
	MerkleRoot string   `json:"merkleRoot"`
}

type API struct {
	mu    sync.Mutex
	chain []Block
}

func sha(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func root(data []string) string {
	if len(data) == 0 {
		return sha("")
	}
	level := make([]string, len(data))
	for i, tx := range data {
		level[i] = sha(tx)
	}
	for len(level) > 1 {
		var next []string
		for i := 0; i < len(level); i += 2 {
			j := i + 1
			if j >= len(level) {
				j = i
			}
			next = append(next, sha(level[i]+level[j]))
		}
		level = next
	}
	return level[0]
}

func mine(index int, data []string, prev string) Block {
	b := Block{
		Index: index, Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data: data, PrevHash: prev, MerkleRoot: root(data),
	}
	for {
		text := fmt.Sprintf("%d|%s|%s|%s|%s|%d",
			b.Index, b.Timestamp, strings.Join(b.Data, ","),
			b.PrevHash, b.MerkleRoot, b.Nonce)
		b.Hash = sha(text)
		if strings.HasPrefix(b.Hash, "0000") {
			return b
		}
		b.Nonce++
	}
}

func (a *API) handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	switch r.URL.Path {
	case "/chain":
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"Use GET"}`, http.StatusMethodNotAllowed)
			return
		}
		json.NewEncoder(w).Encode(a.chain)

	case "/mine":
		if r.Method != http.MethodPost {
			http.Error(w, `{"error":"Use POST"}`, http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Data []string `json:"data"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Data) == 0 {
			http.Error(w, `{"error":"Provide a non-empty data array"}`, http.StatusBadRequest)
			return
		}
		last := a.chain[len(a.chain)-1]
		b := mine(len(a.chain), req.Data, last.Hash)
		a.chain = append(a.chain, b)
		json.NewEncoder(w).Encode(b)

	default:
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Ayesha Imran Blockchain API",
			"endpoints": "GET /chain, POST /mine",
		})
	}
}

func main() {
	genesis := mine(0, []string{"Genesis Block"}, strings.Repeat("0", 64))
	api := &API{chain: []Block{genesis}}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", api.handler)
	log.Println("Blockchain API running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
