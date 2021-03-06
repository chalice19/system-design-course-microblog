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
		_, err := rw.Write([]byte("Ready to work!\n"))
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
	if !ok || len(user_slice) != 1 {
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
	time_now := time.Now()
	iso_timestamp := time_now.In(loc).Format("2006-01-02T15:04:05Z")
	timestamp := time_now.UnixNano()

	id := uuid.NewString()

	var post = storage.Post{
		Id:             id,
		Text:           data.Text,
		AuthorId:       user,
		CreatedAt:      iso_timestamp,
		LastModifiedAt: iso_timestamp,
		Timestamp: 		timestamp,
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

	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *HTTPHandler) HandleChangeThePostText(rw http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	post_id := params["postId"]

	// check if user specified
	user_slice, ok := r.Header["System-Design-User-Id"]
	if !ok || len(user_slice) != 1 {
		http.Error(rw, "No user specified", http.StatusUnauthorized)
		return
	}

	// read new text
	var data PostRequestData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	loc, _ := time.LoadLocation("UTC")
	time_now := time.Now().In(loc).Format("2006-01-02T15:04:05Z")

	post, err := h.Storage.ChangePostText(r.Context(), post_id, user_slice[0], data.Text, time_now)
	if err != nil {
		if errors.Is(err, storage.ErrUnauthorized) {
			http.Error(rw, "Post with this postId created by other user", http.StatusForbidden)
			return
		} else if errors.Is(err, storage.ErrNotFound) {
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

func (h *HTTPHandler) HandleSubscribe(rw http.ResponseWriter, r *http.Request) {
	user_slice, ok := r.Header["System-Design-User-Id"]
	if !ok || len(user_slice) != 1 {
		http.Error(rw, "No user specified", http.StatusUnauthorized)
		return
	}
	user := user_slice[0]

	params := mux.Vars(r)
	to_user := params["userId"]

	// TODO Subscribe in storage
	err := h.Storage.Subscribe(r.Context(), user, to_user)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rw.WriteHeader(200)

	// ???????????? ???? ?????????????????
	// rawResponse, _ := json.Marshal(post)

	// rw.Header().Set("Content-Type", "application/json")
	// _, err = rw.Write(rawResponse)
	// if err != nil {
	// 	http.Error(rw, err.Error(), http.StatusBadRequest)
	// 	return
	// }
}

func (h *HTTPHandler) HandleGetSubscriptions(rw http.ResponseWriter, r *http.Request) {
	user_slice, ok := r.Header["System-Design-User-Id"]
	if !ok || len(user_slice) != 1 {
		// http.Error(rw, "No user specified", http.StatusUnauthorized)
		http.Error(rw, "No user specified", http.StatusBadRequest)
		return
	}
	user := user_slice[0]

	users, err := h.Storage.GetSubscriptions(r.Context(), user)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rawResponse, _ := json.Marshal(users)

	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *HTTPHandler) HandleGetSubscribers(rw http.ResponseWriter, r *http.Request) {
	user_slice, ok := r.Header["System-Design-User-Id"]
	if !ok || len(user_slice) != 1 {
		// http.Error(rw, "No user specified", http.StatusUnauthorized)
		http.Error(rw, "No user specified", http.StatusBadRequest)
		return
	}
	user := user_slice[0]

	users, err := h.Storage.GetSubscribers(r.Context(), user)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rawResponse, _ := json.Marshal(users)

	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}

func (h *HTTPHandler) GetFeed(rw http.ResponseWriter, r *http.Request) {
	user_slice, ok := r.Header["System-Design-User-Id"]
	if !ok || len(user_slice) != 1 {
		// http.Error(rw, "No user specified", http.StatusUnauthorized)
		http.Error(rw, "No user specified", http.StatusBadRequest)
		return
	}
	user := user_slice[0]

	var err error

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
			http.Error(rw, "Wrong size query", http.StatusBadRequest)
			return
		}
	}

	page_token := query_params.Get("page")
	match, _ := regexp.MatchString("^[A-Za-z0-9_\\-]*$", page_token)
	if !match {
		http.Error(rw, "Wrong PageToken format", http.StatusBadRequest)
		return
	}

	posts, err := h.Storage.GetFeed(r.Context(), user, page_token, size)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	rawResponse, _ := json.Marshal(posts)

	rw.Header().Set("Content-Type", "application/json")
	_, err = rw.Write(rawResponse)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
}
