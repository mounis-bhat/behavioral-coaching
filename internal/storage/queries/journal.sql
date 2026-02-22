-- Journal Entries

-- name: UpsertJournalEntry :one
INSERT INTO journal_entries (user_id, content, mood_score, prompt_used, entry_date)
VALUES ($1, $2, $3, $4, CURRENT_DATE)
ON CONFLICT (user_id, entry_date) DO UPDATE
SET content = EXCLUDED.content,
    mood_score = EXCLUDED.mood_score,
    prompt_used = EXCLUDED.prompt_used
RETURNING *;

-- name: GetJournalEntryByDate :one
SELECT * FROM journal_entries
WHERE user_id = $1 AND entry_date = $2;

-- name: GetJournalEntriesByUser :many
SELECT * FROM journal_entries
WHERE user_id = $1
ORDER BY entry_date DESC
LIMIT $2 OFFSET $3;

-- name: GetRecentMoodScores :many
SELECT entry_date, mood_score FROM journal_entries
WHERE user_id = $1
ORDER BY entry_date DESC
LIMIT $2;
