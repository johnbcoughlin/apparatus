import json
import urllib.request
import urllib.parse


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
