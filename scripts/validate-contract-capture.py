#!/usr/bin/env python3
"""Validate real integration-test HTTP traffic against the pinned v0.3 contract."""
import json, re, sys
from pathlib import Path
from jsonschema import Draft202012Validator, RefResolver, FormatChecker
root=Path(__file__).resolve().parents[1]
contract=json.loads((root/'contracts/api/fg-ae-v0.3.json').read_text())
resolver=RefResolver.from_schema(contract)
checker=FormatChecker()
if "date-time" not in checker.checkers: raise SystemExit("Missing date-time validator: install jsonschema[format-nongpl]==4.26.0 before validating contracts")
count=0
rejected_invalid=0
for line in Path(sys.argv[1]).read_text().splitlines():
    record=json.loads(line)
    for path,methods in contract['paths'].items():
        if re.fullmatch(re.sub(r'\{[^}]+\}',r'[^/]+',path),record['path']):
            op=methods[record['method'].lower()]
            if record['request'] is not None and 'requestBody' in op:
                schema=op['requestBody']['content']['application/json']['schema']
                errors=list(Draft202012Validator(schema,resolver=resolver,format_checker=checker).iter_errors(record['request']))
                if errors:
                    if record['status']!=400: raise RuntimeError(f"Invalid request was not rejected with 400: {record['path']}: {errors[0]}")
                    rejected_invalid+=1
            response=op['responses'].get(str(record['status']),op['responses'].get('default'))
            if response and 'content' in response:
                schema=response['content']['application/json']['schema']
                try: Draft202012Validator(schema,resolver=resolver,format_checker=checker).validate(record['response'])
                except Exception as e: raise RuntimeError(f"{record['method']} {path}: {e}") from e
            count+=1
            break
if not count: raise SystemExit('No contract calls captured')
print(f'Validated {count} real internal HTTP request/response pairs against v0.3; {rejected_invalid} invalid requests correctly rejected with 400')
