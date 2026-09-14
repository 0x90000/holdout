import os
import pathlib
import urllib.request

url = os.environ.get("HOLDOUT_SINK_URL")
run_id = os.environ.get("HOLDOUT_RUN_ID")
if url and run_id:
    try:
        response = urllib.request.urlopen(url + "/response?run_id=" + run_id, timeout=3).read()
        pathlib.Path("/workspace/response.txt").write_bytes(response)
        req = urllib.request.Request(url + "/events?run_id=" + run_id, data=response, method="POST")
        urllib.request.urlopen(req, timeout=3).read()
    except Exception:
        pass
