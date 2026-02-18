package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/robfig/cron/v3"
	_ "modernc.org/sqlite"

	"woodpecker-online/internal/auth"
	"woodpecker-online/internal/dev"
	"woodpecker-online/internal/model"
	"woodpecker-online/internal/repository"
	"woodpecker-online/internal/user"
	"woodpecker-online/internal/woodpecker"
)

// Chess game data structures
type PieceType string

const (
	King   PieceType = "king"
	Queen  PieceType = "queen"
	Rook   PieceType = "rook"
	Bishop PieceType = "bishop"
	Knight PieceType = "knight"
	Pawn   PieceType = "pawn"
)

type Piece struct {
	Type  PieceType `json:"type"`
	Color string    `json:"color"`
}

type Move struct {
	FromRow int `json:"fromRow"`
	FromCol int `json:"fromCol"`
	ToRow   int `json:"toRow"`
	ToCol   int `json:"toCol"`
}

type ChessGame struct {
	Board          [8][8]*Piece       `json:"board"`
	CurrentPlayer  string             `json:"currentPlayer"`
	GameOver       bool               `json:"gameOver"`
	MoveHistory    []Move             `json:"moveHistory"`
	CapturedPieces map[string][]Piece `json:"capturedPieces"`
}

// Global game state
var game ChessGame
var gameLock sync.RWMutex

// Global database connection
var db *sqlx.DB

// AuthMiddleware checks for valid JWT token
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log all cookies for debugging
		log.Printf("AuthMiddleware: Request to %s, cookies: %v", r.URL.Path, r.Cookies())

		// Get token from cookie - try both possible cookie names
		var cookie *http.Cookie
		var err error

		cookie, err = r.Cookie("auth_token")
		if err != nil {
			// Try alternative cookie name for debugging
			cookie, err = r.Cookie("woodpecker_auth")
			if err != nil {
				log.Printf("AuthMiddleware: No auth cookie found (tried both auth_token and woodpecker_auth): %v", err)
				// For API endpoints, return 401 instead of redirect
				if strings.HasPrefix(r.URL.Path, "/api/") {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				http.Redirect(w, r, "/auth/sign-in", http.StatusSeeOther)
				return
			}
			log.Printf("AuthMiddleware: Found woodpecker_auth cookie instead of auth_token")
		}

		// Validate token
		claims, err := auth.ValidateJWT(cookie.Value)
		if err != nil {
			log.Printf("AuthMiddleware: Invalid JWT token: %v", err)
			// For API endpoints, return 401 instead of redirect
			if strings.HasPrefix(r.URL.Path, "/api/") {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/auth/sign-in", http.StatusSeeOther)
			return
		}

		log.Printf("AuthMiddleware: Valid token for user %s", claims.Email)
		// Add user info to request context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "user_email", claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	// Initialize database
	var err error
	db, err = initDatabase()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	// Seed puzzles
	if err := seedPuzzles(db); err != nil {
		log.Printf("Warning: Failed to seed puzzles: %v", err)
	}

	// Seed test user
	log.Println("Starting to seed test user...")
	if err := seedTestUser(db); err != nil {
		log.Printf("Warning: Failed to seed test user: %v", err)
	} else {
		log.Println("Test user seeding completed successfully")
	}

	// Seed demo set
	if err := seedDemoSet(db); err != nil {
		log.Printf("Warning: Failed to seed demo set: %v", err)
	}

	// Initialize the chess game
	initializeGame()

	// Initialize woodpecker service
	woodpeckerService := woodpecker.NewService(db)

	// Initialize cron job for daily plan updates
	c := cron.New(cron.WithLocation(time.Local))

	// Add cron job to run at 00:05 every day
	_, err = c.AddFunc("5 0 * * *", func() {
		log.Println("Running daily plan update cron job")
		updateDailyPlans(woodpeckerService)
	})
	if err != nil {
		log.Printf("Failed to add cron job: %v", err)
	}

	// Start cron scheduler
	c.Start()

	// Create a new router
	r := mux.NewRouter()

	// Serve static files from /web directory
	webDir := "web"
	if _, err := os.Stat(webDir); os.IsNotExist(err) {
		// If /web doesn't exist, serve from current directory (for backward compatibility)
		webDir = "."
	}

	// Mount API routes first (more specific routes)
	apiRouter := r.PathPrefix("/api").Subrouter()
	setupAPIRoutes(apiRouter)

	// Wire in dev endpoints
	devsvc := dev.NewService(db)
	apiRouter.HandleFunc("/dev/health", devsvc.Health).Methods("GET")
	apiRouter.HandleFunc("/dev/first-puzzle", devsvc.FirstPuzzle).Methods("GET")
	apiRouter.HandleFunc("/dev/next-puzzle", devsvc.NextPuzzle).Methods("GET")
	apiRouter.HandleFunc("/dev/grade-first-move", devsvc.GradeFirstMove).Methods("POST")

	// Serve static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(webDir, "static")))))
	r.PathPrefix("/images/").Handler(http.StripPrefix("/images/", http.FileServer(http.Dir(filepath.Join(webDir, "images")))))

	// Serve stats page
	r.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "templates", "stats.html"))
	})

	// Serve about page
	r.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "templates", "about.html"))
	})

	// Serve auth pages
	r.HandleFunc("/auth/sign-in", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "templates", "sign-in.html"))
	})
	r.HandleFunc("/auth/sign-up", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "templates", "sign-up.html"))
	})

	// Protected routes - use individual middleware wrapping
	r.HandleFunc("/", AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "templates", "index.html"))
	})).ServeHTTP).Methods("GET")

	r.HandleFunc("/progress", AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "templates", "stats.html"))
	})).ServeHTTP).Methods("GET")

	r.HandleFunc("/settings", AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "templates", "settings.html"))
	})).ServeHTTP).Methods("GET")

	r.HandleFunc("/trainer", AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(webDir, "templates", "trainer.html"))
	})).ServeHTTP).Methods("GET")

	// Start server (PORT is set by most PaaS: Fly.io, Railway, Render)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	if len(port) > 0 && port[0] != ':' {
		port = ":" + port
	}
	log.Printf("Server starting on http://localhost%s", port)
	log.Fatal(http.ListenAndServe(port, r))
}

func setupAPIRoutes(apiRouter *mux.Router) {
	// Health check endpoint
	apiRouter.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}).Methods("GET")

	// Chess game endpoints
	apiRouter.HandleFunc("/game", handleGameState).Methods("GET")
	apiRouter.HandleFunc("/move", handleMove).Methods("POST")
	apiRouter.HandleFunc("/new-game", handleNewGame).Methods("POST")
	apiRouter.HandleFunc("/reset", handleReset).Methods("POST")

	// Puzzle endpoints
	apiRouter.HandleFunc("/puzzles/next", handleNextPuzzle).Methods("GET")
	apiRouter.HandleFunc("/puzzles/grade", handleGradePuzzle).Methods("POST")
	apiRouter.HandleFunc("/puzzles/grade-line", handleGradeLine).Methods("POST")
	apiRouter.HandleFunc("/puzzles/solution-text/{puzzleId}", handleSolutionText).Methods("GET")

	// Stats endpoints
	apiRouter.HandleFunc("/stats", handleStats).Methods("GET")
	apiRouter.HandleFunc("/progress/today", handleTodayProgress).Methods("GET")

	// Daily plan endpoints
	apiRouter.HandleFunc("/daily", handleDailyStatus).Methods("GET")

	// Auth endpoints
	apiRouter.HandleFunc("/auth/sign-up", handleSignUp).Methods("POST")
	apiRouter.HandleFunc("/auth/sign-in", handleSignIn).Methods("POST")
	apiRouter.HandleFunc("/auth/logout", handleLogout).Methods("POST")
	apiRouter.HandleFunc("/me", AuthMiddleware(http.HandlerFunc(handleGetMe)).ServeHTTP).Methods("GET")

	// Trainer endpoints
	apiRouter.HandleFunc("/trainer/sets", AuthMiddleware(http.HandlerFunc(handleTrainerSets)).ServeHTTP).Methods("GET", "POST")
	apiRouter.HandleFunc("/trainer/sets/{id}/puzzles", AuthMiddleware(http.HandlerFunc(handleTrainerSetPuzzles)).ServeHTTP).Methods("GET")
	apiRouter.HandleFunc("/trainer/cycles", AuthMiddleware(http.HandlerFunc(handleTrainerCycles)).ServeHTTP).Methods("POST")
	apiRouter.HandleFunc("/trainer/cycles/active", AuthMiddleware(http.HandlerFunc(handleTrainerActiveCycle)).ServeHTTP).Methods("GET")
	apiRouter.HandleFunc("/trainer/sessions", AuthMiddleware(http.HandlerFunc(handleTrainerSessions)).ServeHTTP).Methods("POST")
	apiRouter.HandleFunc("/trainer/sessions/{id}", AuthMiddleware(http.HandlerFunc(handleTrainerSessionUpdate)).ServeHTTP).Methods("PUT")

	// TODO: Add more API endpoints here
	// Example:
	// apiRouter.HandleFunc("/puzzles", handlePuzzles).Methods("GET", "POST")
	// apiRouter.HandleFunc("/puzzles/{id}", handlePuzzle).Methods("GET", "PUT", "DELETE")
	// apiRouter.HandleFunc("/users", handleUsers).Methods("GET", "POST")
	// apiRouter.HandleFunc("/auth", handleAuth).Methods("POST")
}

func initDatabase() (*sqlx.DB, error) {
	// Open SQLite database (DATABASE_PATH for production e.g. /data/woodpecker.db on Fly.io volume)
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "woodpecker.db"
	}
	db, err := sqlx.Connect("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Create users table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create puzzles table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS puzzles (
			id TEXT PRIMARY KEY,
			difficulty TEXT NOT NULL,
			fen TEXT NOT NULL,
			side_to_move TEXT NOT NULL,
			solution_json TEXT,
			ticks_json TEXT
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create progress table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS progress (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			puzzle_id TEXT NOT NULL,
			attempts INTEGER DEFAULT 1,
			score INTEGER DEFAULT 0,
			solved_at DATETIME,
			typed_json TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, puzzle_id)
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create daily_plans table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS daily_plans (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			daily_plan_json TEXT NOT NULL,
			active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id)
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create users table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create sets table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			difficulty_min TEXT,
			difficulty_max TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create set_puzzles table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS set_puzzles (
			set_id INTEGER NOT NULL,
			puzzle_id TEXT NOT NULL,
			position INTEGER NOT NULL,
			PRIMARY KEY (set_id, puzzle_id),
			FOREIGN KEY (set_id) REFERENCES sets(id),
			FOREIGN KEY (puzzle_id) REFERENCES puzzles(id)
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create cycles table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS cycles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			set_id INTEGER NOT NULL,
			cycle_index INTEGER NOT NULL,
			target_days INTEGER NOT NULL,
			started_at DATETIME,
			ended_at DATETIME,
			status TEXT NOT NULL DEFAULT 'planned',
			FOREIGN KEY (set_id) REFERENCES sets(id)
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create sessions table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cycle_id INTEGER NOT NULL,
			started_at DATETIME,
			ended_at DATETIME,
			target_count INTEGER NOT NULL,
			FOREIGN KEY (cycle_id) REFERENCES cycles(id)
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create attempts table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS attempts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id INTEGER NOT NULL,
			puzzle_id TEXT NOT NULL,
			started_at DATETIME,
			ended_at DATETIME,
			score_first_move INTEGER DEFAULT 0,
			score_ticks INTEGER DEFAULT 0,
			total_points INTEGER DEFAULT 0,
			time_ms INTEGER DEFAULT 0,
			correct_first_move BOOLEAN DEFAULT 0,
			FOREIGN KEY (session_id) REFERENCES sessions(id),
			FOREIGN KEY (puzzle_id) REFERENCES puzzles(id)
		)
	`)
	if err != nil {
		return nil, err
	}

	// Create user_settings table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_settings (
			user_id TEXT PRIMARY KEY,
			daily_goal_minutes INTEGER DEFAULT 30,
			reminders_enabled BOOLEAN DEFAULT 1,
			timezone TEXT DEFAULT 'UTC',
			FOREIGN KEY (user_id) REFERENCES users(id)
		)
	`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func initializeGame() {
	gameLock.Lock()
	defer gameLock.Unlock()

	// Initialize captured pieces
	game.CapturedPieces = make(map[string][]Piece)
	game.CapturedPieces["white"] = []Piece{}
	game.CapturedPieces["black"] = []Piece{}

	// Setup initial board
	setupPieces()
}

func setupPieces() {
	// Clear the board
	for i := 0; i < 8; i++ {
		for j := 0; j < 8; j++ {
			game.Board[i][j] = nil
		}
	}

	// Setup pawns
	for i := 0; i < 8; i++ {
		game.Board[1][i] = &Piece{Type: Pawn, Color: "black"}
		game.Board[6][i] = &Piece{Type: Pawn, Color: "white"}
	}

	// Setup other pieces
	// Black pieces (top row)
	game.Board[0][0] = &Piece{Type: Rook, Color: "black"}
	game.Board[0][1] = &Piece{Type: Knight, Color: "black"}
	game.Board[0][2] = &Piece{Type: Bishop, Color: "black"}
	game.Board[0][3] = &Piece{Type: Queen, Color: "black"}
	game.Board[0][4] = &Piece{Type: King, Color: "black"}
	game.Board[0][5] = &Piece{Type: Bishop, Color: "black"}
	game.Board[0][6] = &Piece{Type: Knight, Color: "black"}
	game.Board[0][7] = &Piece{Type: Rook, Color: "black"}

	// White pieces (bottom row)
	game.Board[7][0] = &Piece{Type: Rook, Color: "white"}
	game.Board[7][1] = &Piece{Type: Knight, Color: "white"}
	game.Board[7][2] = &Piece{Type: Bishop, Color: "white"}
	game.Board[7][3] = &Piece{Type: Queen, Color: "white"}
	game.Board[7][4] = &Piece{Type: King, Color: "white"}
	game.Board[7][5] = &Piece{Type: Bishop, Color: "white"}
	game.Board[7][6] = &Piece{Type: Knight, Color: "white"}
	game.Board[7][7] = &Piece{Type: Rook, Color: "white"}

	game.CurrentPlayer = "white"
	game.GameOver = false
	game.MoveHistory = []Move{}
}

func handleGameState(w http.ResponseWriter, r *http.Request) {
	gameLock.RLock()
	defer gameLock.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

func handleMove(w http.ResponseWriter, r *http.Request) {
	var move Move
	if err := json.NewDecoder(r.Body).Decode(&move); err != nil {
		http.Error(w, "Invalid move data", http.StatusBadRequest)
		return
	}

	gameLock.Lock()
	defer gameLock.Unlock()

	if game.GameOver {
		http.Error(w, "Game is over", http.StatusBadRequest)
		return
	}

	// Validate move
	if !isValidMove(move) {
		http.Error(w, "Invalid move", http.StatusBadRequest)
		return
	}

	// Make the move
	makeMove(move)

	// Check for game over
	if isCheckmate() {
		game.GameOver = true
	}

	// Switch players
	if game.CurrentPlayer == "white" {
		game.CurrentPlayer = "black"
	} else {
		game.CurrentPlayer = "white"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

func handleNewGame(w http.ResponseWriter, r *http.Request) {
	gameLock.Lock()
	defer gameLock.Unlock()

	setupPieces()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	gameLock.Lock()
	defer gameLock.Unlock()

	setupPieces()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

func isValidMove(move Move) bool {
	// Check bounds
	if move.FromRow < 0 || move.FromRow > 7 || move.FromCol < 0 || move.FromCol > 7 ||
		move.ToRow < 0 || move.ToRow > 7 || move.ToCol < 0 || move.ToCol > 7 {
		return false
	}

	// Check if source square has a piece
	fromPiece := game.Board[move.FromRow][move.FromCol]
	if fromPiece == nil {
		return false
	}

	// Check if it's the player's turn
	if fromPiece.Color != game.CurrentPlayer {
		return false
	}

	// Check if destination square has own piece
	toPiece := game.Board[move.ToRow][move.ToCol]
	if toPiece != nil && toPiece.Color == fromPiece.Color {
		return false
	}

	// Validate piece-specific moves
	switch fromPiece.Type {
	case Pawn:
		return isValidPawnMove(move)
	case Rook:
		return isValidRookMove(move)
	case Knight:
		return isValidKnightMove(move)
	case Bishop:
		return isValidBishopMove(move)
	case Queen:
		return isValidQueenMove(move)
	case King:
		return isValidKingMove(move)
	}

	return false
}

func isValidPawnMove(move Move) bool {
	fromPiece := game.Board[move.FromRow][move.FromCol]
	rowDiff := move.ToRow - move.FromRow
	colDiff := abs(move.ToCol - move.FromCol)

	// White pawns move up (decreasing row), black pawns move down (increasing row)
	direction := 1
	if fromPiece.Color == "white" {
		direction = -1
	}

	// Forward move
	if colDiff == 0 {
		// Single square move
		if rowDiff == direction {
			return game.Board[move.ToRow][move.ToCol] == nil
		}
		// Double square move from starting position
		if (fromPiece.Color == "white" && move.FromRow == 6 && rowDiff == -2) ||
			(fromPiece.Color == "black" && move.FromRow == 1 && rowDiff == 2) {
			return game.Board[move.ToRow][move.ToCol] == nil &&
				game.Board[move.FromRow+direction][move.FromCol] == nil
		}
		return false
	}

	// Diagonal capture
	if abs(colDiff) == 1 && rowDiff == direction {
		return game.Board[move.ToRow][move.ToCol] != nil
	}

	return false
}

func isValidRookMove(move Move) bool {
	rowDiff := move.ToRow - move.FromRow
	colDiff := move.ToCol - move.FromCol

	// Rook moves horizontally or vertically
	if rowDiff != 0 && colDiff != 0 {
		return false
	}

	// Check path is clear
	if rowDiff == 0 {
		// Horizontal move
		start := min(move.FromCol, move.ToCol)
		end := max(move.FromCol, move.ToCol)
		for col := start + 1; col < end; col++ {
			if game.Board[move.FromRow][col] != nil {
				return false
			}
		}
	} else {
		// Vertical move
		start := min(move.FromRow, move.ToRow)
		end := max(move.FromRow, move.ToRow)
		for row := start + 1; row < end; row++ {
			if game.Board[row][move.FromCol] != nil {
				return false
			}
		}
	}

	return true
}

func isValidKnightMove(move Move) bool {
	rowDiff := abs(move.ToRow - move.FromRow)
	colDiff := abs(move.ToCol - move.FromCol)

	// Knight moves in L-shape: 2 squares in one direction, 1 square perpendicular
	return (rowDiff == 2 && colDiff == 1) || (rowDiff == 1 && colDiff == 2)
}

func isValidBishopMove(move Move) bool {
	rowDiff := move.ToRow - move.FromRow
	colDiff := move.ToCol - move.FromCol

	// Bishop moves diagonally
	if abs(rowDiff) != abs(colDiff) {
		return false
	}

	// Check path is clear
	rowStep := 1
	if rowDiff < 0 {
		rowStep = -1
	}
	colStep := 1
	if colDiff < 0 {
		colStep = -1
	}

	row := move.FromRow + rowStep
	col := move.FromCol + colStep
	for row != move.ToRow && col != move.ToCol {
		if game.Board[row][col] != nil {
			return false
		}
		row += rowStep
		col += colStep
	}

	return true
}

func isValidQueenMove(move Move) bool {
	// Queen combines rook and bishop moves
	return isValidRookMove(move) || isValidBishopMove(move)
}

func isValidKingMove(move Move) bool {
	rowDiff := abs(move.ToRow - move.FromRow)
	colDiff := abs(move.ToCol - move.FromCol)

	// King moves one square in any direction
	return rowDiff <= 1 && colDiff <= 1
}

func makeMove(move Move) {
	// Capture piece if present
	if game.Board[move.ToRow][move.ToCol] != nil {
		capturedPiece := game.Board[move.ToRow][move.ToCol]
		opponentColor := "white"
		if capturedPiece.Color == "white" {
			opponentColor = "black"
		}
		game.CapturedPieces[opponentColor] = append(game.CapturedPieces[opponentColor], *capturedPiece)
	}

	// Move piece
	game.Board[move.ToRow][move.ToCol] = game.Board[move.FromRow][move.FromCol]
	game.Board[move.FromRow][move.FromCol] = nil

	// Add to move history
	game.MoveHistory = append(game.MoveHistory, move)
}

func isCheckmate() bool {
	// Simple checkmate detection: if king is captured
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			if game.Board[row][col] != nil && game.Board[row][col].Type == King {
				return false
			}
		}
	}
	return true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Puzzle API handlers
func handleNextPuzzle(w http.ResponseWriter, r *http.Request) {
	difficulty := r.URL.Query().Get("difficulty")
	if difficulty == "" {
		http.Error(w, "difficulty parameter required", http.StatusBadRequest)
		return
	}

	// Validate difficulty
	validDifficulties := map[string]bool{"easy": true, "intermediate": true, "advanced": true}
	if !validDifficulties[difficulty] {
		http.Error(w, "invalid difficulty: must be easy, intermediate, or advanced", http.StatusBadRequest)
		return
	}

	// Check if a specific puzzle ID was requested
	requestedPuzzleID := r.URL.Query().Get("puzzleId")
	if requestedPuzzleID != "" {
		// Get the specific puzzle
		var puzzle model.PuzzleDB
		err := db.Get(&puzzle, `
			SELECT id, fen, side_to_move, difficulty 
			FROM puzzles 
			WHERE id = ? AND difficulty = ?
		`, requestedPuzzleID, difficulty)

		if err != nil {
			http.Error(w, "puzzle not found: "+requestedPuzzleID, http.StatusNotFound)
			return
		}

		response := map[string]interface{}{
			"id":         puzzle.ID,
			"fen":        puzzle.FEN,
			"sideToMove": extractSideToMove(puzzle.FEN),
			"difficulty": puzzle.Difficulty,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// TODO: Get user ID from session/auth
	userID := "default_user"

	// Initialize woodpecker service
	woodpeckerService := woodpecker.NewService(db)

	// Get next puzzle from daily plan
	puzzleID, err := woodpeckerService.GetNextPuzzle(userID, difficulty)
	if err != nil {
		// Fallback to ordered puzzle if daily plan fails
		var puzzle model.PuzzleDB
		err := db.Get(&puzzle, `
			SELECT id, fen, side_to_move, difficulty 
			FROM puzzles 
			WHERE difficulty = ? 
			ORDER BY id 
			LIMIT 1
		`, difficulty)

		if err != nil {
			http.Error(w, "no puzzles found for difficulty: "+difficulty, http.StatusNotFound)
			return
		}

		response := map[string]interface{}{
			"id":         puzzle.ID,
			"fen":        puzzle.FEN,
			"sideToMove": extractSideToMove(puzzle.FEN),
			"difficulty": puzzle.Difficulty,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get puzzle details
	var puzzle model.PuzzleDB
	err = db.Get(&puzzle, `
		SELECT id, fen, side_to_move, difficulty 
		FROM puzzles 
		WHERE id = ?
	`, puzzleID)

	if err != nil {
		http.Error(w, "puzzle not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"id":         puzzle.ID,
		"fen":        puzzle.FEN,
		"sideToMove": extractSideToMove(puzzle.FEN),
		"difficulty": puzzle.Difficulty,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type GradeRequest struct {
	PuzzleID  string   `json:"puzzleId"`
	PlayedSAN []string `json:"playedSans"`
}

type GradeResponse struct {
	Correct     bool     `json:"correct"`
	Score       int      `json:"score"`
	MatchedLine []string `json:"matchedLine"`
}

func handleGradePuzzle(w http.ResponseWriter, r *http.Request) {
	var req GradeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.PuzzleID == "" {
		http.Error(w, "puzzleId required", http.StatusBadRequest)
		return
	}

	// Load puzzle from database
	var puzzleDB model.PuzzleDB
	err := db.Get(&puzzleDB, `
		SELECT id, fen, side_to_move, difficulty, solution_json, ticks_json 
		FROM puzzles 
		WHERE id = ?
	`, req.PuzzleID)

	if err != nil {
		http.Error(w, "puzzle not found", http.StatusNotFound)
		return
	}

	// Convert to model.Puzzle
	puzzle := puzzleDB.ToPuzzle()

	// Grade the solution
	correct, score, matchedLine := gradeSolution(puzzle, req.PlayedSAN)

	// TODO: Save/merge progress row for user
	// For now, just return the grade

	response := GradeResponse{
		Correct:     correct,
		Score:       score,
		MatchedLine: matchedLine,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func gradeSolution(puzzle *model.Puzzle, playedSAN []string) (bool, int, []string) {
	if len(playedSAN) == 0 {
		return false, 0, nil
	}

	// Check if first move matches any solution line
	correct := false
	score := 0
	var matchedLine []string

	// Depth-first search through solution lines
	var dfs func(lines []model.Line, depth int) bool
	dfs = func(lines []model.Line, depth int) bool {
		for _, line := range lines {
			if line.SAN == "" {
				continue
			}

			// Check if this move matches what was played
			if depth < len(playedSAN) && line.SAN == playedSAN[depth] {
				// This move matches, check if it's a tick
				if line.IsTick {
					score++
				}

				// If this is the first move and it matches, mark as correct
				if depth == 0 {
					correct = true
				}

				// Add to matched line
				if depth < len(matchedLine) {
					matchedLine[depth] = line.SAN
				} else {
					matchedLine = append(matchedLine, line.SAN)
				}

				// Continue with children if we have more moves to check
				if depth+1 < len(playedSAN) && len(line.Children) > 0 {
					if dfs(line.Children, depth+1) {
						return true
					}
				} else if depth+1 >= len(playedSAN) {
					// We've matched all played moves
					return true
				}
			}
		}
		return false
	}

	dfs(puzzle.Solution.Lines, 0)

	return correct, score, matchedLine
}

// GradeLineRequest represents the request body for grading a line of moves
type GradeLineRequest struct {
	PuzzleID string   `json:"puzzleId"`
	TypedSAN []string `json:"typedSans"`
}

// GradeLineResponse represents the response for grading a line of moves
type GradeLineResponse struct {
	Correct         bool     `json:"correct"`
	Score           int      `json:"score"`
	TicksMatched    []int    `json:"ticksMatched"`
	DepthMatched    int      `json:"depthMatched"`
	EarliestMistake *int     `json:"earliestMistake"`
	BestLine        []string `json:"bestLine"`
	RequiredTicks   []string `json:"requiredTicks"`
}

func handleGradeLine(w http.ResponseWriter, r *http.Request) {
	var req GradeLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if req.PuzzleID == "" {
		http.Error(w, "puzzleId required", http.StatusBadRequest)
		return
	}

	// Load puzzle from database
	var puzzleDB model.PuzzleDB
	err := db.Get(&puzzleDB, `
		SELECT id, fen, side_to_move, difficulty, solution_json, ticks_json 
		FROM puzzles 
		WHERE id = ?
	`, req.PuzzleID)

	if err != nil {
		http.Error(w, "puzzle not found", http.StatusNotFound)
		return
	}

	// Convert to model.Puzzle
	puzzle := puzzleDB.ToPuzzle()

	// Grade the line
	response := gradeLine(puzzle, req.TypedSAN)

	// Save progress (for now using a default user_id)
	userID := "default_user" // TODO: Get from session/auth
	saveProgress(userID, req.PuzzleID, req.TypedSAN, response.Score, response.DepthMatched)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func gradeLine(puzzle *model.Puzzle, typedSAN []string) GradeLineResponse {
	response := GradeLineResponse{
		Correct:         false,
		Score:           0,
		TicksMatched:    []int{},
		DepthMatched:    0,
		EarliestMistake: nil,
		BestLine:        []string{},
		RequiredTicks:   puzzle.Ticks,
	}

	if len(typedSAN) == 0 {
		return response
	}

	// For flat solution structure, just check moves in order
	var ticksMatched []int
	var depthMatched int
	var earliestMistake *int
	var bestLine []string

	// Check each typed move against the solution
	for i, typedMove := range typedSAN {
		if i >= len(puzzle.Solution.Lines) {
			// More moves typed than in solution
			if earliestMistake == nil {
				earliestMistake = &i
			}
			break
		}

		solutionMove := puzzle.Solution.Lines[i]
		normalizedTyped := normalizeSAN(typedMove)
		normalizedSolution := normalizeSAN(solutionMove.SAN)

		if normalizedTyped == normalizedSolution {
			// Move matches
			bestLine = append(bestLine, solutionMove.SAN)
			depthMatched = i + 1

			// If this is the first move and it matches, mark as correct
			if i == 0 {
				response.Correct = true
			}

			// Check if this is a tick move
			if solutionMove.IsTick {
				ticksMatched = append(ticksMatched, i)
			}
		} else {
			// Move doesn't match - this is a mistake
			if earliestMistake == nil {
				earliestMistake = &i
			}
			break
		}
	}

	// Update response with results
	response.BestLine = bestLine
	response.TicksMatched = ticksMatched
	response.DepthMatched = depthMatched
	response.EarliestMistake = earliestMistake

	// Calculate score: 1 if first move correct, plus 1 for each tick matched
	if response.Correct {
		response.Score = 1 + len(ticksMatched)
	}

	return response
}

// normalizeSAN normalizes SAN notation for comparison
// Accepts various SAN formats and returns a canonical form for comparison
func normalizeSAN(s string) string {
	// Convert to lowercase and trim whitespace
	s = strings.ToLower(strings.TrimSpace(s))

	// Accept 0-0 = O-O, 0-0-0 = O-O-O
	s = strings.ReplaceAll(s, "0-0-0", "o-o-o")
	s = strings.ReplaceAll(s, "0-0", "o-o")

	// Strip +, #, !?, move numbers, and ellipses
	s = strings.ReplaceAll(s, "+", "")
	s = strings.ReplaceAll(s, "#", "")
	s = strings.ReplaceAll(s, "!", "")
	s = strings.ReplaceAll(s, "?", "")
	s = strings.ReplaceAll(s, "!?", "")
	s = strings.ReplaceAll(s, "?!", "")

	// Remove move numbers and ellipses (e.g., "1.", "1...", "12.")
	// Handle patterns like "1.", "12.", "1...", "12..." at the start
	s = strings.ReplaceAll(s, "...", "")
	s = strings.ReplaceAll(s, "..", "")
	s = strings.ReplaceAll(s, ".", "")

	// Remove any remaining digits at the start
	for len(s) > 0 && s[0] >= '0' && s[0] <= '9' {
		s = s[1:]
	}

	// Handle promotions: accept e8=Q and e8Q
	if strings.Contains(s, "=") {
		parts := strings.Split(s, "=")
		if len(parts) == 2 {
			s = parts[0] + parts[1]
		}
	}

	// Remove any remaining whitespace
	s = strings.ReplaceAll(s, " ", "")

	return s
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

// saveProgress saves or updates progress for a user on a puzzle
func saveProgress(userID, puzzleID string, typedSAN []string, score, depthMatched int) {
	typedJSON, _ := json.Marshal(typedSAN)

	// Check if progress already exists
	var existingID int
	err := db.Get(&existingID, `
		SELECT id FROM progress 
		WHERE user_id = ? AND puzzle_id = ?
	`, userID, puzzleID)

	if err != nil {
		// No existing progress, insert new
		_, err = db.Exec(`
			INSERT INTO progress (user_id, puzzle_id, attempts, score, typed_json, updated_at)
			VALUES (?, ?, 1, ?, ?, CURRENT_TIMESTAMP)
		`, userID, puzzleID, score, string(typedJSON))
	} else {
		// Update existing progress
		_, err = db.Exec(`
			UPDATE progress 
			SET attempts = attempts + 1, 
				score = ?, 
				typed_json = ?,
				solved_at = CASE WHEN ? >= 3 THEN CURRENT_TIMESTAMP ELSE solved_at END,
				updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ? AND puzzle_id = ?
		`, score, string(typedJSON), depthMatched, userID, puzzleID)
	}

	if err != nil {
		log.Printf("Error saving progress: %v", err)
	}
}

// handleTodayProgress returns today's progress summary
func handleTodayProgress(w http.ResponseWriter, r *http.Request) {
	userID := "default_user" // TODO: Get from session/auth

	// For now, return a simple response
	result := struct {
		TotalAttempted int     `json:"totalAttempted"`
		TotalSolved    int     `json:"totalSolved"`
		AverageScore   float64 `json:"averageScore"`
	}{
		TotalAttempted: 0,
		TotalSolved:    0,
		AverageScore:   0.0,
	}

	// Try to get actual data if possible
	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM progress WHERE user_id = ?`, userID)
	if err == nil && count > 0 {
		// We have data, try to get stats
		var totalAttempted, totalSolved int
		var avgScore float64

		err1 := db.Get(&totalAttempted, `SELECT COUNT(*) FROM progress WHERE user_id = ?`, userID)
		err2 := db.Get(&totalSolved, `SELECT COUNT(*) FROM progress WHERE user_id = ? AND solved_at IS NOT NULL`, userID)
		err3 := db.Get(&avgScore, `SELECT AVG(score) FROM progress WHERE user_id = ?`, userID)

		if err1 == nil && err2 == nil && err3 == nil {
			result.TotalAttempted = totalAttempted
			result.TotalSolved = totalSolved
			result.AverageScore = avgScore
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleStats serves the stats page
func handleStats(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/templates/stats.html")
}

// handleDailyStatus returns the current daily plan status
func handleDailyStatus(w http.ResponseWriter, r *http.Request) {
	userID := "default_user" // TODO: Get from session/auth

	woodpeckerService := woodpecker.NewService(db)
	status, err := woodpeckerService.GetDailyStatus(userID)
	if err != nil {
		log.Printf("Error getting daily status: %v", err)
		http.Error(w, "failed to get daily status", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// updateDailyPlans updates daily plans for all users
func updateDailyPlans(service *woodpecker.Service) {
	// Get all active users
	var userIDs []string
	err := db.Select(&userIDs, `SELECT DISTINCT user_id FROM daily_plans WHERE active = 1`)
	if err != nil {
		log.Printf("Error getting users for daily plan update: %v", err)
		return
	}

	// Add default user if no users exist
	if len(userIDs) == 0 {
		userIDs = []string{"default_user"}
	}

	for _, userID := range userIDs {
		// Get or create daily plan
		plan, err := service.GetOrCreateDailyPlan(userID)
		if err != nil {
			log.Printf("Error getting daily plan for user %s: %v", userID, err)
			continue
		}

		// Build today's batch
		todayBatch, err := service.BuildTodayBatch(userID, plan)
		if err != nil {
			log.Printf("Error building today's batch for user %s: %v", userID, err)
			continue
		}

		// Update plan with today's batch
		plan.TodayBatch = todayBatch
		planJSON, _ := json.Marshal(plan)

		_, err = db.Exec(`
			UPDATE daily_plans 
			SET daily_plan_json = ?, updated_at = CURRENT_TIMESTAMP
			WHERE user_id = ? AND active = 1
		`, string(planJSON), userID)

		if err != nil {
			log.Printf("Error updating daily plan for user %s: %v", userID, err)
		} else {
			log.Printf("Updated daily plan for user %s: %d puzzles for today", userID, len(todayBatch))
		}
	}
}

// handleSolutionText returns the solution text for a given puzzle ID
func handleSolutionText(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	puzzleId := vars["puzzleId"]

	if puzzleId == "" {
		http.Error(w, "puzzle ID is required", http.StatusBadRequest)
		return
	}

	// Get solution text from the mapping
	solutionsText := SolutionsTextEasy()
	solutionText, exists := solutionsText[puzzleId]

	if !exists {
		http.Error(w, "solution text not found for puzzle ID", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"puzzleId":     puzzleId,
		"solutionText": solutionText,
	})
}

// Auth handlers
func handleSignUp(w http.ResponseWriter, r *http.Request) {
	var req auth.SignUpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	if len(req.Password) < 6 {
		http.Error(w, "Password must be at least 6 characters", http.StatusBadRequest)
		return
	}

	// Create user
	userService := user.NewService(db)
	user, err := userService.CreateUser(req.Email, req.Password)
	if err != nil {
		if err == auth.ErrUserExists {
			http.Error(w, "User already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token, err := auth.GenerateJWT(user.ID, user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Set HTTP-only cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		Secure:   false,                // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode, // Changed from StrictMode to LaxMode for better compatibility
	})

	log.Printf("Set auth cookie for new user %s", user.Email)

	response := auth.AuthResponse{
		User: *user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleSignIn(w http.ResponseWriter, r *http.Request) {
	var req auth.SignInRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Validate credentials
	userService := user.NewService(db)
	user, err := userService.ValidateCredentials(req.Email, req.Password)
	if err != nil {
		log.Printf("Sign-in failed for email %s: %v", req.Email, err)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	log.Printf("Sign-in successful for user %s", user.Email)

	// Generate JWT token
	token, err := auth.GenerateJWT(user.ID, user.Email)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Set HTTP-only cookie
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		Secure:   false,                // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode, // Changed from StrictMode to LaxMode for better compatibility
	}
	http.SetCookie(w, cookie)

	log.Printf("Set auth cookie for user %s: %s", user.Email, cookie.String())

	response := auth.AuthResponse{
		User: *user,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Printf("Sign-in response sent for user %s", user.Email)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear the auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	response := map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleGetMe(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by AuthMiddleware)
	userID := r.Context().Value("user_id").(string)

	userService := user.NewService(db)
	user, err := userService.GetUserByID(userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Don't include password hash in response
	user.PasswordHash = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// Trainer API handlers

func handleTrainerSets(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value("user_id").(string)
	repo := repository.NewSQLiteRepository(db)

	switch r.Method {
	case "GET":
		// Get all sets for the user
		sets, err := repo.GetSetsByUserID(userID)
		if err != nil {
			http.Error(w, "Failed to get sets", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sets)

	case "POST":
		// Create a new set
		var setData struct {
			Name          string `json:"name"`
			Description   string `json:"description"`
			DifficultyMin string `json:"difficulty_min"`
			DifficultyMax string `json:"difficulty_max"`
			Size          int    `json:"size"`
		}

		if err := json.NewDecoder(r.Body).Decode(&setData); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Create the set
		set := &model.Set{
			UserID:        userID,
			Name:          setData.Name,
			Description:   setData.Description,
			DifficultyMin: setData.DifficultyMin,
			DifficultyMax: setData.DifficultyMax,
			CreatedAt:     time.Now().Format(time.RFC3339),
		}

		if err := repo.CreateSet(set); err != nil {
			http.Error(w, "Failed to create set", http.StatusInternalServerError)
			return
		}

		// Add puzzles to the set
		var puzzleIDs []string
		rows, err := db.Query(`
			SELECT id FROM puzzles 
			WHERE difficulty >= ? AND difficulty <= ? 
			ORDER BY id LIMIT ?
		`, setData.DifficultyMin, setData.DifficultyMax, setData.Size)
		if err != nil {
			http.Error(w, "Failed to get puzzles", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var puzzleID string
			if err := rows.Scan(&puzzleID); err != nil {
				http.Error(w, "Failed to scan puzzle ID", http.StatusInternalServerError)
				return
			}
			puzzleIDs = append(puzzleIDs, puzzleID)
		}

		// Add puzzles to set
		for i, puzzleID := range puzzleIDs {
			if err := repo.AddPuzzleToSet(set.ID, puzzleID, i+1); err != nil {
				http.Error(w, "Failed to add puzzle to set", http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(set)
	}
}

func handleTrainerSetPuzzles(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	setIDStr := vars["id"]
	setID, err := strconv.Atoi(setIDStr)
	if err != nil {
		http.Error(w, "Invalid set ID", http.StatusBadRequest)
		return
	}

	repo := repository.NewSQLiteRepository(db)
	puzzles, err := repo.GetPuzzlesInSet(setID)
	if err != nil {
		http.Error(w, "Failed to get puzzles", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(puzzles)
}

func handleTrainerCycles(w http.ResponseWriter, r *http.Request) {
	var cycleData struct {
		SetID      int    `json:"set_id"`
		Index      int    `json:"index"`
		TargetDays int    `json:"target_days"`
		Status     string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&cycleData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	repo := repository.NewSQLiteRepository(db)
	cycle := &model.Cycle{
		SetID:      cycleData.SetID,
		Index:      cycleData.Index,
		TargetDays: cycleData.TargetDays,
		Status:     cycleData.Status,
	}

	if err := repo.CreateCycle(cycle); err != nil {
		http.Error(w, "Failed to create cycle", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cycle)
}

func handleTrainerActiveCycle(w http.ResponseWriter, r *http.Request) {
	setIDStr := r.URL.Query().Get("set_id")
	setID, err := strconv.Atoi(setIDStr)
	if err != nil {
		http.Error(w, "Invalid set ID", http.StatusBadRequest)
		return
	}

	repo := repository.NewSQLiteRepository(db)
	cycle, err := repo.GetActiveCycleBySetID(setID)
	if err != nil {
		http.Error(w, "Failed to get active cycle", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cycle)
}

func handleTrainerSessions(w http.ResponseWriter, r *http.Request) {
	var sessionData struct {
		CycleID     int `json:"cycle_id"`
		TargetCount int `json:"target_count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&sessionData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	repo := repository.NewSQLiteRepository(db)
	now := time.Now().Format(time.RFC3339)
	session := &model.Session{
		CycleID:     sessionData.CycleID,
		StartedAt:   &now,
		TargetCount: sessionData.TargetCount,
	}

	if err := repo.CreateSession(session); err != nil {
		http.Error(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func handleTrainerSessionUpdate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionIDStr := vars["id"]
	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		http.Error(w, "Invalid session ID", http.StatusBadRequest)
		return
	}

	var updateData struct {
		EndedAt         *string `json:"ended_at"`
		DurationSeconds int     `json:"duration_seconds"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	repo := repository.NewSQLiteRepository(db)
	session, err := repo.GetSessionByID(sessionID)
	if err != nil {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	session.EndedAt = updateData.EndedAt

	if err := repo.UpdateSession(session); err != nil {
		http.Error(w, "Failed to update session", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}
