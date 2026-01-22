import socket
import threading

HOST = "0.0.0.0"
PORT = 2323


def handle(conn, addr):
    conn.sendall(b"Welcome to test telnet server\r\n")
    try:
        while True:
            data = conn.recv(1024)
            if not data:
                break
            print(f"Received data from {addr}: {data.decode()}")
            conn.sendall(b"pong\r\n")
    finally:
        conn.close()


with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind((HOST, PORT))
    s.listen()
    print(f"Listening on {HOST}:{PORT}")

    while True:
        conn, addr = s.accept()
        threading.Thread(target=handle, args=(conn, addr), daemon=True).start()
