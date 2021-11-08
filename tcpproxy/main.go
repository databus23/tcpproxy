package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// Get preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Printf("Failed to get outbound ip: %s", err)
		os.Exit(1)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func main() {

	forwards := os.Args[1:]
	if len(forwards) == 0 {
		fmt.Printf("usage: %s [listen:port:forward:port ...]\n", os.Args[0])
	}

	for _, forward := range forwards {
		fields := strings.Split(forward, ":")
		switch len(fields) {
		case 3: //format port:forward:port
			go forwardPort(GetOutboundIP().String()+":"+fields[0], fields[1]+":"+fields[2])
		case 4:
			go forwardPort(fields[0]+":"+fields[1], fields[2]+":"+fields[3])
		default:
			log.Fatalf("invalid forward: %s", forward)
		}
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

}
func forwardPort(listen, destination string) {
	log.Printf("Listening on %s forwarding to %s", listen, destination)
	s, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatalf("Failed to create socket: %s", err)
	}
	for {
		src, _ := s.Accept()
		go handleConn(src, destination)
	}

}

func handleConn(src net.Conn, target string) {
	defer src.Close()
	log.Printf("New connection src: %s", src.RemoteAddr())
	start := time.Now()
	defer func() {
		log.Printf("connection closed src %s duration %s", src.RemoteAddr(), time.Now().Sub(start))
	}()
	dest, err := net.Dial("tcp", target)
	if err != nil {
		log.Printf("connection to backend failed: %s", err)
	}
	defer dest.Close()
	errc := make(chan error, 1)
	go proxyCopy(src, dest, errc)
	go proxyCopy(dest, src, errc)

	if err := <-errc; err != nil {
		log.Printf("connection closed: %s", err)
	}
	//give the other side 2 seconds to finish up before closing connections
	select {
	case <-time.After(2 * time.Second):
	case <-errc:
	}
}

func proxyCopy(dest, src net.Conn, errc chan error) {
	_, err := io.Copy(dest, src)
	errc <- err
}
