//go:build !js || !wasm

package network

import (
	"context"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/kivutar/goro/glog"
)

// preferredDNS, when set (e.g. "8.8.8.8"), is queried directly over UDP 53
// before the system resolver. The rg35xx's systemd-resolved stub sits in
// front of DNS that NXDOMAINs DDNS hostnames; asking an external resolver
// directly sidesteps it. Fails soft: any error falls back to the system
// resolver, so behavior elsewhere is unchanged.
var preferredDNS string

// SetPreferredDNS selects the direct resolver server; empty disables it.
func SetPreferredDNS(server string) {
	if server == "" {
		preferredDNS = ""
		return
	}
	if ip := net.ParseIP(server); ip != nil {
		preferredDNS = ip.String()
		return
	}
	glog.Warnf("preferred dns ignored: not an IP address %q", server)
}

type dnsCacheEntry struct {
	ip  string
	at  time.Time
}

var (
	directResolverOnce sync.Once
	directResolver     *net.Resolver
	dnsCache           sync.Map // host -> dnsCacheEntry
)

// dnsCacheTTL bounds how long a resolved DDNS address is reused; home IPs
// can change between sessions, so keep it short.
const dnsCacheTTL = 30 * time.Second

func getDirectResolver() *net.Resolver {
	directResolverOnce.Do(func() {
		directResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: 3 * time.Second}
				return d.DialContext(ctx, "udp", net.JoinHostPort(preferredDNS, "53"))
			},
		}
	})
	return directResolver
}

// resolveDialTarget converts host:port into the address dialGameServer
// should use. IP hosts pass through untouched; hostnames resolve through
// the preferred DNS server first (30s cache), falling back to the hostname
// itself — dial then goes through the system resolver exactly as before.
func resolveDialTarget(ctx context.Context, host string, port int) string {
	portText := strconv.Itoa(port)
	if net.ParseIP(host) != nil || preferredDNS == "" {
		return net.JoinHostPort(host, portText)
	}
	if entry, ok := dnsCache.Load(host); ok {
		if cached := entry.(dnsCacheEntry); time.Since(cached.at) < dnsCacheTTL {
			return net.JoinHostPort(cached.ip, portText)
		}
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	ips, err := getDirectResolver().LookupIP(lookupCtx, "ip", host)
	cancel()
	ip := firstIPv4(ips)
	if err != nil || ip == "" {
		// Fall back to the system resolver by dialing the hostname.
		glog.Debugf("dns direct lookup failed host=%s err=%v; falling back to system resolver", host, err)
		return net.JoinHostPort(host, portText)
	}
	dnsCache.Store(host, dnsCacheEntry{ip: ip, at: time.Now()})
	glog.Infof("dns resolved host=%s ip=%s via %s", host, ip, preferredDNS)
	return net.JoinHostPort(ip, portText)
}

func firstIPv4(ips []net.IP) string {
	for _, candidate := range ips {
		if v4 := candidate.To4(); v4 != nil {
			return v4.String()
		}
	}
	return ""
}
