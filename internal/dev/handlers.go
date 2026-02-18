package dev

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jmoiron/sqlx"
)

type Service struct{ DB *sqlx.DB }

func NewService(db *sqlx.DB) *Service { return &Service{DB: db} }

type line struct {
	SAN      string `json:"san"`
	Children []line `json:"children,omitempty"`
	IsTick   bool   `json:"isTick,omitempty"`
}

type solution struct {
	Lines                []line     `json:"lines"`
	AcceptedAlternatives [][]string `json:"acceptedAlternatives,omitempty"`
}

type puzzleRow struct {
	ID          string `db:"id" json:"id"`
	FEN         string `db:"fen" json:"fen"`
	SideToMove  string `db:"side_to_move" json:"sideToMove"`
	SolutionRaw string `db:"solution_json"`
}

func (s *Service) Health(w http.ResponseWriter, r *http.Request) {
	var n int
	_ = s.DB.Get(&n, `SELECT COUNT(*) FROM puzzles`)
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "puzzles": n})
}

func (s *Service) FirstPuzzle(w http.ResponseWriter, r *http.Request) {
	var p puzzleRow
	err := s.DB.Get(&p, `SELECT id, fen, side_to_move, solution_json FROM puzzles ORDER BY rowid LIMIT 1`)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"id": p.ID, "fen": p.FEN, "sideToMove": extractSideToMove(p.FEN),
	})
}

func (s *Service) NextPuzzle(w http.ResponseWriter, r *http.Request) {
	// Get the current puzzle index from query parameter
	currentID := r.URL.Query().Get("current")
	
	var p puzzleRow
	var err error
	
	if currentID == "" {
		// If no current puzzle, get the first one
		err = s.DB.Get(&p, `SELECT id, fen, side_to_move, solution_json FROM puzzles ORDER BY rowid LIMIT 1`)
	} else {
		// Get the next puzzle after the current one
		err = s.DB.Get(&p, `SELECT id, fen, side_to_move, solution_json FROM puzzles WHERE rowid > (SELECT rowid FROM puzzles WHERE id = ?) ORDER BY rowid LIMIT 1`, currentID)
		if err != nil {
			// If no next puzzle found, wrap around to the first one
			err = s.DB.Get(&p, `SELECT id, fen, side_to_move, solution_json FROM puzzles ORDER BY rowid LIMIT 1`)
		}
	}
	
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	
	json.NewEncoder(w).Encode(map[string]any{
		"id": p.ID, "fen": p.FEN, "sideToMove": extractSideToMove(p.FEN),
	})
}

type gradeReq struct {
	PuzzleID  string   `json:"puzzleId"`
	PlayedSAN []string `json:"playedSans"`
}

func (s *Service) GradeFirstMove(w http.ResponseWriter, r *http.Request) {
	var req gradeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", 400)
		return
	}
	var p puzzleRow
	if err := s.DB.Get(&p, `SELECT id, fen, side_to_move, solution_json FROM puzzles WHERE id=?`, req.PuzzleID); err != nil {
		http.Error(w, "puzzle not found", 404)
		return
	}
	var sol solution
	if err := json.Unmarshal([]byte(p.SolutionRaw), &sol); err != nil {
		http.Error(w, "solution parse error", 500)
		return
	}
	want := map[string]bool{}
	for _, ln := range sol.Lines {
		if ln.SAN != "" {
			want[ln.SAN] = true
		}
	}
	first := ""
	if len(req.PlayedSAN) > 0 {
		first = req.PlayedSAN[0]
	}
	ok := want[first]
	json.NewEncoder(w).Encode(map[string]any{
		"correct":            ok,
		"expectedFirstMoves": want, // for visibility while testing
	})
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
