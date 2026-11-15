#!/usr/bin/env python3
"""Exercise the compiled development host on owned loopback ports.

Postgres mode requires COGNARA_POSTGRES_DSN pointing to a dedicated validation
DB; the script does not erase or migrate unrelated databases.
"""
import argparse
import json
import os
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path


def free_port():
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        return sock.getsockname()[1]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--binary', required=True)
    parser.add_argument('--storage', choices=['json', 'sqlite', 'postgres'], required=True)
    args = parser.parse_args()
    if args.storage == 'postgres' and not os.environ.get('COGNARA_POSTGRES_DSN'):
        raise SystemExit('Postgres acceptance requires a dedicated COGNARA_POSTGRES_DSN')
    binary = str(Path(args.binary).resolve())
    port, mock_port = free_port(), free_port()
    while mock_port == port:
        mock_port = free_port()
    base = f'http://127.0.0.1:{port}'

    def request(path, body=None, identity='demo-user'):
        data = json.dumps(body).encode() if body is not None else None
        req = urllib.request.Request(base+path, data=data, headers={
            'Content-Type': 'application/json', 'Authorization': 'Bearer '+identity})
        with urllib.request.urlopen(req, timeout=15) as response:
            raw = response.read()
            return json.loads(raw) if 'application/json' in response.headers.get('Content-Type', '') else raw.decode()

    with tempfile.TemporaryDirectory(prefix='cognara-runtime-') as directory:
        command = [binary, '--demo', '--storage', args.storage, '--port', str(port),
                   '--mock-port', str(mock_port), '--data', str(Path(directory)/'state')]
        process = None
        log = open(Path(directory)/'process.log', 'wb')

        def stop():
            nonlocal process
            if process is None:
                return
            process.terminate()
            try:
                code = process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait()
                raise RuntimeError('development host did not shut down gracefully')
            process = None
            if code != 0:
                raise RuntimeError('development host exited unsuccessfully')

        def start():
            nonlocal process
            process = subprocess.Popen(command, stdout=log, stderr=log)
            deadline = time.monotonic()+20
            while time.monotonic() < deadline:
                if process.poll() is not None:
                    raise RuntimeError('development host exited before readiness')
                try:
                    if request('/ready')['status'] == 'UP':
                        return
                except (urllib.error.URLError, TimeoutError):
                    pass
                time.sleep(0.1)
            raise RuntimeError('development host did not become ready')

        try:
            start()
            session = request('/api/v1/cognitive/apps/channel-replenishment/sessions', {})['session_id']
            stream = request('/api/v1/agent-chat/'+session, {'query': 'SO1002需求增加到350'})
            if '276.6' not in stream or 'event: done' not in stream:
                raise RuntimeError('compiled host did not return the verified business result')
            messages = request('/api/v1/messages/'+session+'/load')['data']
            answer = messages[-1]
            before = len(request('/api/v1/cognitive/governance/usage', identity='demo-admin')['items'])
            stop()
            start()
            replay = request('/api/v1/sessions/continue-stream/'+session+'?'+urllib.parse.urlencode({'message_id': answer['id']}))
            after = len(request('/api/v1/cognitive/governance/usage', identity='demo-admin')['items'])
            if '276.6' not in replay or 'event: done' not in replay or before != after:
                raise RuntimeError('restart replay changed the result or repeated usage')
            stop()
            print(f'PASS {args.storage}: compiled startup, business call, graceful restart, authorized replay, unchanged usage')
        finally:
            if process is not None:
                process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait()
            log.close()


if __name__ == '__main__':
    main()
