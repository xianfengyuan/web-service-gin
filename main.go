package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// album represents data about a record album.
type album struct {
	ID     string  `json:"id"`
	Title  string  `json:"title"`
	Artist string  `json:"artist"`
	Price  float64 `json:"price"`
}

type appConfig struct {
	URI string `json:"uri"`
}

// albums slice to seed record album data.
var albums = []album{}
var client *mongo.Client
var db string
var cname string

func main() {
	db = "media"
	cname = "albums"
	client, albums = init_db(db, cname)

	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	router := gin.Default()
	router.GET("/albums", getAlbums)
	router.GET("/albums/:id", getAlbumByID)
	router.POST("/albums", postAlbums)
	router.DELETE("/albums/:id", deleteAlbumByID)

	router.Run(":8080")
}

func init_db(db string, c string) (*mongo.Client, []album) {
	path, present := os.LookupEnv("CONFIG_PATH")
	if !present {
		path = "./"
	}
	configFile, err := os.Open(path + "config.json")
	if err != nil {
		panic("Could not open config file: " + err.Error())
	}
	defer configFile.Close()
	var cfg appConfig
	if err := json.NewDecoder(configFile).Decode(&cfg); err != nil {
		panic("Could not decode config file: " + err.Error())
	}

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(cfg.URI).SetServerAPIOptions(serverAPI)
	client, err := mongo.Connect(opts)
	if err != nil {
		panic(err)
	}

	// Send a ping to confirm a successful connection
	if err := client.Ping(context.TODO(), readpref.Primary()); err != nil {
		panic(err)
	}
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	var albums = []album{}
	collection := client.Database(db).Collection(c)
	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		panic("Could not find albums in MongoDB: " + err.Error())
	}
	if err := cursor.All(context.TODO(), &albums); err != nil {
		panic("Could not decode albums from MongoDB: " + err.Error())
	}
	for _, album := range albums {
		fmt.Printf("Album: %s by %s, Price: %.2f\n", album.Title, album.Artist, album.Price)
	}

	return client, albums
}

func getAlbums(c *gin.Context) {
	c.IndentedJSON(http.StatusOK, albums)
}

func postAlbums(c *gin.Context) {
	var newAlbum album

	// Call BindJSON to bind the received JSON to newAlbum.
	if err := c.BindJSON(&newAlbum); err != nil {
		return
	}

	// Add the new album to the slice.
	collection := client.Database(db).Collection(cname)
	_, err := collection.InsertOne(context.TODO(), newAlbum)
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "could not insert album"})
		return
	}
	albums = append(albums, newAlbum)
	c.IndentedJSON(http.StatusCreated, newAlbum)
}

func getAlbumByID(c *gin.Context) {
	id := c.Param("id")

	// Loop through the list of albums, looking for
	// an album whose ID value matches the parameter.
	for _, a := range albums {
		if a.ID == id {
			c.IndentedJSON(http.StatusOK, a)
			return
		}
	}

	// If not found, return a 404.
	c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
}

func deleteAlbumByID(c *gin.Context) {
	id := c.Param("id")

	// Find the album by ID and delete it
	collection := client.Database(db).Collection(cname)
	result, err := collection.DeleteOne(context.TODO(), bson.M{"id": id})
	if err != nil {
		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "could not delete album"})
		return
	}

	if result.DeletedCount == 0 {
		c.IndentedJSON(http.StatusNotFound, gin.H{"message": "album not found"})
		return
	}

	// Remove the album from the slice
	for i, a := range albums {
		if a.ID == id {
			if i < 1 {
				albums = albums[1:]
			} else if i == len(albums)-1 {
				albums = albums[:len(albums)-1]
			} else {
				albums = append(albums[:i], albums[i+1:]...)
			}
			break
		}
	}
	c.IndentedJSON(http.StatusOK, gin.H{"message": "album deleted"})
}
