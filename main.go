package main

import (
	"log"
	"microblog/handlers"
	"microblog/storage/mongostore"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

func NewServer() *http.Server {
	r := mux.NewRouter()

	mongoUrl := os.Getenv("MONGO_URL")
	mongostorage := mongostore.NewStorage(mongoUrl)

	handler := &handlers.HTTPHandler{
		Storage: mongostorage,
	}

	r.HandleFunc("/", handlers.HandleRoot)
	r.HandleFunc("/api/v1/posts", handler.HandlePostAPost).Methods("POST")
	r.HandleFunc("/api/v1/posts/{postId:[A-Za-z0-9_\\-]+}", handler.HandleGetThePost).Methods("GET")
	r.HandleFunc("/api/v1/users/{userId:[0-9a-f]+}/posts", handler.HandleGetThePostLine).Methods("GET")
	r.HandleFunc("/maintenance/ping", handler.PingHandler).Methods("GET")

	r.HandleFunc("/api/v1/users/{userId:[0-9a-f]+}/subscribe", handler.HandleSubscribe).Methods("POST")
	r.HandleFunc("/api/v1/subscriptions", handler.HandleGetSubscriptions).Methods("GET")
	r.HandleFunc("/api/v1/subscribers", handler.HandleGetSubscribers).Methods("GET")
	r.HandleFunc("/api/v1/feed", handler.GetFeed).Methods("GET")   // behave like posts

	return &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8080",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
}

func main() {
	srv := NewServer()
	log.Printf("Start serving on %s", srv.Addr)
	log.Fatal(srv.ListenAndServe())
}
