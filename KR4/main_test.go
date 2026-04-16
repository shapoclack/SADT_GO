package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/go-chi/chi/v5"
)

// Setup Router для тестов (изоляция)
func setupRouter() *chi.Mux {
	// Сброс хранилища перед каждым тестом (Изоляция состояния)
	mu.Lock()
	dbMock = make(map[int]UserOut)
	idSeq = 1
	mu.Unlock()

	r := chi.NewRouter()
	r.Post("/users", createUser)
	r.Get("/users/{id}", getUser)
	r.Delete("/users/{id}", deleteUser)
	return r
}

func TestCreateUser(t *testing.T) {
	r := setupRouter()

	// Генерация данных через Faker
	payload := map[string]interface{}{
		"username": gofakeit.Username(),
		"age":      gofakeit.Number(19, 60), // Условие > 18
		"email":    gofakeit.Email(),
		"password": gofakeit.Password(true, true, true, true, false, 10), // Длина между 8 и 16
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder() // Аналог httpx.AsyncClient + ASGITransport

	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Errorf("Expected 201 Created, got %v", rr.Code)
	}

	var out UserOut
	json.NewDecoder(rr.Body).Decode(&out)
	if out.ID != 1 || out.Username != payload["username"] {
		t.Errorf("Response body mismatch")
	}
}

func TestGetUser(t *testing.T) {
	r := setupRouter()

	// Сначала создаем
	mu.Lock()
	dbMock[1] = UserOut{ID: 1, Username: gofakeit.Username(), Age: 25, Email: gofakeit.Email()}
	mu.Unlock()

	// Успешный GET
	req, _ := http.NewRequest("GET", "/users/1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %v", rr.Code)
	}

	// 404 GET
	reqNotFound, _ := http.NewRequest("GET", "/users/999", nil)
	rrNotFound := httptest.NewRecorder()
	r.ServeHTTP(rrNotFound, reqNotFound)

	if rrNotFound.Code != http.StatusNotFound {
		t.Errorf("Expected 404, got %v", rrNotFound.Code)
	}
}

func TestDeleteUser(t *testing.T) {
	r := setupRouter()

	mu.Lock()
	dbMock[1] = UserOut{ID: 1, Username: "test", Age: 20}
	mu.Unlock()

	// Успешный DELETE
	req, _ := http.NewRequest("DELETE", "/users/1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected 204 No Content, got %v", rr.Code)
	}

	// Повторный DELETE (уже удален -> 404)
	rrRetry := httptest.NewRecorder()
	r.ServeHTTP(rrRetry, req)

	if rrRetry.Code != http.StatusNotFound {
		t.Errorf("Expected 404 on second delete, got %v", rrRetry.Code)
	}
}
