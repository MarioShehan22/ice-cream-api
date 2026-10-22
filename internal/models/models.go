package models

import "time"

type Role string

const (
	RoleOwner   Role = "owner"
	RoleManager Role = "manager"
	RoleCashier Role = "cashier"
)

type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:100;not null" json:"name"`
	Email        string    `gorm:"size:150;uniqueIndex;not null" json:"email"`
	PasswordHash string    `gorm:"size:100;not null" json:"-"`
	Role         Role      `gorm:"size:20;not null;index" json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Category is an ice cream flavour (Chocolate, Vanilla, Strawberry, ...).
type Category struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;uniqueIndex;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type Item struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:150;not null;index" json:"name"`
	Description string    `gorm:"size:1000" json:"description"`
	CategoryID  uint      `gorm:"not null;index" json:"category_id"`
	Category    *Category `gorm:"foreignKey:CategoryID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"category,omitempty"`
	Price       float64   `gorm:"not null;default:0" json:"price"`
	Quantity    int       `gorm:"not null;default:0" json:"quantity"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Customer struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:100;not null;index" json:"name"`
	Email     string    `gorm:"size:150;uniqueIndex;not null" json:"email"`
	Phone     string    `gorm:"size:20;not null" json:"phone"`
	Address   string    `gorm:"size:255" json:"address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RevokedToken stores the JTI of logged-out JWTs until they expire naturally.
type RevokedToken struct {
	ID        string    `gorm:"primaryKey;size:64"`
	ExpiresAt time.Time `gorm:"not null;index"`
}
