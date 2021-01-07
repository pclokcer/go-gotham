package models

import (
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gotham/app"
	"time"
)

type User struct {
	ID                uint    `gorm:"primaryKey;auto_increment" json:"id"`
	Name              string  `gorm:"size:255;not null" json:"name"`
	Email             string  `gorm:"size:100;not null;unique;unique_index" json:"email"`
	Password          string  `gorm:"size:100" json:"-"`
	Verified          uint8   `gorm:"type:boolean" json:"verified"`
	VerificationToken *string `gorm:"size:50;" json:"-"`
	Image             *string `gorm:"size:500;" json:"image"`
	Admin             uint8   `gorm:"type:boolean;not null;default:0" json:"admin"`

	// Time
	CreatedAt time.Time      `gorm:"type:datetime(0)" json:"created_at"`
	UpdatedAt time.Time      `gorm:"type:datetime(0)"  json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

/**
 * VerifyPassword
 *
 * @param string , string
 * @return error
 */
func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}


/**
 * Create
 *
 * @return *gorm.DB
 */
func (u *User) Create() *gorm.DB {
	return app.Application.Container.GetDb().Create(&u)
}

/**
 * Save
 *
 * @return *gorm.DB
 */
func (u *User) Save() *gorm.DB {
	return app.Application.Container.GetDb().Save(&u)
}

/**
 * Delete
 *
 * @return string, error
 */
func (u *User) Delete() *gorm.DB {
	return app.Application.Container.GetDb().Delete(&u)
}


/**
 * IsVerified
 *
 * @return bool
 */
func (u User) IsVerified() bool {
	return u.Verified == 1
}

/**
 * IsAdmin
 *
 * @return bool
 */
func (u User) IsAdmin() bool {
	return u.Admin == 1
}
