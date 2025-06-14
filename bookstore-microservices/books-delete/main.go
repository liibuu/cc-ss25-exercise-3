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

func deleteBook(coll *mongo.Collection, id string) error {
	filter := bson.M{"id": id}
	result, err := coll.DeleteOne(context.TODO(), filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("book with ID %s not found", id)
	}

	return nil
}

func main() {
	fmt.Println("Waiting for MongoDB to be ready...")
	time.Sleep(15 * time.Second)
	
	var client *mongo.Client
	var coll *mongo.Collection
	var err error

	// Retry connection to MongoDB
	for i := 0; i < 10; i++ {
		client, coll, err = connectToMongoDB()
		if err == nil {
			break
		}
		fmt.Printf("Failed to connect to MongoDB (attempt %d/10): %v\n", i+1, err)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MongoDB after 10 attempts: %v", err))
	}

	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			fmt.Printf("Error disconnecting from MongoDB: %v\n", err)
		}
	}()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	e.DELETE("/api/books/:id", func(c echo.Context) error {
		id := c.Param("id")

		if err := deleteBook(coll, id); err != nil {
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
			"message": "Book deleted successfully",
		})
	})

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	fmt.Println("Books DELETE service starting on port 8080")
	e.Logger.Fatal(e.Start(":8080"))
}