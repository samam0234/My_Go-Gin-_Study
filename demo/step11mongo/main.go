package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type note struct {
	Title string `bson:"title"`
}

func main() {
	if err := run(os.Getenv("MONGO_URL")); err != nil {
		panic(err)
	}
}

func run(uri string) error {
	if uri == "" {
		fmt.Println("MONGO_URL이 없어 연결은 건너뜁니다. 이 파일은 컴파일만 확인합니다.")
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	defer client.Disconnect(context.Background())
	coll := client.Database("gin_study").Collection("notes")
	if _, err := coll.InsertOne(ctx, note{Title: "mongo"}); err != nil {
		return err
	}
	var found note
	return coll.FindOne(ctx, bson.D{{Key: "title", Value: "mongo"}}).Decode(&found)
}
