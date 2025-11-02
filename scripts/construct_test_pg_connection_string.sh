#!/usr/bin/env bash
set -euo pipefail

# Script to construct PostgreSQL connection string
# If APPARATUS_USE_EPHEMERAL_PG is set, uses pg_tmp for ephemeral database
# Otherwise, constructs connection string from standard PG_ environment variables

if [ -n "${APPARATUS_USE_EPHEMERAL_PG:-}" ]; then
    # Use pg_tmp for ephemeral PostgreSQL instance
    # pg_tmp outputs a connection string directly
    pg_tmp
else
    # Construct connection string from PG_ environment variables
    # Standard PostgreSQL environment variables:
    # PGHOST, PGPORT, PGDATABASE, PGUSER, PGPASSWORD

    PGHOST="${PGHOST:-localhost}"
    PGPORT="${PGPORT:-5432}"
    PGDATABASE="${PGDATABASE:-postgres}"
    PGUSER="${PGUSER:-postgres}"
    PGPASSWORD="${PGPASSWORD:-}"

    # Construct connection string
    # Format: postgresql://user:password@host:port/database
    if [ -n "$PGPASSWORD" ]; then
        echo "postgresql://${PGUSER}:${PGPASSWORD}@${PGHOST}:${PGPORT}/${PGDATABASE}?sslmode=disable"
    else
        echo "postgresql://${PGUSER}@${PGHOST}:${PGPORT}/${PGDATABASE}?sslmode=disable"
    fi
fi

