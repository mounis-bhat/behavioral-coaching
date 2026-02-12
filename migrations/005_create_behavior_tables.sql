-- +goose Up
-- +goose StatementBegin

-- Behavior profiles: one per user, stores AI-populated behavioral data
CREATE TABLE behavior_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    goals JSONB NOT NULL DEFAULT '[]',
    constraints JSONB NOT NULL DEFAULT '{}',
    psychological_state JSONB NOT NULL DEFAULT '{}',
    difficulty_level NUMERIC(3,1) NOT NULL DEFAULT 5.0,
    onboarding_completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_behavior_profiles_updated_at
    BEFORE UPDATE ON behavior_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Daily plans: one plan per user per day
CREATE TABLE daily_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_date DATE NOT NULL,
    difficulty_score NUMERIC(3,1) NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'completed', 'expired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, plan_date)
);

CREATE INDEX idx_daily_plans_user_id ON daily_plans (user_id);
CREATE INDEX idx_daily_plans_user_date ON daily_plans (user_id, plan_date DESC);

CREATE TRIGGER update_daily_plans_updated_at
    BEFORE UPDATE ON daily_plans
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Plan tasks: individual tasks within a daily plan
CREATE TABLE plan_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    daily_plan_id UUID NOT NULL REFERENCES daily_plans(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    category TEXT NOT NULL CHECK (category IN ('health', 'mindset', 'discipline', 'relationships', 'productivity')),
    difficulty NUMERIC(3,1) NOT NULL DEFAULT 5.0,
    position INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plan_tasks_daily_plan_id ON plan_tasks (daily_plan_id);

-- Execution logs: one log per task, tracks completion
CREATE TABLE execution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_task_id UUID NOT NULL UNIQUE REFERENCES plan_tasks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_execution_logs_user_id ON execution_logs (user_id);

-- Adherence states: one record per user, upserted on recomputation
CREATE TABLE adherence_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    completion_rate NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    streak_count INTEGER NOT NULL DEFAULT 0,
    total_tasks INTEGER NOT NULL DEFAULT 0,
    completed_tasks INTEGER NOT NULL DEFAULT 0,
    difficulty_mismatch BOOLEAN NOT NULL DEFAULT FALSE,
    last_computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TRIGGER update_adherence_states_updated_at
    BEFORE UPDATE ON adherence_states
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Adaptation logs: append-only log of plan adaptation events
CREATE TABLE adaptation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    adaptation_reason TEXT NOT NULL,
    difficulty_change NUMERIC(3,1) NOT NULL,
    previous_difficulty NUMERIC(3,1) NOT NULL,
    new_difficulty NUMERIC(3,1) NOT NULL,
    trigger_metrics JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_adaptation_logs_user_id ON adaptation_logs (user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS adaptation_logs;
DROP TABLE IF EXISTS adherence_states;
DROP TABLE IF EXISTS execution_logs;
DROP TABLE IF EXISTS plan_tasks;
DROP TABLE IF EXISTS daily_plans;
DROP TABLE IF EXISTS behavior_profiles;
-- +goose StatementEnd
