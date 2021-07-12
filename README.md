### dbsproxy
Proxy server for dbs2go backends.

Every DBS query can be speed up by adding timestamp to it.
Therefore, we need a proxy server which will consume DBS
queries and pass them as goroutines to dbs2go backend servers.
For that we can use `Accept: application/ndjson` (new line
delimiter JSON) which will allow backend servers to send
only JSON results. The proxy server will use GoLang `sync.WaitGroup`.
Here is schematic view of the architecture:
```
                    |-> dbs2go (timerange t1-t2)
client -> dbsproxy -|-> dbs2go (timerange t2-t3)
                    |-> dbs2go ...
```
