"""Validate an archived result against schema/result.v1.json offline."""

import json
import sys
from pathlib import Path

try:
    import jsonschema
except ImportError as exc:  # pragma: no cover - environment contract
    raise SystemExit("jsonschema is required for schema validation") from exc


def main() -> int:
    path = Path(sys.argv[1] if len(sys.argv) > 1 else "results.json")
    schema_path = Path(__file__).resolve().parents[1] / "schema" / "result.v1.json"
    with path.open(encoding="utf-8") as result_file:
        result = json.load(result_file)
    with schema_path.open(encoding="utf-8") as schema_file:
        schema = json.load(schema_file)
    jsonschema.Draft202012Validator(schema, format_checker=jsonschema.FormatChecker()).validate(result)
    print(f"schema-valid: {path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
