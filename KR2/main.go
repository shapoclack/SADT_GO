package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Секретный ключ для подписи сессий (в реальном проекте должен храниться в .env)
var secretKey = []byte("super_secret_key_12345")

// --- Модели для Задания 3.1 ---
type UserCreate struct {
	Name         string `json:"name" binding:"required"`
	Email        string `json:"email" binding:"required,email"`
	Age          *int   `json:"age,omitempty" binding:"omitempty,gt=0"`
	IsSubscribed *bool  `json:"is_subscribed,omitempty"`
}

// --- Данные для Задания 3.2 ---
type Product struct {
	ProductID int     `json:"product_id"`
	Name      string  `json:"name"`
	Category  string  `json:"category"`
	Price     float64 `json:"price"`
}

var sampleProducts = []Product{
	{ProductID: 123, Name: "Smartphone", Category: "Electronics", Price: 599.99},
	{ProductID: 456, Name: "Phone Case", Category: "Accessories", Price: 19.99},
	{ProductID: 789, Name: "Iphone", Category: "Electronics", Price: 1299.99},
	{ProductID: 101, Name: "Headphones", Category: "Accessories", Price: 99.99},
	{ProductID: 202, Name: "Smartwatch", Category: "Electronics", Price: 299.99},
}

// --- Модели для Задания 5 ---
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// CommonHeaders для Задания 5.5
type CommonHeaders struct {
	UserAgent      string `header:"User-Agent" binding:"required"`
	AcceptLanguage string `header:"Accept-Language" binding:"required"`
}

func main() {
	router := gin.Default()

	// 🌟 ЗАДАНИЕ 3.1: Создание пользователя
	router.POST("/create_user", func(c *gin.Context) {
		var user UserCreate
		// ShouldBindJSON автоматически проверяет типы и правила (email, gt=0)
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, user)
	})

	// 🌟 ЗАДАНИЕ 3.2: Продукты
	// 1. Поиск продуктов (маршрут объявлен ДО /product/:product_id во избежание конфликтов)
	router.GET("/products/search", func(c *gin.Context) {
		keyword := strings.ToLower(c.Query("keyword"))
		category := c.Query("category")
		limitStr := c.DefaultQuery("limit", "10")

		if keyword == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "keyword is required"})
			return
		}

		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 10
		}

		var results []Product
		for _, p := range sampleProducts {
			if strings.Contains(strings.ToLower(p.Name), keyword) {
				if category == "" || p.Category == category {
					results = append(results, p)
					if len(results) == limit {
						break
					}
				}
			}
		}

		if results == nil {
			results = []Product{} // Возвращаем пустой массив, а не null
		}
		c.JSON(http.StatusOK, results)
	})

	// 2. Получение продукта по ID
	router.GET("/product/:product_id", func(c *gin.Context) {
		idStr := c.Param("product_id")
		productID, err := strconv.Atoi(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product_id"})
			return
		}

		for _, p := range sampleProducts {
			if p.ProductID == productID {
				c.JSON(http.StatusOK, p)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
	})

	// 🌟 ЗАДАНИЯ 5.1, 5.2, 5.3: Аутентификация, куки и сессии
	router.POST("/login", func(c *gin.Context) {
		var loginData LoginRequest
		if err := c.ShouldBindJSON(&loginData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Простая проверка (моковая)
		if loginData.Username != "user123" || loginData.Password != "password123" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid credentials"})
			return
		}

		userID := uuid.New().String()
		timestamp := time.Now().Unix()
		signature := generateSignature(userID, timestamp)

		sessionToken := fmt.Sprintf("%s.%d.%s", userID, timestamp, signature)

		// Установка куки: name, value, maxAge (300 сек = 5 мин), path, domain, secure, httpOnly
		c.SetCookie("session_token", sessionToken, 300, "/", "", false, true)

		c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
	})

	router.GET("/profile", sessionMiddleware(), func(c *gin.Context) {
		// Извлекаем userID, который middleware заботливо положил в контекст
		userID := c.MustGet("user_id").(string)
		c.JSON(http.StatusOK, gin.H{
			"user_profile": "This is protected data",
			"user_id":      userID,
		})
	})

	// 🌟 ЗАДАНИЯ 5.4, 5.5: Работа с заголовками
	router.GET("/headers", func(c *gin.Context) {
		var headers CommonHeaders
		if err := c.ShouldBindHeader(&headers); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required headers"})
			return
		}

		if !isValidAcceptLanguage(headers.AcceptLanguage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Accept-Language format"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"User-Agent":      headers.UserAgent,
			"Accept-Language": headers.AcceptLanguage,
		})
	})

	router.GET("/info", func(c *gin.Context) {
		var headers CommonHeaders
		if err := c.ShouldBindHeader(&headers); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required headers"})
			return
		}

		if !isValidAcceptLanguage(headers.AcceptLanguage) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Accept-Language format"})
			return
		}

		// Добавляем кастомный заголовок ответа
		c.Header("X-Server-Time", time.Now().Format(time.RFC3339))

		c.JSON(http.StatusOK, gin.H{
			"message": "Добро пожаловать! Ваши заголовки успешно обработаны.",
			"headers": gin.H{
				"User-Agent":      headers.UserAgent,
				"Accept-Language": headers.AcceptLanguage,
			},
		})
	})

	// Запуск сервера на порту 8080
	fmt.Println("Starting server on :8080...")
	router.Run(":8080")
}

// --- Вспомогательные функции ---

// Генерация HMAC-SHA256 подписи
func generateSignature(userID string, timestamp int64) string {
	data := fmt.Sprintf("%s.%d", userID, timestamp)
	mac := hmac.New(sha256.New, secretKey)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// Проверка валидности Accept-Language через регулярное выражение
func isValidAcceptLanguage(al string) bool {
	// Простая проверка формата, например: en-US,en;q=0.9
	matched, _ := regexp.MatchString(`^[a-zA-Z\-]+(,[a-zA-Z\-]+;q=[0-9\.]+)*`, al)
	return matched
}

// Middleware для проверки сессии (Задание 5.3)
func sessionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("session_token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			c.Abort()
			return
		}

		parts := strings.Split(cookie, ".")
		if len(parts) != 3 {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid session"})
			c.Abort()
			return
		}

		userID := parts[0]
		timestampStr := parts[1]
		signature := parts[2]

		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid session"})
			c.Abort()
			return
		}

		// Проверяем целостность (подпись)
		expectedSignature := generateSignature(userID, timestamp)
		if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid session"})
			c.Abort()
			return
		}

		// Проверяем время активности
		now := time.Now().Unix()
		elapsed := now - timestamp

		if elapsed >= 300 { // Прошло >= 5 минут
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Session expired"})
			c.Abort()
			return
		}

		// Продлеваем сессию, если прошло от 3 до 5 минут
		if elapsed >= 180 && elapsed < 300 {
			newTimestamp := time.Now().Unix()
			newSignature := generateSignature(userID, newTimestamp)
			newSessionToken := fmt.Sprintf("%s.%d.%s", userID, newTimestamp, newSignature)
			c.SetCookie("session_token", newSessionToken, 300, "/", "", false, true)
		}

		// Передаем userID в контекст для использования в обработчике маршрута
		c.Set("user_id", userID)
		c.Next()
	}
}
