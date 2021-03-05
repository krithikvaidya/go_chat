# Client-Server Chat Application in Golang

## How to run:

- Clone the repository
- Change directory to the repo's root folder

#### Client:

```
./go_chat -h "<server ip:port>" -p "<server-password>"
```

#### Server

```
./go_chat -s -p "<server-password>"
```

## Message Protocol:

1. pm \<username\> \<message\>  --> from client to server, unicast message
2. broadcast \<message\> --> from client to server, broadcast message

## Screenshots:

#### Execution with 1 server, 3 clients:

- Server:
![](https://i.imgur.com/VTqLUEK.png)

- Client 1:
![](https://i.imgur.com/EXVsyFk.png)

- Client 2:
 ![](https://i.imgur.com/kYMWF8r.png)

- Client 3:
 ![image](https://user-images.githubusercontent.com/43742553/110122794-0e40a600-7de6-11eb-91c9-73f741871f2b.png)


## Features

- Personal messages and Broadcast messages.
- Multithreading at Server and Client sides using goroutines.
- Multithread synchronization using Mutex locks.
- Network communication using raw sockets.
