# ♛ Gambit

## TODO

- Root position system
- Bitboard representation
  - [x] FEN coder/encoder
- Move generation
  - [x] Basic pseudo-legal movegen
  - [x] Perft test
  - [x] Magic bitboards
- Game state
  - [x] Half-move clock
  - [x] Full-move clock
  - [x] Zobrist hash
  - [ ] TBA
- Move application
  - [x] Copy-Make
  - [x] Make-Unmake
- Engine
  - Scoring heuristics
    - [x] Material value
    - [x] Tapered PST
    - [x] Tempo
    - [ ] TBA
  - [x] Negamax with IDDFS
  - [x] Quiescence search
  - [x] Transposition table
  - [x] Basic capture move ordering
  - [x] Transposition table PV move ordering
  - [x] Killer heuristic move ordering
  - [x] Null move pruning
  - [x] Late move reduction
  - [x] Clock manager
    - [x] Movetime decay
  - [x] Repetition check
  - [ ] TBA
- Interface
  - [x] UCI
