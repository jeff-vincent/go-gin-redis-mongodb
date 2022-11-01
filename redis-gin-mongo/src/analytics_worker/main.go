package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
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

func getDoc(title string, mongo2 *mongo.Client) AnalyticsData {
	coll := mongo2.Database(databaseName).Collection(collectionName)
	var result AnalyticsData
	err := coll.FindOne(context.TODO(), bson.D{{"title", title}}).Decode(&result)
	if err != nil {
		log.Print(err)
	}
	return result
}

func insertDoc(title string, mongo2 *mongo.Client) *mongo.InsertOneResult {
	coll := mongo2.Database(databaseName).Collection(collectionName)
	data := AnalyticsData{Title: title, Views: 1}
	res, err := coll.InsertOne(context.TODO(), data)

	if err != nil {
		panic(err)
	}
	return res
}

func updateAnalytics(title string, mongo2 *mongo.Client) {
	existingDoc := getDoc(title, mongo2)
	if existingDoc.Title == "" {
		insertDoc(title, mongo2)
	} else {
		views := existingDoc.Views + 1
		coll := mongo2.Database("blog").Collection("views")
		result, err := coll.UpdateOne(
			context.TODO(),
			bson.M{"title": existingDoc.Title},
			bson.D{
				{"$set", bson.D{{"views", views}}},
			},
		)
		if err != nil {
			panic(err)
		}
		fmt.Println(result)
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
	opt, err := redis.ParseURL(REDIS_URI)
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(opt)
	pubsub := rdb.Subscribe(ctx, "Analytics")
	defer pubsub.Close()
	ch := pubsub.Channel()
	for msg := range ch {
		title := msg.Payload
		updateAnalytics(title, mongo2)
	}
}
