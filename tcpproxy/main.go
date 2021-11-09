package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
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

	var monitorStdin bool
	flag.BoolVar(&monitorStdin, "i", false, "terminate when stdin is closed")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: %s [listen:port:destination ...]\n       destination can be either host:port (tcp) or file path (unix socket)\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()
	forwards := flag.Args()
	if len(forwards) == 0 {
		fmt.Printf("usage: %s [listen:port:destination ...]\n\n       destination: host:port (tcp) or file path (unix socket)\n", os.Args[0])
		os.Exit(1)
	}

	wg := new(sync.WaitGroup)
	for _, forward := range forwards {
		wg.Add(1)
		fields := strings.Split(forward, ":")
		switch len(fields) {
		case 3: //format listen:port:unix_path
			go forwardConn(wg, fields[0]+":"+fields[1], fields[2])
		case 4: //format list:port:destination:port
			go forwardConn(wg, fields[0]+":"+fields[1], fields[2]+":"+fields[3])
		default:
			log.Fatalf("invalid forward: %s", forward)
		}
	}
	wg.Wait()
	os.Stdout.Close() //signal that listening sockets are up
	if monitorStdin {
		go func() {
			io.ReadAll(os.Stdin)
			log.Fatal("Stdin closed, terminating")
		}()
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

}
func forwardConn(wg *sync.WaitGroup, listen, destination string) {
	log.Printf("Listening on %s, forwarding to %s", listen, destination)
	s, err := net.Listen("tcp", listen)
	if err != nil {
		log.Fatalf("Failed to create socket: %s", err)
	}
	wg.Done()
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
	var dest net.Conn
	var err error
	if _, err2 := os.Stat(target); os.IsNotExist(err2) {
		dest, err = net.Dial("tcp", target)
	} else {
		dest, err = net.Dial("unix", target)
	}
	if err != nil {
		log.Printf("connection to backend failed: %s", err)
		return
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
