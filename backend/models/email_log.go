package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EmailLog records a book sent to a Kindle email by a user.
type EmailLog struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	BookID    primitive.ObjectID `bson:"bookId" json:"bookId"`
	FileTitle string             `bson:"fileTitle" json:"fileTitle"`
	ToEmail   string             `bson:"toEmail" json:"toEmail"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	UserEmail string             `bson:"userEmail" json:"userEmail"`
	SentAt    time.Time          `bson:"sentAt" json:"sentAt"`
}
