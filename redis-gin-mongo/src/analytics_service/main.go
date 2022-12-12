package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	databaseName   = "blog"
	collectionName = "views"
)

var (
	mongo_host = os.Getenv("MONGO2_HOST")
	mongo_port = os.Getenv("MONGO2_PORT")
	mongo_uri  = fmt.Sprintf("mongodb://%s:%s", mongo_host, mongo_port)
)

func getAnalyticsDataByTitle(ctx *gin.Context, mongoClient *mongo.Client, title string) bson.D {
	coll := mongoClient.Database(databaseName).Collection(collectionName)
	var result bson.D
	err := coll.FindOne(ctx, bson.D{{"title", title}}).Decode(&result)
	if err != nil {
		log.Error().Err(err).Msg("error occured while connecting to redis")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "user friendly error - without exposing internal logic"})
	}
	return result
}

func getAllBlogViews(ctx *gin.Context, mongoClient *mongo.Client) []bson.M {
	coll := mongoClient.Database(databaseName).Collection(collectionName)
	cursor, err := coll.Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}
	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		log.Error().Err(err).Msg("error occured while connecting to redis")
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "user friendly error - without exposing internal logic"})
	}
	return results
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

	router.GET("/views/:title", func(ctx *gin.Context) {
		title := ctx.Param("title")
		result := getAnalyticsDataByTitle(ctx, mongoClient, title)
		ctx.JSON(http.StatusOK, gin.H{
			"Data": result,
		})
	})

	router.GET("/views", func(ctx *gin.Context) {
		result := getAllBlogViews(ctx, mongoClient)
		ctx.JSON(http.StatusOK, gin.H{
			"Data": result,
		})
	})
	router.Run(":8082")
}
