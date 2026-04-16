package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func initDB() {
	var err error
	db, err = gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database")
	}
	// Автомиграция (Задание 8.1, 8.2)
	db.AutoMigrate(&User{}, &Todo{})
}

func main() {
	godotenv.Load()
	initDB()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Задание 6.3: Защита или скрытие документации
	r.With(DocsBasicAuth).Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"message": "Welcome to protected docs (DEV mode)!"}`))
	})

	// Задания 6.1, 6.2, 6.4, 6.5: Аутентификация
	// Лимит: 1 запрос в минуту
	r.With(limitMiddleware(rate.Every(time.Minute), 1)).Post("/register", registerHandler)
	// Лимит: 5 запросов в минуту
	r.With(limitMiddleware(rate.Every(time.Minute/5), 5)).Post("/login", loginHandler)

	// Защищенные маршруты
	r.Group(func(r chi.Router) {
		r.Use(JWTAuth)

		r.Get("/protected_resource", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"message": "Access granted to protected resource"}`))
		})

		// Задание 8.2: CRUD Todos
		r.Route("/todos", func(r chi.Router) {
			r.Post("/", createTodo) // Создать
			r.Get("/{id}", getTodo) // Получить

			// Задание 7.1: Доступ к обновлению и удалению только для admin/user
			r.With(RequireRole("admin", "user")).Put("/{id}", updateTodo)
			r.With(RequireRole("admin")).Delete("/{id}", deleteTodo)
		})
	})

	fmt.Println("Server running on port 8000...")
	http.ListenAndServe(":8000", r)
}

// --- Хендлеры ---

func registerHandler(w http.ResponseWriter, r *http.Request) {
	var input UserInput
	json.NewDecoder(r.Body).Decode(&input)

	var existing User
	if db.Where("username = ?", input.Username).First(&existing).Error == nil {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"detail": "User already exists"})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	newUser := User{Username: input.Username, Password: string(hashedPassword), Role: "user"}
	db.Create(&newUser)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "New user created"})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var input UserInput
	json.NewDecoder(r.Body).Decode(&input)

	var user User
	if db.Where("username = ?", input.Username).First(&user).Error != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"detail": "User not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"detail": "Authorization failed"})
		return
	}

	// Генерация JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  user.Username,
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, _ := token.SignedString([]byte(os.Getenv("JWT_SECRET")))

	json.NewEncoder(w).Encode(map[string]string{"access_token": tokenString, "token_type": "bearer"})
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	var todo Todo
	json.NewDecoder(r.Body).Decode(&todo)
	db.Create(&todo)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(todo)
}

func getTodo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var todo Todo
	if db.First(&todo, id).Error != nil {
		http.Error(w, `{"detail": "Not found"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(todo)
}

func updateTodo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var todo Todo
	if db.First(&todo, id).Error != nil {
		http.Error(w, `{"detail": "Not found"}`, http.StatusNotFound)
		return
	}

	var input Todo
	json.NewDecoder(r.Body).Decode(&input)
	todo.Title = input.Title
	todo.Description = input.Description
	todo.Completed = input.Completed
	db.Save(&todo)
	json.NewEncoder(w).Encode(todo)
}

func deleteTodo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	db.Delete(&Todo{}, id)
	json.NewEncoder(w).Encode(map[string]string{"message": "Successfully deleted"})
}
