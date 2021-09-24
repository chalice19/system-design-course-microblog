package storage

import (
	"context"
	"errors"
	"fmt"
)

var (
	ErrStorage = errors.New("storage")
	ErrCollision = fmt.Errorf("%w.collision", ErrStorage)
	ErrNotFound = fmt.Errorf("%w.not_found", ErrStorage)
)

type Post struct {
	Id        string `json:"id" bson:"_id"`
	Text      string `json:"text" bson:"text"`
	AuthorId  string `json:"authorId" bson:"authorId"`
	CreatedAt string `json:"createdAt" bson:"createdAt"`
}

type PostLineAnswer struct {
	Posts []Post `json:"posts"`
	Token string `json:"nextPage,omitempty"`
}

type Storage interface {
	PostPost(ctx context.Context, post Post) error
	GetPost(ctx context.Context, postId string) (Post, error)
	GetPostLine(ctx context.Context, user string, post_token string, size int) (PostLineAnswer, error)
}