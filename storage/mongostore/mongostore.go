package mongostore

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"

	"context"
	"errors"
	"fmt"
	"microblog/storage"
	"os"
	"time"
)

const collName = "Posts"
var IsReady bool = false

type storage_struct struct {
	posts *mongo.Collection
}

func NewStorage(mongoURL string) *storage_struct {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		panic(err)
	}

	collection := client.Database(os.Getenv("MONGO_DBNAME")).Collection(collName)
	configureIndexes(ctx, collection)

	IsReady = true

	return &storage_struct{
		posts: collection,
	}
}

func configureIndexes(ctx context.Context, collection *mongo.Collection) {
	indexModels := []mongo.IndexModel{
		{
			Keys: bsonx.Doc{{Key: "authorId", Value: bsonx.Int32(1)},
							{Key: "_id", Value: bsonx.Int32(1)}},
		},
	}
	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)

	_, err := collection.Indexes().CreateMany(ctx, indexModels, opts)
	if err != nil {
		panic(fmt.Errorf("failed to ensure indexes %w", err))
	}
}

func (s *storage_struct) PostPost(ctx context.Context, post storage.Post) error {
	for attempt := 0; attempt < 5; attempt++ {
		_, err := s.posts.InsertOne(ctx, post)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				continue
			}
			return fmt.Errorf("something went wrong - %w", storage.ErrStorage)
		}

		return nil
	}

	return fmt.Errorf("too much attempts during inserting - %w", storage.ErrCollision)
}

func (s *storage_struct) GetPost(ctx context.Context, postId string) (storage.Post, error) {
	var result storage.Post
	err := s.posts.FindOne(ctx, bson.M{"_id": postId}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return result, fmt.Errorf("no post with id %v - %w", postId, storage.ErrNotFound)
		}

		return result, fmt.Errorf("somehting went wrong - %w", storage.ErrStorage)
	}

	return result, nil
}

func (s *storage_struct) GetPostLine(ctx context.Context, user string, post_token string, size int) (storage.PostLineAnswer, error) {
	var answer storage.PostLineAnswer
	answer.Posts = make([]storage.Post, 0)
	var post storage.Post

	cursor, err := s.posts.Find(ctx, bson.M{"authorId": user, "_id": bson.M{"$gte": post_token}})
	if err != nil {
		panic(err)
	}
	defer cursor.Close(ctx)

	i := 0
	cursor_ok := cursor.Next(ctx)
	for cursor_ok && i < size {
		if err = cursor.Decode(&post); err != nil {
			panic(err)
		}
		answer.Posts = append(answer.Posts, post)
		i++
		cursor_ok = cursor.Next(ctx)
	}

	if cursor_ok && cursor.Decode(&post) == nil {
		answer.Token = post.Id
	}

	// count, err := s.posts.CountDocuments(ctx, bson.D{})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("numm of docs:", count)

	return answer, nil
}