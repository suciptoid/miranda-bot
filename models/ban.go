package models

import "github.com/jinzhu/gorm"

// Ban models
type Ban struct {
	gorm.Model

	MessageID    int `gorm:"unique_index"` // Same message can't be ban-voted more than once
	ReporterID   int64
	BannedUserID int64
	VoteUp       int `gorm:"default:'0'"`
	VoteDown     int `gorm:"default:'0'"`
	UserBans     []*UserBan
}
