#!/usr/bin/env python3

import subprocess
import sys
import time
import urllib.request
from pathlib import Path
import pytest
import apparatus

def wait_for_server(url, max_attempts=30, interval=0.01):
    """Wait for server to be ready by polling the health endpoint."""
    for attempt in range(max_attempts):
        try:
            urllib.request.urlopen(url, timeout=1)
            return True
        except Exception:
            time.sleep(interval)
    return False

@pytest.fixture(scope="module")
def running_server():
    """Pytest fixture to start and stop the apparatus server for all tests in the module."""
    server_path = Path(__file__).parent.parent / "server" / "apparatus-server"

    server_process = subprocess.Popen(
        [str(server_path)],
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
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

    # Test home page
    with urllib.request.urlopen(f"http://localhost:8080/runs/{id}", timeout=5) as response:
        content = response.read().decode('utf-8')
        assert f"Run: my great run" in content
        assert id in content
        assert "musa" in content

