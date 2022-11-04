package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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

func getPost(ctx *gin.Context) {
	title := ctx.Query("title")
	address := fmt.Sprintf("http://%s:%s/get-post?title=%s", BLOG_SERVICE_HOST, BLOG_SERVICE_PORT, title)
	resp, _ := http.Get(address)
	defer resp.Body.Close()
	val := &Doc{}
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(val)

	if err != nil {
		log.Print(err)
	}
	ctx.JSON(http.StatusOK, val)
}

func getAllPosts(ctx *gin.Context) {
	address := fmt.Sprintf("http://%s:%s/get-all-posts", BLOG_SERVICE_HOST, BLOG_SERVICE_PORT)
	resp, _ := http.Get(address)
	defer resp.Body.Close()
	val := &Docs{}
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(val)
	if err != nil {
		log.Print(err)
	}
	ctx.JSON(http.StatusOK, val)
}

func index(ctx *gin.Context) {
	ctx.HTML(http.StatusOK, "index.html", gin.H{})
}

func newPost(ctx *gin.Context, t string, a string, b string) {
	ctx.HTML(http.StatusOK, "post.html", gin.H{
		"title":  t,
		"author": a,
		"body":   b,
	})
}

func getPostViews(ctx *gin.Context) {
	title := ctx.Query("title")
	address := fmt.Sprintf("http://%s:%s/get-analytics-data-by-title?title=%s", ANALYTICS_SERVICE_HOST, ANALYTICS_SERVICE_PORT, title)
	resp, _ := http.Get(address)
	defer resp.Body.Close()
	val := &Doc{}
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(val)

	if err != nil {
		log.Print(err)
	}
	ctx.JSON(http.StatusOK, val)
}

func getAllViews(ctx *gin.Context) {
	address := fmt.Sprintf("http://%s:%s/get-all-blog-views", ANALYTICS_SERVICE_HOST, ANALYTICS_SERVICE_PORT)
	resp, _ := http.Get(address)
	defer resp.Body.Close()
	val := &Docs{}
	decoder := json.NewDecoder(resp.Body)
	err := decoder.Decode(val)

	if err != nil {
		log.Print(err)
	}
	ctx.JSON(http.StatusOK, val)
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
		err = json.Unmarshal(payload, &new_post)
		if err != nil {
			log.Print(err)
		}
		ctx := context.Background()
		if err := rdb.RPush(ctx, "Upload", payload).Err(); err != nil {
			panic(err)
		}

		newPost(c, title, author, body)
	})
	router.GET("/post", getPost)
	router.GET("/posts", getAllPosts)
	router.GET("/views", getPostViews)
	router.GET("/all-views", getAllViews)
	router.Run(":8081")
}
