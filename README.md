# ♛ Gambit

## TODO

- Root position system
- Bitboard representation
  - [x] FEN coder/encoder
- Move generation
  - [x] Basic pseudo-legal movegen
  - [x] Perft test
  - [ ] Magic bitboards
- Game state
  - [ ] Repetition check
  - [x] Half-move clock
  - [x] Full-move clock
  - [x] Zobrist hash
  - [ ] TBA
- Move application
  - [x] Copy-Make
  - [x] Make-Unmake
- Engine
  - Scoring heuristics
    - [x] Piece value
    - [x] PST
    - [ ] Tapered PST
  - [x] Search runner
  - [x] Negamax with IDDFS
  - [x] Transposition table
  - [x] Basic capture move ordering
  - [x] Transposition table PV move ordering
  - [x] Killer heuristic move ordering
  - [x] Null move pruning
  - [x] Clock manager
  - [ ] TBA
- Interface
  - [x] UCI
