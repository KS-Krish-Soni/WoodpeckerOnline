// Woodpecker Chess Puzzles API
window.app = {
    // Current puzzle state
    currentFEN: '',
    currentTurn: 'white',
    playedMoves: [],
    playedSANs: [],
    chess: null,
    originalFEN: '',
    calcSAN: [], // Array of user-entered SAN moves for calculation pad
    
    /**
     * Load a new puzzle into the board
     * @param {string} fen - FEN string representing the puzzle position
     * @param {string} turn - 'white' or 'black' indicating whose turn it is
     */
    setFEN(fen, turn) {
        // Check if Chess constructor is available
        if (typeof Chess === 'undefined') {
            console.error('Chess.js library not loaded!');
            document.getElementById('status').textContent = 'Error: Chess.js not loaded';
            return;
        }
        
        this.currentFEN = fen;
        this.originalFEN = fen;
        this.currentTurn = turn;
        this.playedMoves = [];
        this.playedSANs = [];
        
        // Initialize chess.js with the FEN
        this.chess = new Chess(fen);
        
                    // Update UI
        document.getElementById('status').textContent = 'Puzzle loaded';
        document.getElementById('btn-show-solution').disabled = false;
        
        // Update the board display - wait for chessPuzzleInstance to be available
        const updateBoard = () => {
            if (window.chessPuzzleInstance) {
                window.chessPuzzleInstance.loadGameState();
            } else {
                setTimeout(updateBoard, 50);
            }
        };
        updateBoard();
    },
    
    /**
     * Get the SAN sequence the user just played
     * @returns {string} SAN move sequence
     */
    getPlayedSAN() {
        return [...this.playedSANs]; // Return a copy
    },
    
    /**
     * Reset the current puzzle attempt
     */
    resetAttempt() {
        this.playedMoves = [];
        this.playedSANs = [];
        
        // Reset chess.js to original FEN
        if (this.originalFEN) {
            this.chess = new Chess(this.originalFEN);
        }
        
                    // Reset UI
        document.getElementById('status').textContent = 'Ready';
        document.getElementById('btn-show-solution').disabled = false;
        
        // Reset board to original puzzle position
        if (window.chessPuzzleInstance) {
            window.chessPuzzleInstance.loadGameState();
        }
    },
    
    /**
     * Load the next puzzle
     */
    loadNextPuzzle() {
        // TODO: Implement API call to get next puzzle
        // For now, just log the action
        document.getElementById('status').textContent = 'Loading next puzzle...';
    },
    
    /**
     * Submit the current move sequence
     */
    submitMove() {
        const sanMoves = this.getPlayedSAN();
        
        // TODO: Implement API call to submit moves
        document.getElementById('status').textContent = 'Move submitted';
        document.getElementById('btn-submit').disabled = true;
    },
    
    /**
     * Show the solution for the current puzzle
     */
    async showSolution() {
        if (!currentPuzzle?.id) {
            document.getElementById('status').textContent = 'No puzzle loaded';
            return;
        }
        
        try {
            const res = await fetch(`/api/puzzles/solution-text/${currentPuzzle.id}`);
            if (!res.ok) {
                throw new Error(`HTTP ${res.status}: ${res.statusText}`);
            }
            
            const j = await res.json();
            
            // Display the solution text in the dedicated solution text span
            const solutionTextEl = document.getElementById('solution-text');
            if (solutionTextEl) {
                solutionTextEl.textContent = j.solutionText;
            }
            
            // Update status to show success
            const statusEl = document.getElementById('status');
            if (statusEl) {
                statusEl.textContent = 'Solution loaded';
            }
            
        } catch (error) {
            console.error('Error loading solution text:', error);
            const solutionTextEl = document.getElementById('solution-text');
            if (solutionTextEl) {
                solutionTextEl.textContent = `Error loading solution: ${error.message}`;
            }
            
            const statusEl = document.getElementById('status');
            if (statusEl) {
                statusEl.textContent = 'Error loading solution';
            }
        }
    },
    
    /**
     * Update the today counter
     * @param {number} solved - Number of puzzles solved today
     * @param {number} total - Total number of puzzles available today
     */
    updateTodayCounter(solved, total) {
        document.getElementById('today-counter').textContent = `${solved}/${total}`;
    },
    
    /**
     * Add a move to the played moves array
     * @param {string} move - Move in algebraic notation
     */
    addMove(move) {
        this.playedMoves.push(move);
        document.getElementById('status').textContent = 'Move made';
    },
    
    /**
     * Handle piece drop/move on the board
     * @param {string} from - Source square (e.g., 'e2')
     * @param {string} to - Target square (e.g., 'e4')
     * @param {string} promotion - Promotion piece (default 'q')
     * @returns {boolean} - Whether the move was legal and applied
     */
    handleMove(from, to, promotion = 'q') {
        if (!this.chess) {
            console.error('Chess instance not initialized');
            return false;
        }
        
        try {
            // Attempt to make the move
            const move = this.chess.move({
                from: from,
                to: to,
                promotion: promotion
            });
            
            if (move) {
                // Move was legal, add to played moves
                this.playedSANs.push(move.san);
                this.playedMoves.push(`${from}-${to}`);
                
                // Update UI
                document.getElementById('status').textContent = 'Move made';
                
                // Update the board display
                if (window.chessPuzzleInstance) {
                    window.chessPuzzleInstance.loadGameState();
                }
                
                return true;
            } else {
                return false;
            }
        } catch (error) {
            console.error('Error making move:', error);
            return false;
        }
    },
    
    /**
     * Get current FEN from chess.js
     * @returns {string} Current FEN string
     */
    getCurrentFEN() {
        return this.chess ? this.chess.fen() : '';
    },
    
    /**
     * Get current turn from chess.js
     * @returns {string} 'w' for white, 'b' for black
     */
    getCurrentTurn() {
        return this.chess ? this.chess.turn() : 'w';
    },
    
    /**
     * Check if the game is over
     * @returns {boolean} Whether the game is over
     */
    isGameOver() {
        return this.chess ? this.chess.game_over() : false;
    },
    
    /**
     * Get the board state as a 2D array
     * @returns {Array} 2D array representing the board
     */
    getBoard() {
        if (!this.chess) {
            return [];
        }
        
        // Create a board array manually by iterating through squares
        const board = [];
        const squares = ['a8', 'b8', 'c8', 'd8', 'e8', 'f8', 'g8', 'h8',
                        'a7', 'b7', 'c7', 'd7', 'e7', 'f7', 'g7', 'h7',
                        'a6', 'b6', 'c6', 'd6', 'e6', 'f6', 'g6', 'h6',
                        'a5', 'b5', 'c5', 'd5', 'e5', 'f5', 'g5', 'h5',
                        'a4', 'b4', 'c4', 'd4', 'e4', 'f4', 'g4', 'h4',
                        'a3', 'b3', 'c3', 'd3', 'e3', 'f3', 'g3', 'h3',
                        'a2', 'b2', 'c2', 'd2', 'e2', 'f2', 'g2', 'h2',
                        'a1', 'b1', 'c1', 'd1', 'e1', 'f1', 'g1', 'h1'];
        
        for (let row = 0; row < 8; row++) {
            board[row] = [];
            for (let col = 0; col < 8; col++) {
                const square = squares[row * 8 + col];
                const piece = this.chess.get(square);
                
                if (piece && piece.type && piece.color) {
                    board[row][col] = {
                        type: piece.type,
                        color: piece.color === 'w' ? 'white' : 'black'
                    };
                } else {
                    board[row][col] = null;
                }
            }
        }
        return board;
    }
};

// Puzzle functionality
let currentPuzzle = null;
let matchedLine = []; // Store the matched solution line for animation
let currentPuzzleIndex = 0; // Track current puzzle index (0-4 for puzzles 1-5)

// Global variables for calculation pad
let calcSAN = []; // Array of user-entered SAN moves for calculation pad

async function loadNextPuzzle() {
    try {
        console.log('=== LOAD NEXT PUZZLE DEBUG ===');
        console.log('Current puzzle index:', currentPuzzleIndex);
        
        // Calculate the next puzzle number (1-5)
        const nextPuzzleNumber = currentPuzzleIndex + 1;
        const puzzleId = `wpm_easy_${String(nextPuzzleNumber).padStart(3, '0')}`;
        
        console.log('Loading puzzle ID:', puzzleId);
        
        // Get the puzzle data from the API
        const res = await fetch(`/api/puzzles/next?difficulty=easy&puzzleId=${puzzleId}`);
        if (!res.ok) {
            throw new Error(`HTTP ${res.status}: ${res.statusText}`);
        }
        
        const j = await res.json();
        console.log('API response received:', j);
        
        currentPuzzle = j;
        console.log('Set currentPuzzle to:', currentPuzzle);
        console.log('currentPuzzle.id:', currentPuzzle.id);
        
        window.app.setFEN(j.fen, j.sideToMove);
        
        const el = document.getElementById('status');
        if (el) el.textContent = `Loaded puzzle ${j.id} • side to move: ${j.sideToMove}`;
        
        // Update puzzle number display
        updatePuzzleNumber(j.id);
        
        // Update difficulty display
        updateDifficultyDisplay(j.difficulty);
        
        // Clear previous matched line
        matchedLine = [];
        
        // Reset calculation pad
        resetLine();
        
        // Clear previous solution text
        const solutionTextEl = document.getElementById('solution-text');
        if (solutionTextEl) {
            solutionTextEl.textContent = 'Click "Show solution" to display puzzle solution';
        }
        
        // Increment the puzzle index for next time (cycle back to 0 after 5)
        currentPuzzleIndex = (currentPuzzleIndex + 1) % 5;
        
    } catch (error) {
        console.error('Error loading puzzle:', error);
        const el = document.getElementById('status');
        if (el) el.textContent = `Error loading puzzle: ${error.message}`;
    }
}



async function showSolution() {
    console.log('=== SHOW SOLUTION DEBUG ===');
    console.log('currentPuzzle:', currentPuzzle);
    console.log('currentPuzzle?.id:', currentPuzzle?.id);
    
    if (!currentPuzzle?.id) {
        console.log('No puzzle loaded, showing error message');
        document.getElementById('status').textContent = 'No puzzle loaded';
        return;
    }
    
    console.log('Making API call to:', `/api/puzzles/solution-text/${currentPuzzle.id}`);
    
    try {
        const res = await fetch(`/api/puzzles/solution-text/${currentPuzzle.id}`);
        console.log('API response status:', res.status);
        
        if (!res.ok) {
            throw new Error(`HTTP ${res.status}: ${res.statusText}`);
        }
        
        const j = await res.json();
        console.log('API response data:', j);
        
        // Display the solution text in the dedicated solution text span
        const solutionTextEl = document.getElementById('solution-text');
        if (solutionTextEl) {
            solutionTextEl.textContent = j.solutionText;
            console.log('Solution text displayed successfully in dedicated span');
        } else {
            console.error('solution-text element not found!');
        }
        
        // Update status to show success
        const statusEl = document.getElementById('status');
        if (statusEl) {
            statusEl.textContent = 'Solution loaded';
        }
        
    } catch (error) {
        console.error('Error loading solution text:', error);
        const solutionTextEl = document.getElementById('solution-text');
        if (solutionTextEl) {
            solutionTextEl.textContent = `Error loading solution: ${error.message}`;
        }
        
        const statusEl = document.getElementById('status');
        if (statusEl) {
            statusEl.textContent = 'Error loading solution';
        }
    }
}



// Dev harness functionality (keeping for backward compatibility)
async function devHealth() {
    const res = await fetch('/api/dev/health');
    const j = await res.json();
    const el = document.getElementById('status');
    if (el) el.textContent = `DB OK • ${j.puzzles} puzzles seeded`;
}

// Update puzzle number display
function updatePuzzleNumber(puzzleId) {
    const puzzleNumberEl = document.getElementById('puzzleNumber');
    if (!puzzleNumberEl) return;
    
    // Extract number from puzzle ID (e.g., "wpm_easy_005" -> "5")
    const match = puzzleId.match(/_(\d+)$/);
    if (match) {
        const number = parseInt(match[1], 10);
        puzzleNumberEl.textContent = `${number}.`;
    } else {
        puzzleNumberEl.textContent = 'No puzzle loaded';
    }
}

// Update difficulty display
function updateDifficultyDisplay(difficulty) {
    console.log('=== DIFFICULTY DEBUG: updateDifficultyDisplay called ===');
    console.log('Parameter received:', difficulty);
    console.log('Type of parameter:', typeof difficulty);
    
    const difficultyEl = document.getElementById('difficulty-display');
    console.log('Element found:', difficultyEl);
    console.log('Element HTML:', difficultyEl ? difficultyEl.outerHTML : 'null');
    
    if (!difficultyEl) {
        console.error('difficulty-display element not found!');
        return;
    }
    
    const newText = difficulty || '-';
    console.log('Setting text to:', newText);
    
    difficultyEl.textContent = newText;
    
    console.log('After setting text, element textContent:', difficultyEl.textContent);
    console.log('After setting text, element innerHTML:', difficultyEl.innerHTML);
    console.log('=== DIFFICULTY DEBUG: updateDifficultyDisplay completed ===');
}

// Dev harness functions for backward compatibility
async function devLoadFirst() {
    const res = await fetch('/api/dev/first-puzzle');
    const j = await res.json();
    currentPuzzle = j;
    window.app.setFEN(j.fen, j.sideToMove);
    resetLine();
    const el = document.getElementById('status');
    if (el) el.textContent = `Loaded puzzle ${j.id} • side to move: ${j.sideToMove}`;
    
    // Update puzzle number display
    updatePuzzleNumber(j.id);
    
    // Update difficulty display
    updateDifficultyDisplay(j.difficulty);
}

async function devSubmitFirstMove() {
    const moves = window.app.getPlayedSAN();
    const body = JSON.stringify({ puzzleId: currentPuzzle?.id, playedSans: moves });
    const res = await fetch('/api/dev/grade-first-move', { method:'POST', headers:{'Content-Type':'application/json'}, body });
    const j = await res.json();
    const el = document.getElementById('status');
    if (j.correct) el.textContent = `✔ Correct first move (${moves[0]})`;
    else el.textContent = `✖ Wrong first move (${moves[0] || 'none'}) • expected one of: ${Object.keys(j.expectedFirstMoves).join(', ')}`;
}

// Load chess.js script
function loadChessJS() {
    if (typeof Chess !== 'undefined') {
        return;
    }
    
    const script = document.createElement('script');
    script.src = '/static/chess.js';
    script.onload = () => {
        // Chess.js loaded successfully
    };
    script.onerror = () => {
        console.error('Failed to load Chess.js');
    };
    document.head.appendChild(script);
}



// Calculation Pad Functions
function normalizeSAN(s) {
    if (!s || typeof s !== 'string') {
        return '';
    }
    
    // Convert to lowercase and trim whitespace
    s = s.toLowerCase().trim();
    
    // Accept 0-0 = O-O, 0-0-0 = O-O-O
    s = s.replace(/0-0-0/g, 'o-o-o');
    s = s.replace(/0-0/g, 'o-o');
    
    // Strip +, #, !?, move numbers, and ellipses
    s = s.replace(/[+#!?]/g, '');
    s = s.replace(/[!?]+/g, '');
    
    // Remove move numbers and ellipses (e.g., "1.", "1...", "12.")
    s = s.replace(/^\d+\.+\s*/, ''); // remove '12.' or '12...' at start
    s = s.replace(/\.\.\./g, '');
    s = s.replace(/\.\./g, '');
    s = s.replace(/\./g, '');
    
    // Remove any remaining digits at the start
    s = s.replace(/^\d+/, '');
    
    // Handle promotions: accept e8=Q and e8Q
    if (s.includes('=')) {
        const parts = s.split('=');
        if (parts.length === 2) {
            s = parts[0] + parts[1];
        }
    }
    
    // Remove any remaining whitespace
    s = s.replace(/\s+/g, '');
    
    return s;
}

async function addMoveFromInput() {
    const raw = document.getElementById('calc-san').value;
    if (!raw) return;
    
    // Normalize the input
    const san = normalizeSAN(raw);
    
    if (!san) {
        return uiError('Invalid move format');
    }
    
    // For the calculation pad, we'll be more lenient and just add the move
    // The server-side grading will handle the actual validation
    calcSAN.push(raw.trim()); // Keep the original input for display
    
    renderCalcChips();
    document.getElementById('calc-san').value = '';
    uiOK('Move added');
}

function undoMove(){ 
    if (calcSAN.length){
        calcSAN.pop(); 
        renderCalcChips(); 
        uiOK(''); 
    } 
}

function resetLine(){
    calcSAN = []; 
    renderCalcChips(); 
    uiOK(''); 
}

function renderCalcChips(){
    const ol = document.getElementById('calc-line'); 
    if (!ol) {
        console.error('calc-line element not found!');
        return;
    }
    
    ol.innerHTML = '';
    
    calcSAN.forEach((m,i)=>{
        const li = document.createElement('li');
        li.textContent = `${Math.floor(i/2)+1}${i%2? '...':' .' } ${m}`;
        ol.appendChild(li);
    });
}

function uiOK(msg){ 
    const feedback = document.getElementById('calc-feedback');
    feedback.textContent = msg; 
    feedback.className = 'calc-feedback success'; 
}

function uiError(msg){ 
    const feedback = document.getElementById('calc-feedback');
    feedback.textContent = msg; 
    feedback.className = 'calc-feedback error'; 
}

async function submitLine() {
    if (!currentPuzzle?.id) return uiError('No puzzle loaded');
    const res = await fetch('/api/puzzles/grade-line', {
        method:'POST',
        headers:{'Content-Type':'application/json'},
        body: JSON.stringify({ puzzleId: currentPuzzle.id, typedSans: calcSAN })
    });
    if (!res.ok) return uiError('Server error');
    const j = await res.json();
    // Feedback
    const ticks = j.requiredTicks || [];
    const matched = (j.ticksMatched||[]).length;
    const all = ticks.length;
    const msg = j.correct
        ? `✔ First move correct. Ticks: ${matched}/${all}. Depth matched: ${j.depthMatched}.`
        : `✖ First move wrong. Expected one of: ${(j.bestLine && j.bestLine[0]) ? j.bestLine[0] : '—'}`;
    uiOK(msg);

    // Optional: color the chips up to earliest mistake; highlight tick chips
    highlightChips(j);
    
    // Refresh today's progress counter
    loadTodayProgress();
}

function highlightChips(j) {
    const chips = document.querySelectorAll('#calc-line li');
    chips.forEach((chip, index) => {
        chip.classList.remove('tick-ok', 'tick-miss', 'mistake');
        
        if (j.earliestMistake !== null && index >= j.earliestMistake) {
            chip.classList.add('mistake');
        } else if (j.ticksMatched && j.ticksMatched.includes(index)) {
            chip.classList.add('tick-ok');
        } else if (j.requiredTicks && j.requiredTicks.includes(calcSAN[index])) {
            chip.classList.add('tick-miss');
        }
    });
}

function setupCalculationPad() {
    document.getElementById('btn-add').onclick = addMoveFromInput;
    document.getElementById('calc-san').addEventListener('keydown', e=>{
        if (e.key === 'Enter') { addMoveFromInput(); }
        if (e.key === 'Backspace' && !e.target.value) { undoMove(); }
    });
    document.getElementById('btn-undo').onclick = undoMove;
    document.getElementById('btn-reset-line').onclick = resetLine;
    document.getElementById('btn-submit-line').onclick = submitLine;
}

// Load today's progress and update the counter
async function loadTodayProgress() {
    try {
        const response = await fetch('/api/daily');
        const data = await response.json();
        
        const todayCounter = document.getElementById('today-counter');
        if (todayCounter) {
            todayCounter.textContent = `${data.doneToday}/${data.perDay}`;
        }
    } catch (error) {
        console.error('Error loading today\'s progress:', error);
    }
}

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    console.log('=== DIFFICULTY DEBUG: DOM loaded, checking elements ===');
    
    // Check if difficulty display element exists
    const difficultyEl = document.getElementById('difficulty-display');
    console.log('Difficulty element on page load:', difficultyEl);
    if (difficultyEl) {
        console.log('Difficulty element HTML:', difficultyEl.outerHTML);
        console.log('Current text content:', difficultyEl.textContent);
        
        // Test setting difficulty directly
        console.log('=== DIFFICULTY DEBUG: Testing direct difficulty setting ===');
        difficultyEl.textContent = 'TEST-EASY';
        console.log('After setting test text, element textContent:', difficultyEl.textContent);
        
        // Reset to original
        difficultyEl.textContent = '-';
        console.log('After resetting to "-", element textContent:', difficultyEl.textContent);
    } else {
        console.error('DIFFICULTY DEBUG: difficulty-display element NOT FOUND on page load!');
    }
    
    // Load chess.js script
    loadChessJS();
    
    // Setup calculation pad
    setupCalculationPad();
    
    // Setup puzzle functionality
    devHealth(); // Show database status
    document.getElementById('btn-next').onclick = loadNextPuzzle;
    document.getElementById('btn-show-solution').onclick = showSolution;
    
    console.log('=== BUTTON SETUP DEBUG ===');
    console.log('btn-next element:', document.getElementById('btn-next'));
    console.log('btn-show-solution element:', document.getElementById('btn-show-solution'));
    
    
    
    // Set initial button states
    document.getElementById('btn-show-solution').disabled = false;
    
    // Load today's progress
    loadTodayProgress();
    
    // Initialize puzzle number display
    const puzzleNumberEl = document.getElementById('puzzleNumber');
    if (puzzleNumberEl) {
        puzzleNumberEl.textContent = 'No puzzle loaded';
    }
});
