package store

import (
	"context"

	"github.com/kevinaaaquil/books/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (db *DB) UserByEmail(ctx context.Context, email string) (*models.User, error) {
	var u models.User
	err := db.Users().FindOne(ctx, bson.M{"email": email}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) CreateUser(ctx context.Context, user *models.User) (primitive.ObjectID, error) {
	res, err := db.Users().InsertOne(ctx, user, options.InsertOne())
	if err != nil {
		return primitive.NilObjectID, err
	}
	return res.InsertedID.(primitive.ObjectID), nil
}

func (db *DB) UserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var u models.User
	err := db.Users().FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
