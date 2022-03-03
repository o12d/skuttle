package main

import (
	"bytes"
	"encoding/base32"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/256dpi/newdns"
	"github.com/google/uuid"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/miekg/dns"
	"golang.org/x/net/html/charset"
)

var client influxdb2.Client = influxdb2.NewClient("https://westeurope-1.azure.cloud2.influxdata.com", os.Getenv("INFLUX_TOKEN"))
var writeAPI api.WriteAPI = client.WriteAPI("michael_riedmann@live.com", "skuttle")

func report(appId uuid.UUID, requestId uuid.UUID, version string, arch string) {
	defer writeAPI.Flush()

	p := influxdb2.NewPointWithMeasurement("stat").
		AddField("requestId", requestId.String()).
		AddTag("appId", appId.String()).
		AddTag("version", version).
		AddTag("arch", arch).
		SetTime(time.Now())

	writeAPI.WritePoint(p)
}

func convertToUTF8(strBytes []byte, origEncoding string) string {
	byteReader := bytes.NewReader(strBytes)
	reader, _ := charset.NewReaderLabel(origEncoding, byteReader)
	strBytes, _ = ioutil.ReadAll(reader)
	return string(strBytes)
}

func addPadding(s string) string {
	d := strings.ToUpper(s)
	padding := ""
	switch len(d) % 8 {
	case 2:
		padding = "======"
	case 4:
		padding = "===="
	case 5:
		padding = "==="
	case 7:
		padding = "=="
	}
	return d + padding
}

func decodeData(s string) ([]byte, error) {
	d := addPadding(s)

	e, err := base32.StdEncoding.DecodeString(d)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func convertSliceToUUID(slice []byte) uuid.UUID {
	var bId [16]byte

	copy(bId[:], slice[:16])
	return uuid.UUID(bId)
}

func processData(d []byte) (string, uuid.UUID, error) {
	id := convertSliceToUUID(d[:16])
	bPayload := d[16:]
	payload := convertToUTF8(bPayload, "utf8")

	return payload, id, nil
}

func main() {
	// Create a client
	// You can generate an API Token from the "API Tokens Tab" in the UI

	// Get errors channel
	errorsCh := writeAPI.Errors()
	// Create go proc for reading and logging errors
	go func() {
		for err := range errorsCh {
			log.Printf("write error: %s\n", err.Error())
		}
	}()

	defer client.Close()

	// create zone
	zone := &newdns.Zone{
		Name:             "t.cncr.io.",
		MasterNameServer: "alex.ns.cloudflare.com.",
		AllNameServers: []string{
			"alex.ns.cloudflare.com.",
			"zoe.ns.cloudflare.com.",
		},
		Handler: func(name string) ([]newdns.Set, error) {
			defer func() {
				if err := recover(); err != nil {
					log.Println("panic occurred:", err)
				}
			}()

			s := strings.Split(name, ".")
			apiId := s[2]
			decAppId, _ := decodeData(s[1])
			appId := uuid.UUID(convertSliceToUUID(decAppId))
			decPayload, _ := decodeData(s[0])
			tags, id, err := processData(decPayload)
			if err != nil {
				panic(err)
			}
			sTags := strings.Split(tags, ";")
			log.Printf("%v %v %v %v %v\n", apiId, appId, tags, id, err)
			report(appId, id, sTags[0], sTags[1])

			var data = []string{name}
			return []newdns.Set{
				{
					Name: "t.cncr.io.",
					Type: newdns.TXT,
					Records: []newdns.Record{
						{Data: data},
					},
				},
			}, nil

			//return nil, nil
		},
	}

	// create server
	server := newdns.NewServer(newdns.Config{
		Handler: func(name string) (*newdns.Zone, error) {
			// check name
			if newdns.InZone("t.cncr.io.", name) {
				return zone, nil
			}

			return nil, nil
		},
		Logger: func(e newdns.Event, msg *dns.Msg, err error, reason string) {
			// log.Printf("%-8s %v %s\n", e, err, reason)
		},
	})

	// run server
	go func() {
		err := server.Run(":15353")
		if err != nil {
			panic(err)
		}
	}()

	// print info
	log.Println("running on port 15353")

	// wait forever
	select {}
}
