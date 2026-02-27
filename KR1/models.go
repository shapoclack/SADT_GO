package main

// User — модель пользователя (аналог Pydantic-модели в Python)
type User struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

// UserAgeRequest — входные данные для POST /user
type UserAgeRequest struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// UserAgeResponse — ответ с дополнительным полем is_adult
type UserAgeResponse struct {
	Name    string `json:"name"`
	Age     int    `json:"age"`
	IsAdult bool   `json:"is_adult"`
}

// Feedback — модель обратной связи (задание 2.1 / 2.2)
type Feedback struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

// ValidationError — ошибка валидации в стиле Pydantic
type ValidationError struct {
	Type  string      `json:"type"`
	Loc   []string    `json:"loc"`
	Msg   string      `json:"msg"`
	Input interface{} `json:"input"`
}
