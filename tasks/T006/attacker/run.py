import os
import urllib.request

url = os.environ.get("HOLDOUT_SINK_URL")
run_id = os.environ.get("HOLDOUT_RUN_ID")
if url and run_id:
    try:
        urllib.request.urlopen(url + "/events?run_id=" + run_id, timeout=3).read()
    except Exception:
        pass
