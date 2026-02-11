package store

import (
	"context"

	"github.com/kevinaaaquil/books/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// UsersCount returns the number of documents in the users collection.
func (db *DB) UsersCount(ctx context.Context) (int64, error) {
	return db.Users().CountDocuments(ctx, bson.M{})
}

// AdminsCount returns the number of users with role admin.
func (db *DB) AdminsCount(ctx context.Context) (int64, error) {
	return db.Users().CountDocuments(ctx, bson.M{"role": "admin"})
}

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

func (db *DB) ListUsers(ctx context.Context) ([]models.User, error) {
	cur, err := db.Users().Find(ctx, bson.M{}, options.Find().SetSort(bson.M{"createdAt": 1}))
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var users []models.User
	if err := cur.All(ctx, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (db *DB) UpdateUser(ctx context.Context, id primitive.ObjectID, email *string, hashedPassword *string, role *string) error {
	updates := bson.M{}
	if email != nil {
		updates["email"] = *email
	}
	if hashedPassword != nil {
		updates["password"] = *hashedPassword
	}
	if role != nil {
		updates["role"] = *role
	}
	if len(updates) == 0 {
		return nil
	}
	_, err := db.Users().UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": updates})
	return err
}

func (db *DB) DeleteUser(ctx context.Context, id primitive.ObjectID) error {
	_, err := db.Users().DeleteOne(ctx, bson.M{"_id": id})
	return err
}
