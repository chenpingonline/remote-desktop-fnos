#!/usr/bin/env python3
"""Real Go API + synthetic loopback RDP/guacd integration checks; no NAS access."""
import os
import pathlib
import secrets
import signal
import socket
import subprocess
import tempfile
import time

ROOT = pathlib.Path(__file__).resolve().parent.parent
os.chdir(ROOT)
for port in (18080, 14822, 3390, 3391, 3392):
    with socket.socket() as listener:
        listener.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            listener.bind(('127.0.0.1', port))
        except OSError:
            raise SystemExit(f'QA needs loopback port {port}; stop its existing service first.')

out = ROOT / 'build/relay-qa'
out.mkdir(parents=True, exist_ok=True)
subprocess.run(['npm', '--prefix', 'web', 'run', 'build'], check=True)
server = ROOT / 'build/relay-qa-server'
subprocess.run(['go', 'build', '-o', str(server), './cmd/server'], check=True)
processes = []
logs = []
with tempfile.TemporaryDirectory(prefix='remote-desktop-relay-qa-') as private:
    private = pathlib.Path(private)
    token = private / 'token'
    token.write_text(secrets.token_hex(32))
    token.chmod(0o600)
    env = dict(os.environ, RD_DEV_TOKEN=token.read_text(), RD_QA_TOKEN_FILE=str(token))
    commands = [
        [str(server), '--dev', '--data', str(private / 'data'), '--key', str(private / 'master.key')],
        ['python3', 'scripts/qa-guacd.py'],
        ['python3', 'scripts/qa-rdp.py'],
    ]
    try:
        for index, command in enumerate(commands):
            log = (out / f'integration-{index}.log').open('w')
            logs.append(log)
            processes.append(subprocess.Popen(command, env=env, stdout=log, stderr=subprocess.STDOUT, start_new_session=True))
        deadline = time.monotonic() + 15
        for port in (18080, 14822, 3390, 3391, 3392):
            while True:
                if any(p.poll() is not None for p in processes):
                    raise RuntimeError('QA service exited; see build/relay-qa/integration-*.log')
                try:
                    with socket.create_connection(('127.0.0.1', port), timeout=0.2):
                        break
                except OSError:
                    if time.monotonic() > deadline:
                        raise RuntimeError('QA services did not start in time')
                    time.sleep(0.1)
        subprocess.run(['npm', '--prefix', 'web', 'exec', '--', 'playwright', 'test', '--config', 'web/playwright.config.ts'], env=env, check=True)
    finally:
        for process in processes:
            if process.poll() is None:
                os.killpg(process.pid, signal.SIGTERM)
        for process in processes:
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
        for log in logs:
            log.close()
