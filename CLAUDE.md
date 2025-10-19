# Apparatus - Experiment Tracking System

## Project Overview
Alternative to MLflow for experiment tracking without AI-specific cruft.

## Architecture
- Go server backend with standard HTML rendering (multipage webapp using htmx/turbolinks)
- Python logging library for experiment tracking
- SQL datastores (SQLite initially, Postgres later)
- Frontend code lives in server/ subdirectory

## Project Structure
```
apparatus/
├── server/          # Go backend + frontend
└── logging/         # Python logging library (uv project)
```

## Development Commands

### Go Server
```bash
cd server
# TODO: Add go commands once Go is available in PATH
```

### Python Logging Library
```bash
cd logging
uv run python -m logging
```

## Next Steps
- [ ] Initialize Go module once Go is available
- [ ] Set up basic server structure
- [ ] Implement experiment tracking models
- [ ] Create frontend templates