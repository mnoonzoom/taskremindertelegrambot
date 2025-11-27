package main

import (
	"context"
	"flag"
	"log"
	"time"

	tgClient "read-adviser-bot/clients/telegram"
	"read-adviser-bot/consumer/event-consumer"
	"read-adviser-bot/events/telegram"
	mongoStorage "read-adviser-bot/storage/mongo"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	tgBotHost           = "api.telegram.org"
	batchSize           = 100
	mongoURI            = "mongodb://localhost:27017"
	mongoDBName         = "read_adviser_bot"
	mongoCollectionName = "pages"
)

func main() {
	// Контекст для подключения
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// === Подключение к MongoDB ===
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("can't connect to mongo: ", err)
	}

	// Проверка соединения
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("can't ping mongo: ", err)
	}

	// Берём базу и коллекцию
	db := client.Database(mongoDBName)
	col := db.Collection(mongoCollectionName)

	// === Инициализируем Mongo Storage ===
	s := mongoStorage.New(col)

	if err := s.Init(ctx); err != nil {
		log.Fatal("can't init storage: ", err)
	}

	// === Telegram Processor ===
	eventsProcessor := telegram.New(
		tgClient.New(tgBotHost, mustToken()),
		s,
	)

	log.Print("service started")

	consumer := event_consumer.New(eventsProcessor, eventsProcessor, batchSize)

	if err := consumer.Start(); err != nil {
		log.Fatal("service is stopped: ", err)
	}
}

// Получение TG токена
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
