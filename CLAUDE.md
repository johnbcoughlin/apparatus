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
