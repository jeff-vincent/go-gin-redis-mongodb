package main

import (
	"context"
	"fmt"
	"log"
	"os"

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
	ctx := context.Background()
	mongo2, err := mongo.NewClient(options.Client().ApplyURI(MONGO2_URI))
	if err != nil {
		panic(err)
	}
	err = mongo2.Connect(ctx)
	if err != nil {
		panic(err)
	}
	defer mongo2.Disconnect(ctx)
	// Rpush Blpop

	opt, err := redis.ParseURL(REDIS_URI)
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(opt)
	for {
		// use `rdb.BLPop(0, "queue")` for infinite waiting time
		result, err := rdb.BLPop(ctx, 0, "Analytics").Result()
		if err != nil {
			panic(err)
		}
		updateAnalytics(mongo2, result[1])
		fmt.Println(result[0], result[1])
	}

}
