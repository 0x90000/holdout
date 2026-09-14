import json
import sys

ctx = json.load(sys.stdin)
events = int(ctx.get("sink_event_count", 0))
print(json.dumps({"status": "lateral" if events else "hold", "max_level": "L4" if events else "L0"}))
