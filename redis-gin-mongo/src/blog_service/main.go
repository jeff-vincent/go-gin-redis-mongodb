package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongo_host = os.Getenv("MONGO1_HOST")
	mongo_port = os.Getenv("MONGO1_PORT")
	redis_host = os.Getenv("REDIS_HOST")
	redis_port = os.Getenv("REDIS_PORT")
	mongo_uri  = fmt.Sprintf("mongodb://%s:%s", mongo_host, mongo_port)
	redis_uri  = fmt.Sprintf("redis://%s:%s/0", redis_host, redis_port)
)

const (
	databaseName   = "blog"
	collectionName = "posts"
)

func getPost(ctx *gin.Context, mongoClient *mongo.Client, title string) (bson.D, error) {
	coll := mongoClient.Database(databaseName).Collection(collectionName)
	var result bson.D
	err := coll.FindOne(ctx, bson.D{{"title", title}}).Decode(&result)
	if err != nil {
		log.Error().Err(err).Msg("error occured while fetching posts from posts mongo")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Get post failed"})
		return nil, err
	}
	Publish(ctx, title)
	return result, err
}

func Publish(ctx *gin.Context, payload string) {
	opt, err := redis.ParseURL(redis_uri)
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to redis")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Analytics error"})
		return
	}
	rdb := redis.NewClient(opt)
	if err := rdb.RPush(ctx, "queue:blog-view", payload).Err(); err != nil {
		log.Error().Err(err).Msg("error occured while publishing to redis")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Analytics error"})
		return
	}
}

func main() {
	mongoClient, err := mongo.NewClient(options.Client().ApplyURI(mongo_uri))
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to mongo")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = mongoClient.Connect(ctx)
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to mongo")
	}
	defer mongoClient.Disconnect(ctx)
	router := gin.Default()

	router.GET("/posts/:title", func(ctx *gin.Context) {
		title := ctx.Param("title")
		result, err := getPost(ctx, mongoClient, title)
		if err != nil {
			log.Error().Err(err).Msg("error occured while fetching post from mongo")
		}
		ctx.JSON(http.StatusOK, gin.H{
			"Data": result,
		})
	})
	router.Run()
}
