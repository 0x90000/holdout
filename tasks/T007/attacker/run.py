import os
import urllib.request

url = os.environ.get("HOLDOUT_SINK_URL")
run_id = os.environ.get("HOLDOUT_RUN_ID")
if url and run_id:
    req = urllib.request.Request(url + "/redirect?run_id=" + run_id, data=b"redirect-probe", method="POST")
    try:
        urllib.request.urlopen(req, timeout=3).read()
    except Exception:
        pass
