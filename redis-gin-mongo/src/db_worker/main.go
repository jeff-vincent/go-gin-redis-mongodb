package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BlogPost struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Body   string `json:"body"`
}

const (
	databaseName   = "blog"
	collectionName = "posts"
)

var (
	mongo_host = os.Getenv("MONGO1_HOST")
	mongo_port = os.Getenv("MONGO1_PORT")
	redis_host = os.Getenv("REDIS_HOST")
	redis_port = os.Getenv("REDIS_PORT")
	mongo_uri  = fmt.Sprintf("mongodb://%s:%s", mongo_host, mongo_port)
	redis_uri  = fmt.Sprintf("redis://%s:%s/0", redis_host, redis_port)
)

func insertDoc(mongoClient *mongo.Client, post BlogPost) (*mongo.InsertOneResult, error) {
	coll := mongoClient.Database(databaseName).Collection(collectionName)
	result, err := coll.InsertOne(context.TODO(), post)

	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to redis")
		return nil, err
	}
	return result, err
}

func main() {
	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(mongo_uri))
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to mongo")
	}
	ctx := context.Background()
	err = mongoClient.Connect(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to mongo")
	}
	defer mongoClient.Disconnect(ctx)
	opt, err := redis.ParseURL(redis_uri)
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to redis")
	}
	rdb := redis.NewClient(opt)
	for {
		result, err := rdb.BLPop(ctx, 0, "queue:new-post").Result()
		if err != nil {
			log.Error().Err(err).Msg("error occured while reading from redis")
			continue
		}

		post := BlogPost{}
		err = json.Unmarshal([]byte(result[1]), &post)
		if err != nil {
			log.Error().Err(err).Msg("error occured while decoding response into Post object")
		}
		insertDoc(mongoClient, post)
		fmt.Println(result)

	}
}
