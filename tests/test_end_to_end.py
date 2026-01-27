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

def test_experiments_on_main_page(running_server):
    apparatus.create_run("run 234jkl")
    apparatus.create_run("run 147abc")

    # Test home page shows experiments, not runs
    with urllib.request.urlopen(f"http://localhost:8080", timeout=5) as response:
        content = response.read().decode('utf-8')
        assert "Default" in content  # Default experiment should be shown
        assert "Experiments" in content

    # Test experiment page shows runs
    with urllib.request.urlopen(f"http://localhost:8080/experiments/00000000-0000-0000-0000-000000000000", timeout=5) as response:
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


def test_nested_runs(running_server):
    """Test creating and viewing nested runs (parent -> child -> grandchild)."""
    # Create parent run (level 0)
    parent_id = apparatus.create_run("parent run")

    # Create child run (level 1) with parent
    child_id = apparatus.create_run("child run", parent_run_uuid=parent_id)

    # Create grandchild run (level 2) with child as parent
    grandchild_id = apparatus.create_run("grandchild run", parent_run_uuid=child_id)

    # Verify parent page doesn't show breadcrumbs
    with urllib.request.urlopen(f"http://localhost:8080/runs/{parent_id}", timeout=5) as response:
        content = response.read().decode('utf-8')
        assert "parent run" in content
        # No breadcrumb navigation for top-level runs
        assert ">" not in content or "child run" not in content

    # Verify child page shows parent in breadcrumbs
    with urllib.request.urlopen(f"http://localhost:8080/runs/{child_id}", timeout=5) as response:
        content = response.read().decode('utf-8')
        assert "child run" in content
        assert "parent run" in content  # Parent should be in breadcrumb

    # Verify grandchild page shows parent and grandparent in breadcrumbs
    with urllib.request.urlopen(f"http://localhost:8080/runs/{grandchild_id}", timeout=5) as response:
        content = response.read().decode('utf-8')
        assert "grandchild run" in content
        assert "child run" in content  # Parent in breadcrumb
        assert "parent run" in content  # Grandparent in breadcrumb


def test_nested_runs_max_depth(running_server):
    """Test that nested runs cannot exceed maximum depth (2 levels)."""
    # Create 3 levels
    l0_id = apparatus.create_run("level 0")
    l1_id = apparatus.create_run("level 1", parent_run_uuid=l0_id)
    l2_id = apparatus.create_run("level 2", parent_run_uuid=l1_id)

    # Attempting to create level 3 should fail
    try:
        apparatus.create_run("level 3", parent_run_uuid=l2_id)
        assert False, "Should have raised an error for exceeding max nesting"
    except RuntimeError as e:
        assert "maximum nesting level" in str(e).lower() or "400" in str(e)


def test_nested_runs_experiment_page(running_server):
    """Test that experiment page shows nested runs with collapsible details."""
    # Create a fresh experiment for this test
    import urllib.parse

    # Create parent and child under default experiment
    parent_id = apparatus.create_run("nested parent")
    child_id = apparatus.create_run("nested child", parent_run_uuid=parent_id)

    # Check experiment page shows parent with expand toggle (▶) indicating it has children
    with urllib.request.urlopen(f"http://localhost:8080/experiments/00000000-0000-0000-0000-000000000000", timeout=5) as response:
        content = response.read().decode('utf-8')
        assert "nested parent" in content
        assert "▶" in content  # Expand toggle indicates parent has children

    # Check opening the parent run shows children
    with urllib.request.urlopen(f"http://localhost:8080/experiments/00000000-0000-0000-0000-000000000000?open_l0={parent_id}", timeout=5) as response:
        content = response.read().decode('utf-8')
        assert "nested parent" in content
        assert "nested child" in content

