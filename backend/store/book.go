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

func (db *DB) AllBooks(ctx context.Context) ([]models.Book, error) {
	cur, err := db.Books().Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"createdAt": -1}))
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

// BooksVisibleToGuest returns books where viewByGuest is true (for guest-role users).
func (db *DB) BooksVisibleToGuest(ctx context.Context) ([]models.Book, error) {
	cur, err := db.Books().Find(ctx, bson.M{"viewByGuest": true}, options.Find().SetSort(bson.M{"createdAt": -1}))
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

func (db *DB) BookByID(ctx context.Context, id primitive.ObjectID) (*models.Book, error) {
	var book models.Book
	err := db.Books().FindOne(ctx, bson.M{"_id": id}).Decode(&book)
	if err != nil {
		return nil, err
	}
	return &book, nil
}

// DeleteBook removes a book by ID. Returns the deleted book's S3Key, CoverS3Key (if any), and any error.
func (db *DB) DeleteBook(ctx context.Context, id primitive.ObjectID) (s3Key, coverS3Key string, err error) {
	var book models.Book
	err = db.Books().FindOneAndDelete(ctx, bson.M{"_id": id}).Decode(&book)
	if err != nil {
		return "", "", err
	}
	return book.S3Key, book.CoverS3Key, nil
}

// UpdateBookMetadata updates a book's metadata fields by ID.
func (db *DB) UpdateBookMetadata(ctx context.Context, id primitive.ObjectID, book *models.Book) error {
	update := bson.M{
		"title":          book.Title,
		"authors":        book.Authors,
		"publisher":      book.Publisher,
		"publishDate":    book.PublishDate,
		"isbn":           book.ISBN,
		"pageCount":      book.PageCount,
		"coverUrl":       book.CoverURL,
		"thumbnailUrl":   book.ThumbnailURL,
		"edition":        book.Edition,
		"preface":        book.Preface,
		"category":       book.Category,
		"categories":     book.Categories,
		"ratingAverage": book.RatingAverage,
		"ratingCount":    book.RatingCount,
	}
	_, err := db.Books().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": update})
	return err
}

// UpdateBookViewByGuest sets viewByGuest for a book (admin only).
func (db *DB) UpdateBookViewByGuest(ctx context.Context, id primitive.ObjectID, viewByGuest bool) error {
	_, err := db.Books().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"viewByGuest": viewByGuest}})
	return err
}
