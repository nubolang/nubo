import socket
import threading

import paramiko

HOST_KEY = paramiko.RSAKey.generate(2048)  # in-memory key
USERNAME = "testuser"
PASSWORD = "password"
PORT = 2222  # SSH port


class Server(paramiko.ServerInterface):
    def check_auth_password(self, username, password):
        if username == USERNAME and password == PASSWORD:
            return paramiko.AUTH_SUCCESSFUL
        return paramiko.AUTH_FAILED

    def get_allowed_auths(self, username):
        return "password"

    def check_channel_request(self, kind, chanid):
        if kind == "session":
            return paramiko.OPEN_SUCCEEDED
        return paramiko.OPEN_FAILED_ADMINISTRATIVELY_PROHIBITED


def handle_client(client_socket):
    transport = paramiko.Transport(client_socket)
    transport.add_server_key(HOST_KEY)
    server = Server()
    try:
        transport.start_server(server=server)
        chan = transport.accept(20)
        if chan is None:
            return
        chan.send(b"Welcome to test SSH server!\n")
        while True:
            data = chan.recv(1024)
            if not data:
                break
            print(f"Received: {data.decode().strip()}")
            chan.send(b"pong\n")
    except Exception as e:
        print("Error:", e)
    finally:
        transport.close()


with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind(("0.0.0.0", PORT))
    s.listen()
    print(f"SSH server listening on port {PORT}")
    while True:
        client, addr = s.accept()
        threading.Thread(target=handle_client, args=(client,), daemon=True).start()
