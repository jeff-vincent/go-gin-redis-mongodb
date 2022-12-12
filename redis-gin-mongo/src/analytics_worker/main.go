package main

import (
	"context"
	"fmt"
	"os"

	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	databaseName     = "blog"
	collectionName   = "views"
	resultKeyIndex   = 0
	resultValueIndex = 1
)

var (
	mongo_host = os.Getenv("MONGO2_HOST")
	mongo_port = os.Getenv("MONGO2_PORT")
	redis_host = os.Getenv("REDIS_HOST")
	redis_port = os.Getenv("REDIS_PORT")
	mongo_uri  = fmt.Sprintf("mongodb://%s:%s", mongo_host, mongo_port)
	redis_uri  = fmt.Sprintf("redis://%s:%s/0", redis_host, redis_port)
)

type AnalyticsData struct {
	Title string
	Views int
}

func getDoc(mongoClient *mongo.Client, title string) (AnalyticsData, error) {
	coll := mongoClient.Database(databaseName).Collection(collectionName)
	var result AnalyticsData
	err := coll.FindOne(context.TODO(), bson.D{{"title", title}}).Decode(&result)
	if err != nil {
		log.Error().Err(err).Msg("error occured while fetching post from mongo")
		return result, err
	}
	return result, err
}

func insertDoc(mongoClient *mongo.Client, title string) (*mongo.InsertOneResult, error) {
	coll := mongoClient.Database(databaseName).Collection(collectionName)
	data := AnalyticsData{Title: title, Views: 1}
	result, err := coll.InsertOne(context.TODO(), data)

	if err != nil {
		log.Error().Err(err).Msg("error occured while inserting post to mongo")
		return result, err
	}
	return result, err
}

func updateAnalytics(mongoClient *mongo.Client, title string) {
	existingDoc, err := getDoc(mongoClient, title)
	if err != nil {
		log.Error().Err(err).Msg("error occured while fetching post from mongo")
	}
	if existingDoc.Title == "" {
		insertDoc(mongoClient, title)
	} else {
		views := existingDoc.Views + 1
		coll := mongoClient.Database("blog").Collection("views")
		_, err := coll.UpdateOne(
			context.TODO(),
			bson.M{"title": existingDoc.Title},
			bson.D{
				{"$set", bson.D{{"views", views}}},
			},
		)
		if err != nil {
			log.Error().Err(err).Msg("error occured while updating analytics")
		}
	}
}

func main() {
	ctx := context.Background()
	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(mongo_uri))
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to mongo")
	}
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
		result, err := rdb.BLPop(ctx, 0, "queue:blog-view").Result()
		if err != nil {
			log.Error().Err(err).Msg("error occured while fetching data from posts redis")
			continue
		}
		updateAnalytics(mongoClient, result[1])
		fmt.Println(result[resultKeyIndex], result[resultValueIndex])
	}
}
