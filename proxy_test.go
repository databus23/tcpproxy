package tcpproxy

import (
	"net/http"
	"testing"
	"time"
)

func TestToUnix(t *testing.T) {

	listener, err := ToUnix("127.0.0.1:6789")
	if err != nil {
		t.Error("Failed to create listener", err)
	}
	defer listener.Close()

	called := false
	srv := http.Server{Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	})}
	go srv.Serve(listener)
	time.Sleep(10 * time.Millisecond) // allow the Serve go routine to be called
	_, err = http.Get("http://127.0.0.1:6789")
	if err != nil {
		t.Fatal("http request failed", err)
	}
	if !called {
		t.Fatal("handler not called")
	}

}
