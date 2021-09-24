package localstorage

import (
	"context"
	"math"
	"microblog/storage"
	"sync"
)

type storage_struct struct {
	storageMu sync.RWMutex
	storage   map[string]storage.Post
	lines     map[string][]string
}

func NewStorage() *storage_struct {
	storage := storage_struct{
		storage: make(map[string]storage.Post),
		lines: make(map[string][]string),
	}

	return &storage
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

func (s *storage_struct) GetPostLine(ctx context.Context, user string, page_token string, size int) (storage.PostLineAnswer, error) {
	var answer storage.PostLineAnswer
	answer.Posts = make([]storage.Post, 0)

	s.storageMu.Lock()
	num_of_posts := len(s.lines[user])
	s.storageMu.Unlock()

	if num_of_posts == 0 {
		if page_token != "" {
			return answer, storage.ErrNotFound
		}

		return answer, nil
	}

	var i = num_of_posts - 1

	if page_token != "" {
		s.storageMu.Lock()
		for i >= 0 && page_token != s.lines[user][i] {
			i--
		}
		s.storageMu.Unlock()
	}

	if i == -1 {
		return answer, storage.ErrNotFound
	}

	var end = int(math.Max(-1, float64(i-size)))

	for ; i >= 0 && i > end; i-- {
		s.storageMu.Lock()
		key := s.lines[user][i]
		answer.Posts = append(answer.Posts, s.storage[key])
		s.storageMu.Unlock()
	}

	if i > -1 {
		s.storageMu.Lock()
		answer.Token = s.lines[user][i]
		s.storageMu.Unlock()
	}

	return answer, nil
}