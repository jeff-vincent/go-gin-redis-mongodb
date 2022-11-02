package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/thoas/bokchoy"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	databaseName   = "blog"
	collectionName = "views"
)

var (
	MONGO2_HOST = os.Getenv("MONGO2_HOST")
	MONGO2_PORT = os.Getenv("MONGO2_PORT")
	REDIS_HOST  = os.Getenv("REDIS_HOST")
	REDIS_PORT  = os.Getenv("REDIS_PORT")
	MONGO2_URI  = fmt.Sprintf("mongodb://%s:%s", MONGO2_HOST, MONGO2_PORT)
	REDIS_URI   = fmt.Sprintf("redis://%s:%s/0", REDIS_HOST, REDIS_PORT)
)

type AnalyticsData struct {
	Title string
	Views int
}

func getDoc(mongo2 *mongo.Client, title string) AnalyticsData {
	coll := mongo2.Database(databaseName).Collection(collectionName)
	var result AnalyticsData
	err := coll.FindOne(context.TODO(), bson.D{{"title", title}}).Decode(&result)
	if err != nil {
		log.Print(err)
	}
	return result
}

func insertDoc(mongo2 *mongo.Client, title string) *mongo.InsertOneResult {
	coll := mongo2.Database(databaseName).Collection(collectionName)
	data := AnalyticsData{Title: title, Views: 1}
	result, err := coll.InsertOne(context.TODO(), data)

	if err != nil {
		log.Print(err)
	}
	return result
}

func updateAnalytics(mongo2 *mongo.Client, title string) {
	existingDoc := getDoc(mongo2, title)
	if existingDoc.Title == "" {
		insertDoc(mongo2, title)
	} else {
		views := existingDoc.Views + 1
		coll := mongo2.Database("blog").Collection("views")
		_, err := coll.UpdateOne(
			context.TODO(),
			bson.M{"title": existingDoc.Title},
			bson.D{
				{"$set", bson.D{{"views", views}}},
			},
		)
		if err != nil {
			log.Print(err)
		}
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongo2, err := mongo.NewClient(options.Client().ApplyURI(MONGO2_URI))
	if err != nil {
		panic(err)
	}
	err = mongo2.Connect(ctx)
	if err != nil {
		panic(err)
	}
	defer mongo2.Disconnect(ctx)
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
	engine.Queue("Analytics").HandleFunc(func(r *bokchoy.Request) error {
		resultString := fmt.Sprintf("%v", r.Task.Payload)
		updateAnalytics(mongo2, resultString)

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
