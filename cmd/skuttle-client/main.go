package main

import (
	"context"
	"encoding/base32"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
)

func main() {
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, network, "0.0.0.0:15353")
		},
	}

	enc := base32.StdEncoding

	id := uuid.New()
	userId, _ := uuid.Parse("1B6A7B73-86F0-4DAC-B3F1-636B1803F5A6")
	encUserId := enc.EncodeToString(userId[:])

	payload := "1.0.0;amd64"
	encPayload := enc.EncodeToString(append(id[:], []byte(payload)...))

	targetname := encPayload + "." + encUserId + ".v1.t.cncr.io"
	//drop padding to save space and also avoid confusing dns resolvers
	targetname = strings.ReplaceAll(targetname, "=", "")
	//converting to lowercase to avoid problems with case-sensitive resolvers
	targetname = strings.ToLower(targetname)

	fmt.Println(targetname)
	txtrecords, _ := r.LookupTXT(context.Background(), targetname)
	fmt.Println(len(txtrecords))
	for _, txt := range txtrecords {
		fmt.Println(txt)
	}
}
