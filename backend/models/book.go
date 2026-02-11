package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Book struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title         string             `bson:"title" json:"title"`
	Authors       []string           `bson:"authors,omitempty" json:"authors,omitempty"`
	Publisher     string             `bson:"publisher,omitempty" json:"publisher,omitempty"`
	PublishDate   string             `bson:"publishDate,omitempty" json:"publishDate,omitempty"`
	ISBN          string             `bson:"isbn,omitempty" json:"isbn,omitempty"`
	PageCount     int                `bson:"pageCount,omitempty" json:"pageCount,omitempty"`
	CoverURL      string             `bson:"coverUrl,omitempty" json:"coverUrl,omitempty"`
	ThumbnailURL  string             `bson:"thumbnailUrl,omitempty" json:"thumbnailUrl,omitempty"`
	CoverS3Key       string             `bson:"coverS3Key,omitempty" json:"-"` // extracted from EPUB, served via /api/books/:id/cover
	ExtractedCoverURL string            `bson:"-" json:"extractedCoverUrl,omitempty"` // set when serializing if CoverS3Key set; lets frontend toggle
	Edition       string             `bson:"edition,omitempty" json:"edition,omitempty"`
	Preface       string             `bson:"preface,omitempty" json:"preface,omitempty"`
	Category      string             `bson:"category,omitempty" json:"category,omitempty"`
	Categories    []string           `bson:"categories,omitempty" json:"categories,omitempty"`
	RatingAverage float64            `bson:"ratingAverage,omitempty" json:"ratingAverage,omitempty"`
	RatingCount   int                `bson:"ratingCount,omitempty" json:"ratingCount,omitempty"`
	Format           string             `bson:"format" json:"format"`                     // "epub" or "pdf"
	S3Key            string             `bson:"s3Key" json:"-"`                         // object key in S3
	OriginalName     string             `bson:"originalName" json:"originalName"`
	UploadedByEmail  string             `bson:"uploadedByEmail,omitempty" json:"uploadedByEmail,omitempty"`
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
}
