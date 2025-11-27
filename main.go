package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	tgClient "read-adviser-bot/clients/telegram"
	"read-adviser-bot/consumer/event-consumer"
	"read-adviser-bot/events/telegram"
	mongoStorage "read-adviser-bot/storage/mongo"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		log.Fatal("MONGO_URI not set")
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("can't connect to mongo atlas:", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("can't ping mongo atlas:", err)
	}

	log.Println("Connected to MongoDB Atlas")

	db := client.Database("read_adviser_bot")
	col := db.Collection("pages")

	storage := mongoStorage.New(col)
	if err := storage.Init(ctx); err != nil {
		log.Fatal("can't init storage:", err)
	}

	eventsProcessor := telegram.New(
		tgClient.New("api.telegram.org", mustToken()),
		storage,
	)

	consumer := event_consumer.New(eventsProcessor, eventsProcessor, 100)

	go func() {
		if err := consumer.Start(); err != nil {
			log.Fatal("bot stopped:", err)
		}
	}()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Bot is running.\n"))
	})

	log.Println("HTTP server running on port", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
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
