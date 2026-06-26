import socket
import select
import sys
import termios
import tty
import time

HOST = "udp-service"
PORT = 9001
SEND_INTERVAL = 0.1



def redraw(text):
    sys.stdout.write(f"\r> {text}\033[K") 
    sys.stdout.flush()

with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as client_socket:
    client_socket.sendto(b"JOIN", (HOST, PORT))

    current_text = ""
    last_sequence = -1
    text_changed = False 
    next_send_time = time.monotonic() + SEND_INTERVAL

    stdin_fd = sys.stdin.fileno()#we get the number for the keyboard input stream
    old_terminal_settings = termios.tcgetattr(stdin_fd)#we take a snapshot of the terminal settings before the client uses it

    try:
        tty.setcbreak(stdin_fd)

        while True:
            # wait for either:
            # - a keypress
            # - a server message
            # - the next 0.1-second send time
            now = time.monotonic()
            timeout = max(0, next_send_time - now) #calculate how long until we should send, we use max with 0 because negative time doesnt make sense,                                                         negative time means we need to send now
            readable, _, _ = select.select( #readable becomes a list of the watched sources that currently have data ready to read:
                [stdin_fd, client_socket],
                [],
                [],
                timeout
            )

            if stdin_fd in readable:
                key = sys.stdin.read(1)

                
                if key in ("\x7f", "\b"): #if backspace we remove one chracter from current text
                    if current_text:
                        current_text = current_text[:-1]
                        text_changed = True
                        redraw(current_text)

                elif key.isprintable(): #otherwise we add
                    current_text += key
                    text_changed = True
                    redraw(current_text)

            if client_socket in readable: 
                data, _address = client_socket.recvfrom(1024)
                message = data.decode("utf-8")

                command, separator, payload = message.partition(" ") #split incoming datagram

                if command == "TEXT" and separator:
                    sequence_text, separator, received_text = payload.partition(" ")

                    if separator:
                        try:
                            received_sequence = int(sequence_text)
                        except ValueError:
                            continue
                
                        if received_sequence > last_sequence: #if the message is newer we update, this is checked with the sequence number
                            last_sequence = received_sequence
                            current_text = received_text
                            redraw(current_text)

            if time.monotonic() >= next_send_time:
                if text_changed: #if its time to send, we send, and then we tell the program that the text is up to date
                    update = f"UPDATE {current_text}"
                    client_socket.sendto(update.encode("utf-8"), (HOST, PORT))
                    text_changed = False

                next_send_time = time.monotonic() + SEND_INTERVAL #and we schedule the next send time

    except KeyboardInterrupt: #catches ctrl+c
        pass


    finally: 
        try:
            client_socket.sendto(b"LEAVE", (HOST, PORT)) #send leave udp
        
        finally:
            termios.tcsetattr( #reset settings
                stdin_fd,
                termios.TCSADRAIN,
                old_terminal_settings
            )
            print()
