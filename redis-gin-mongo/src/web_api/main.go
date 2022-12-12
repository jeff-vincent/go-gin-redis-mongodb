package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/rs/zerolog/log"
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
	redis_host             = os.Getenv("REDIS_HOST")
	redis_port             = os.Getenv("REDIS_PORT")
	analytics_service_host = os.Getenv("ANALYTICS_SERVICE_HOST")
	analytics_service_port = os.Getenv("ANALYTICS_SERVICE_PORT")
	blog_service_host      = os.Getenv("BLOG_SERVICE_HOST")
	blog_service_port      = os.Getenv("BLOG_SERVICE_PORT")
	redis_uri              = fmt.Sprintf("redis://%s:%s/0", redis_host, redis_port)
)

func getPost(ctx *gin.Context) {
	title := ctx.Param("title")
	address := fmt.Sprintf("http://%s:%s/posts/%s", blog_service_host, blog_service_port, title)
	resp, err := http.Get(address)
	if err != nil {
		log.Error().Err(err).Msg("error occured while fetching posts from posts service")                                 // this log will get stored internally
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "user friendly error - without exposing internal logic"}) // you have the granullarity for choosing your own status code (instead of panicing)
	}
	defer resp.Body.Close()
	val := &Doc{}
	decoder := json.NewDecoder(resp.Body) // I will usually also check the status code of the response before hand (like for 404 error for example)
	err = decoder.Decode(val)
	if err != nil {
		log.Error().Err(err).Msg("error occured while decoding response into Doc object") // this log will get stored internally - usually you also log additional data like resp.Body etc..
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "user friendly error - without exposing internal logic"})
	}
	ctx.JSON(http.StatusOK, val)
}

func getAllPosts(ctx *gin.Context) {
	address := fmt.Sprintf("http://%s:%s/posts", blog_service_host, blog_service_port)
	resp, err := http.Get(address)
	if err != nil {
		log.Error().Err(err).Msg("error occured while fetching posts from posts service")                                 // this log will get stored internally
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "user friendly error - without exposing internal logic"}) // you have the granullarity for choosing your own status code (instead of panicing)
	}
	defer resp.Body.Close()
	val := &Docs{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(val)
	if err != nil {
		log.Error().Err(err).Msg("error occured while decoding response into Doc object") // this log will get stored internally - usually you also log additional data like resp.Body etc..
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "user friendly error - without exposing internal logic"})
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
	title := ctx.Param("title")
	address := fmt.Sprintf("http://%s:%s/views/%s", analytics_service_host, analytics_service_port, title)
	resp, err := http.Get(address)
	if err != nil {
		log.Error().Err(err).Msg("error occured while fetching views from views service")                                 // this log will get stored internally
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "user friendly error - without exposing internal logic"}) // you have the granullarity for choosing your own status code (instead of panicing)
	}
	defer resp.Body.Close()
	val := &Doc{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(val)
	if err != nil {
		log.Error().Err(err).Msg("error occured while decoding response into Doc object") // this log will get stored internally - usually you also log additional data like resp.Body etc..
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "user friendly error - without exposing internal logic"})
	}
	ctx.JSON(http.StatusOK, val)
}

func getAllViews(ctx *gin.Context) {
	address := fmt.Sprintf("http://%s:%s/views", analytics_service_host, analytics_service_port)
	resp, err := http.Get(address)
	if err != nil {
		log.Error().Err(err).Msg("error occured while fetching views from views service")                                 // this log will get stored internally
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "user friendly error - without exposing internal logic"}) // you have the granullarity for choosing your own status code (instead of panicing)
	}
	defer resp.Body.Close()
	val := &Docs{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(val)
	if err != nil {
		log.Error().Err(err).Msg("error occured while decoding response into Doc object") // this log will get stored internally - usually you also log additional data like resp.Body etc..
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "user friendly error - without exposing internal logic"})
	}
	ctx.JSON(http.StatusOK, val)
}

func main() {
	opt, err := redis.ParseURL(redis_uri)
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(opt)

	router := gin.Default()
	router.LoadHTMLGlob("templates/*.html")
	router.GET("/", index)
	router.POST("/posts", func(ctx *gin.Context) {
		title := ctx.PostForm("title")
		author := ctx.PostForm("author")
		body := ctx.PostForm("body")
		new_post := BlogPost{Title: title, Author: author, Body: body}
		payload, err := json.Marshal(new_post)
		if err != nil {
			log.Error().Err(err).Msg("error occured while decoding response into Doc object") // this log will get stored internally - usually you also log additional data like resp.Body etc..
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "user friendly error - without exposing internal logic"})
		}
		err = json.Unmarshal(payload, &new_post)
		if err != nil {
			log.Error().Err(err).Msg("error occured while decoding response into Doc object") // this log will get stored internally - usually you also log additional data like resp.Body etc..
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "user friendly error - without exposing internal logic"})
		}
		if err := rdb.RPush(ctx, "queue:new-post", payload).Err(); err != nil {
			log.Error().Err(err).Msg("error occured while decoding response into Doc object") // this log will get stored internally - usually you also log additional data like resp.Body etc..
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "user friendly error - without exposing internal logic"})
		}

		newPost(ctx, title, author, body)
	})
	router.GET("/posts/:title", getPost)
	router.GET("/posts", getAllPosts)
	router.GET("/views/:title", getPostViews)
	router.GET("/views", getAllViews)
	router.Run(":8081")
}
