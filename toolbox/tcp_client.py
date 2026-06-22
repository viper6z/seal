import socket

HOST = "tcp-service"
PORT = 9000

while True:
    request = input("write your request, or type quit to exit: ")

    if request.lower() == "quit":
        break

    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as client_socket:
        client_socket.connect((HOST, PORT))
        client_socket.sendall((request + "\n").encode("utf-8"))

        data = client_socket.recv(1024)
        output = data.decode("utf-8")
        print(output, end="")
