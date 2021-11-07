package cacheredis

import (
	"context"
	"encoding/json"
	"fmt"
	"microblog/storage"
	"time"

	"github.com/go-redis/redis/v8"
)

type storage_struct struct {
	client            *redis.Client
	persistentStorage storage.Storage
}

func NewStorage(persistentStorage storage.Storage, client *redis.Client) *storage_struct {
	return &storage_struct{
		client:            client,
		persistentStorage: persistentStorage,
	}
}

func (s *storage_struct) save_to_cache(ctx context.Context, post storage.Post) {
	json_post, _ := json.Marshal(post)
	err := s.client.Set(ctx, post.Id, string(json_post), time.Hour).Err()
	if err != nil {
		fmt.Println("Failed to insert key ", post.Id, " into cache due to an error: ", err)
	} else {
		fmt.Println("Saved to cash: ", string(json_post))
	}
}

func (s *storage_struct) PostPost(ctx context.Context, post storage.Post) error {
	// save to db
	err := s.persistentStorage.PostPost(ctx, post)
	if err != nil {
		return err
	}

	s.save_to_cache(ctx, post)

	return nil
}

func (s *storage_struct) GetPost(ctx context.Context, postId string) (storage.Post, error) {
	//try to get from cache

	str_post, err := s.client.Get(ctx, postId).Result();
	var post storage.Post

	fmt.Println("From cache read: ", str_post)

	switch {
		case err == redis.Nil:
			// continue execution
		case err != nil:
			fmt.Println("From cache we couldn't take", postId, "because of error: ", err)
		default:
			json.Unmarshal([]byte(str_post), &post)
			return post, nil
	}

	// get from db
	post, err = s.persistentStorage.GetPost(ctx, postId)
	if err != nil {
		return post, err
	}

	s.save_to_cache(ctx, post)

	return post, nil
}

func (s *storage_struct) GetPostLine(ctx context.Context, user string, page_token string, size int) (storage.PostLineAnswer, error) {
	answer, err := s.persistentStorage.GetPostLine(ctx, user, page_token, size)
	if err != nil {
		return answer, err
	}

	for _, post := range answer.Posts {
		s.save_to_cache(ctx, post)
	}

	return answer, nil
}

func (s *storage_struct) ChangePostText(ctx context.Context, postId string, user string, new_text string, new_time string) (storage.Post, error) {
	post, err := s.persistentStorage.ChangePostText(ctx, postId, user, new_text, new_time)
	if err != nil {
		return post, err
	}

	s.save_to_cache(ctx, post)

	return post, nil
}