package mongostore

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"

	"context"
	"errors"
	"fmt"
	"log"
	"microblog/storage"
	"os"
	"time"
)

type storage_struct struct {
	posts *mongo.Collection
	subscriptions *mongo.Collection
	feeds *mongo.Collection
}

func NewStorage(mongoURL string) *storage_struct {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
	if err != nil {
		panic(err)
	}

	posts := client.Database(os.Getenv("MONGO_DBNAME")).Collection("Posts")
	configurePostsIndexes(ctx, posts)

	subscriptions := client.Database(os.Getenv("MONGO_DBNAME")).Collection("Subscribes")
	configureSubscribesIndexes(ctx, subscriptions)

	feeds := client.Database(os.Getenv("MONGO_DBNAME")).Collection("Feeds")
	configureFeedsIndexes(ctx, feeds)

	storage.IsReady = true

	return &storage_struct{
		posts: posts,
		subscriptions: subscriptions,
		feeds: feeds,
	}
}

func configurePostsIndexes(ctx context.Context, collection *mongo.Collection) {
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

func configureSubscribesIndexes(ctx context.Context, collection *mongo.Collection) {
	indexModels := []mongo.IndexModel{
		{
			Keys: bsonx.Doc{{Key: "user", Value: bsonx.Int32(1)},
			{Key: "toUser", Value: bsonx.Int32(1)}},
		},
	}
	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)

	_, err := collection.Indexes().CreateMany(ctx, indexModels, opts)
	if err != nil {
		panic(fmt.Errorf("failed to ensure indexes %w", err))
	}
}

func configureFeedsIndexes(ctx context.Context, collection *mongo.Collection) {
	indexModels := []mongo.IndexModel{
		{
			Keys: bsonx.Doc{{Key: "user", Value: bsonx.Int32(1)},
				{Key: "postId", Value: bsonx.Int32(-1)},
				{Key: "time", Value: bsonx.Int32(-1)}},
		},
		{
			Keys: bsonx.Doc{{Key: "user", Value: bsonx.Int32(1)},
			{Key: "time", Value: bsonx.Int32(-1)}},
		},
	}
	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)

	_, err := collection.Indexes().CreateMany(ctx, indexModels, opts)
	if err != nil {
		panic(fmt.Errorf("failed to ensure indexes %w", err))
	}
}

func (s *storage_struct) PostPost(ctx context.Context, post storage.Post) error {
	log.Println("User", post.AuthorId, "created post", post)

	for attempt := 0; attempt < 5; attempt++ {
		_, err := s.posts.InsertOne(ctx, post)
		if err != nil {
			if mongo.IsDuplicateKeyError(err) {
				continue
			}
			return fmt.Errorf("something went wrong - %w", storage.ErrStorage)
		}

		// добавить также в feed всем, кто подписан на post.authorId
		var subscription storage.Subscription
		user := post.AuthorId

		// find all subscribers of the user
		cursor, err := s.subscriptions.Find(ctx, bson.M{"toUser": user})
		if err != nil {
			return err
		}
		defer cursor.Close(ctx)

		cursor_ok := cursor.Next(ctx)
		for cursor_ok{
			if err = cursor.Decode(&subscription); err != nil {
				return err
			}
			
			feedpost := storage.FeedPost {
				User: subscription.User,
				PostId: post.MongoID,
				Timestamp: post.Timestamp,
				Post: post,
			}
			err = s.addFeedPost(ctx, feedpost)
			if err != nil {
				return err
			}

			cursor_ok = cursor.Next(ctx)
		}

		return nil
	}

	return fmt.Errorf("too much attempts during inserting - %w", storage.ErrCollision)
}

func (s *storage_struct) GetPost(ctx context.Context, postId string) (storage.Post, error) {
	var result storage.Post

	err := s.posts.FindOne(ctx, bson.M{"id": postId}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return result, fmt.Errorf("no post with id %v - %w", postId, storage.ErrNotFound)
		}

		return result, fmt.Errorf("somehting went wrong - %w", storage.ErrStorage)
	}

	return result, nil
}

func (s *storage_struct) GetPostLine(ctx context.Context, user string, page_token string, size int) (storage.PostLineAnswer, error) {
	var answer storage.PostLineAnswer
	answer.Posts = make([]storage.Post, 0)
	var post storage.Post
	var err error
	var page_token_decoded primitive.ObjectID

	// check if page_token is correct
	if page_token != "" {
		count, err := s.posts.CountDocuments(ctx, bson.M{"authorId": user})
		if err != nil {
			panic(err)
		}
		if count == 0 {
			return answer, storage.ErrNotFound
		}

		// make ObjectID from page_token
		page_token_decoded, err = primitive.ObjectIDFromHex(page_token)
		if err != nil {
			panic(err)
		}
	}

	// sort posts so that latest goes first
	opts := options.Find()
	opts.SetSort(bson.M{"_id": -1})

	// find all posts of the user sorted beginning from page_token post
	var cursor *mongo.Cursor

	if page_token != "" {
		cursor, err = s.posts.Find(ctx, bson.M{"authorId": user, "_id": bson.M{"$lte": page_token_decoded}}, opts)
	} else {
		cursor, err = s.posts.Find(ctx, bson.M{"authorId": user}, opts)
	}

	if err != nil {
		panic(err)
	}
	defer cursor.Close(ctx)

	// read and save no more than size posts
	i := 0
	cursor_ok := cursor.Next(ctx)
	for cursor_ok && i < size {
		if err = cursor.Decode(&post); err != nil {
			panic(err)
		}
		answer.Posts = append(answer.Posts, post)

		// decode, err := primitive.ObjectIDFromHex(post.MongoID.Hex())
		// if err != nil {
		// 	panic(err)
		// }

		// fmt.Println(post.Text, post.MongoID.Hex(), decode)

		i++
		cursor_ok = cursor.Next(ctx)
	}

	if page_token != "" && len(answer.Posts) == 0 {
		return answer, storage.ErrNotFound
	}

	// save the next page_token
	if cursor_ok && cursor.Decode(&post) == nil {
		answer.Token = post.MongoID.Hex()
	}

	// count, err := s.posts.CountDocuments(ctx, bson.D{})
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("numm of docs:", count)

	return answer, nil
}

func (s *storage_struct) ChangePostText(ctx context.Context, postId string, user string, new_text string, new_time string) (storage.Post, error) {
	post, err := s.GetPost(ctx, postId)
	if err != nil {
		return post, err
	}

	if post.AuthorId != user {
		return post, storage.ErrUnauthorized
	}

	_, err = s.posts.UpdateOne(
		ctx,
		bson.M{"id": postId},
		bson.M{"$set": bson.M{"text": new_text, "lastModifiedAt": new_time}},
	)

	if err != nil {
		return post, nil
	}

	post.Text = new_text
	post.LastModifiedAt = new_time
	
	// а еще изменить во всех копиях в feed

	_, err = s.feeds.UpdateMany(
		ctx,
		bson.M{"postId": post.MongoID},
		bson.M{"$set": bson.M{"post": post}},
	)

	if err != nil {
		return post, nil
	}

	return post, err
}

func (s *storage_struct) Subscribe(ctx context.Context, user string, to_user string) error {
	log.Printf("Called Subscribe user %s to %s\n", user, to_user)

	count, err := s.subscriptions.CountDocuments(ctx, bson.M{"user": user, "toUser": to_user})
	if err != nil {
		panic(err)
	}
	already_subed := count != 0

	if (!already_subed) {

		for attempt := 0; attempt < 5; attempt++ {
			// Вставить без дупликатов
			opts := options.Update().SetUpsert(true)
			_, err = s.subscriptions.UpdateOne(
				ctx, 
				bson.M{"user": user, "toUser": to_user},
				bson.M{"$set": bson.M{"user": user, "toUser": to_user}},
				opts,
			)

			if err != nil {
				if mongo.IsDuplicateKeyError(err) {
					continue
				}
				log.Fatalln("error: ", err.Error())
				return fmt.Errorf("something went wrong - %w", storage.ErrStorage)
			}

			err = s.copyPostsToSubscriber(ctx, user, to_user)
			if err != nil {
				log.Fatalln("error: ", err.Error())
				return fmt.Errorf("something went wrong - %w", storage.ErrStorage)
			}

			return nil
		}

	}

	return fmt.Errorf("too much attempts during inserting - %w", storage.ErrCollision)
}

func (s *storage_struct) GetSubscriptions(ctx context.Context, user string) (storage.Subscriptions, error) {
	log.Printf("Called Subscriptions of user %s\n", user)

	var answer storage.Subscriptions
	var subscription storage.Subscription

	// find all subscriptions of the user
	cursor, err := s.subscriptions.Find(ctx, bson.M{"user": user})
	if err != nil {
		return answer, err
	}
	defer cursor.Close(ctx)

	// read and save
	cursor_ok := cursor.Next(ctx)
	for cursor_ok{
		if err = cursor.Decode(&subscription); err != nil {
			return answer, err
		}
		answer.Users = append(answer.Users, subscription.ToUser)

		log.Println("subscribed to subscription")

		cursor_ok = cursor.Next(ctx)
	}

	return answer, nil
}

func (s *storage_struct) GetSubscribers(ctx context.Context, user string) (storage.Subscribers, error) {
	log.Printf("Called Subscribers of user %s\n", user)

	var answer storage.Subscribers
	var subscription storage.Subscription
	
	// find all subscriptions of the user
	cursor, err := s.subscriptions.Find(ctx, bson.M{"toUser": user})
	if err != nil {
		return answer, err
	}
	defer cursor.Close(ctx)

	// read and save
	cursor_ok := cursor.Next(ctx)
	for cursor_ok{
		if err = cursor.Decode(&subscription); err != nil {
			return answer, err
		}
		answer.Users = append(answer.Users, subscription.User)

		cursor_ok = cursor.Next(ctx)
	}

	return answer, nil
}

func (s *storage_struct) GetFeed(ctx context.Context, user string, page_token string, size int) (storage.PostLineAnswer, error) {
	log.Println("You send GetFeed requset for", size, "posts, we're working on it")
	var answer storage.PostLineAnswer
	answer.Posts = make([]storage.Post, 0)
	var err error

	var page_token_decoded primitive.ObjectID

	// check if page_token is correct
	if page_token != "" {
		count, err := s.posts.CountDocuments(ctx, bson.M{"user": user})
		if err != nil {
			panic(err)
		}
		if count == 0 {
			return answer, storage.ErrNotFound
		}

		// make ObjectID from page_token
		page_token_decoded, err = primitive.ObjectIDFromHex(page_token)
		if err != nil {
			panic(err)
		}
	}

	opts := options.Find()
	opts.SetSort(bson.M{"time": -1})

	// find all posts of the user sorted by time beginning from page_token post
	var cursor *mongo.Cursor

	if page_token != "" {
		cursor, err = s.feeds.Find(ctx, bson.M{"user": user, "postId": bson.M{"$lte": page_token_decoded}}, opts)
	} else {
		cursor, err = s.feeds.Find(ctx, bson.M{"user": user}, opts)
	}

	if err != nil {
		return answer, err
	}
	defer cursor.Close(ctx)

	// count how many for this user
	count, err := s.feeds.CountDocuments(ctx, bson.M{"user": user})
	if err != nil {
		panic(err)
	}
	log.Println("num of posts for user to see:", count)

	var feedpost storage.FeedPost
	i := 0
	cursor_ok := cursor.Next(ctx)
	for cursor_ok && i < size {
		if err = cursor.Decode(&feedpost); err != nil {
			return answer, err
		}

		answer.Posts = append(answer.Posts, feedpost.Post)

		i++
		cursor_ok = cursor.Next(ctx)
	}

	if page_token != "" && len(answer.Posts) == 0 {
		return answer, storage.ErrNotFound
	}

	// save the next page_token
	if cursor_ok && cursor.Decode(&feedpost) == nil {
		answer.Token = feedpost.PostId.Hex()
	}

	return answer, nil
}

func (s *storage_struct) copyPostsToSubscriber(ctx context.Context, user string, to_user string) error {
	cursor, err := s.posts.Find(ctx, bson.M{"authorId": to_user})
	if err != nil {
		return err
	}
	defer cursor.Close(ctx)

	var post storage.Post

	cursor_ok := cursor.Next(ctx)
	for cursor_ok{
		if err = cursor.Decode(&post); err != nil {
			return err
		}

		feedpost := storage.FeedPost{
			User: user,
			Timestamp: post.Timestamp,
			PostId: post.MongoID,
			Post: post,
		}
		err = s.addFeedPost(ctx, feedpost)
		if err != nil {
			return err
		}

		log.Println("copied post", feedpost.PostId, "to feed of", user)

		cursor_ok = cursor.Next(ctx)
	}

	return nil
}

func (s *storage_struct) addFeedPost(ctx context.Context, feedpost storage.FeedPost) error {
	for attempt := 0; attempt < 5; attempt++ {
		_, err := s.feeds.InsertOne(ctx, feedpost)
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