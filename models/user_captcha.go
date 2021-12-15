package models

import "github.com/jinzhu/gorm"

// UserCaptcha model
type UserCaptcha struct {
	gorm.Model

	UserID    int64  `gorm:"index"`
	Code      string `gorm:"size:5"`
	MessageID int
}
