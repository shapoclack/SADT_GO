package main

import (
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

type CalculateRequest struct {
	Num1 float64 `json:"num1"`
	Num2 float64 `json:"num2"`
}

var server = gin.Default()

// Экземпляр модели User (аналог создания объекта Pydantic-модели)
var user = User{
	Name: "Антон Савельев",
	ID:   1,
}

// Хранилище отзывов (аналог списка feedbacks в Python)
var feedbacks []Feedback

// Список запрещённых корней слов
var bannedWords = []string{"кринж", "рофл", "вайб"}

// validateFeedback проверяет данные отзыва и возвращает список ошибок
func validateFeedback(fb Feedback) []ValidationError {
	var errors []ValidationError

	nameLen := utf8.RuneCountInString(fb.Name)
	if nameLen < 2 {
		errors = append(errors, ValidationError{
			Type:  "string_too_short",
			Loc:   []string{"body", "name"},
			Msg:   "String should have at least 2 characters",
			Input: fb.Name,
		})
	} else if nameLen > 50 {
		errors = append(errors, ValidationError{
			Type:  "string_too_long",
			Loc:   []string{"body", "name"},
			Msg:   "String should have at most 50 characters",
			Input: fb.Name,
		})
	}

	msgLen := utf8.RuneCountInString(fb.Message)
	if msgLen < 10 {
		errors = append(errors, ValidationError{
			Type:  "string_too_short",
			Loc:   []string{"body", "message"},
			Msg:   "String should have at least 10 characters",
			Input: fb.Message,
		})
	} else if msgLen > 500 {
		errors = append(errors, ValidationError{
			Type:  "string_too_long",
			Loc:   []string{"body", "message"},
			Msg:   "String should have at most 500 characters",
			Input: fb.Message,
		})
	}

	// Проверка на запрещённые слова
	lower := strings.ToLower(fb.Message)
	for _, word := range bannedWords {
		if strings.Contains(lower, word) {
			errors = append(errors, ValidationError{
				Type:  "value_error",
				Loc:   []string{"body", "message"},
				Msg:   "Value error, Использование недопустимых слов",
				Input: fb.Message,
			})
			break
		}
	}

	return errors
}

func main() {
	server.GET("/", func(c *gin.Context) {
		c.File("index.html")
	})

	server.GET("/users", func(c *gin.Context) {
		c.JSON(http.StatusOK, user)
	})

	server.POST("/calculate", func(c *gin.Context) {
		var req CalculateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"result": req.Num1 + req.Num2})
	})

	server.POST("/user", func(c *gin.Context) {
		var req UserAgeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		resp := UserAgeResponse{
			Name:    req.Name,
			Age:     req.Age,
			IsAdult: req.Age >= 18,
		}
		c.JSON(http.StatusOK, resp)
	})

	// Задание 2.1 / 2.2: POST /feedback с валидацией
	server.POST("/feedback", func(c *gin.Context) {
		var fb Feedback
		if err := c.ShouldBindJSON(&fb); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Валидация (задание 2.2)
		if errs := validateFeedback(fb); len(errs) > 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"detail": errs})
			return
		}

		// Сохраняем отзыв
		feedbacks = append(feedbacks, fb)
		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Спасибо, %s! Ваш отзыв сохранён.", fb.Name),
		})
	})

	// GET /feedbacks — посмотреть все отзывы
	server.GET("/feedbacks", func(c *gin.Context) {
		c.JSON(http.StatusOK, feedbacks)
	})

	server.Run(":8000")
}
