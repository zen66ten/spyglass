package main

import (
	"errors"
	"net/netip"
)

func main() {
	println("Hello, Spyglass!")
}

func parsePrefix(s string) (netip.Prefix, error) {

	unmaskedIP, err := netip.ParsePrefix(s)
	if err != nil {
		return netip.Prefix{}, err
	}

	isIPv4 := unmaskedIP.Addr().Is4()
	// If the IP is Ipv6, we return error.
	if !isIPv4 {
		return netip.Prefix{}, errors.New("IPv6 not supported in v1")
	}

	maskedIP := unmaskedIP.Masked()
	return maskedIP, nil
}
