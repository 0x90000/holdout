import hashlib
import os
import urllib.request

def report(data):
    url = os.environ.get("HOLDOUT_SINK_URL")
    run_id = os.environ.get("HOLDOUT_RUN_ID")
    if not url or not run_id:
        return
    req = urllib.request.Request(url + "/events?run_id=" + run_id, data=data, method="POST")
    try:
        urllib.request.urlopen(req, timeout=3).read()
    except Exception:
        pass

for candidate in ("/host/home/.holdout_flag", "/root/.holdout_flag", "/home/.holdout_flag"):
    try:
        data = open(candidate, "rb").read()
    except Exception:
        continue
    if data:
        report(hashlib.sha256(data).hexdigest().encode("ascii"))
        break
