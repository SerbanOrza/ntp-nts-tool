This tool is mainly used by NTPinfo. 

You can perform NTP and NTS measurements.

You can compile the project with "GOOS=linux GOARCH=amd64 go build -o ntpnts_linux_amd64" (similar on windows)

OBS:
1) NTPv5 is still in draft mode and our tool tries to measure "draft-ietf-ntp-ntpv5-05" and "draft-ietf-ntp-ntpv5-06". At the moment, it should correctly send draft NTPv5 requests to a server,
   but the work is still in progress. If you find a bug in my implementation, please tell me
2) -d will not show too much info for NTS 
3) Currently, "draft_ntpv5" mode and "ntpv5" mode are exactly the same.
4) In all ntp versions, offset and rtt are calculated from the values that were recorded before and after the measurement (so they does not use t1 and t4 from the response as they may be invalid).
  Current usage:
```
Usage:
    <mode> <host> [-draft <string>] [-t <timeout>] [-d] [-ipv <4|6>]

draft modes (available):
    draft_ntpv5 <host> <draft>
    draft_ntpv5 <host_ip> <draft> <timeout_s>

where:
        - <mode> can be "nts" (with ntpv4) or an NTP version: ntpv1,ntpv2,ntpv3,ntpv4,ntpv5, draft_ntpv5
        - <host> can be a domain name or an IP address
        - timeout is a float64 in seconds
        - [-draft <string>] the string can be "draft-ietf-ntp-ntpv5-05" or "draft-ietf-ntp-ntpv5-06"
        - [-d] means debug mode. More data will be shown on screen.
        - [-ipv <4|6>] can be -ipv 4 or -ipv 6. Only for NTS!! (at the moment). It will try that ip type version. If it fails, it tries the other one

Obs:
        - we support both IPv4 and IPv6
        - by default timeout is 7 seconds in both NTP and NTS
        - Use -d if you want pretty data. Otherwise you will see it in JSON format

Example: ./program_name ntpv5 ntp0.testdns.nl -d -draft "draft-ietf-ntp-ntpv5-05" -t 8.2
         ./program_name ntpv2 time.google.com
```
