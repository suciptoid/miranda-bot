package models

import "github.com/jinzhu/gorm"

// Report models
type Report struct {
	gorm.Model

	MessageID  int `gorm:"unique_index"` // Same message can't be reported more than once
	ReporterID int64
	VoteUp     int `gorm:"default:'0'"`
	VoteDown   int `gorm:"default:'0'"`
	// Users      []User `gorm:"many2many:report_users"`
	UserReports []*UserReport
}
