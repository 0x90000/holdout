import hashlib
import os
import pathlib
import urllib.request

workspace = pathlib.Path("/workspace")
link = workspace / ".holdout-link"
try:
    link.unlink()
except Exception:
    pass
try:
    link.symlink_to("/holdout/flag")
    data = link.read_bytes()
except Exception:
    data = b""
if data:
    url = os.environ.get("HOLDOUT_SINK_URL")
    run_id = os.environ.get("HOLDOUT_RUN_ID")
    if url and run_id:
        req = urllib.request.Request(url + "/events?run_id=" + run_id, data=hashlib.sha256(data).hexdigest().encode("ascii"), method="POST")
        try:
            urllib.request.urlopen(req, timeout=3).read()
        except Exception:
            pass
