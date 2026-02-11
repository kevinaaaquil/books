package store

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDB(ctx context.Context, uri, dbName string) (*DB, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	log.Println("Connected to MongoDB")
	return &DB{
		Client:   client,
		Database: client.Database(dbName),
	}, nil
}

func (db *DB) Users() *mongo.Collection {
	return db.Database.Collection("users")
}

func (db *DB) Books() *mongo.Collection {
	return db.Database.Collection("books")
}

func (db *DB) EmailConfig() *mongo.Collection {
	return db.Database.Collection("kindle_config")
}

func (db *DB) EmailLogs() *mongo.Collection {
	return db.Database.Collection("email_logs")
}

func (db *DB) Disconnect(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return db.Client.Disconnect(ctx)
}
