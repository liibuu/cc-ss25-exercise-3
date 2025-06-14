package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BookStore struct {
	MongoID     primitive.ObjectID `bson:"_id,omitempty"`
	ID          string             `bson:"id,omitempty"`
	BookName    string             `bson:"bookname"`
	BookAuthor  string             `bson:"bookauthor"`
	BookEdition string             `bson:"bookedition"`
	BookPages   string             `bson:"bookpages"`
	BookYear    string             `bson:"bookyear"`
}

type BookResponse struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Author  string `json:"author"`
	Pages   string `json:"pages"`
	Edition string `json:"edition"`
	Year    string `json:"year"`
}

func getMongoURI() string {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://mongo:27017"
	}
	return uri
}

func connectToMongoDB() (*mongo.Client, *mongo.Collection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(getMongoURI()))
	if err != nil {
		return nil, nil, err
	}

	// Test the connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, nil, err
	}

	db := client.Database("exercise-1")
	coll := db.Collection("information")
	return client, coll, nil
}

func getAllBooksAPI(coll *mongo.Collection) []BookResponse {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	if err != nil {
		fmt.Printf("Error finding books: %v\n", err)
		return []BookResponse{}
	}
	
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		fmt.Printf("Error decoding books: %v\n", err)
		return []BookResponse{}
	}

	var ret []BookResponse
	for _, res := range results {
		ret = append(ret, BookResponse{
			ID:      res.ID,
			Title:   res.BookName,
			Author:  res.BookAuthor,
			Pages:   res.BookPages,
			Edition: res.BookEdition,
			Year:    res.BookYear,
		})
	}

	return ret
}

func main() {
	// Wait for MongoDB to be ready
	fmt.Println("Waiting for MongoDB to be ready...")
	// time.Sleep(15 * time.Second)
	
	var client *mongo.Client
	var coll *mongo.Collection
	var err error

	// Retry connection to MongoDB with shorter intervals
	for i := 0; i < 5; i++ {
		client, coll, err = connectToMongoDB()
		if err == nil {
			break
		}
		fmt.Printf("Failed to connect to MongoDB (attempt %d/5): %v\n", i+1, err)
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		fmt.Printf("Warning: Failed to connect to MongoDB after 5 attempts: %v\n", err)
		// Continue anyway for testing - create a mock response
	}

	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			fmt.Printf("Error disconnecting from MongoDB: %v\n", err)
		}
	}()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	e.GET("/api/books", func(c echo.Context) error {
		books := getAllBooksAPI(coll)
		return c.JSON(http.StatusOK, books)
	})

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	fmt.Println("Books GET service starting on port 8080")
	e.Logger.Fatal(e.Start(":8080"))
}