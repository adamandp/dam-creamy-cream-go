// Package model contains manually maintained mappings of the NestJS Prisma schema.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type User struct {
	ID                string             `gorm:"column:id;type:uuid;primaryKey;not null" json:"id"`
	RoleID            string             `gorm:"column:role_id;type:uuid;not null" json:"roleId"`
	Username          string             `gorm:"column:username;type:varchar(50);not null;uniqueIndex:users_username_key" json:"username"`
	Email             *string            `gorm:"column:email;type:varchar(255);uniqueIndex:users_email_key" json:"email"`
	PhoneNumber       *string            `gorm:"column:phone_number;type:varchar(100);uniqueIndex:users_phone_number_key" json:"phoneNumber"`
	Password          string             `gorm:"column:password;type:varchar(255);not null" json:"password"`
	FullName          string             `gorm:"column:full_name;type:varchar(100);not null" json:"fullName"`
	BirthDate         *time.Time         `gorm:"column:birth_date;type:date" json:"birthDate"`
	ImageURL          *string            `gorm:"column:image_url;type:varchar(255)" json:"imageUrl"`
	Gender            *GenderType        `gorm:"column:gender;type:\"GENDER_TYPE\"" json:"gender"`
	IsActive          *bool              `gorm:"column:is_active;type:boolean;not null;default:false" json:"isActive"`
	CreatedAt         time.Time          `gorm:"column:created_at;type:timestamp(3);not null;default:CURRENT_TIMESTAMP;autoCreateTime" json:"createdAt"`
	UpdatedAt         time.Time          `gorm:"column:updated_at;type:timestamp(3);not null;autoUpdateTime" json:"updatedAt"`
	Roles             Role               `gorm:"foreignKey:RoleID;references:ID" json:"-"`
	Cart              *Cart              `gorm:"foreignKey:UserID;references:ID" json:"-"`
	UserTokens        []UserToken        `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Orders            []Order            `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Reviews           []Review           `gorm:"foreignKey:UserID;references:ID" json:"-"`
	UserVouchers      []UserVoucher      `gorm:"foreignKey:UserID;references:ID" json:"-"`
	ProductAnalytics  []ProductAnalytics `gorm:"foreignKey:UserID;references:ID" json:"-"`
	UserNotifications []UserNotification `gorm:"foreignKey:UserID;references:ID" json:"-"`
	Addresses         []Address          `gorm:"foreignKey:UserID;references:ID" json:"-"`
}

func (User) TableName() string {
	return "users"
}

func (m *User) BeforeCreate(_ *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	return nil
}
