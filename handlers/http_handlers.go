package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"microblog/storage"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func HandleRoot(w http.ResponseWriter, _ *http.Request) {
	_, err := w.Write([]byte("Benvenuti nel nostro MicroBlog!"))
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	w.Header().Set("Content-Type", "plain/text")
}

type HTTPHandler struct {
	Storage storage.Storage
}

type PostRequestData struct {
	Text string `json:"text"`
}

func (h *HTTPHandler) PingHandler(rw http.ResponseWriter, r *http.Request) {
	if storage.IsReady {
		_, err := rw.Write([]byte("Ready!\n"))
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		rw.Header().Set("Content-Type", "plain/text")
		return
	}

	http.Error(rw, "Not ready yet", http.StatusBadRequest)
}

func (h *HTTPHandler) HandlePostAPost(rw http.ResponseWriter, r *http.Request) {
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
	time_now := time.Now().In(loc).Format("2006-01-02T15:04:05Z")

	id := uuid.NewString()

	var post = storage.Post{
		Id:        id,
		Text:      data.Text,
		AuthorId:  user,
		CreatedAt: time_now,
	}

	err = h.Storage.PostPost(r.Context(), post)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rawResponse, _ := json.Marshal(post)

	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *HTTPHandler) HandleGetThePost(rw http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	post_id := params["postId"]

	post, err := h.Storage.GetPost(r.Context(), post_id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(rw, "Post with this postId does not exist", 404)
			return
		} else {
			http.Error(rw, err.Error(), http.StatusBadRequest)
			return
		}
	}

	rawResponse, _ := json.Marshal(post)

	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *HTTPHandler) HandleGetThePostLine(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	user := params["userId"]

	var err error
	var answer storage.PostLineAnswer
	answer.Posts = make([]storage.Post, 0)

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

	answer, err = h.Storage.GetPostLine(r.Context(), user, page_token, size)
	if err != nil {
		http.Error(rw, "Something bad has hapened: "+err.Error(), 400)
		return
	}

	rawResponse, _ := json.Marshal(answer)

	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}
