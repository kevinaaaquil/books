package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// EmailConfig holds iCloud/Kindle email settings for sending books. Each document has its own _id and a userId linking to the user.
type EmailConfig struct {
	ID                  primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID              primitive.ObjectID `bson:"userId" json:"userId"`
	AppSpecificPassword string             `bson:"appSpecificPassword" json:"appSpecificPassword"`
	ICloudMail          string             `bson:"icloudMail" json:"icloudMail"`
	SenderMail          string             `bson:"senderMail" json:"senderMail"`
	KindleMail          string             `bson:"kindleMail" json:"kindleMail"`
}
