-- +goose Up
-- +goose StatementBegin

ALTER TABLE behavior_profiles ADD COLUMN last_notified_at TIMESTAMPTZ;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE behavior_profiles DROP COLUMN IF EXISTS last_notified_at;

-- +goose StatementEnd
