package main

import (
	"context"
	"fmt"
	"net"
	"time"
)

func main() {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, network, "0.0.0.0:1337")
		},
	}

	txtrecords, _ := r.LookupTXT(context.Background(), "blub.t.cncr.io")

	for _, txt := range txtrecords {
		fmt.Println(txt)
	}
}
