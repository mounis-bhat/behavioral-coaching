-- Behavior Profiles

-- name: CreateBehaviorProfile :one
INSERT INTO behavior_profiles (user_id, goals, constraints, psychological_state, difficulty_level, onboarding_completed)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetBehaviorProfileByUserID :one
SELECT * FROM behavior_profiles WHERE user_id = $1;

-- name: UpdateBehaviorProfile :one
UPDATE behavior_profiles
SET goals = COALESCE(sqlc.narg('goals'), goals),
    constraints = COALESCE(sqlc.narg('constraints'), constraints),
    psychological_state = COALESCE(sqlc.narg('psychological_state'), psychological_state),
    difficulty_level = COALESCE(sqlc.narg('difficulty_level'), difficulty_level),
    onboarding_completed = COALESCE(sqlc.narg('onboarding_completed'), onboarding_completed)
WHERE user_id = sqlc.arg('user_id')
RETURNING *;

-- name: DeleteBehaviorProfile :exec
DELETE FROM behavior_profiles WHERE user_id = $1;

-- Daily Plans

-- name: CreateDailyPlan :one
INSERT INTO daily_plans (user_id, plan_date, difficulty_score, status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetDailyPlanByUserAndDate :one
SELECT * FROM daily_plans WHERE user_id = $1 AND plan_date = $2;

-- name: GetActiveDailyPlan :one
SELECT * FROM daily_plans
WHERE user_id = $1 AND plan_date = CURRENT_DATE AND status != 'expired';

-- name: UpdateDailyPlanStatus :one
UPDATE daily_plans SET status = $2 WHERE id = $1
RETURNING *;

-- name: GetRecentDailyPlans :many
SELECT * FROM daily_plans
WHERE user_id = $1
ORDER BY plan_date DESC
LIMIT $2;

-- Plan Tasks

-- name: CreatePlanTask :one
INSERT INTO plan_tasks (daily_plan_id, title, description, category, difficulty, position)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetPlanTasksByDailyPlan :many
SELECT * FROM plan_tasks
WHERE daily_plan_id = $1
ORDER BY position ASC;

-- name: GetPlanTaskByID :one
SELECT * FROM plan_tasks WHERE id = $1;

-- Execution Logs

-- name: UpsertExecutionLog :one
INSERT INTO execution_logs (plan_task_id, user_id, completed, completed_at, notes)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (plan_task_id) DO UPDATE
SET completed = EXCLUDED.completed,
    completed_at = EXCLUDED.completed_at,
    notes = EXCLUDED.notes
RETURNING *;

-- name: GetExecutionLogsByDailyPlan :many
SELECT el.* FROM execution_logs el
JOIN plan_tasks pt ON el.plan_task_id = pt.id
WHERE pt.daily_plan_id = $1
ORDER BY pt.position ASC;

-- name: GetExecutionLogsByUser :many
SELECT * FROM execution_logs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- Adherence States

-- name: UpsertAdherenceState :one
INSERT INTO adherence_states (user_id, completion_rate, streak_count, total_tasks, completed_tasks, difficulty_mismatch, last_computed_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (user_id) DO UPDATE
SET completion_rate = EXCLUDED.completion_rate,
    streak_count = EXCLUDED.streak_count,
    total_tasks = EXCLUDED.total_tasks,
    completed_tasks = EXCLUDED.completed_tasks,
    difficulty_mismatch = EXCLUDED.difficulty_mismatch,
    last_computed_at = NOW()
RETURNING *;

-- name: GetAdherenceStateByUserID :one
SELECT * FROM adherence_states WHERE user_id = $1;

-- Adaptation Logs

-- name: CreateAdaptationLog :one
INSERT INTO adaptation_logs (user_id, adaptation_reason, difficulty_change, previous_difficulty, new_difficulty, trigger_metrics)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetAdaptationLogsByUser :many
SELECT * FROM adaptation_logs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;
