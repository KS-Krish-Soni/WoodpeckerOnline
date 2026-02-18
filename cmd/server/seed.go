package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"woodpecker-online/internal/model"
	"woodpecker-online/internal/repository"
	"woodpecker-online/internal/user"

	"github.com/jmoiron/sqlx"
)

// readPuzzlesFromFile reads puzzles from a FEN list file
func readPuzzlesFromFile(filename string, maxPuzzles int) ([]*model.Puzzle, error) {
	// Read the entire file content first
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Split into lines and process
	lines := strings.Split(string(content), "\n")
	var puzzles []*model.Puzzle

	for _, line := range lines {
		if len(puzzles) >= maxPuzzles {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip lines with placeholder text
		if strings.Contains(line, "<insert fen value here>") || strings.Contains(line, "<insert fen here>") {
			continue
		}

		// Parse line format: "number. FEN_string side_to_move"
		parts := strings.SplitN(line, ". ", 2)
		if len(parts) != 2 {
			continue
		}

		// Extract puzzle number
		puzzleNum, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		// Extract FEN and side to move
		fenParts := strings.Fields(parts[1])
		if len(fenParts) < 2 {
			continue
		}

		// Get the side to move (last part)
		sideToMove := fenParts[len(fenParts)-1]

		// Get the board position (all parts except the last one)
		boardPosition := strings.Join(fenParts[:len(fenParts)-1], " ")

		// Reconstruct full FEN with proper side to move
		fen := fmt.Sprintf("%s %s - - 0 1", boardPosition, sideToMove)

		// Create puzzle ID
		puzzleID := fmt.Sprintf("wpm_easy_%03d", puzzleNum)

		puzzle := &model.Puzzle{
			ID:         puzzleID,
			Difficulty: "easy",
			FEN:        fen,
		}

		puzzles = append(puzzles, puzzle)
	}

	return puzzles, nil
}

// seedPuzzles inserts sample puzzles into the database
func seedPuzzles(db *sqlx.DB) error {
	log.Println("Seeding puzzles...")

	// Read puzzles from fen_list_easy.txt file
	puzzles, err := readPuzzlesFromFile("fen_list_easy.txt", 5)
	if err != nil {
		return fmt.Errorf("failed to read puzzles from file: %v", err)
	}

	// Load solutions and ticks from easy_solutions.go
	easySolutions := SolutionsEasy()

	// Merge solutions and ticks with puzzle data
	for _, puzzle := range puzzles {
		if solutionData, exists := easySolutions[puzzle.ID]; exists {
			puzzle.Solution = solutionData.Solution
			puzzle.Ticks = solutionData.Ticks
		}
	}

	// Check if puzzles already exist (idempotent)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM puzzles").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Printf("Found %d existing puzzles, skipping seed", count)
		return nil
	}

	// Insert puzzles
	for _, puzzle := range puzzles {
		puzzleDB := model.FromPuzzle(puzzle)

		_, err := db.Exec(`
			INSERT INTO puzzles (id, difficulty, fen, side_to_move, solution_json, ticks_json)
			VALUES (?, ?, ?, ?, ?, ?)
		`, puzzleDB.ID, puzzleDB.Difficulty, puzzleDB.FEN,
			extractSideToMove(puzzleDB.FEN), puzzleDB.SolutionJSON, puzzleDB.TicksJSON)

		if err != nil {
			return err
		}

		log.Printf("Inserted puzzle: %s (%s)", puzzle.ID, puzzle.Difficulty)
	}

	log.Printf("Successfully seeded %d puzzles", len(puzzles))
	return nil
}

// seedTestUser creates a test user for development
func seedTestUser(db *sqlx.DB) error {
	log.Println("Seeding test user...")

	// Check if test user already exists
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "test@example.com").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("Test user already exists, skipping")
		return nil
	}

	// Create test user
	userService := user.NewService(db)
	testUser, err := userService.CreateUser("test@example.com", "password123")
	if err != nil {
		return fmt.Errorf("failed to create test user: %v", err)
	}

	log.Printf("Created test user: %s (ID: %s)", testUser.Email, testUser.ID)
	return nil
}

// seedDemoSet creates a demo set with the first 5 easy puzzles for the test user
func seedDemoSet(db *sqlx.DB) error {
	log.Println("Seeding demo set...")

	// Get the test user
	var testUserID string
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", "test@example.com").Scan(&testUserID)
	if err != nil {
		return fmt.Errorf("failed to find test user: %v", err)
	}

	// Check if demo set already exists
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sets WHERE user_id = ? AND name = ?", testUserID, "Demo Easy Set").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		log.Println("Demo set already exists, skipping")
		return nil
	}

	// Create repository
	repo := repository.NewSQLiteRepository(db)

	// Create the demo set
	demoSet := &model.Set{
		UserID:        testUserID,
		Name:          "Demo Easy Set",
		Description:   "A demonstration set containing the first 5 easy puzzles for testing the Woodpecker Method",
		DifficultyMin: "easy",
		DifficultyMax: "easy",
		CreatedAt:     time.Now().Format(time.RFC3339),
	}

	err = repo.CreateSet(demoSet)
	if err != nil {
		return fmt.Errorf("failed to create demo set: %v", err)
	}

	// Get the first 5 easy puzzles
	var puzzleIDs []string
	rows, err := db.Query("SELECT id FROM puzzles WHERE difficulty = 'easy' ORDER BY id LIMIT 5")
	if err != nil {
		return fmt.Errorf("failed to query puzzles: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var puzzleID string
		if err := rows.Scan(&puzzleID); err != nil {
			return fmt.Errorf("failed to scan puzzle ID: %v", err)
		}
		puzzleIDs = append(puzzleIDs, puzzleID)
	}

	if len(puzzleIDs) == 0 {
		return fmt.Errorf("no easy puzzles found to add to demo set")
	}

	// Add puzzles to the set
	for i, puzzleID := range puzzleIDs {
		err = repo.AddPuzzleToSet(demoSet.ID, puzzleID, i+1)
		if err != nil {
			return fmt.Errorf("failed to add puzzle %s to set: %v", puzzleID, err)
		}
	}

	// Create the first cycle for the set
	cycle := &model.Cycle{
		SetID:      demoSet.ID,
		Index:      1,
		TargetDays: 1,
		Status:     "planned",
	}

	err = repo.CreateCycle(cycle)
	if err != nil {
		return fmt.Errorf("failed to create demo cycle: %v", err)
	}

	// Create default user settings for the test user
	settings := &model.UserSettings{
		UserID:           testUserID,
		DailyGoalMinutes: 30,
		RemindersEnabled: true,
		Timezone:         "UTC",
	}

	err = repo.CreateUserSettings(settings)
	if err != nil {
		return fmt.Errorf("failed to create user settings: %v", err)
	}

	log.Printf("Created demo set '%s' with %d puzzles and initial cycle", demoSet.Name, len(puzzleIDs))
	return nil
}
