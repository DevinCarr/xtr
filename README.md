xtr
====

Cross-Traceroute (xtr) runs both ICMPv4 and ICMPv6 message echo requests to the target with the intent to cross refrerence the reverse IP results and check for routers that serve both IPv4 and IPv6 traffic to the destination endpoint.

Example:
```shell
$ ./xtr devincarr.com
v4: 104.21.34.245
 1: 10.0.0.1
 2: <omitted for privacy>
 3: <omitted for privacy>
 4: be-232-rar01.santaclara.ca.sfba.comcast.net. (162.151.78.253)
 5: be-39911-cs01.sunnyvale.ca.ibone.comcast.net. (96.110.41.113)
 6: be-1112-cr12.sunnyvale.ca.ibone.comcast.net. (96.110.46.6)
 7: be-302-cr12.9greatoaks.ca.ibone.comcast.net. (96.110.37.174)
 8: be-1412-cs04.9greatoaks.ca.ibone.comcast.net. (68.86.166.169)
 9: be-1411-cr11.9greatoaks.ca.ibone.comcast.net. (68.86.166.166)
10: be-303-cr11.losangeles.ca.ibone.comcast.net. (96.110.36.153)
11: be-1211-cs02.losangeles.ca.ibone.comcast.net. (96.110.45.169)
12: be-3211-pe11.600wseventh.ca.ibone.comcast.net. (96.110.33.54)
13: 50.242.151.226
14: 172.70.212.2
15: 104.21.34.245

v6: 2606:4700:3030::6815:22f5
 1: <omitted for privacy>
 2: <omitted for privacy>
 3: <omitted for privacy>
 4: be-1-rur01.santaclara.ca.sfba.comcast.net. (2001:558:80:402::1)
 5: be-232-rar01.santaclara.ca.sfba.comcast.net. (2001:558:80:213::1)
 6: *
 7: *
 8: *
 9: *
10: *
11: 2001:559::2e6
12: 2606:4700:3030::6815:22f5

xtr: 1
be-232-rar01.santaclara.ca.sfba.comcast.net. (162.151.78.253) (2001:558:80:213::1)
```

In the above example, the `be-232-rar01.santaclara.ca.sfba.comcast.net` hostname serves both A (162.151.78.253) and AAAA (2001:558:80:213::1) addresses along the route to the final destination `devincarr.com`.

## Why

I created this tool to answer the question: "Does my route along the internet share any routers in both IPv4 and IPv6 routes?"

As it turns out, to some destinations, my ISP provides a router into their network that actually provides both a A and AAAA address. There may be other routers along the route that share both addresses but they don't resolve when I search the IPs via reverse DNS lookups.

In the process I also learned a lot about the ICMP ([RFC 792](https://datatracker.ietf.org/doc/html/rfc792)) and ICMPv6 ([RFC 4443](https://datatracker.ietf.org/doc/html/rfc4443)).