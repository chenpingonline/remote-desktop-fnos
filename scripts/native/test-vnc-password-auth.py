#!/usr/bin/env python3
"""Real packaged guacd/LibVNCClient against synthetic RFB servers; no NAS/passwords.
Run inside remote-desktop-native:1.6.0, mounting this script read-only.
"""
import os
import socket
import struct
import subprocess
import threading
import tempfile
import sys


def exact(conn, length):
    data = b''
    while len(data) < length:
        chunk = conn.recv(length - len(data))
        if not chunk:
            raise EOFError()
        data += chunk
    return data


def encode(*parts):
    return (','.join(f'{len(p)}.{p}' for p in parts) + ';').encode()


def read_instruction(conn):
    result = []
    while True:
        length = b''
        while True:
            char = exact(conn, 1)
            if char == b'.':
                break
            length += char
        result.append(exact(conn, int(length)).decode())
        if exact(conn, 1) == b';':
            return result


def connect_engine(timeout):
    if engine_socket is None:
        return socket.create_connection(('127.0.0.1', 14829), timeout)
    conn = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
    conn.settimeout(timeout)
    try:
        conn.connect(engine_socket)
        return conn
    except OSError:
        conn.close()
        raise


def check(schemes, should_connect):
    listener = socket.socket()
    listener.bind(('127.0.0.1', 0))
    listener.listen()
    selected = []
    failures = []
    finished = threading.Event()

    def rfb_server():
        try:
            with listener.accept()[0] as conn:
                conn.settimeout(10)
                conn.sendall(b'RFB 003.889\n')
                assert exact(conn, 12).startswith(b'RFB 003.')
                conn.sendall(bytes([len(schemes), *schemes]))
                try:
                    selected.append(exact(conn, 1)[0])
                except EOFError:
                    assert not should_connect
                    return
                assert should_connect and selected == [2], selected
                conn.sendall(bytes(range(16)))  # synthetic VNC password challenge
                exact(conn, 16)  # synthetic server accepts the response, no real secret
                conn.sendall(struct.pack('!I', 0))
                exact(conn, 1)  # ClientInit shared flag
                pixel_format = struct.pack('!BBBBHHHBBBxxx', 32, 24, 0, 1, 255, 255, 255, 16, 8, 0)
                name = b'Synthetic VNC regression'
                conn.sendall(struct.pack('!HH', 64, 48) + pixel_format + struct.pack('!I', len(name)) + name)
                while True:
                    opcode = exact(conn, 1)[0]
                    if opcode == 0:
                        exact(conn, 19)
                    elif opcode == 2:
                        count = struct.unpack('!xH', exact(conn, 3))[0]
                        exact(conn, count * 4)
                    elif opcode == 3:
                        exact(conn, 9)
                        conn.sendall(struct.pack('!BBHHHHHi', 0, 0, 1, 0, 0, 64, 48, 0) + b'\x30\x60\x90\x00' * (64 * 48))
                    elif opcode == 4:
                        exact(conn, 7)
                    elif opcode == 5:
                        exact(conn, 5)
                    else:
                        raise AssertionError(f'unexpected RFB opcode {opcode}')
        except (EOFError, ConnectionResetError, BrokenPipeError):
            pass
        except Exception as error:
            failures.append(repr(error))
        finally:
            finished.set()
            listener.close()

    threading.Thread(target=rfb_server, daemon=True).start()
    with connect_engine(10) as conn:
        conn.settimeout(10)
        conn.sendall(encode('select', 'vnc'))
        args = read_instruction(conn)
        assert args[0] == 'args', args
        for parts in [('size', '800', '600', '96'), ('audio',), ('video',), ('image', 'image/png', 'image/jpeg'), ('timezone', 'UTC')]:
            conn.sendall(encode(*parts))
        values = {'hostname': '127.0.0.1', 'port': str(listener.getsockname()[1]), 'password': 'synthetic-only'}
        conn.sendall(encode('connect', *(name if name.startswith('VERSION_') else values.get(name, '') for name in args[1:])))
        size_seen = False
        while True:
            parts = read_instruction(conn)
            if parts[0] == 'error':
                assert not should_connect, parts
                break
            if parts[0] == 'size' and parts[1:] == ['0', '64', '48']:
                size_seen = True
            if parts[0] == 'sync':
                assert should_connect and size_seen, parts
                break
        conn.sendall(encode('disconnect'))
    assert finished.wait(10), 'RFB fixture did not finish'
    assert not failures, failures
    assert selected == ([2] if should_connect else []), selected
    print(f'offered={schemes} selected={selected}: ' + ('frame received' if should_connect else 'unsupported account auth rejected'))


env = dict(os.environ, LD_LIBRARY_PATH='/out/lib', OPENSSL_MODULES='/out/lib/ossl-modules')
runtime = tempfile.TemporaryDirectory()
engine_socket = os.path.join(runtime.name, 'guacd.sock') if '--unix' in sys.argv[1:] else None
bind_args = ['-b', engine_socket] if engine_socket else ['-b', '127.0.0.1', '-l', '14829']
process = subprocess.Popen(['/out/bin/guacd', '-f', *bind_args], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
try:
    import time
    for _ in range(100):
        try:
            with connect_engine(.1):
                break
        except OSError:
            time.sleep(.05)
    check([30, 2], True)
    check([2], True)
    check([30], False)
finally:
    process.terminate()
    process.wait(timeout=5)
    runtime.cleanup()
