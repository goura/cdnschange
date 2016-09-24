package main

import (
	"flag"
	"fmt"
	"os"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/dns/v1"
)

func main() {
	const defaultTtl = 60

	var ttl int
	flag.IntVar(&ttl, "ttl", defaultTtl, "TTL for the record")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s PROJECT MANAGEDZONE RECORDNAME IPADDR\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 4 {
		flag.Usage()
		os.Exit(64)
	}

	project := flag.Arg(0)
	managedZone := flag.Arg(1)
	recordName := flag.Arg(2)
	ipAddr := flag.Arg(3)

	ctx := context.Background()
	hc, err := google.DefaultClient(ctx, dns.CloudPlatformScope)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get Application Default Credentials: %s\n", err)
		os.Exit(1)
	}
	c, err := dns.New(hc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create Cloud DNS client: %s\n", err)
		os.Exit(1)
	}

	// Find exisiting record
	var targetRrdata *dns.ResourceRecordSet
	targetRrdata = nil

	call := c.ResourceRecordSets.List(project, managedZone)
	if err := call.Pages(ctx, func(page *dns.ResourceRecordSetsListResponse) error {
		for _, v := range page.Rrsets {
			if v.Name == recordName {
				targetRrdata = v // Need to send this when changing the record
				break
			}
		}
		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error occured while looping through resource records: %s\n", err)
		os.Exit(1)
	}

	// Build up deletions
	var deletions []*dns.ResourceRecordSet
	if targetRrdata == nil {
		deletions = []*dns.ResourceRecordSet{}
	} else {
		deletions = []*dns.ResourceRecordSet{targetRrdata}
	}

	// Build up additions
	rr := dns.ResourceRecordSet{
		Kind:    "dns#resourceRecordSet",
		Name:    recordName,
		Rrdatas: []string{ipAddr},
		Ttl:     int64(ttl),
		Type:    "A",
	}

	additions := []*dns.ResourceRecordSet{&rr}

	// Call create
	resp, err := c.Changes.Create(project, managedZone, &dns.Change{
		Kind:      "dns#change",
		Additions: additions,
		Deletions: deletions,
	}).Context(ctx).Do()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not change: %s\n", err)
		os.Exit(1)
	}

	_ = resp

	os.Exit(0)
}
