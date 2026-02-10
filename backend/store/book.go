package store

import (
	"context"

	"github.com/kevinaaaquil/books/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (db *DB) InsertBook(ctx context.Context, book *models.Book) (primitive.ObjectID, error) {
	res, err := db.Books().InsertOne(ctx, book, options.InsertOne())
	if err != nil {
		return primitive.NilObjectID, err
	}
	return res.InsertedID.(primitive.ObjectID), nil
}

func (db *DB) BooksByUserID(ctx context.Context, userID primitive.ObjectID) ([]models.Book, error) {
	cur, err := db.Books().Find(ctx, bson.M{"userId": userID}, options.Find().SetSort(bson.M{"createdAt": -1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var books []models.Book
	if err := cur.All(ctx, &books); err != nil {
		return nil, err
	}
	return books, nil
}

// BookByID returns a book by ID only if it belongs to the given user.
func (db *DB) BookByID(ctx context.Context, id, userID primitive.ObjectID) (*models.Book, error) {
	var book models.Book
	err := db.Books().FindOne(ctx, bson.M{"_id": id, "userId": userID}).Decode(&book)
	if err != nil {
		return nil, err
	}
	return &book, nil
}
