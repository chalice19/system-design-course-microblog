package storage

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrStorage      = errors.New("storage")
	ErrCollision    = fmt.Errorf("%w: collision", ErrStorage)
	ErrNotFound     = fmt.Errorf("%w: not found", ErrStorage)
	ErrUnauthorized = fmt.Errorf("%w: unauthorized action", ErrStorage)
)

var IsReady bool = false

type Post struct {
	Id             string `json:"id" bson:"id"`
	Text           string `json:"text" bson:"text"`
	AuthorId       string `json:"authorId" bson:"authorId"`
	CreatedAt      string `json:"createdAt" bson:"createdAt"`
	LastModifiedAt string `json:"lastModifiedAt" bson:"lastModifiedAt"`

	// mongo id to read docs in the right order
	MongoID primitive.ObjectID `json:"mongoId,omitempty" bson:"_id,omitempty"`
}

type PostLineAnswer struct {
	Posts []Post `json:"posts"`
	Token string `json:"nextPage,omitempty"`
}

type Subscription struct {
	User string `bson:"user"`
	ToUser string `bson:"toUser"`
}

type Subscriptions struct {
	Users []string `json:"users"`
}

type Subscribers struct {
	Users []string `json:"users"`
}

type Storage interface {
	PostPost(ctx context.Context, post Post) error
	GetPost(ctx context.Context, postId string) (Post, error)
	GetPostLine(ctx context.Context, user string, page_token string, size int) (PostLineAnswer, error)
	ChangePostText(ctx context.Context, postId string, user string, new_text string, new_time string) (Post, error)

	Subscribe(ctx context.Context, user string, to_user string) error
	GetSubscriptions(ctx context.Context, user string) (Subscriptions, error)
	GetSubscribers(ctx context.Context, user string) (Subscribers, error)
	GetFeed(ctx context.Context, user string, page_token string, size int) (PostLineAnswer, error)
}
