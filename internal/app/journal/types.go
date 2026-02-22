package journal

// UpsertEntryInput represents journal entry creation/update input
// @Description Journal entry upsert request
type UpsertEntryInput struct {
	Content    string `json:"content"`
	MoodScore  int    `json:"mood_score"`
	PromptUsed string `json:"prompt_used"`
}

// JournalEntryResponse represents a journal entry
// @Description Journal entry response
type JournalEntryResponse struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	Content    string `json:"content"`
	MoodScore  int    `json:"mood_score"`
	PromptUsed string `json:"prompt_used"`
	EntryDate  string `json:"entry_date"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

// MoodDataPoint represents a single mood score on a given date
// @Description Mood data point
type MoodDataPoint struct {
	Date  string `json:"date"`
	Score int    `json:"score"`
}

// MoodTrendResponse contains the 14-day mood trend analysis
// @Description Mood trend response
type MoodTrendResponse struct {
	AvgScore   float64         `json:"avg_score"`
	Trend      string          `json:"trend"`
	DataPoints []MoodDataPoint `json:"data_points"`
}
