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

const (
	databaseName   = "blog"
	collectionName = "posts"
)

var (
	MONGO1_HOST = os.Getenv("MONGO1_HOST")
	MONGO1_PORT = os.Getenv("MONGO1_PORT")
	REDIS_HOST  = os.Getenv("REDIS_HOST")
	REDIS_PORT  = os.Getenv("REDIS_PORT")
	MONGO1_URI  = fmt.Sprintf("mongodb://%s:%s", MONGO1_HOST, MONGO1_PORT)
	REDIS_URI   = fmt.Sprintf("redis://%s:%s/0", REDIS_HOST, REDIS_PORT)
)

func insertDoc(mongo1 *mongo.Client, post BlogPost) *mongo.InsertOneResult {
	coll := mongo1.Database(databaseName).Collection(collectionName)
	res, err := coll.InsertOne(context.TODO(), post)

	if err != nil {
		log.Print(err)
	}
	return res
}

func main() {
	mongo1, err := mongo.NewClient(options.Client().ApplyURI(MONGO1_URI))
	if err != nil {
		log.Print(err)
	}
	ctx := context.Background()
	err = mongo1.Connect(ctx)
	if err != nil {
		log.Print(err)
	}
	defer mongo1.Disconnect(ctx)
	opt, err := redis.ParseURL(REDIS_URI)
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(opt)
	for {
		result, err := rdb.BLPop(ctx, 0, "Upload").Result()
		if err != nil {
			panic(err)
		}

		post := BlogPost{}
		err = json.Unmarshal([]byte(result[1]), &post)
		if err != nil {
			log.Print(err)
		}
		insertDoc(mongo1, post)
		fmt.Println(result)

	}
}
