# Create Run Workflow Spec

## SQLite Schema
```sql
CREATE TABLE runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uuid TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

- Auto-incrementing ID for internal use
- UUID for external references
- Run name (required)
- Timestamp for tracking when created

## Go Server Implementation

### Database initialization (`server/db.go`)
- Open SQLite connection to `apparatus.db`
- Create runs table if not exists
- Store global db connection

### POST /runs endpoint
- Accept query string: `POST /runs?name=experiment-1`
- Generate UUID
- Insert into runs table
- Return JSON: `{"id": "<uuid>", "name": "experiment-1"}`

### GET /runs/{uuid} endpoint
- Parse UUID from URL path
- Query runs table by UUID
- Return HTML page showing run name

## Python Logging Library

Create `logging/apparatus/__init__.py`:
- `create_run(name, tracking_uri="http://localhost:8080")` function
- Make POST to `{tracking_uri}/runs?name={name}`
- Return run UUID
- **No external dependencies** - stdlib only

## End-to-End Test
Update `tests/end_to_end.py`:
- Start server
- Call `apparatus.create_run("test-run")`
- Verify we get back a UUID
- GET `/runs/{uuid}` and verify page contains "test-run"

## Notes
- No error handling - happy path only
- No abstractions or alternatives
- Simple and direct implementation
