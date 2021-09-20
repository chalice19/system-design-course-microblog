package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

func handleRoot(w http.ResponseWriter, _ *http.Request) {
	_, err := w.Write([]byte("Benvenuti nel nostro MicroBlog!"))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	w.Header().Set("Content-Type", "plain/text")
}

type Post struct {
	Id        string `json:"id"`
	Text      string `json:"text"`
	AuthorId  string `json:"authorId"`
	CreatedAt string `json:"createdAt"`
}

type HTTPHandler struct {
	storageMu sync.RWMutex
	storage   map[string]Post
	lines     map[string][]string
}

type PostRequestData struct {
	Text string `json:"text"`
}

func (h *HTTPHandler) handlePostAPost(rw http.ResponseWriter, r *http.Request) {
	var data PostRequestData

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	user_slice, ok := r.Header["System-Design-User-Id"]
	if !ok {
		http.Error(rw, "No user specified", http.StatusUnauthorized)
		return
	}
	user := user_slice[0]
	match, _ := regexp.MatchString("^[0-9a-f]+$", user)
	if !match {
		http.Error(rw, "Wrong UserId format", http.StatusUnauthorized)
		return
	}

	loc, _ := time.LoadLocation("UTC")

	time_now := time.Now().In(loc).Format("01-02-2006T15:04:05Z")
	rand.Seed(time.Now().UnixNano())
	time_sec_string := strconv.Itoa(rand.Intn(100))
	var id_text = time_now + user + time_sec_string

	// id := b64.StdEncoding.EncodeToString([]byte(id_text))

	// id_hash := sha1.New()
	// id_hash.Write([]byte(id_text))
	// id := hex.EncodeToString(id_hash.Sum(nil))

	id := base64.RawStdEncoding.EncodeToString([]byte(id_text))

	var post = Post{id, data.Text, user, time_now}

	h.storageMu.Lock()
	h.storage[id] = post

	if h.lines == nil {
		h.lines = make(map[string][]string)
	}
	user_posts := h.lines[user]
	h.lines[user] = append(user_posts, id)

	h.storageMu.Unlock()

	rawResponse, _ := json.Marshal(post)

	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *HTTPHandler) handleGetThePost(rw http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	post_id := params["postId"]

	h.storageMu.Lock()
	post, ok := h.storage[post_id]
	h.storageMu.Unlock()

	if !ok {
		http.Error(rw, "Post with this postId does not exist", 404)
		return
	}

	rawResponse, _ := json.Marshal(post)

	rw.Header().Set("Content-Type", "application/json")
	var err error
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

type PostLineAnswer struct {
	Posts []Post `json:"posts"`
	Token string `json:"nextPage,omitempty"`
}

func (h *HTTPHandler) handleGetThePostLine(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	user := params["userId"]

	var err error
	var num_of_posts int
	var answer PostLineAnswer
	answer.Posts = make([]Post, 0)

	h.storageMu.Lock()
	num_of_posts = len(h.lines[user])
	h.storageMu.Unlock()
	if num_of_posts == 0 {
		rawResponse, _ := json.Marshal(answer)
		_, err = rw.Write(rawResponse)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
		return
	}

	query_params := r.URL.Query()
	size_query := query_params.Get("size")
	var size int
	if size_query == "" {
		size = 10
	} else {
		size, err = strconv.Atoi(size_query)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		} else if size < 0 {
			http.Error(rw, "Wrong size query", 400)
			return
		}
	}

	page_token := query_params.Get("page")
	match, _ := regexp.MatchString("^[A-Za-z0-9_\\-]*$", page_token)
	if !match {
		http.Error(rw, "Wrong PageToken format", 400)
		return
	}

	var i = num_of_posts - 1

	if page_token != "" {
		h.storageMu.Lock()
		for i >= 0 && page_token != h.lines[user][i] {
			i--
		}
		h.storageMu.Unlock()
	}

	if i == -1 {
		http.Error(rw, "Bad PageToken", 400)
		return
	}

	var end = int(math.Max(-1, float64(i-size)))

	for ; i >= 0 && i > end; i-- {
		h.storageMu.Lock()
		key := h.lines[user][i]
		answer.Posts = append(answer.Posts, h.storage[key])
		h.storageMu.Unlock()
	}

	if i > -1 {
		h.storageMu.Lock()
		answer.Token = h.lines[user][i]
		h.storageMu.Unlock()
	}

	rawResponse, _ := json.Marshal(answer)

	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func NewServer() *http.Server {
	r := mux.NewRouter()

	handler := &HTTPHandler{
		storage: make(map[string]Post),
	}

	r.HandleFunc("/", handleRoot)
	r.HandleFunc("/api/v1/posts", handler.handlePostAPost).Methods("POST")
	r.HandleFunc("/api/v1/posts/{postId:[A-Za-z0-9_\\-]+}", handler.handleGetThePost).Methods("GET")
	r.HandleFunc("/api/v1/users/{userId:[0-9a-f]+}/posts", handler.handleGetThePostLine).Methods("GET")

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
