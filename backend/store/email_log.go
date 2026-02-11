package store

import (
	"context"

	"github.com/kevinaaaquil/books/backend/models"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// InsertEmailLog records that a book was sent to an email by a user.
func (db *DB) InsertEmailLog(ctx context.Context, log *models.EmailLog) error {
	_, err := db.EmailLogs().InsertOne(ctx, log, options.InsertOne())
	return err
}
