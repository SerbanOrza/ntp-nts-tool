This tool is mainly used by NTPinfo.
This tool provides you the raw results from performing NTP and NTS measurements.

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
### Examples

Example of an **NTPv4** response format:
```json
{
  "Host": <string>,
  "Measured server IP": <string>,
  "client_recv_time": <unsigned_int64>,
  "leap": <int>,
  "mode": <int>,
  "offset": <double>,
  "orig_timestamp": <unsigned_int64>,
  "poll": <int8>,
  "precision": <double>,
  "recv_timestamp": <unsigned_int64>,
  "ref_id": <uint32>,
  "ref_timestamp": <unsigned_int64>,
  "root_delay": <double>,
  "root_disp": <double>,
  "rtt": <double>,
  "stratum": <int>,
  "tx_timestamp": <unsigned_int64>,
  "version": 4
}
```

Example of a **draft NTPv5** response format:
```json
{
  "Host": <string>,
  "Measured server IP": <string>,
  "client_cookie": <unsigned_int64>,
  "client_cookie_valid": <bool>,
  "client_recv_time": <unsigned_int64>,
  "draft": <string>,
  "era": <int>,
  "flags_decoded": {
    "auth_nak": <bool>,
    "interleaved": <bool>,
    "synchronized": <bool>
  },
  "flags_raw": <int>,
  "leap": <int>,
  "mode": <int>,
  "offset": <double>,
  "orig_timestamp": <unsigned_int64>,
  "poll": <int8>,
  "precision": <double>,
  "recv_timestamp": <unsigned_int64>,
  "root_delay": <double>,
  "root_disp": <double>,
  "rtt": <double>,
  "server_cookie": <unsigned_int64>,
  "stratum": <int>,
  "timescale": <int>,
  "tx_timestamp": <unsigned_int64>,
  "version": 5
}
```

Example of an **NTS** response:
```json
{
  "Host": "ntpd-rs.sidnlabs.nl",
  "Measured server IP": "2401:c080:3000:2945:5400:4ff:fe69:f923",
  "Measured server port": "123",
  "client_recv_time": 17052749908339855186,
  "client_sent_time": 17052749907615849113,
  "kissCode": "",
  "leap": 0,
  "minError": 0.741145376,
  "mode": 4,
  "offset": 0.825159143,
  "poll": 1,
  "precision": 0.000003814,
  "ref_id": "133.243.238.243",
  "ref_id_raw": "0x85f3eef3",
  "ref_time": 17052749696910491648,
  "root_delay": 0.043182373,
  "root_disp": 0.00062561,
  "root_dist": 0.106230563,
  "rtt": 0.168027534,
  "server_recv_time": 17052749917887107331,
  "server_sent_time": 17052749911523050338,
  "stratum": 2,
  "version": 4
}
```