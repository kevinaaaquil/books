package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Book struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       primitive.ObjectID `bson:"userId" json:"userId"`
	Title        string             `bson:"title" json:"title"`
	Authors      []string           `bson:"authors,omitempty" json:"authors,omitempty"`
	Publisher    string             `bson:"publisher,omitempty" json:"publisher,omitempty"`
	PublishDate  string             `bson:"publishDate,omitempty" json:"publishDate,omitempty"`
	ISBN         string             `bson:"isbn,omitempty" json:"isbn,omitempty"`
	PageCount    int                `bson:"pageCount,omitempty" json:"pageCount,omitempty"`
	CoverURL     string             `bson:"coverUrl,omitempty" json:"coverUrl,omitempty"`
	Edition      string             `bson:"edition,omitempty" json:"edition,omitempty"`
	Format       string             `bson:"format" json:"format"` // "epub" or "pdf"
	S3Key        string             `bson:"s3Key" json:"-"`       // object key in S3
	OriginalName string             `bson:"originalName" json:"originalName"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
}
