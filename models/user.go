package models

import "github.com/jinzhu/gorm"

// User model
type User struct {
	gorm.Model

	TelegramID int64  `gorm:"unique_index"`
	Name       string `gorm:"size:255"`
	Username   string `gorm:"size:255;unique_index"`
	Point      int    `gorm:"default:'10'"`
	RoleID     int    `gorm:"default:'3'"` // 1 Admin 2 Moderator 3 Member
}
