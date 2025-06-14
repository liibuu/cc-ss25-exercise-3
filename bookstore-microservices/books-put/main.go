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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type BookRequest struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Author  string `json:"author"`
	Pages   string `json:"pages,omitempty"`
	Edition string `json:"edition,omitempty"`
	Year    string `json:"year,omitempty"`
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

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, nil, err
	}

	db := client.Database("exercise-1")
	coll := db.Collection("information")
	return client, coll, nil
}

func updateBook(coll *mongo.Collection, id string, bookReq BookRequest) error {
	// Find the book by ID (custom ID, not MongoDB _id)
	filter := bson.M{"id": id}
	
	// Create update document
	update := bson.M{
		"$set": bson.M{
			"bookname":    bookReq.Title,
			"bookauthor":  bookReq.Author,
			"bookedition": bookReq.Edition,
			"bookpages":   bookReq.Pages,
			"bookyear":    bookReq.Year,
		},
	}

	result, err := coll.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("book with ID %s not found", id)
	}

	return nil
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

	e.PUT("/api/books/:id", func(c echo.Context) error {
		id := c.Param("id")
		var bookReq BookRequest
		if err := c.Bind(&bookReq); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request body",
			})
		}

		// Validate required fields
		if bookReq.Title == "" || bookReq.Author == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Title and author are required fields",
			})
		}

		if err := updateBook(coll, id, bookReq); err != nil {
			if err.Error() == fmt.Sprintf("book with ID %s not found", id) {
				return c.JSON(http.StatusNotFound, map[string]string{
					"error": err.Error(),
				})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusOK, map[string]string{
			"message": "Book updated successfully",
		})
	})

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	fmt.Println("Books PUT service starting on port 8080")
	e.Logger.Fatal(e.Start(":8080"))
}