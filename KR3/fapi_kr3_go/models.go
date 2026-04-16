package main

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null" json:"username"`
	Password string `gorm:"not null" json:"-"`          // Пароль не отдаем в JSON
	Role     string `gorm:"default:'user'" json:"role"` // Для Задания 7.1 (RBAC)
}

// UserInput - DTO для регистрации/логина
type UserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Todo struct {
	gorm.Model
	Title       string `json:"title"`
	Description string `json:"description"`
	Completed   bool   `gorm:"default:false" json:"completed"`
}
