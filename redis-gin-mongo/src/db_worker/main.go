package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/thoas/bokchoy"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = mongo1.Connect(ctx)
	if err != nil {
		log.Print(err)
	}
	defer mongo1.Disconnect(ctx)
	engine, err := bokchoy.New(ctx, bokchoy.Config{
		Broker: bokchoy.BrokerConfig{
			Type: "redis",
			Redis: bokchoy.RedisConfig{
				Type: "client",
				Client: bokchoy.RedisClientConfig{
					Addr: "localhost:6379",
				},
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	engine.Queue("Upload").HandleFunc(func(r *bokchoy.Request) error {
		fmt.Printf("%T", r.Task.Payload)
		post := BlogPost{}
		payload, _ := json.Marshal(r.Task.Payload)
		err := json.Unmarshal(payload, &post)

		if err != nil {
			log.Print(err)
		}
		insertDoc(mongo1, post)

		return nil
	})
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for range c {
			log.Print("Received signal, gracefully stopping")
			engine.Stop(ctx)
		}
	}()

	engine.Run(ctx)

}
