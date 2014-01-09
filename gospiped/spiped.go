package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"

	"github.com/dchest/spipe"
)

var (
	fSourceAddr = flag.String("s", "", "source address")
	fTargetAddr = flag.String("t", "", "target address")
	fKeyFile    = flag.String("k", "", "key file")
	fMaxConn    = flag.Int("n", 100, "maximum number of connections")
)

func main() {
	flag.Parse()
	if *fSourceAddr == "" || *fTargetAddr == "" || *fKeyFile == "" {
		flag.Usage()
		return
	}
	key, err := ioutil.ReadFile(*fKeyFile)
	if err != nil {
		log.Fatal(err)
	}
	s, err := spipe.Listen(key, "tcp", *fSourceAddr)
	if err != nil {
		log.Fatal(err)
	}
	waiting := make(chan net.Conn, *fMaxConn)
	spaces := make(chan bool, *fMaxConn)
	for i := 0; i < *fMaxConn; i++ {
		spaces <- true
	}

	go match(key, waiting, spaces)

	for {
		c, err := s.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		log.Printf("Received connection: %s", c.RemoteAddr())
		waiting <- c
	}
}

func match(key []byte, waiting chan net.Conn, spaces chan bool) {
	for c := range waiting {
		<-spaces
		go func(c net.Conn) {
			handleConn(key, c)
			spaces <- true
			log.Printf("Closed connection: %s", c.RemoteAddr())
		}(c)
	}
}

func handleConn(key []byte, c net.Conn) {
	defer c.Close()

	t, err := spipe.Dial(key, "tcp", *fTargetAddr)
	if err != nil {
		log.Print(err)
		return
	}
	defer t.Close()

	fin := make(chan bool, 2)
	ch1, ch2 := make(chan bool, 1), make(chan bool, 1)

	go copyContent(t, c, fin, ch1, ch2)
	go copyContent(c, t, fin, ch2, ch1)

	<-fin
	<-fin
}

func copyContent(dst io.Writer, src io.Reader, fin, done, otherDone chan bool) {
	var b [1024]byte
	for {
		select {
		case <-otherDone:
			fin <- true
			return
		default:
			n, err := src.Read(b[:])
			if err != nil {
				fin <- true
				done <- true
				return
			}
			_, err = dst.Write(b[:n])
			if err != nil {
				fin <- true
				done <- true
				return
			}
		}
	}
}
