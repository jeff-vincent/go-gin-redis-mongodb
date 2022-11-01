package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MONGO1_HOST = os.Getenv("MONGO1_HOST")
var MONGO1_PORT = os.Getenv("MONGO1_PORT")
var REDIS_HOST = os.Getenv("REDIS_HOST")
var REDIS_PORT = os.Getenv("REDIS_PORT")

func getDoc(client mongo.Client, title string) bson.D {
	coll := client.Database("blog").Collection("posts")
	var result bson.D
	err := coll.FindOne(context.TODO(), bson.D{{"title", title}}).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}
	Publish(title)
	return result
}

func Publish(payload string) {
	redis_uri := fmt.Sprintf("redis://%s:%s/0", REDIS_HOST, REDIS_PORT)
	opt, err := redis.ParseURL(redis_uri)
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(opt)
	ctx := context.Background()
	err = rdb.Publish(ctx, "Analytics", payload).Err()
	if err != nil {
		panic(err)
	}

}

func main() {
	mongo_uri := fmt.Sprintf("mongodb://%s:%s", MONGO1_HOST, MONGO1_PORT)
	client, err := mongo.NewClient(options.Client().ApplyURI(mongo_uri))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)
	r := gin.Default()

	r.GET("/get-doc", func(c *gin.Context) {
		title := c.Query("title")
		result := getDoc(*client, title)

		c.JSON(http.StatusOK, gin.H{
			"Data": result,
		})
	})

	r.Run(":8082")
}
