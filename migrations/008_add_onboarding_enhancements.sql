-- +goose Up

-- Add delivery mode, emotional state, and AI profile summary to behavior_profiles
ALTER TABLE behavior_profiles
    ADD COLUMN delivery_mode TEXT NOT NULL DEFAULT 'flexible_list'
        CHECK (delivery_mode IN ('strict_schedule', 'flexible_list', 'gentle_structure')),
    ADD COLUMN emotional_state TEXT NOT NULL DEFAULT ''
        CHECK (emotional_state IN ('', 'burned_out', 'anxious', 'stuck')),
    ADD COLUMN ai_profile_summary JSONB NOT NULL DEFAULT '{}';

-- Add task type, suggested time, and anchor label to plan_tasks
ALTER TABLE plan_tasks
    ADD COLUMN task_type TEXT NOT NULL DEFAULT 'regular'
        CHECK (task_type IN ('regular', 'anchor')),
    ADD COLUMN suggested_time TEXT NOT NULL DEFAULT '',
    ADD COLUMN anchor_label TEXT NOT NULL DEFAULT ''
        CHECK (anchor_label IN ('', 'wake_window', 'activation_activity', 'sleep_winddown'));

-- +goose Down

ALTER TABLE plan_tasks
    DROP COLUMN anchor_label,
    DROP COLUMN suggested_time,
    DROP COLUMN task_type;

ALTER TABLE behavior_profiles
    DROP COLUMN ai_profile_summary,
    DROP COLUMN emotional_state,
    DROP COLUMN delivery_mode;
