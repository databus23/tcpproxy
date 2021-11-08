tcpproxy
========
Is very a very simple tcpproxy to listen on multiple addresses and forward do other endpoints.

I wrote this as a simple dependency free helper for developing on macOS.
Its main purpose is to avoid the "Do you want to accept incoming connections" every time a compiled binary is started.
By only listening on localhost in the binary and using the proxy to expose to a wider audience the message only needs to accepted once for the proxy. \o/

```
usage: tcpproxy listen:port:target:port ...
```
