package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
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
	MONGO2_URI  = fmt.Sprintf("mongodb://%s:%s", MONGO2_HOST, MONGO2_PORT)
)

func getAnalyticsDataByTitle(ctx *gin.Context, mongo2 *mongo.Client, title string) bson.D {
	coll := mongo2.Database(databaseName).Collection(collectionName)
	var result bson.D
	err := coll.FindOne(ctx, bson.D{{"title", title}}).Decode(&result)
	if err != nil {
		panic(err)
	}
	return result
}

func getAllBlogViews(ctx *gin.Context, mongo2 *mongo.Client) []bson.M {
	coll := mongo2.Database(databaseName).Collection(collectionName)
	cursor, err := coll.Find(context.TODO(), bson.D{})
	if err != nil {
		panic(err)
	}
	var results []bson.M
	if err = cursor.All(ctx, &results); err != nil {
		panic(err)
	}
	return results
}

func main() {
	mongo2, err := mongo.NewClient(options.Client().ApplyURI(MONGO2_URI))
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = mongo2.Connect(ctx)
	if err != nil {
		panic(err)
	}
	defer mongo2.Disconnect(ctx)
	router := gin.Default()

	router.GET("/get-analytics-data-by-title", func(ctx *gin.Context) {
		title := ctx.Query("title")
		result := getAnalyticsDataByTitle(ctx, mongo2, title)

		ctx.JSON(http.StatusOK, gin.H{
			"Data": result,
		})
	})

	router.GET("/get-all-blog-views", func(ctx *gin.Context) {
		result := getAllBlogViews(ctx, mongo2)
		ctx.JSON(http.StatusOK, gin.H{
			"Data": result,
		})
	})
	router.Run()
}
