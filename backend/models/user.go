package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Role constants for user authorization.
const (
	RoleAdmin  = "admin"
	RoleViewer = "viewer"
	RoleEditor = "editor"
	RoleGuest  = "guest"
)

var ValidRoles = []string{RoleAdmin, RoleViewer, RoleEditor, RoleGuest}

type User struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email            string             `bson:"email" json:"email"`
	Password         string             `bson:"password" json:"-"` // bcrypt hash
	Role             string             `bson:"role" json:"role"`   // admin, viewer, editor, guest
	UseExtractedCover bool              `bson:"useExtractedCover" json:"useExtractedCover"` // prefer EPUB-extracted thumbnail over API cover
	CreatedAt        time.Time          `bson:"createdAt" json:"createdAt"`
}
