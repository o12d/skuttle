package main

import (
	"fmt"
	"os"
	"time"

	"github.com/256dpi/newdns"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/miekg/dns"
)

func report(writeAPI api.WriteAPI) {
	defer writeAPI.Flush()

	p := influxdb2.NewPointWithMeasurement("stat").
		AddTag("unit", "temperature").
		AddField("avg", 23.2).
		AddField("max", 45).
		SetTime(time.Now())

	writeAPI.WritePoint(p)
}

func main() {
	// Create a client
	// You can generate an API Token from the "API Tokens Tab" in the UI
	client := influxdb2.NewClient("https://westeurope-1.azure.cloud2.influxdata.com", os.Getenv("INFLUX_TOKEN"))
	writeAPI := client.WriteAPI("michael_riedmann@live.com", "skuttle")

	// Get errors channel
	errorsCh := writeAPI.Errors()
	// Create go proc for reading and logging errors
	go func() {
		for err := range errorsCh {
			fmt.Printf("write error: %s\n", err.Error())
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
			// return apex records

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
			fmt.Println(e, err, reason)
		},
	})

	// run server
	go func() {
		err := server.Run(":1337")
		if err != nil {
			panic(err)
		}
	}()

	// print info
	fmt.Println("Query apex: dig example.com @0.0.0.0 -p 1337")
	fmt.Println("Query other: dig foo.example.com @0.0.0.0 -p 1337")

	// wait forever
	select {}
}
