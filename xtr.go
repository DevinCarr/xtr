package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	IcmpReplySuccess = 0
	IcmpReplyTimeout = 1
	RouteUnknown     = "*"
	RouteEnd         = "END"
)

type Routes struct {
	ipv4 string
	ipv6 string
}

func sendIcmpEcho(socket *icmp.PacketConn, icmpType icmp.Type, dst net.IP, hops int) error {
	var err error
	if icmpType == ipv4.ICMPTypeEcho {
		err = socket.IPv4PacketConn().SetTTL(hops)
	} else {
		err = socket.IPv6PacketConn().SetHopLimit(hops)
	}
	if err != nil {
		return err
	}

	wm := icmp.Message{
		Type: icmpType, Code: 0,
		Body: &icmp.Echo{
			ID: os.Getpid() & 0xffff, Seq: 1,
			Data: []byte("xtr-echo"),
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return err
	}
	if _, err := socket.WriteTo(wb, &net.UDPAddr{IP: dst, Zone: "en0"}); err != nil {
		return err
	}
	return nil
}

func icmpPing(socket *icmp.PacketConn, icmpType icmp.Type, dst net.IP, hops int, tries int, timeout time.Duration) (net.IP, icmp.Type, int, error) {
	var rm *icmp.Message
	rb := make([]byte, 1500)
	for attempt := 0; attempt < tries; attempt++ {
		err := sendIcmpEcho(socket, icmpType, dst, hops)
		if err != nil {
			return nil, nil, -1, err
		}
		end := time.Now().Add(timeout)
		for {
			if time.Now().After(end) {
				return nil, nil, IcmpReplyTimeout, nil
			}
			n, peer, err := socket.ReadFrom(rb)
			if err != nil {
				return nil, nil, -1, err
			}
			if icmpType == ipv4.ICMPTypeEcho {
				rm, err = icmp.ParseMessage(1, rb[:n])
			} else {
				rm, err = icmp.ParseMessage(58, rb[:n])
			}
			if err != nil {
				return nil, nil, -1, err
			}

			switch rm.Type {
			case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
				host, _, _ := net.SplitHostPort(peer.String())
				return net.ParseIP(host), rm.Type, IcmpReplySuccess, nil
			case ipv4.ICMPTypeTimeExceeded, ipv6.ICMPTypeTimeExceeded:
				host, _, _ := net.SplitHostPort(peer.String())
				return net.ParseIP(host), rm.Type, IcmpReplySuccess, nil
			default:
				// drop packet if we don't care about the response type
				//log.Printf("invalid: %v", rm.Type)
			}
		}
	}
	return nil, nil, IcmpReplyTimeout, nil
}

func runRoute(socket *icmp.PacketConn, icmpType icmp.Type, dst net.IP, maxHops int, attempts int, timeout time.Duration, routes chan<- string) error {
	for hops := 1; hops <= maxHops; hops++ {
		peer, resp, ret, err := icmpPing(socket, icmpType, dst, hops, attempts, timeout)
		if err != nil {
			return err
		}

		if ret == 1 {
			routes <- RouteUnknown
			//log.Printf("%2d: *", hops)
			continue
		}

		switch resp {
		case ipv4.ICMPTypeEchoReply, ipv4.ICMPTypeTimeExceeded,
			ipv6.ICMPTypeEchoReply, ipv6.ICMPTypeTimeExceeded:
			routes <- peer.String()
			//log.Printf("%2d: %s", hops, peer.String())
		default:
			log.Fatal(resp)
		}

		if dst.Equal(peer) {
			routes <- RouteEnd
			return nil
		}
	}
	routes <- RouteEnd
	return fmt.Errorf("exceeded max hops: %d", maxHops)
}

func main() {
	if len(os.Args) <= 1 || len(os.Args[1]) <= 0 {
		panic(fmt.Errorf("missing host"))
	}
	host := os.Args[1]
	addrs, err := net.LookupIP(host)
	if err != nil {
		panic(err)
	}

	var dst4 net.IP
	var dst6 net.IP
	for _, addr := range addrs {
		if dst4 == nil && addr.To4() != nil {
			dst4 = addr
		}
		if dst6 == nil && addr.To16() != nil {
			dst6 = addr
		}
		if dst4 != nil && dst6 != nil {
			break
		}
	}
	timeout, _ := time.ParseDuration("1s")
	attempts := 3
	maxHops := 64

	r4 := make(chan string, maxHops)
	r6 := make(chan string, maxHops)
	var c4 *icmp.PacketConn
	if dst4 != nil {
		c4, err = icmp.ListenPacket("udp4", "0.0.0.0")
		if err != nil {
			panic(err)
		}
		defer c4.Close()
		go runRoute(c4, ipv4.ICMPTypeEcho, dst4, maxHops, attempts, timeout, r4)
	}

	var c6 *icmp.PacketConn
	if dst6 != nil {
		c6, err = icmp.ListenPacket("udp6", "::")
		if err != nil {
			panic(err)
		}
		defer c6.Close()
		go runRoute(c6, ipv6.ICMPTypeEchoRequest, dst6, maxHops, attempts, timeout, r6)
	}

	routes := make(map[string]Routes)
	shared := 0
	fmt.Printf("v4: %v\n", dst4)
	for hops := 1; hops <= maxHops; hops++ {
		ip := <-r4
		if ip == RouteEnd {
			break
		}
		if ip == RouteUnknown {
			fmt.Printf("%2d: %s\n", hops, ip)
			continue
		}
		hosts, err := net.LookupAddr(ip)
		if err != nil {
			fmt.Printf("%2d: %s\n", hops, ip)
			continue
		}
		hostname := ""
		for _, host := range hosts {
			if hostname == "" {
				hostname = host
			}
			routes[host] = Routes{ip, ""}
		}
		if hostname != "" {
			fmt.Printf("%2d: %s (%s)\n", hops, hostname, ip)
		} else {
			fmt.Printf("%2d: %s\n", hops, ip)
		}
	}
	fmt.Println()
	fmt.Printf("v6: %v\n", dst6)
	for hops := 1; hops <= maxHops; hops++ {
		ip := <-r6
		if ip == RouteEnd {
			break
		}
		if ip == RouteUnknown {
			fmt.Printf("%2d: %s\n", hops, ip)
			continue
		}
		hosts, err := net.LookupAddr(ip)
		if err != nil {
			fmt.Printf("%2d: %s\n", hops, ip)
			continue
		}
		hostname := ""
		for _, host := range hosts {
			if hostname == "" {
				hostname = host
			}
			route, exists := routes[host]
			if exists {
				route.ipv6 = ip
				shared++
				routes[host] = route
			}
		}
		if hostname != "" {
			fmt.Printf("%2d: %s (%s)\n", hops, hostname, ip)
		} else {
			fmt.Printf("%2d: %s\n", hops, ip)
		}
	}
	fmt.Println()
	fmt.Printf("xtr: %d\n", shared)
	if shared > 0 {
		for host, route := range routes {
			if route.ipv4 != "" && route.ipv6 != "" {
				fmt.Printf("%s (%s) (%s)\n", host, route.ipv4, route.ipv6)
			}
		}
	}
}
