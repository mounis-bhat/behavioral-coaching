-- +goose Up
ALTER TABLE daily_plans ADD COLUMN prompt_version TEXT NOT NULL DEFAULT 'v1';
CREATE INDEX idx_daily_plans_plan_date ON daily_plans (plan_date DESC);
CREATE INDEX idx_adaptation_logs_created_at ON adaptation_logs (created_at DESC);
CREATE INDEX idx_plan_tasks_category ON plan_tasks (category);

-- +goose Down
DROP INDEX IF EXISTS idx_plan_tasks_category;
DROP INDEX IF EXISTS idx_adaptation_logs_created_at;
DROP INDEX IF EXISTS idx_daily_plans_plan_date;
ALTER TABLE daily_plans DROP COLUMN IF EXISTS prompt_version;
