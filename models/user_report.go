package models

import (
	"github.com/jinzhu/gorm"
)

// UserReport struct
type UserReport struct {
	gorm.Model

	User     *User
	UserID   int
	Report   *Report
	ReportID int
	Vote     int // 1 Vote Up; 0 Vote Down
}
