#!/usr/bin/env python3

import subprocess
import sys
import time
import urllib.request
from pathlib import Path

def wait_for_server(url, max_attempts=30, interval=0.01):
    """Wait for server to be ready by polling the health endpoint."""
    for attempt in range(max_attempts):
        try:
            urllib.request.urlopen(url, timeout=1)
            return True
        except Exception:
            time.sleep(interval)
    return False

def main():
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

        # Test home page
        with urllib.request.urlopen("http://localhost:8080", timeout=5) as response:
            content = response.read().decode('utf-8')
            assert "Welcome to Apparatus" in content

    finally:
        if server_process.poll() is None:
            server_process.terminate()
            server_process.wait()

if __name__ == "__main__":
    main()
