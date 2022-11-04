package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	MONGO1_HOST = os.Getenv("MONGO1_HOST")
	MONGO1_PORT = os.Getenv("MONGO1_PORT")
	REDIS_HOST  = os.Getenv("REDIS_HOST")
	REDIS_PORT  = os.Getenv("REDIS_PORT")
	MONGO1_URI  = fmt.Sprintf("mongodb://%s:%s", MONGO1_HOST, MONGO1_PORT)
	REDIS_URI   = fmt.Sprintf("redis://%s:%s/0", REDIS_HOST, REDIS_PORT)
)

const (
	databaseName   = "blog"
	collectionName = "posts"
)

func getPost(ctx *gin.Context, mongo1 *mongo.Client, title string) bson.D {
	coll := mongo1.Database(databaseName).Collection(collectionName)
	var result bson.D
	err := coll.FindOne(context.TODO(), bson.D{{"title", title}}).Decode(&result)
	if err != nil {
		panic(err)
	}
	Publish(title)
	return result
}

func Publish(payload string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Rpush Blpop
	opt, err := redis.ParseURL(REDIS_URI)
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(opt)
	if err := rdb.RPush(ctx, "Analytics", payload).Err(); err != nil {
		panic(err)
	}

}

func main() {
	mongo1, err := mongo.NewClient(options.Client().ApplyURI(MONGO1_URI))
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = mongo1.Connect(ctx)
	if err != nil {
		panic(err)
	}
	defer mongo1.Disconnect(ctx)
	router := gin.Default()

	router.GET("/get-post", func(ctx *gin.Context) {
		title := ctx.Query("title")
		result := getPost(ctx, mongo1, title)
		ctx.JSON(http.StatusOK, gin.H{
			"Data": result,
		})
	})
	router.Run()
}
