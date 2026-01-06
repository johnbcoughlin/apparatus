# Apparatus - Experiment Tracking System

## Project Overview
Alternative to MLflow for experiment tracking without AI-specific cruft.

## Architecture
- Go server backend with standard HTML rendering (multipage webapp using htmx/turbolinks)
- Python logging library for experiment tracking
- SQL datastores (SQLite initially, Postgres later)
- Frontend code lives in server/ subdirectory

## Database Conventions
- **No foreign key constraints** - Use integer references but don't enforce with FOREIGN KEY constraints
- Keep schema simple and compatible across SQLite and Postgres

## Project Structure
```
apparatus/
├── server/          # Go backend + frontend
└── logging/         # Python logging library (uv project)
```

## Development Commands

### Go Server
```bash
mise run-server
```

### Tests

#### Go tests
```
mise run go-tests
```

#### Playwright tests
```
PLAYWRIGHT_HTML_OPEN=never mise run playwright-tests
```

#### All tests
```
PLAYWRIGHT_HTML_OPEN=never mise test
```

## Jujutsu Workflow: New Change First

**Always run `jj new -m "description"` BEFORE making changes.**

Create a new change when:
- Starting any new task or modification
- Switching between different tasks
- Breaking out a small refactoring before the main work
- Any time you'd consider making a commit

Why: Squashing changes together (`jj squash`) is trivial. Splitting changes apart is harder and error-prone. Creating changes proactively is cheap and keeps work isolated.

Pattern:
```bash
jj new -m "fix: update error handling"
# now make your changes
```

Not:
```bash
# make changes first, then realize you need a new change
```

This keeps the working copy clean, makes abandoning work easy (`jj abandon`), and maintains clear change boundaries throughout development.
