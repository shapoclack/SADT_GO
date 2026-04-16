package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-playground/validator/v10"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

// --- 10.1 Модели ошибок ---

type ErrorResponse struct {
	StatusCode int    `json:"status_code"`
	Detail     string `json:"detail"`
}

// Кастомные ошибки
var (
	ErrConditionNotMet  = errors.New("custom_a: condition not met")
	ErrResourceNotFound = errors.New("custom_b: resource not found")
)

// Централизованный обработчик ошибок
func respondWithError(w http.ResponseWriter, err error, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		StatusCode: statusCode,
		Detail:     err.Error(),
	})
}

// --- 10.2 Валидация (Аналог Pydantic) ---

var validate *validator.Validate

func init() {
	validate = validator.New()
}

type UserIn struct {
	Username string `json:"username" validate:"required"`
	Age      int    `json:"age" validate:"required,gt=18"`             // gt=18 (больше 18)
	Email    string `json:"email" validate:"required,email"`           // EmailStr
	Password string `json:"password" validate:"required,min=8,max=16"` // constr
	Phone    string `json:"phone" validate:"omitempty"`                // Optional
}

type UserOut struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Age      int    `json:"age"`
	Email    string `json:"email"`
}

// In-Memory БД для заданий 11.1 - 11.2
var (
	dbMock = make(map[int]UserOut)
	mu     sync.Mutex
	idSeq  = 1
)

func main() {
	// Применение миграций (Задание 9.1)
	db, err := sql.Open("sqlite3", "./app.db")
	if err != nil {
		log.Fatal(err)
	}
	goose.SetBaseFS(embedMigrations)
	if err := goose.Up(db, "migrations"); err != nil {
		log.Println("Migration output:", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	// Задание 10.1: Эндпоинты, генерирующие кастомные ошибки
	r.Get("/error-a", func(w http.ResponseWriter, r *http.Request) {
		respondWithError(w, ErrConditionNotMet, http.StatusBadRequest)
	})
	r.Get("/error-b", func(w http.ResponseWriter, r *http.Request) {
		respondWithError(w, ErrResourceNotFound, http.StatusNotFound)
	})

	// Задание 10.2 и 11.2: CRUD пользователей с валидацией
	r.Post("/users", createUser)
	r.Get("/users/{id}", getUser)
	r.Delete("/users/{id}", deleteUser)

	fmt.Println("Server listening on :8000")
	http.ListenAndServe(":8000", r)
}

// --- Хендлеры CRUD ---

func createUser(w http.ResponseWriter, r *http.Request) {
	var input UserIn
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondWithError(w, errors.New("invalid json payload"), http.StatusUnprocessableEntity)
		return
	}

	// Запуск валидации
	if err := validate.Struct(input); err != nil {
		respondWithError(w, err, http.StatusUnprocessableEntity) // 422 Unprocessable Entity
		return
	}

	// Дефолтное значение для телефона (как Optional[str] = 'Unknown')
	if input.Phone == "" {
		input.Phone = "Unknown"
	}

	mu.Lock()
	defer mu.Unlock()
	userOut := UserOut{
		ID:       idSeq,
		Username: input.Username,
		Age:      input.Age,
		Email:    input.Email,
	}
	dbMock[idSeq] = userOut
	idSeq++

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(userOut)
}

func getUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	mu.Lock()
	user, exists := dbMock[id]
	mu.Unlock()

	if !exists {
		respondWithError(w, ErrResourceNotFound, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func deleteUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(chi.URLParam(r, "id"))

	mu.Lock()
	defer mu.Unlock()

	if _, exists := dbMock[id]; !exists {
		respondWithError(w, ErrResourceNotFound, http.StatusNotFound)
		return
	}

	delete(dbMock, id)
	w.WriteHeader(http.StatusNoContent)
}
