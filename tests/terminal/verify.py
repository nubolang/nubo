"""POSIX PTY integration tests. Run after: go build -o bin/nubo ./cmd/nubo."""
import fcntl
import os
from pathlib import Path
import pty
import select
import signal
import struct
import subprocess
import tempfile
import termios
import time

ROOT = Path(__file__).resolve().parents[2]


def run_case(name, source=None, action=None, keyboard=False, expected=None):
    master, slave = pty.openpty()
    fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", 24, 80, 0, 0))
    original = termios.tcgetattr(slave)
    output = bytearray()
    with tempfile.TemporaryDirectory() as directory:
        path = ROOT / "examples/terminal/move.nubo"
        if source is not None:
            path = Path(directory) / "main.nubo"
            path.write_text(source)
        proc = subprocess.Popen([os.environ.get("NUBO_BIN", str(ROOT / "bin/nubo")), str(path), "--nocolor"],
                                cwd=ROOT, stdin=slave, stdout=slave, stderr=slave)
        try:
            deadline = time.monotonic() + 5
            acted = action is None
            answered = False
            while time.monotonic() < deadline:
                if select.select([master], [], [], 0.02)[0]:
                    output.extend(os.read(master, 65536))
                if keyboard and not answered and b"\x1b[?u" in output:
                    os.write(master, b"\x1b[?11u")
                    answered = True
                if not acted and b"\x1b[?25l" in output and (source is not None or b"FPS:" in output):
                    action(proc, master)
                    acted = True
                if proc.poll() is not None:
                    while select.select([master], [], [], 0)[0]:
                        output.extend(os.read(master, 65536))
                    break
            assert proc.poll() is not None, f"{name}: process did not exit"
            assert b"DATA RACE" not in output, f"{name}: data race: {output!r}"
            assert acted, f"{name}: never reached interactive state: {output!r}"
            restored = termios.tcgetattr(slave)
            # Darwin sets PENDIN while reprocessing input after canonical restoration.
            restored[3] &= ~getattr(termios, "PENDIN", 0)
            original[3] &= ~getattr(termios, "PENDIN", 0)
            assert restored == original, f"{name}: terminal mode not restored: {output!r}"
            assert b"\x1b[?25h" in output and b"\x1b[?1049l" in output, f"{name}: missing visual restoration: {output!r}"
            if source is None:
                assert b"error[" not in output, f"{name}: demo error: {output!r}"
                assert b"FPS:" in output and b"80x24" in output, f"{name}: no demo frame"
            if keyboard:
                assert answered and b"\x1b[<u" in output, f"{name}: protocol not restored"
                assert b"error[" not in output, f"{name}: {output!r}"
            if expected is not None:
                assert b"error[" not in output and expected in output, f"{name}: {output!r}"
            print(f"PASS {name}")
        finally:
            if proc.poll() is None:
                proc.kill()
                proc.wait()
            os.close(master)
            os.close(slave)


setup = '''import terminal from "@std/terminal"
let session = terminal.TerminalSession()
session.setRaw(true)
session.setAlternateScreen(true)
session.setCursorVisible(false)
'''
run_case("demo Escape", action=lambda proc, fd: os.write(fd, b"\x1b[Aa\x1b"))
run_case("demo Ctrl+C handler", action=lambda proc, fd: os.write(fd, b"\x03"))
run_case("normal exit without defer", setup)
run_case("runtime error without defer", setup + 'panic("expected failure")\n')
run_case("default Ctrl+C", setup + 'while true {}\n', lambda proc, fd: os.write(fd, b"\x03"))
run_case("SIGTERM", setup + 'while true {}\n', lambda proc, fd: proc.send_signal(signal.SIGTERM))

run_case("held-key negotiation and cleanup", setup + """
expect(session.setKeyEvents(true), "keyboard negotiation")
expect(session.setKeyEvents(true), "repeated enable")
expect(!session.isKeyDown("w"), "initial held state")
expect(!session.setKeyEvents(false), "disable")
expect(!session.isKeyDown("w"), "disabled held state")
""", keyboard=True)
run_case("unsupported key events", setup + """
expect(!session.setKeyEvents(true), "unsupported terminal must return false")
expect(!session.isKeyDown("w"), "unsupported terminal has no held state")
""")
run_case("protocol demo", keyboard=True, action=lambda proc, fd: os.write(fd, b"\x1b[119;1:1u\x1b[97;1:1u\x1b[119;1:3u\x1b[97;1:3u\x1b[27u"))
run_case("protocol Ctrl+C handler", keyboard=True, action=lambda proc, fd: os.write(fd, b"\x1b[99;5u"))

run_case("Nubo session wrapper", setup + """
let alias = session
let size = alias.size()
expect(size.width == 80 && size.height == 24, "session size")
expect(isNil(alias.pollKey()), "default poll must be nonblocking")
expect(isNil(alias.pollResize()), "unchanged terminal size")
alias.clear()
alias.setCursorPosition(4, 3)
alias.flush()
alias.close()
session.close()
""", expected=b"\x1b[2J\x1b[H\x1b[3;4H")
