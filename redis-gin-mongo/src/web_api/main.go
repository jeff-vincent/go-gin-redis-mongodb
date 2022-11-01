package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/bson"
)

type BlogPost struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Body   string `json:"body"`
}

type Doc struct {
	Data bson.D `json:"data"`
}

type Docs struct {
	Data []bson.M `json:"data"`
}

var (
	REDIS_HOST             = os.Getenv("REDIS_HOST")
	REDIS_PORT             = os.Getenv("REDIS_PORT")
	ANALYTICS_SERVICE_HOST = os.Getenv("ANALYTICS_SERVICE_HOST")
	ANALYTICS_SERVICE_PORT = os.Getenv("ANALYTICS_SERVICE_PORT")
	BLOG_SERVICE_HOST      = os.Getenv("BLOG_SERVICE_HOST")
	BLOG_SERVICE_PORT      = os.Getenv("BLOG_SERVICE_PORT")
	REDIS_URI              = fmt.Sprintf("redis://%s:%s/0", REDIS_HOST, REDIS_PORT)
)

func getPost(c *gin.Context) {
	title := c.Query("title")
	address := fmt.Sprintf("http://%s:%s/get-post?title=%s", BLOG_SERVICE_HOST, BLOG_SERVICE_PORT, title)
	resp, _ := http.Get(address)
	defer resp.Body.Close()
	val := &Doc{}
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(val)

	if err != nil {
		panic(err)
	}
	c.JSON(http.StatusOK, val)
}

func getAllPosts(c *gin.Context) {
	address := fmt.Sprintf("http://%s:%s/get-all-posts", BLOG_SERVICE_HOST, BLOG_SERVICE_PORT)
	resp, _ := http.Get(address)
	defer resp.Body.Close()
	val := &Docs{}
	fmt.Println(resp.Body)
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(val)
	if err != nil {
		fmt.Println(err)
	}
	c.JSON(http.StatusOK, val)
}

func index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}

func newPost(c *gin.Context, t string, a string, b string) {
	c.HTML(http.StatusOK, "post.html", gin.H{
		"title":  t,
		"author": a,
		"body":   b,
	})
}

func getPostViews(c *gin.Context) {
	title := c.Query("title")
	address := fmt.Sprintf("http://%s:%s/get-analytics-data-by-title?title=%s", ANALYTICS_SERVICE_HOST, ANALYTICS_SERVICE_PORT, title)
	resp, _ := http.Get(address)
	defer resp.Body.Close()
	val := &Doc{}
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(val)

	if err != nil {
		fmt.Println(err)
	}
	c.JSON(http.StatusOK, val)
}

func main() {
	opt, err := redis.ParseURL(REDIS_URI)
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(opt)
	router := gin.Default()
	router.LoadHTMLGlob("templates/*.html")
	router.GET("/", index)
	router.POST("/post", func(c *gin.Context) {
		title := c.PostForm("title")
		author := c.PostForm("author")
		body := c.PostForm("body")
		new_post := BlogPost{Title: title, Author: author, Body: body}
		payload, err := json.Marshal(new_post)
		if err != nil {
			panic(err)
		}
		ctx := context.Background()
		err = rdb.Publish(ctx, "Upload", payload).Err()
		if err != nil {
			panic(err)
		}
		newPost(c, title, author, body)
	})
	router.GET("/post", getPost)
	router.GET("/posts", getAllPosts)
	router.GET("/views", getPostViews)
	router.Run(":8081")
}
