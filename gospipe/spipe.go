package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	"github.com/dchest/spipe"
)

var (
	fTargetAddr = flag.String("t", "", "target address")
	fKeyFile    = flag.String("k", "", "key file")
	fTimeout    = flag.Int("o", 0, "connection timeout in seconds")
)

func main() {
	flag.Parse()
	if *fTargetAddr == "" || *fKeyFile == "" {
		flag.Usage()
		return
	}
	key, err := ioutil.ReadFile(*fKeyFile)
	if err != nil {
		log.Fatal(err)
	}
	c, err := spipe.Dial(key, "tcp", *fTargetAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	errch := make(chan error, 2)
	ch1, ch2 := make(chan bool, 1), make(chan bool, 1)
	timeout := time.Duration(*fTimeout) * time.Second

	go send(c, errch, ch1, ch2, timeout)
	go receive(c, errch, ch2, ch1, timeout)
  
	for i := 0; i < 2; i++ {
		err = <-errch
		if err != nil && err != io.EOF {
			log.Print(err)
		}
	}
}

func send(c net.Conn, errch chan error, done, otherDone chan bool, timeout time.Duration) {
	for {
		select {
		case <-otherDone:
			errch <- io.EOF
			return
		default:
			if timeout > 0 {
				c.SetReadDeadline(time.Now().Add(timeout))
			}
			if err := copyBytes(c, os.Stdin); err != nil {
				errch <- err
				done <- true
				return
			}
		}
	}
}

func receive(c net.Conn, errch chan error, done, otherDone chan bool, timeout time.Duration) {
	for {
		select {
		case <-otherDone:
			errch <- io.EOF
			return
		default:
			if timeout > 0 {
				c.SetWriteDeadline(time.Now().Add(timeout))
			}
			if err := copyBytes(os.Stdout, c); err != nil {
				errch <- err
				done <- true
				return
			}
		}
	}
}

func copyBytes(dst io.Writer, src io.Reader) (err error) {
	var buf [1024]byte
	var nr, nw int
	nr, err = src.Read(buf[:])
	if nr > 0 {
		nw, err = dst.Write(buf[:nr])
		if nr != nw {
			err = io.ErrShortWrite
		}

	}
	return
}
