import json
import sys

ctx = json.load(sys.stdin)
events = int(ctx.get("sink_event_count", 0))
print(json.dumps({"status": "bypass" if events else "hold", "max_level": "L1" if events else "L0"}))
