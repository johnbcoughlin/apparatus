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
