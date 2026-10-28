package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	store := NewSecretStore(os.Getenv("VAULT_ADDR"), os.Getenv("VAULT_TOKEN"))

	key, source, err := LoadAPIKey(store)
	if err != nil {
		log.Fatalf("cannot load api key: %v", err)
	}

	srv := NewServer(key, source)
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", srv.Handler()))
}
