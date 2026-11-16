#!/usr/bin/env python3
"""Verify the compiled exporter against the CI-only MySQL database after runtime acceptance."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import tempfile

parser = argparse.ArgumentParser()
parser.add_argument('--binary', required=True)
args = parser.parse_args()
if not os.environ.get('COGNARA_MYSQL_DSN'):
    raise SystemExit('Dedicated MySQL test DSN required')
with tempfile.TemporaryDirectory(prefix='cognara-export-check-') as temp:
    output = Path(temp) / 'snapshot.json'
    command = [args.binary, '--output', str(output)]
    first = subprocess.run(command, capture_output=True, text=True, timeout=20)
    if first.returncode:
        raise SystemExit('Compiled MySQL snapshot export failed')
    report = json.loads(first.stdout)
    data = output.read_bytes()
    snapshot = json.loads(data)
    if report['sha256'] != hashlib.sha256(data).hexdigest() or report['bytes'] != len(data):
        raise SystemExit('Snapshot integrity summary mismatch')
    if not snapshot['sessions'] or output.stat().st_mode & 0o777 != 0o600:
        raise SystemExit('Snapshot contents or permissions invalid')
    second = subprocess.run(command, capture_output=True, text=True, timeout=20)
    if second.returncode == 0 or output.read_bytes() != data:
        raise SystemExit('Existing snapshot overwritten')
    print('PASS compiled MySQL snapshot: checksum, records, permissions, overwrite refusal')
