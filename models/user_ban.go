package models

import (
	"github.com/jinzhu/gorm"
)

// UserBan struct
type UserBan struct {
	gorm.Model

	User   *User
	UserID int
	Ban    *Ban
	BanID  int
	Vote   int // 1 Vote Up; 0 Vote Down
}
