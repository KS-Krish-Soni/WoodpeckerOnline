package repository

import (
	"database/sql"

	"woodpecker-online/internal/model"

	"github.com/jmoiron/sqlx"
)

// SQLiteRepository implements the Repository interface using SQLite
type SQLiteRepository struct {
	db *sqlx.DB
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(db *sqlx.DB) Repository {
	return &SQLiteRepository{db: db}
}

// UserRepository implementation

func (r *SQLiteRepository) CreateUser(user *model.User) error {
	query := `
		INSERT INTO users (id, email, password_hash, created_at)
		VALUES (?, ?, ?, ?)
	`
	_, err := r.db.Exec(query, user.ID, user.Email, user.PasswordHash, user.CreatedAt)
	return err
}

func (r *SQLiteRepository) GetUserByID(id string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, email, password_hash, created_at FROM users WHERE id = ?`
	err := r.db.Get(user, query, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *SQLiteRepository) GetUserByEmail(email string) (*model.User, error) {
	user := &model.User{}
	query := `SELECT id, email, password_hash, created_at FROM users WHERE email = ?`
	err := r.db.Get(user, query, email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *SQLiteRepository) UpdateUser(user *model.User) error {
	query := `
		UPDATE users 
		SET email = ?, password_hash = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := r.db.Exec(query, user.Email, user.PasswordHash, user.ID)
	return err
}

func (r *SQLiteRepository) DeleteUser(id string) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

// SetRepository implementation

func (r *SQLiteRepository) CreateSet(set *model.Set) error {
	query := `
		INSERT INTO sets (user_id, name, description, difficulty_min, difficulty_max, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query, set.UserID, set.Name, set.Description, set.DifficultyMin, set.DifficultyMax, set.CreatedAt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	set.ID = int(id)
	return nil
}

func (r *SQLiteRepository) GetSetByID(id int) (*model.Set, error) {
	set := &model.Set{}
	query := `SELECT id, user_id, name, description, difficulty_min, difficulty_max, created_at FROM sets WHERE id = ?`
	err := r.db.Get(set, query, id)
	if err != nil {
		return nil, err
	}
	return set, nil
}

func (r *SQLiteRepository) GetSetsByUserID(userID string) ([]*model.Set, error) {
	var sets []*model.Set
	query := `SELECT id, user_id, name, description, difficulty_min, difficulty_max, created_at FROM sets WHERE user_id = ? ORDER BY created_at DESC`
	err := r.db.Select(&sets, query, userID)
	if err != nil {
		return nil, err
	}
	return sets, nil
}

func (r *SQLiteRepository) UpdateSet(set *model.Set) error {
	query := `
		UPDATE sets 
		SET name = ?, description = ?, difficulty_min = ?, difficulty_max = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query, set.Name, set.Description, set.DifficultyMin, set.DifficultyMax, set.ID)
	return err
}

func (r *SQLiteRepository) DeleteSet(id int) error {
	query := `DELETE FROM sets WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *SQLiteRepository) AddPuzzleToSet(setID int, puzzleID string, position int) error {
	query := `
		INSERT INTO set_puzzles (set_id, puzzle_id, position)
		VALUES (?, ?, ?)
	`
	_, err := r.db.Exec(query, setID, puzzleID, position)
	return err
}

func (r *SQLiteRepository) GetPuzzlesInSet(setID int) ([]*model.SetPuzzle, error) {
	var puzzles []*model.SetPuzzle
	query := `SELECT set_id, puzzle_id, position FROM set_puzzles WHERE set_id = ? ORDER BY position`
	err := r.db.Select(&puzzles, query, setID)
	if err != nil {
		return nil, err
	}
	return puzzles, nil
}

func (r *SQLiteRepository) RemovePuzzleFromSet(setID int, puzzleID string) error {
	query := `DELETE FROM set_puzzles WHERE set_id = ? AND puzzle_id = ?`
	_, err := r.db.Exec(query, setID, puzzleID)
	return err
}

// CycleRepository implementation

func (r *SQLiteRepository) CreateCycle(cycle *model.Cycle) error {
	query := `
		INSERT INTO cycles (set_id, cycle_index, target_days, started_at, ended_at, status)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query, cycle.SetID, cycle.Index, cycle.TargetDays, cycle.StartedAt, cycle.EndedAt, cycle.Status)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	cycle.ID = int(id)
	return nil
}

func (r *SQLiteRepository) GetCycleByID(id int) (*model.Cycle, error) {
	cycle := &model.Cycle{}
	query := `SELECT id, set_id, cycle_index, target_days, started_at, ended_at, status FROM cycles WHERE id = ?`
	err := r.db.Get(cycle, query, id)
	if err != nil {
		return nil, err
	}
	return cycle, nil
}

func (r *SQLiteRepository) GetCyclesBySetID(setID int) ([]*model.Cycle, error) {
	var cycles []*model.Cycle
	query := `SELECT id, set_id, cycle_index, target_days, started_at, ended_at, status FROM cycles WHERE set_id = ? ORDER BY cycle_index`
	err := r.db.Select(&cycles, query, setID)
	if err != nil {
		return nil, err
	}
	return cycles, nil
}

func (r *SQLiteRepository) UpdateCycle(cycle *model.Cycle) error {
	query := `
		UPDATE cycles 
		SET set_id = ?, cycle_index = ?, target_days = ?, started_at = ?, ended_at = ?, status = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query, cycle.SetID, cycle.Index, cycle.TargetDays, cycle.StartedAt, cycle.EndedAt, cycle.Status, cycle.ID)
	return err
}

func (r *SQLiteRepository) DeleteCycle(id int) error {
	query := `DELETE FROM cycles WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *SQLiteRepository) GetActiveCycleBySetID(setID int) (*model.Cycle, error) {
	cycle := &model.Cycle{}
	query := `SELECT id, set_id, index, target_days, started_at, ended_at, status FROM cycles WHERE set_id = ? AND status = 'active'`
	err := r.db.Get(cycle, query, setID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return cycle, nil
}

// SessionRepository implementation

func (r *SQLiteRepository) CreateSession(session *model.Session) error {
	query := `
		INSERT INTO sessions (cycle_id, started_at, ended_at, target_count)
		VALUES (?, ?, ?, ?)
	`
	result, err := r.db.Exec(query, session.CycleID, session.StartedAt, session.EndedAt, session.TargetCount)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	session.ID = int(id)
	return nil
}

func (r *SQLiteRepository) GetSessionByID(id int) (*model.Session, error) {
	session := &model.Session{}
	query := `SELECT id, cycle_id, started_at, ended_at, target_count FROM sessions WHERE id = ?`
	err := r.db.Get(session, query, id)
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (r *SQLiteRepository) GetSessionsByCycleID(cycleID int) ([]*model.Session, error) {
	var sessions []*model.Session
	query := `SELECT id, cycle_id, started_at, ended_at, target_count FROM sessions WHERE cycle_id = ? ORDER BY started_at`
	err := r.db.Select(&sessions, query, cycleID)
	if err != nil {
		return nil, err
	}
	return sessions, nil
}

func (r *SQLiteRepository) UpdateSession(session *model.Session) error {
	query := `
		UPDATE sessions 
		SET cycle_id = ?, started_at = ?, ended_at = ?, target_count = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query, session.CycleID, session.StartedAt, session.EndedAt, session.TargetCount, session.ID)
	return err
}

func (r *SQLiteRepository) DeleteSession(id int) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *SQLiteRepository) GetActiveSessionByCycleID(cycleID int) (*model.Session, error) {
	session := &model.Session{}
	query := `SELECT id, cycle_id, started_at, ended_at, target_count FROM sessions WHERE cycle_id = ? AND ended_at IS NULL`
	err := r.db.Get(session, query, cycleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return session, nil
}

// AttemptRepository implementation

func (r *SQLiteRepository) CreateAttempt(attempt *model.Attempt) error {
	query := `
		INSERT INTO attempts (session_id, puzzle_id, started_at, ended_at, score_first_move, score_ticks, total_points, time_ms, correct_first_move)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.Exec(query, attempt.SessionID, attempt.PuzzleID, attempt.StartedAt, attempt.EndedAt, attempt.ScoreFirstMove, attempt.ScoreTicks, attempt.TotalPoints, attempt.TimeMs, attempt.CorrectFirstMove)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	attempt.ID = int(id)
	return nil
}

func (r *SQLiteRepository) GetAttemptByID(id int) (*model.Attempt, error) {
	attempt := &model.Attempt{}
	query := `SELECT id, session_id, puzzle_id, started_at, ended_at, score_first_move, score_ticks, total_points, time_ms, correct_first_move FROM attempts WHERE id = ?`
	err := r.db.Get(attempt, query, id)
	if err != nil {
		return nil, err
	}
	return attempt, nil
}

func (r *SQLiteRepository) GetAttemptsBySessionID(sessionID int) ([]*model.Attempt, error) {
	var attempts []*model.Attempt
	query := `SELECT id, session_id, puzzle_id, started_at, ended_at, score_first_move, score_ticks, total_points, time_ms, correct_first_move FROM attempts WHERE session_id = ? ORDER BY started_at`
	err := r.db.Select(&attempts, query, sessionID)
	if err != nil {
		return nil, err
	}
	return attempts, nil
}

func (r *SQLiteRepository) UpdateAttempt(attempt *model.Attempt) error {
	query := `
		UPDATE attempts 
		SET session_id = ?, puzzle_id = ?, started_at = ?, ended_at = ?, score_first_move = ?, score_ticks = ?, total_points = ?, time_ms = ?, correct_first_move = ?
		WHERE id = ?
	`
	_, err := r.db.Exec(query, attempt.SessionID, attempt.PuzzleID, attempt.StartedAt, attempt.EndedAt, attempt.ScoreFirstMove, attempt.ScoreTicks, attempt.TotalPoints, attempt.TimeMs, attempt.CorrectFirstMove, attempt.ID)
	return err
}

func (r *SQLiteRepository) DeleteAttempt(id int) error {
	query := `DELETE FROM attempts WHERE id = ?`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *SQLiteRepository) GetAttemptsByPuzzleID(puzzleID string) ([]*model.Attempt, error) {
	var attempts []*model.Attempt
	query := `SELECT id, session_id, puzzle_id, started_at, ended_at, score_first_move, score_ticks, total_points, time_ms, correct_first_move FROM attempts WHERE puzzle_id = ? ORDER BY started_at`
	err := r.db.Select(&attempts, query, puzzleID)
	if err != nil {
		return nil, err
	}
	return attempts, nil
}

// UserSettingsRepository implementation

func (r *SQLiteRepository) CreateUserSettings(settings *model.UserSettings) error {
	query := `
		INSERT INTO user_settings (user_id, daily_goal_minutes, reminders_enabled, timezone)
		VALUES (?, ?, ?, ?)
	`
	_, err := r.db.Exec(query, settings.UserID, settings.DailyGoalMinutes, settings.RemindersEnabled, settings.Timezone)
	return err
}

func (r *SQLiteRepository) GetUserSettingsByUserID(userID string) (*model.UserSettings, error) {
	settings := &model.UserSettings{}
	query := `SELECT user_id, daily_goal_minutes, reminders_enabled, timezone FROM user_settings WHERE user_id = ?`
	err := r.db.Get(settings, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Return default settings if none exist
			return &model.UserSettings{
				UserID:           userID,
				DailyGoalMinutes: 30,
				RemindersEnabled: true,
				Timezone:         "UTC",
			}, nil
		}
		return nil, err
	}
	return settings, nil
}

func (r *SQLiteRepository) UpdateUserSettings(settings *model.UserSettings) error {
	query := `
		UPDATE user_settings 
		SET daily_goal_minutes = ?, reminders_enabled = ?, timezone = ?
		WHERE user_id = ?
	`
	_, err := r.db.Exec(query, settings.DailyGoalMinutes, settings.RemindersEnabled, settings.Timezone, settings.UserID)
	return err
}

func (r *SQLiteRepository) DeleteUserSettings(userID string) error {
	query := `DELETE FROM user_settings WHERE user_id = ?`
	_, err := r.db.Exec(query, userID)
	return err
}
