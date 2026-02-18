# Woodpecker Online Chess Puzzles

A modern chess puzzle application built with Go, featuring a beautiful web interface and robust backend.

## Features

- **Chess Puzzle Interface**: Solve chess puzzles with a modern, responsive UI
- **Move Validation**: Real-time move validation using chess.js
- **SAN Notation**: Standard Algebraic Notation for move recording
- **Puzzle Management**: Load, submit, and reset puzzle attempts
- **Session Management**: User authentication and session handling
- **Database Integration**: SQLite database for puzzle and user data
- **Scheduled Jobs**: Daily puzzle updates and maintenance tasks

## Project Structure

```
WoodpeckerOnline/
├── cmd/
│   └── server/
│       └── main.go          # Main server application
├── web/
│   ├── templates/
│   │   └── index.html       # Main chess puzzle interface
│   ├── static/
│   │   ├── css/
│   │   │   └── style.css    # Application styling
│   │   └── app.js           # Frontend JavaScript API
│   └── images/
│       └── *.svg            # Chess piece sprites
├── go.mod                   # Go module definition
├── go.sum                   # Dependency checksums
└── README.md               # This file
```

## Dependencies

- **modernc.org/sqlite**: CGO-free SQLite driver
- **github.com/jmoiron/sqlx**: Enhanced database helpers
- **github.com/gorilla/mux**: HTTP router and URL matcher
- **github.com/gorilla/sessions**: Cookie-based sessions
- **golang.org/x/crypto/bcrypt**: Password hashing
- **github.com/robfig/cron/v3**: Scheduled job management

## Getting Started

### Prerequisites

- Go 1.21 or later
- Modern web browser

### Installation

1. Clone the repository:
   ```bash
   git clone <repository-url>
   cd WoodpeckerOnline
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Run the server (run the package so all files in cmd/server are compiled):
   ```bash
   go run ./cmd/server
   ```

4. Open your browser and navigate to:
   ```
   http://localhost:8080
   ```

## API Endpoints

### Health Check
- `GET /api/health` - Server health status

### Chess Game (Legacy)
- `GET /api/game` - Get current game state
- `POST /api/move` - Make a chess move
- `POST /api/new-game` - Start a new game
- `POST /api/reset` - Reset current game

### Puzzle Management (Planned)
- `GET /api/puzzles` - Get available puzzles
- `POST /api/puzzles/{id}/submit` - Submit puzzle solution
- `GET /api/puzzles/{id}/solution` - Get puzzle solution

## Development

### Running the Server

```bash
# Development mode (run package, not a single file)
go run ./cmd/server

# Build and run
go build -o woodpecker ./cmd/server
./woodpecker
```

### File Structure

- **`cmd/server/main.go`**: Main server application with routing and middleware
- **`web/templates/index.html`**: Main chess puzzle interface
- **`web/static/app.js`**: Frontend JavaScript with chess.js integration
- **`web/static/css/style.css`**: Application styling and responsive design

## Features

### Chess Puzzle Interface
- Full-width chess board with fixed sidebar
- Puzzle controls: Load, Submit, Reset, Show Solution
- Real-time status updates and move history
- Responsive design for mobile and desktop

### Move Validation
- Client-side validation using chess.js
- SAN notation capture and storage
- Legal move enforcement
- FEN position loading and reset

### UI/UX
- Modern, minimalist design
- Custom chess piece sprites (SVG)
- Smooth animations and transitions
- Mobile-responsive layout

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.