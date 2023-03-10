package privnet

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

// Look up the 6PN addresses for all instances of the given app
func AllPeerIPs(ctx context.Context, appName string) ([]net.IPAddr, error) {
	return Get6PN(ctx, fmt.Sprintf("%s.internal", appName))
}

// Load all allocation IDs from the vms.{app}.internal DNS record
func AllPeerAllocIDs(ctx context.Context, appName string) ([]string, error) {
	records, err := newResolver().LookupTXT(ctx, fmt.Sprintf("vms.%s.internal", appName))
	if err != nil {
		return nil, err
	}

	allocIDs := make([]string, 0)
	for _, record := range records {
		for _, alloc := range strings.Split(record, ",") {
			allocID, _, ok := strings.Cut(alloc, " ")
			if ok {
				// We should truncate the alloc IDs to the 8 character
				// prefix that is used in fly's VM DNS.
				allocIDs = append(allocIDs, allocID[:8])
			}
		}
	}
	return allocIDs, nil
}

// Load all regions the app is deployed in, from the regions.{app}.internal DNS record
func GetRegions(ctx context.Context, appName string) ([]string, error) {
	records, err := newResolver().LookupTXT(ctx, fmt.Sprintf("regions.%s.internal", appName))
	if err != nil {
		return nil, err
	}

	regions := make([]string, 0)
	for _, record := range records {
		regions = append(regions, strings.Split(record, ",")...)
	}
	return regions, nil
}

func Get6PN(ctx context.Context, hostname string) ([]net.IPAddr, error) {
	res := newResolver()
	ips, err := res.LookupIPAddr(ctx, hostname)
	if err != nil {
		return ips, err
	}

	// make sure we're including the local ip, just in case it's not in service discovery yet
	local, err := res.LookupIPAddr(ctx, "fly-local-6pn")
	if err != nil || len(local) < 1 {
		return ips, err
	}

	localExists := false
	for _, v := range ips {
		if v.IP.String() == local[0].IP.String() {
			localExists = true
		}
	}

	if !localExists {
		ips = append(ips, local[0])
	}
	return ips, err
}

func PrivateIPv6() (net.IP, error) {
	ips, err := net.LookupIP("fly-local-6pn")
	if err != nil && !strings.HasSuffix(err.Error(), "no such host") && !strings.HasSuffix(err.Error(), "server misbehaving") {
		return nil, err
	}

	if len(ips) > 0 {
		return ips[0], nil
	}

	return net.ParseIP("127.0.0.1"), nil
}

func newResolver() *net.Resolver {
	// Get the nameserver to use from the environment
	// or fall back to fdaa::3
	nameserver := os.Getenv("FLY_NAMESERVER")
	if nameserver == "" {
		nameserver = "fdaa::3"
	}
	nameserver = net.JoinHostPort(nameserver, "53")

	// We can use this DNS resolver to look up fly-based DNS
	// records. This can be used for clustering information
	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 1 * time.Second,
			}
			return d.DialContext(ctx, "udp6", nameserver)
		},
	}
}
