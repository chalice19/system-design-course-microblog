package main

import (
	"log"
	"microblog/handlers"
	"microblog/storage"
	"microblog/storage/cacheredis"
	"microblog/storage/localstorage"
	"microblog/storage/mongostore"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

const inmemory_mode = "inmemory"
const mongo_db_mode = "mongo"
const cached = "cached"

func NewStorage() storage.Storage {
	storage_mode := os.Getenv("STORAGE_MODE")
	if storage_mode == inmemory_mode {
		local_storage := localstorage.NewStorage()
		return local_storage

	} else if storage_mode == mongo_db_mode {
		mongoUrl := os.Getenv("MONGO_URL")
		mongostorage := mongostore.NewStorage(mongoUrl)
		return mongostorage

	} else if storage_mode == cached {
		mongoUrl := os.Getenv("MONGO_URL")
		mongostorage := mongostore.NewStorage(mongoUrl)

		redisUrl := os.Getenv("REDIS_URL")
		redisClient := redis.NewClient(&redis.Options{ Addr: redisUrl })
		cached_storage := cacheredis.NewStorage(mongostorage, redisClient)

		return cached_storage
	}

	return nil
}

func NewServer() *http.Server {
	r := mux.NewRouter()

	handler := &handlers.HTTPHandler{
		Storage: NewStorage(),
	}

	r.HandleFunc("/", handlers.HandleRoot)
	r.HandleFunc("/api/v1/posts", handler.HandlePostAPost).Methods("POST")
	r.HandleFunc("/api/v1/posts/{postId:[A-Za-z0-9_\\-]+}", handler.HandleGetThePost).Methods("GET")
	r.HandleFunc("/api/v1/posts/{postId:[A-Za-z0-9_\\-]+}", handler.HandleChangeThePostText).Methods("PATCH")
	r.HandleFunc("/api/v1/users/{userId:[0-9a-f]+}/posts", handler.HandleGetThePostLine).Methods("GET")
	r.HandleFunc("/maintenance/ping", handler.PingHandler).Methods("GET")

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
