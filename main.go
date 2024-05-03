package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/user"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/websocket"
)

var local bool
var pass string

func init() {
	flag.BoolVar(&local, "local", false, "listen locally - do not connect to any host")
	pass = "root"
	if os.Getenv("SSH_PASSWORD") != "" {
		pass = os.Getenv("SSH_PASSWORD")
	}
}

func main() {
	flag.Parse()

	mux := &http.ServeMux{}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, WEBPAGE)
	})

	mux.Handle("/echo", websocket.Handler(func(c *websocket.Conn) {
		io.Copy(c, c)
	}))

	s := http.Server{Handler: mux}

	// select between a standard listener, or a listener forwarded though ssh
	var l net.Listener
	if local {
		var err error
		l, err = net.Listen("tcp", ":13337")
		if err != nil {
			panic(err)
		}
	} else {
		if len(os.Args) <= 1 {
			log.Fatalf("no ssh address specified")
		}

		u, err := user.Current()
		if err != nil {
			panic(err)
		}

		conn, err := ssh.Dial("tcp", os.Args[1],
			&ssh.ClientConfig{
				User: u.Username,
				Auth: []ssh.AuthMethod{
					ssh.Password(pass),
				},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			},
		)
		if err != nil {
			log.Panicf("unable to dial ssh host %s: %s", os.Args[1], err)
		}
		defer conn.Close()

		l, err = conn.Listen("tcp", "localhost:13337")
		if err != nil {
			log.Panicf("unable to forward embedded webserver: %s", err)
		}
	}

	log.Printf("%T listening on %s", l, l.Addr())
	err := s.Serve(l)
	log.Fatalf("Webserver exited with error: %s", err)
}

const WEBPAGE = `
<html>
<head>
<script>
var i = 0;
const socket = new WebSocket("ws://"+window.location.host+"/echo");
socket.addEventListener("message", (event) => {
	console.log("Message from server", event.data);
});
setInterval(function() {
	if (socket.readyState != WebSocket.OPEN) {
		console.log("Socket not Open:", socket.readyState, "!=", WebSocket.OPEN)
		return
	}
	socket.send("Hello websocket, this is number " + i);
	i++;
}, 1000)
</script>
</head>
<body>
</body>
`
