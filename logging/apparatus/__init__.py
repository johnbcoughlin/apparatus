import json
import urllib.request
import urllib.parse
import time
from datetime import datetime


def create_run(name, tracking_uri="http://localhost:8080"):
    """Create a new run and return its UUID."""
    url = f"{tracking_uri}/api/runs?name={urllib.parse.quote(name)}"

    req = urllib.request.Request(url, method="POST")
    with urllib.request.urlopen(req) as response:
        data = json.loads(response.read().decode('utf-8'))
        return data["id"]


def log_param(run_uuid, key, value, tracking_uri="http://localhost:8080"):
    """Log a parameter for a run. Value can be str, bool, float, or int."""
    # Detect type
    if isinstance(value, bool):
        value_type = "bool"
        value_str = "true" if value else "false"
    elif isinstance(value, int):
        value_type = "int"
        value_str = str(value)
    elif isinstance(value, float):
        value_type = "float"
        value_str = str(value)
    elif isinstance(value, str):
        value_type = "string"
        value_str = value
    else:
        raise TypeError(f"Unsupported parameter type: {type(value)}")

    url = f"{tracking_uri}/api/params?run_uuid={urllib.parse.quote(run_uuid)}&key={urllib.parse.quote(key)}&value={urllib.parse.quote(value_str)}&type={value_type}"

    req = urllib.request.Request(url, method="POST")
    try:
        with urllib.request.urlopen(req) as response:
            if response.status != 200:
                raise RuntimeError(f"Failed to log parameter: HTTP {response.status}")
            data = json.loads(response.read().decode('utf-8'))
            return data["status"]
    except urllib.error.HTTPError as e:
        raise RuntimeError(f"Failed to log parameter: HTTP {e.code}")


def log_metric(run_uuid, key, value, logged_at=None, time_value=None, step=None, tracking_uri="http://localhost:8080"):
    """Log a metric for a run.

    Args:
        run_uuid: The UUID of the run
        key: The metric name
        value: The metric value (must be numeric)
        logged_at: Timestamp in milliseconds since epoch (defaults to current time)
        time_value: Optional time value (e.g., training time in seconds)
        step: Optional step/iteration number
        tracking_uri: The tracking server URI

    Note: If neither time_value nor step is provided, time will default to logged_at.
    """
    if logged_at is None:
        logged_at = int(time.time() * 1000)

    # If neither time nor step is supplied, default time to logged_at
    if time_value is None and step is None:
        time_value = logged_at

    payload = {
        "run_uuid": run_uuid,
        "key": key,
        "value": float(value),
        "logged_at": logged_at,
    }

    if time_value is not None:
        payload["time"] = float(time_value)
    if step is not None:
        payload["step"] = int(step)

    url = f"{tracking_uri}/api/metrics"
    data = json.dumps(payload).encode('utf-8')

    req = urllib.request.Request(url, data=data, method="POST")
    req.add_header('Content-Type', 'application/json')

    try:
        with urllib.request.urlopen(req) as response:
            if response.status != 200:
                raise RuntimeError(f"Failed to log metric: HTTP {response.status}")
            result = json.loads(response.read().decode('utf-8'))
            return result["status"]
    except urllib.error.HTTPError as e:
        error_body = e.read().decode('utf-8')
        try:
            error_data = json.loads(error_body)
            if "missing_fields" in error_data:
                raise RuntimeError(f"Failed to log metric: {error_data['error']} - {error_data['missing_fields']}")
            else:
                raise RuntimeError(f"Failed to log metric: {error_data.get('error', 'Unknown error')}")
        except json.JSONDecodeError:
            raise RuntimeError(f"Failed to log metric: HTTP {e.code}")


def log_artifact(run_uuid, path, file_path, tracking_uri="http://localhost:8080"):
    """Log an artifact (file) for a run.

    Args:
        run_uuid: The UUID of the run
        path: Logical path for the artifact (e.g., "model.pkl", "plots/accuracy.png")
        file_path: Local filesystem path to the file to upload
        tracking_uri: The tracking server URI
    """
    import os
    from urllib.request import Request, urlopen
    from urllib.error import HTTPError

    if not os.path.exists(file_path):
        raise FileNotFoundError(f"File not found: {file_path}")

    # Prepare multipart form data with a simple boundary string
    boundary = "----ApparatusBoundary7MA4YWxkTrZu0gW"

    # Read file content
    with open(file_path, "rb") as f:
        file_content = f.read()

    # Build multipart body
    body_parts = []

    # Add run_uuid field
    body_parts.append(f"--{boundary}\r\n".encode())
    body_parts.append(b'Content-Disposition: form-data; name="run_uuid"\r\n\r\n')
    body_parts.append(run_uuid.encode())

    # Add path field
    body_parts.append(f"\r\n--{boundary}\r\n".encode())
    body_parts.append(b'Content-Disposition: form-data; name="path"\r\n\r\n')
    body_parts.append(path.encode())

    # Add file field
    filename = os.path.basename(file_path)
    body_parts.append(f"\r\n--{boundary}\r\n".encode())
    body_parts.append(f'Content-Disposition: form-data; name="file"; filename="{filename}"\r\n'.encode())
    body_parts.append(b'Content-Type: application/octet-stream\r\n\r\n')
    body_parts.append(file_content)

    # Final boundary
    body_parts.append(f"\r\n--{boundary}--\r\n".encode())

    body = b"".join(body_parts)

    url = f"{tracking_uri}/api/artifacts"
    req = Request(url, data=body, method="POST")
    req.add_header("Content-Type", f"multipart/form-data; boundary={boundary}")

    try:
        with urlopen(req) as response:
            if response.status != 200:
                raise RuntimeError(f"Failed to log artifact: HTTP {response.status}")
            result = json.loads(response.read().decode('utf-8'))
            return result["status"]
    except HTTPError as e:
        error_body = e.read().decode('utf-8')
        try:
            error_data = json.loads(error_body)
            raise RuntimeError(f"Failed to log artifact: {error_data.get('error', 'Unknown error')}")
        except json.JSONDecodeError:
            raise RuntimeError(f"Failed to log artifact: HTTP {e.code}")
