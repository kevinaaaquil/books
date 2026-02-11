package store

import (
	"context"

	"github.com/kevinaaaquil/books/backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// EnsureEmailConfigIndex creates a unique index on userId so each user has at most one Kindle config.
func (db *DB) EnsureEmailConfigIndex(ctx context.Context) error {
	idx := mongo.IndexModel{
		Keys:    bson.D{{Key: "userId", Value: 1}},
		Options: options.Index().SetUnique(true),
	}
	_, err := db.EmailConfig().Indexes().CreateOne(ctx, idx)
	return err
}

// GetEmailConfig returns the Kindle/email config for the given user, or nil if none exists.
func (db *DB) GetEmailConfig(ctx context.Context, userID primitive.ObjectID) (*models.EmailConfig, error) {
	var cfg models.EmailConfig
	err := db.EmailConfig().FindOne(ctx, bson.M{"userId": userID}).Decode(&cfg)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpsertEmailConfig creates or updates the Kindle/email config for the given user. Config has its own _id; userId links it to the user.
func (db *DB) UpsertEmailConfig(ctx context.Context, userID primitive.ObjectID, cfg *models.EmailConfig) error {
	set := bson.M{
		"userId":             userID,
		"appSpecificPassword": cfg.AppSpecificPassword,
		"icloudMail":          cfg.ICloudMail,
		"senderMail":          cfg.SenderMail,
		"kindleMail":          cfg.KindleMail,
	}
	opts := options.Update().SetUpsert(true)
	_, err := db.EmailConfig().UpdateOne(ctx, bson.M{"userId": userID}, bson.M{"$set": set}, opts)
	return err
}
