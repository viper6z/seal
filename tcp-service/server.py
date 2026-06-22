#application protocol running on TCP using socket

import socket

with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as server_socket:
    port = 9000
    server_socket.bind(('0.0.0.0', port))
    server_socket.listen(5)

    while True:
        client_socket, client_address = server_socket.accept()
        data = client_socket.recv(1024) #raw bytes from tcp
        if not data:
            break
        request = data.decode("utf-8").strip()#decode and strip
        command, separator, argument = request.partition(" ")#split
        #responses:
        if command == "PING" and not argument:
            response = "PONG\n" 
        elif command == "ECHO" and argument:
            response = f"ECHO {argument}\n"
        else:
            response = "ERROR: invalid request\n"

        client_socket.sendall(response.encode("utf-8"))

        client_socket.close()

