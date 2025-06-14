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

func createBook(coll *mongo.Collection, bookReq BookRequest) error {
	// Check if book with same ID already exists
	cursor, err := coll.Find(context.TODO(), bson.M{"id": bookReq.ID})
	if err != nil {
		return err
	}
	
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		return err
	}
	
	if len(results) > 0 {
		return fmt.Errorf("book with ID %s already exists", bookReq.ID)
	}

	// Create new book
	newBook := BookStore{
		ID:          bookReq.ID,
		BookName:    bookReq.Title,
		BookAuthor:  bookReq.Author,
		BookEdition: bookReq.Edition,
		BookPages:   bookReq.Pages,
		BookYear:    bookReq.Year,
	}

	_, err = coll.InsertOne(context.TODO(), newBook)
	return err
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

	e.POST("/api/books", func(c echo.Context) error {
		var bookReq BookRequest
		if err := c.Bind(&bookReq); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "Invalid request body",
			})
		}

		// Validate required fields
		if bookReq.ID == "" || bookReq.Title == "" || bookReq.Author == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": "ID, title, and author are required fields",
			})
		}

		if err := createBook(coll, bookReq); err != nil {
			return c.JSON(http.StatusConflict, map[string]string{
				"error": err.Error(),
			})
		}

		return c.JSON(http.StatusCreated, map[string]string{
			"message": "Book created successfully",
		})
	})

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "healthy"})
	})

	fmt.Println("Books POST service starting on port 8080")
	e.Logger.Fatal(e.Start(":8080"))
}