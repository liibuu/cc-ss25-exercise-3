package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

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

func getMongoURI() string {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://mongo:27017"
	}
	return uri
}

func prepareDatabase(client *mongo.Client, dbName string, collecName string) (*mongo.Collection, error) {
	db := client.Database(dbName)

	names, err := db.ListCollectionNames(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, err
	}
	if !slices.Contains(names, collecName) {
		cmd := bson.D{{"create", collecName}}
		var result bson.M
		if err = db.RunCommand(context.TODO(), cmd).Decode(&result); err != nil {
			log.Fatal(err)
			return nil, err
		}
	}

	coll := db.Collection(collecName)
	return coll, nil
}

func prepareData(coll *mongo.Collection) {
	startData := []BookStore{
		{
			ID:          "example1",
			BookName:    "The Vortex",
			BookAuthor:  "José Eustasio Rivera",
			BookEdition: "958-30-0804-4",
			BookPages:   "292",
			BookYear:    "1924",
		},
		{
			ID:          "example2",
			BookName:    "Frankenstein",
			BookAuthor:  "Mary Shelley",
			BookEdition: "978-3-649-64609-9",
			BookPages:   "280",
			BookYear:    "1818",
		},
		{
			ID:          "example3",
			BookName:    "The Black Cat",
			BookAuthor:  "Edgar Allan Poe",
			BookEdition: "978-3-99168-238-7",
			BookPages:   "280",
			BookYear:    "1843",
		},
	}

	for _, book := range startData {
		cursor, err := coll.Find(context.TODO(), bson.M{"id": book.ID})
		var results []BookStore
		if err = cursor.All(context.TODO(), &results); err != nil {
			panic(err)
		}
		if len(results) > 1 {
			log.Fatal("more records were found")
		} else if len(results) == 0 {
			result, err := coll.InsertOne(context.TODO(), book)
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("Inserted book: %+v\n", result)
			}
		} else {
			fmt.Printf("Book already exists: %s\n", book.ID)
		}
	}
}

func main() {
	fmt.Println("Data seeder starting...")
	
	// Wait for MongoDB to be ready
	time.Sleep(20 * time.Second)
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var client *mongo.Client
	var err error

	// Retry connection to MongoDB
	for i := 0; i < 10; i++ {
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(getMongoURI()))
		if err == nil {
			err = client.Ping(ctx, nil)
			if err == nil {
				break
			}
		}
		fmt.Printf("Failed to connect to MongoDB (attempt %d/10): %v\n", i+1, err)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		panic(fmt.Sprintf("Failed to connect to MongoDB after 10 attempts: %v", err))
	}

	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	coll, err := prepareDatabase(client, "exercise-1", "information")
	if err != nil {
		log.Fatal(err)
	}

	prepareData(coll)
	
	fmt.Println("Data seeding completed successfully!")
}