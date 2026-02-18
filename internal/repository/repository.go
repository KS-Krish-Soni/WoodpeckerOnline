package repository

import (
	"woodpecker-online/internal/model"
)

// Repository defines the interface for all repository operations
type Repository interface {
	UserRepository
	SetRepository
	CycleRepository
	SessionRepository
	AttemptRepository
	UserSettingsRepository
}

// UserRepository defines operations for user management
type UserRepository interface {
	CreateUser(user *model.User) error
	GetUserByID(id string) (*model.User, error)
	GetUserByEmail(email string) (*model.User, error)
	UpdateUser(user *model.User) error
	DeleteUser(id string) error
}

// SetRepository defines operations for set management
type SetRepository interface {
	CreateSet(set *model.Set) error
	GetSetByID(id int) (*model.Set, error)
	GetSetsByUserID(userID string) ([]*model.Set, error)
	UpdateSet(set *model.Set) error
	DeleteSet(id int) error
	AddPuzzleToSet(setID int, puzzleID string, position int) error
	GetPuzzlesInSet(setID int) ([]*model.SetPuzzle, error)
	RemovePuzzleFromSet(setID int, puzzleID string) error
}

// CycleRepository defines operations for cycle management
type CycleRepository interface {
	CreateCycle(cycle *model.Cycle) error
	GetCycleByID(id int) (*model.Cycle, error)
	GetCyclesBySetID(setID int) ([]*model.Cycle, error)
	UpdateCycle(cycle *model.Cycle) error
	DeleteCycle(id int) error
	GetActiveCycleBySetID(setID int) (*model.Cycle, error)
}

// SessionRepository defines operations for session management
type SessionRepository interface {
	CreateSession(session *model.Session) error
	GetSessionByID(id int) (*model.Session, error)
	GetSessionsByCycleID(cycleID int) ([]*model.Session, error)
	UpdateSession(session *model.Session) error
	DeleteSession(id int) error
	GetActiveSessionByCycleID(cycleID int) (*model.Session, error)
}

// AttemptRepository defines operations for attempt management
type AttemptRepository interface {
	CreateAttempt(attempt *model.Attempt) error
	GetAttemptByID(id int) (*model.Attempt, error)
	GetAttemptsBySessionID(sessionID int) ([]*model.Attempt, error)
	UpdateAttempt(attempt *model.Attempt) error
	DeleteAttempt(id int) error
	GetAttemptsByPuzzleID(puzzleID string) ([]*model.Attempt, error)
}

// UserSettingsRepository defines operations for user settings management
type UserSettingsRepository interface {
	CreateUserSettings(settings *model.UserSettings) error
	GetUserSettingsByUserID(userID string) (*model.UserSettings, error)
	UpdateUserSettings(settings *model.UserSettings) error
	DeleteUserSettings(userID string) error
}
