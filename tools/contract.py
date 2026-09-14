"""Run the same fail-fast, offline checks locally and in contract CI."""

import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]


def run(arguments: list[str], expected: int = 0) -> None:
    print("+ " + " ".join(arguments), flush=True)
    completed = subprocess.run(arguments, cwd=ROOT, check=False)
    if completed.returncode != expected:
        raise SystemExit(
            f"contract failed: expected exit {expected}, got {completed.returncode}"
        )


def main() -> None:
    binary = ROOT / "bin" / ("holdout-contract.exe" if os.name == "nt" else "holdout-contract")
    result = ROOT / "bin" / "contract-results.json"
    binary.parent.mkdir(exist_ok=True)
    run(["go", "test", "./..."])
    run(["go", "vet", "./..."])
    run([sys.executable, "-m", "compileall", "-q", "tasks", "tools"])
    run([sys.executable, "-m", "ruff", "check", "tasks", "tools"])
    run([sys.executable, "tools/oracle_contract.py"])
    run([sys.executable, "tools/task_docs_contract.py"])
    run(["go", "build", "-o", str(binary), "./cmd/holdout"])
    run([str(binary), "validate", "--suite", "public-v0"])
    run([str(binary), "run", "--sut", "docker-runc", "--suite", "smoke-v0",
         "--dry-run", "--out", str(result)], expected=3)
    run([str(binary), "validate-result", "--file", str(result), "--strict"])
    run([sys.executable, "tools/validate_result_schema.py", str(result)])
    print("contract checks passed")


if __name__ == "__main__":
    main()
