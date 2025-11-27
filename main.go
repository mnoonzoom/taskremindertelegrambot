package main

import (
	"context"
	"flag"
	"log"
	"os"
	"time"

	tgClient "read-adviser-bot/clients/telegram"
	"read-adviser-bot/consumer/event-consumer"
	"read-adviser-bot/events/telegram"
	mongoStorage "read-adviser-bot/storage/mongo"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	tgBotHost   = "api.telegram.org"
	batchSize   = 100
	dbName      = "read_adviser_bot" 
	collection  = "pages"            
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI is not set")
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("can't connect to mongo atlas: ", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("can't ping mongo atlas: ", err)
	}

	log.Println("Connected to MongoDB Atlas")

	db := client.Database(dbName)
	col := db.Collection(collection)

	storage := mongoStorage.New(col)
	if err := storage.Init(ctx); err != nil {
		log.Fatal("can't init storage: ", err)
	}

	eventsProcessor := telegram.New(
		tgClient.New(tgBotHost, mustToken()),
		storage,
	)

	log.Print("service started")

	consumer := event_consumer.New(eventsProcessor, eventsProcessor, batchSize)

	if err := consumer.Start(); err != nil {
		log.Fatal("service is stopped", err)
	}
}

func mustToken() string {
	token := flag.String(
		"tg-bot-token",
		"",
		"token for access to telegram bot",
	)

	flag.Parse()

	if *token == "" {
		log.Fatal("token is not specified")
	}

	return *token
}
