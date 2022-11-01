package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BlogPost struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Body   string `json:"body"`
}

var MONGO1_HOST = os.Getenv("MONGO1_HOST")
var MONGO1_PORT = os.Getenv("MONGO1_PORT")
var REDIS_HOST = os.Getenv("REDIS_HOST")
var REDIS_PORT = os.Getenv("REDIS_PORT")

func insertDoc(client *mongo.Client, post BlogPost) *mongo.InsertOneResult {
	coll := client.Database("blog").Collection("posts")
	res, err := coll.InsertOne(context.TODO(), post)

	if err != nil {
		log.Fatal(err)
	}
	return res
}

func main() {
	mongo_uri := fmt.Sprintf("mongodb://%s:%s", MONGO1_HOST, MONGO1_PORT)
	client, err := mongo.NewClient(options.Client().ApplyURI(mongo_uri))
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	redis_uri := fmt.Sprintf("redis://%s:%s/0", REDIS_HOST, REDIS_PORT)
	opt, err := redis.ParseURL(redis_uri)
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(opt)
	pubsub := rdb.Subscribe(ctx, "Upload")
	defer pubsub.Close()
	ch := pubsub.Channel()
	for msg := range ch {
		post := BlogPost{}
		if err := json.Unmarshal([]byte(msg.Payload), &post); err != nil {
			panic(err)
		}
		insertDoc(client, post)
		fmt.Println(post)

	}
}
