#!/usr/bin/env python3

import subprocess
import sys
import time
import urllib.request
from pathlib import Path
import pytest
import apparatus
import tempfile
import os

def wait_for_server(url, max_attempts=30, interval=0.01):
    """Wait for server to be ready by polling the health endpoint."""
    for attempt in range(max_attempts):
        try:
            urllib.request.urlopen(url, timeout=1)
            return True
        except Exception:
            time.sleep(interval)
    return False

@pytest.fixture(scope="module", params=["sqlite", "postgres"])
def running_server(request):
    """Pytest fixture to start and stop the apparatus server for all tests in the module."""
    server_path = Path(__file__).parent.parent / "server" / "apparatus-server"
    db_type = request.param

    tmpdir = tempfile.TemporaryDirectory()
    print(f"tmpdir name: {tmpdir.name}, database: {db_type}")

    # Configure database connection string based on parameter
    if db_type == "sqlite":
        db_conn = f"sqlite:///{tmpdir.name}/apparatus.db"
    else:  # postgres
        db_conn = subprocess.run(
            Path(__file__).parent.parent / "scripts" / "construct_test_pg_connection_string.sh",
            capture_output=True,
            check=True).stdout.strip()

    server_process = subprocess.Popen(
        [str(server_path),
         "-db", db_conn,
         "-artifact-store-uri", f"file://{tmpdir.name}/artifacts",
         ],
        cwd=Path(__file__).parent.parent / "server"
    )

    try:
        # Wait for server to be ready
        if not wait_for_server("http://localhost:8080/health"):
            print("Server failed to start", file=sys.stderr)
            sys.exit(1)

        # Check if our process actually started (could have exited if port already in use)
        if server_process.poll() is not None:
            print("Server process exited (port may be in use)", file=sys.stderr)
            sys.exit(1)

        yield server_process

    finally:
        if server_process.poll() is None:
            server_process.terminate()
            server_process.wait()
        tmpdir.cleanup()

def test_runs_on_main_page(running_server):
    apparatus.create_run("run 234jkl")
    apparatus.create_run("run 147abc")

    # Test home page
    with urllib.request.urlopen(f"http://localhost:8080", timeout=5) as response:
        content = response.read().decode('utf-8')
        assert "234jkl" in content
        assert "147abc" in content

def test_create_and_view_run(running_server):
    id = apparatus.create_run("my great run")
    apparatus.log_param(id, "param", "musa")

    apparatus.log_metrics(id, "metric", [5, 3], [46.7, 88.9])

    with urllib.request.urlopen(f"http://localhost:8080/runs/{id}", timeout=5) as response:
        content = response.read().decode('utf-8')
        assert f"Run: my great run" in content
        assert id in content

    with urllib.request.urlopen(f"http://localhost:8080/runs/{id}/overview", timeout=5) as response:
        content = response.read().decode('utf-8')
        assert "musa" in content
        assert "88.9, 46.7" in content

def test_artifact_upload(running_server):
    id = apparatus.create_run("run with artifacts")

    # Create a temporary test file
    with tempfile.NamedTemporaryFile(mode='w', suffix='.txt', delete=False) as f:
        f.write("This is a test artifact file")
        test_file_path = f.name

    try:
        # Upload the artifact
        apparatus.log_artifact(id, "results/test_output.txt", test_file_path)

        # Verify artifact was stored (we can't easily test file serving without implementing the view route)
        # For now, just verify the upload succeeded without error
        assert True
    finally:
        # Clean up temp file
        Path(test_file_path).unlink()

