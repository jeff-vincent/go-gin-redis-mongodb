package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MONGO2_HOST = os.Getenv("MONGO2_HOST")
var MONGO2_PORT = os.Getenv("MONGO2_PORT")

func getDoc(client mongo.Client, title string) bson.D {
	coll := client.Database("blog").Collection("views")
	var result bson.D
	err := coll.FindOne(context.TODO(), bson.D{{"title", title}}).Decode(&result)
	if err != nil {
		log.Fatal(err)
	}
	return result
}

func getAllDocs(client mongo.Client) []bson.M {
	coll := client.Database("blog").Collection("views")
	cursor, err := coll.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	var results []bson.M
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}
	return results
}

func main() {
	mongo_uri := fmt.Sprintf("mongodb://%s:%s", MONGO2_HOST, MONGO2_PORT)
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

	r.GET("/get-all-docs", func(c *gin.Context) {
		result := getAllDocs(*client)
		c.JSON(http.StatusOK, gin.H{
			"Data": result,
		})
	})
	r.Run()
}
