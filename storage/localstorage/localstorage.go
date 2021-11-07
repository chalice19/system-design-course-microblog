package localstorage

import (
	"context"
	"microblog/storage"
	"strconv"
	"strings"
	"sync"
	"time"
)

type storage_struct struct {
	storageMu sync.RWMutex
	storage   map[string]storage.Post
	lines     map[string][]string
}

func NewStorage() *storage_struct {
	new_storage := storage_struct{
		storage: make(map[string]storage.Post),
		lines:   make(map[string][]string),
	}

	storage.IsReady = true
	return &new_storage
}

func (s *storage_struct) PostPost(ctx context.Context, post storage.Post) error {
	s.storageMu.Lock()
	
	s.storage[post.Id] = post

	user_posts := s.lines[post.AuthorId]
	s.lines[post.AuthorId] = append(user_posts, post.Id)

	s.storageMu.Unlock()

	return nil
}

func (s *storage_struct) GetPost(ctx context.Context, postId string) (storage.Post, error) {
	s.storageMu.Lock()
	post, ok := s.storage[postId]
	s.storageMu.Unlock()

	if !ok {
		return post, storage.ErrNotFound
	}

	return post, nil
}

// func (s *storage_struct) GetPostLine(ctx context.Context, user string, page_token string, size int) (storage.PostLineAnswer, error) {
// 	var answer storage.PostLineAnswer
// 	answer.Posts = make([]storage.Post, 0)

// 	s.storageMu.Lock()
//	defer s.storageMu.Unlock()

// 	num_of_posts := len(s.lines[user])

// 	if num_of_posts == 0 {
// 		if page_token != "" {
// 			return answer, storage.ErrNotFound
// 		}

// 		return answer, nil
// 	}

// 	var i = num_of_posts - 1

// 	if page_token != "" {
// 		for i >= 0 && page_token != s.lines[user][i] {
// 			i--
// 		}
// 	}

// 	if i == -1 {
// 		return answer, storage.ErrNotFound
// 	}

// 	var end = i - size

// 	for ; i >= 0 && i > end; i-- {
// 		key := s.lines[user][i]
// 		answer.Posts = append(answer.Posts, s.storage[key])
// 	}

// 	if i > -1 {
// 		answer.Token = s.lines[user][i]
// 	}

// 	return answer, nil
// }

func (s *storage_struct) GetPostLine(ctx context.Context, user string, page_token string, size int) (storage.PostLineAnswer, error) {
	var answer storage.PostLineAnswer
	answer.Posts = make([]storage.Post, 0)

	s.storageMu.Lock()
	defer s.storageMu.Unlock()

	num_of_posts := len(s.lines[user])

	if num_of_posts == 0 {
		if page_token == "" {
			return answer, nil
		}

		return answer, storage.ErrNotFound
	}

	var index int
	var err error

	if page_token == "" {
		index = num_of_posts - 1
	} else {
		token := strings.Split(page_token, "_")
		if len(token) != 2 || token[0] != user {
			return answer, storage.ErrNotFound
		}

		index, err = strconv.Atoi(token[1])
		if err != nil {
			return answer, err
		}
		if index < 0 || index >= num_of_posts {
			return answer, storage.ErrNotFound
		}
	}

	end := index - size

	for ; index > end && index >= 0; index-- {
		post := s.storage[s.lines[user][index]]
		answer.Posts = append(answer.Posts, post)
	}

	if index >= 0 {
		answer.Token = user + "_" + strconv.Itoa(index)
	}

	return answer, nil
}

func (s *storage_struct) ChangePostText(ctx context.Context, postId string, user string, new_text string) (storage.Post, error) {
	s.storageMu.Lock()
	defer s.storageMu.Unlock()

	post, ok := s.storage[postId]

	if !ok {
		return post, storage.ErrNotFound
	}
	if post.AuthorId != user {
		return post, storage.ErrUnauthorized
	}

	loc, _ := time.LoadLocation("UTC")
	time_now := time.Now().In(loc).Format("2006-01-02T15:04:05Z")

	post.Text = new_text
	post.CreatedAt = time_now

	s.storage[postId] = post

	return post, nil
}