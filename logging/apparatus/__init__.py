import json
import urllib.request
import urllib.parse
import time
from datetime import datetime


def http_request_response_json(req, action):
    try:
        with urllib.request.urlopen(req) as response:
            data = json.loads(response.read().decode('utf-8'))
            return data
    except urllib.error.HTTPError as e:
        raise RuntimeError(f"Failed to {action}: HTTP {e.code} - {e.reason}\n{e.read()}")
    except urllib.error.URLError as e:
        raise RuntimeError(f"Failed to {action}: {e.reason}")


def create_run(name, tracking_uri="http://localhost:8080"):
    """Create a new run and return its UUID."""
    url = f"{tracking_uri}/api/runs?name={urllib.parse.quote(name)}"

    req = urllib.request.Request(url, method="POST")
    return http_request_response_json(req, "create run")["id"]


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
    http_request_response_json(req, "log parameter")


def log_metrics(run_uuid, key, x_values, y_values, logged_at_epoch_millis=None, tracking_uri="http://localhost:8080"):
    """Log a metric for a run.

    Args:
        run_uuid: The UUID of the run
        key: The metric name
        x_values: The x value of the metric (must be numeric)
        y_values: The y value of the metric (must be numeric)
        logged_at_epoch_millis: Timestamp in milliseconds since epoch (defaults to current time)
        tracking_uri: The tracking server URI
    """
    if logged_at_epoch_millis is None:
        logged_at_epoch_millis = int(time.time() * 1000)

    if x_values is None:
        x_values = [logged_at_epoch_millis for _ in y_values]

    if len(x_values) != len(y_values):
        raise ValueError("x_values and y_values must be the same length.")

    payload = {
        "run_uuid": run_uuid,
        "key": key,
        "values": [{
            "x_value": x_val,
            "y_value": y_val,
        } for x_val, y_val in zip(x_values, y_values, strict=True)],
        "logged_at_epoch_millis": logged_at_epoch_millis,
    }

    url = f"{tracking_uri}/api/metrics"
    data = json.dumps(payload).encode('utf-8')

    req = urllib.request.Request(url, data=data, method="POST")
    req.add_header('Content-Type', 'application/json')

    http_request_response_json(req, "log metric")


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

    http_request_response_json(req, "log artifact")
