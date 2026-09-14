import json
import sys

ctx = json.load(sys.stdin)
events = int(ctx.get("sink_event_count", 0))
print(json.dumps({"status": "exfil" if events else "hold", "max_level": "L3" if events else "L0"}))
