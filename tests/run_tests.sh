#!/bin/bash

set -euo pipefail

uv run --with "./logging",pytest pytest tests
