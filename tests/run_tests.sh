#!/bin/bash

set -euo pipefail

uv run --with-editable "./logging" --with pytest pytest tests "$@"
