package storage

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrStorage = errors.New("storage")
	ErrCollision = fmt.Errorf("%w.collision", ErrStorage)
	ErrNotFound = fmt.Errorf("%w.not_found", ErrStorage)
)

var IsReady bool = false

type Post struct {
	Id        string `json:"inmemory_id"`
	Text      string `json:"text" bson:"text"`
	AuthorId  string `json:"authorId" bson:"authorId"`
	CreatedAt string `json:"createdAt" bson:"createdAt"`
	MongoID	  primitive.ObjectID `json:"mongoId,omitempty" bson:"_id,omitempty"`
}

type PostLineAnswer struct {
	Posts []Post `json:"posts"`
	Token string `json:"nextPage,omitempty"`
}

type Storage interface {
	PostPost(ctx context.Context, post Post) error
	GetPost(ctx context.Context, postId string) (Post, error)
	GetPostLine(ctx context.Context, user string, page_token string, size int) (PostLineAnswer, error)
}