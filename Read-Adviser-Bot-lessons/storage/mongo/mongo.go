package mongo

import (
    "context"
    "errors"
    "fmt"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

    "read-adviser-bot/storage"
)

type Storage struct {
    col *mongo.Collection
}

func New(col *mongo.Collection) *Storage {
    return &Storage{col: col}
}

func (s *Storage) Save(ctx context.Context, p *storage.Page) error {
    // p.URL = текст задачи
    _, err := s.col.InsertOne(ctx, bson.M{
        "url":       p.URL,      // тут просто задача, не ссылка
        "user_name": p.UserName,
    })
    if err != nil {
        // ловим дубликаты, если есть уникальный индекс
        var writeErr mongo.WriteException
        if errors.As(err, &writeErr) {
            for _, e := range writeErr.WriteErrors {
                if e.Code == 11000 {
                    // уже есть такая задача – можно вернуть nil или спец. ошибку
                    return nil
                }
            }
        }
        return fmt.Errorf("can't save task: %w", err)
    }

    return nil
}

func (s *Storage) PickRandom(ctx context.Context, userName string) (*storage.Page, error) {
    pipeline := mongo.Pipeline{
        {{"$match", bson.D{{"user_name", userName}}}},
        {{"$sample", bson.D{{"size", 1}}}},
    }

    cursor, err := s.col.Aggregate(ctx, pipeline)
    if err != nil {
        return nil, fmt.Errorf("can't pick random task: %w", err)
    }
    defer cursor.Close(ctx)

    if !cursor.Next(ctx) {
        if err := cursor.Err(); err != nil {
            return nil, fmt.Errorf("cursor error: %w", err)
        }
        return nil, storage.ErrNoSavedPages
    }

    var doc struct {
        URL      string `bson:"url"`
        UserName string `bson:"user_name"`
    }

    if err := cursor.Decode(&doc); err != nil {
        return nil, fmt.Errorf("can't decode task: %w", err)
    }

    return &storage.Page{
        URL:      doc.URL,
        UserName: doc.UserName,
    }, nil
}

func (s *Storage) Remove(ctx context.Context, p *storage.Page) error {
    res, err := s.col.DeleteOne(ctx, bson.M{
        "url":       p.URL,
        "user_name": p.UserName,
    })
    if err != nil {
        return fmt.Errorf("can't remove task: %w", err)
    }

    if res.DeletedCount == 0 {
        return storage.ErrNoSavedPages
    }

    return nil
}

func (s *Storage) IsExists(ctx context.Context, p *storage.Page) (bool, error) {
    filter := bson.M{
        "url":       p.URL,
        "user_name": p.UserName,
    }

    err := s.col.FindOne(ctx, filter).Err()
    if err == mongo.ErrNoDocuments {
        return false, nil
    }
    if err != nil {
        return false, fmt.Errorf("can't check task existence: %w", err)
    }

    return true, nil
}

// ➕ новый метод: получить все задачи юзера
func (s *Storage) GetAll(ctx context.Context, userName string) ([]*storage.Page, error) {
    cursor, err := s.col.Find(ctx, bson.M{"user_name": userName})
    if err != nil {
        return nil, fmt.Errorf("can't get tasks: %w", err)
    }
    defer cursor.Close(ctx)

    var tasks []*storage.Page

    for cursor.Next(ctx) {
        var doc struct {
            URL      string `bson:"url"`
            UserName string `bson:"user_name"`
        }

        if err := cursor.Decode(&doc); err != nil {
            return nil, fmt.Errorf("can't decode task: %w", err)
        }

        tasks = append(tasks, &storage.Page{
            URL:      doc.URL,
            UserName: doc.UserName,
        })
    }

    if err := cursor.Err(); err != nil {
        return nil, fmt.Errorf("cursor error: %w", err)
    }

    if len(tasks) == 0 {
        return nil, storage.ErrNoSavedPages
    }

    return tasks, nil
}

// Индексы
func (s *Storage) Init(ctx context.Context) error {
    idx := mongo.IndexModel{
        Keys: bson.D{
            {Key: "user_name", Value: 1},
            {Key: "url", Value: 1},
        },
        Options: options.Index().SetUnique(true),
    }

    _, err := s.col.Indexes().CreateOne(ctx, idx)
    if err != nil {
        return fmt.Errorf("can't create index: %w", err)
    }

    return nil
}
