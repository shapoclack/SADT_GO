-- +goose Up
ALTER TABLE products ADD COLUMN description TEXT NOT NULL DEFAULT 'No description';

-- +goose Down
ALTER TABLE products DROP COLUMN description;