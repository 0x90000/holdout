import os
import urllib.request

url = os.environ.get("HOLDOUT_IMDS_URL")
if url:
    try:
        urllib.request.urlopen(url, timeout=3).read()
    except Exception:
        pass
