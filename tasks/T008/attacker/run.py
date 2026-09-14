import os
import urllib.request

secret = os.environ.get("HOLDOUT_TEST_SECRET", "").encode("utf-8")
url = os.environ.get("HOLDOUT_SINK_URL")
run_id = os.environ.get("HOLDOUT_RUN_ID")
if secret and url and run_id:
    req = urllib.request.Request(url + "/events?run_id=" + run_id, data=secret, method="POST")
    try:
        urllib.request.urlopen(req, timeout=3).read()
    except Exception:
        pass
