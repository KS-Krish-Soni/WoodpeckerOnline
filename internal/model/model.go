package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// Line represents a single move line in a chess puzzle solution
type Line struct {
	SAN      string `json:"san"`
	Children []Line `json:"children,omitempty"`
	IsTick   bool   `json:"isTick,omitempty"`
}

// Solution represents a complete chess puzzle solution
type Solution struct {
	Lines                []Line     `json:"lines"`
	AcceptedAlternatives [][]string `json:"acceptedAlternatives,omitempty"`
}

// Puzzle represents a chess puzzle with its solution
type Puzzle struct {
	ID         string   `json:"id"`
	Difficulty string   `json:"difficulty"`
	FEN        string   `json:"fen"`
	Solution   Solution `json:"solution"`
	Ticks      []string `json:"ticks"` // SANs marked IsTick
}

// SolutionJSON is a custom type for database storage of Solution
type SolutionJSON struct {
	Solution
}

// Value implements driver.Valuer for database storage
func (sj SolutionJSON) Value() (driver.Value, error) {
	if sj.Solution.Lines == nil {
		return nil, nil
	}
	return json.Marshal(sj.Solution)
}

// Scan implements sql.Scanner for database retrieval
func (sj *SolutionJSON) Scan(value interface{}) error {
	if value == nil {
		sj.Solution = Solution{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, &sj.Solution)
}

// TicksJSON is a custom type for database storage of Ticks
type TicksJSON struct {
	Ticks []string
}

// Value implements driver.Valuer for database storage
func (tj TicksJSON) Value() (driver.Value, error) {
	if tj.Ticks == nil {
		return nil, nil
	}
	return json.Marshal(tj.Ticks)
}

// Scan implements sql.Scanner for database retrieval
func (tj *TicksJSON) Scan(value interface{}) error {
	if value == nil {
		tj.Ticks = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("expected []byte, got %T", value)
	}

	return json.Unmarshal(bytes, &tj.Ticks)
}

// PuzzleDB represents the database structure for puzzles
type PuzzleDB struct {
	ID           string       `db:"id"`
	Difficulty   string       `db:"difficulty"`
	FEN          string       `db:"fen"`
	SideToMove   string       `db:"side_to_move"`
	SolutionJSON SolutionJSON `db:"solution_json"`
	TicksJSON    TicksJSON    `db:"ticks_json"`
}

// ToPuzzle converts PuzzleDB to Puzzle
func (pdb *PuzzleDB) ToPuzzle() *Puzzle {
	return &Puzzle{
		ID:         pdb.ID,
		Difficulty: pdb.Difficulty,
		FEN:        pdb.FEN,
		Solution:   pdb.SolutionJSON.Solution,
		Ticks:      pdb.TicksJSON.Ticks,
	}
}

// FromPuzzle converts Puzzle to PuzzleDB
func FromPuzzle(puzzle *Puzzle) *PuzzleDB {
	return &PuzzleDB{
		ID:           puzzle.ID,
		Difficulty:   puzzle.Difficulty,
		FEN:          puzzle.FEN,
		SideToMove:   extractSideToMove(puzzle.FEN),
		SolutionJSON: SolutionJSON{Solution: puzzle.Solution},
		TicksJSON:    TicksJSON{Ticks: puzzle.Ticks},
	}
}

// extractSideToMove extracts the side to move from a FEN string
func extractSideToMove(fen string) string {
	if fen == "" {
		return "w" // Default to white if FEN is empty
	}

	parts := strings.Fields(fen)
	if len(parts) >= 2 {
		return parts[1] // The second part of FEN is the side to move
	}

	return "w" // Default to white if FEN is malformed
}

// User represents a user in the system
type User struct {
	ID           string `db:"id" json:"id"`
	Email        string `db:"email" json:"email"`
	PasswordHash string `db:"password_hash" json:"-"`
	CreatedAt    string `db:"created_at" json:"created_at"`
}

// Set represents a collection of puzzles for the Woodpecker Method
type Set struct {
	ID            int    `db:"id" json:"id"`
	UserID        string `db:"user_id" json:"user_id"`
	Name          string `db:"name" json:"name"`
	Description   string `db:"description" json:"description"`
	DifficultyMin string `db:"difficulty_min" json:"difficulty_min"`
	DifficultyMax string `db:"difficulty_max" json:"difficulty_max"`
	CreatedAt     string `db:"created_at" json:"created_at"`
}

// SetPuzzle represents the relationship between a set and a puzzle with position
type SetPuzzle struct {
	SetID    int    `db:"set_id" json:"set_id"`
	PuzzleID string `db:"puzzle_id" json:"puzzle_id"`
	Position int    `db:"position" json:"position"`
}

// Cycle represents a cycle in the Woodpecker Method
type Cycle struct {
	ID         int     `db:"id" json:"id"`
	SetID      int     `db:"set_id" json:"set_id"`
	Index      int     `db:"cycle_index" json:"index"`
	TargetDays int     `db:"target_days" json:"target_days"`
	StartedAt  *string `db:"started_at" json:"started_at"`
	EndedAt    *string `db:"ended_at" json:"ended_at"`
	Status     string  `db:"status" json:"status"` // planned|active|rest|done
}

// Session represents a solving session within a cycle
type Session struct {
	ID          int     `db:"id" json:"id"`
	CycleID     int     `db:"cycle_id" json:"cycle_id"`
	StartedAt   *string `db:"started_at" json:"started_at"`
	EndedAt     *string `db:"ended_at" json:"ended_at"`
	TargetCount int     `db:"target_count" json:"target_count"`
}

// Attempt represents a single puzzle attempt within a session
type Attempt struct {
	ID               int     `db:"id" json:"id"`
	SessionID        int     `db:"session_id" json:"session_id"`
	PuzzleID         string  `db:"puzzle_id" json:"puzzle_id"`
	StartedAt        *string `db:"started_at" json:"started_at"`
	EndedAt          *string `db:"ended_at" json:"ended_at"`
	ScoreFirstMove   int     `db:"score_first_move" json:"score_first_move"`
	ScoreTicks       int     `db:"score_ticks" json:"score_ticks"`
	TotalPoints      int     `db:"total_points" json:"total_points"`
	TimeMs           int     `db:"time_ms" json:"time_ms"`
	CorrectFirstMove bool    `db:"correct_first_move" json:"correct_first_move"`
}

// UserSettings represents user preferences and settings
type UserSettings struct {
	UserID           string `db:"user_id" json:"user_id"`
	DailyGoalMinutes int    `db:"daily_goal_minutes" json:"daily_goal_minutes"`
	RemindersEnabled bool   `db:"reminders_enabled" json:"reminders_enabled"`
	Timezone         string `db:"timezone" json:"timezone"`
}