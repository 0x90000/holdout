"""Check the safety and reproducibility sections required in every task doc."""

from pathlib import Path
import re
import sys


ROOT = Path(__file__).resolve().parents[1]
CATALOG = ROOT / "tasks" / "catalog.yaml"
REQUIRED_SECTIONS = (
    "## 前置条件",
    "## 预期失败模式",
    "## 授权范围与证据",
    "## 复现命令",
)


def tasks() -> list[tuple[str, bool]]:
    result: list[tuple[str, bool]] = []
    current_id = ""
    current_dangerous = False
    for line in CATALOG.read_text(encoding="utf-8").splitlines():
        match = re.match(r"\s*- id:\s*([A-Za-z0-9][A-Za-z0-9._-]*)\s*$", line)
        if match:
            if current_id:
                result.append((current_id, current_dangerous))
            current_id = match.group(1)
            current_dangerous = False
            continue
        if current_id and re.match(r"\s+dangerous:\s*true\s*$", line):
            current_dangerous = True
    if current_id:
        result.append((current_id, current_dangerous))
    return result


def main() -> int:
    failures: list[str] = []
    entries = tasks()
    for task_id, dangerous in entries:
        task_root = ROOT / "tasks" / task_id
        doc = task_root / "TASK.md"
        if not doc.is_file():
            failures.append(f"{task_id}: missing TASK.md")
            continue
        text = doc.read_text(encoding="utf-8")
        for section in REQUIRED_SECTIONS:
            if section not in text:
                failures.append(f"{task_id}: missing {section}")
        if "HOLDOUT_RUN_ID" not in text or "sink" not in text.lower():
            failures.append(f"{task_id}: must document run ID and sink evidence")
        if "go run ./cmd/holdout" not in text:
            failures.append(f"{task_id}: missing holdout reproduction command")
        if dangerous:
            for marker in ("Firecracker", "HOLDOUT_I_UNDERSTAND", "--allow-dangerous"):
                if marker not in text:
                    failures.append(f"{task_id}: dangerous task must document {marker}")
    if failures:
        print("task docs contract failed:", file=sys.stderr)
        print("\n".join(failures), file=sys.stderr)
        return 1
    print(f"task docs contract passed: {len(entries)} tasks")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
