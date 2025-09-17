package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

/*
	old format:

nts <host>

	nts <host> <timeout_s>
	nts <host_ip>
	nts <host_ip> <timeout_s>
	nts <host> ipv4/ipv6     (what IP type you want if possible. If not, you will get the type that is available for NTS)
	nts <host> ipv4/ipv6 <timeout_s>

	<NTP_version> <host>
	<NTP_version> <host> <timeout_s>
	<NTP_version> <host_ip>
	<NTP_version> <host_ip> <timeout_s>
*/
var usage_info = `Usage:
    <mode> <host> [-draft <string>] [-t <timeout>] [-d] [-ipv <4|6>]

draft modes (available):
    draft_ntpv5 <host> <draft>
    draft_ntpv5 <host_ip> <draft> <timeout_s>

where:
	- <mode> can be "nts" (with ntpv4) or an NTP version: ntpv1,ntpv2,ntpv3,ntpv4,ntpv5
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
`

/*
Return codes for measuring NTS:

	    0 -> ok, NTS measurement succeeded
		1 -> KE failed
		2 -> DNS problem, "Could not deduct NTP host and port"
		3 -> KE succeeded, but measurement timeout
		4 -> invalid NTP response (it violates the RFC rules)
		5 -> KE succeeded, but KissCode detected
		6 -> NTS measurement succeeded, but not on the wanted IP family (ex: domain name NTS only works on ipv4)

OBS: 0 and 6 mean the measurement succeeded. (6 has a warning)
OBS: if you measure directly an IP, then the TLS certificate is not validated (because this tool does not know to

	which domain name this IP belongs)

Return codes for measuring NTP:

		-100 -> commands is malformed
		0 -> success, measurement performed. See results on screen
	    1 -> error connecting to the server
	    2 -> could not send data to the connection with the server
	    3 -> measurement timeout
	    4 -> error parsing response

Warning:
 1. In both cases (NTP and NTS) where you use a domain name as the host, consider that this tool does not resolve
    the domain name in terms of the client IP. It resolves the domain name based on the machine that executes this code.
    If you want to use an IP address (for server) near the client, then resolve it somewhere else and use that IP in this code.
    (The aim of this tool is to perform NTP and NTS measurement, not to solve DNS problems)
 2. In NTS measurements performed on a specific IP address, KE may redirect to another IP address. If this is the case, a warning
    will be shown in the response. The measurement succeeded, but KE redirected us to another IP. (you can also see this in
    host vs measured server ip
*/
func main() {
	//mainn()
	//os.Exit(0)
	args := os.Args[1:]
	//server := "2001:4860:4806:c::" //"time.cloudflare.com" //"time.apple.com" //"ntp0.testdns.nl" //"ntp0.testdns.nl" //"time.apple.com"
	//server := "139.84.137.244" //"ntpd-rs.sidnlabs.nl" //args := os.Args[1:]
	//server := "ntpd-rs.sidnlabs.nl"
	//server := "ntp5.example.net" //args := os.Args[1:]
	//server := "ntp0.testdns.nl" //args := os.Args[1:]
	//server := "127.0.0.1"
	//server := "time.android.com" //args := os.Args[1:]
	//args := []string{"ntpv5", server}

	if len(args) < 2 {
		fmt.Println(usage_info)
		os.Exit(-100)
	}
	//parsing command line
	mode := args[0]
	host := args[1]
	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	draft := flagSet.String("draft", "", "draft version for NTPv5 (string)")
	timeout := flagSet.Float64("t", 7.0, "timeout in seconds")
	debugArg := flagSet.Bool("d", false, "enable debug output")
	ipv := flagSet.String("ipv", "", "force IP version (4 or 6)")

	// Parse only args after <mode> and <host>
	flagSet.Parse(args[2:])

	// Validate ipv
	if *ipv != "" && *ipv != "4" && *ipv != "6" {
		fmt.Println("Error: -ipv must be 4 or 6")
		os.Exit(-100)
	}
	// Validate timeout
	if *timeout <= 0 {
		fmt.Println("Error: timeout must be >0 ")
		os.Exit(-100)
	}

	// --- Example usage ---
	//fmt.Println("Mode:", mode)
	//fmt.Println("Host:", host)
	//fmt.Println("Draft:", *draft)
	//fmt.Println("Timeout:", *timeout)
	//fmt.Println("Debug:", *debug_arg)
	//fmt.Println("IP version:", *ipv)
	//if len(args) < 1 || len(args) > 3 {
	//	fmt.Println(usage_info)
	//	os.Exit(-100)
	//}
	if mode == "nts" {
		measureNTS(host, *ipv, *timeout)
		os.Exit(0)
	}
	//ntp versions part
	//time.Sleep(400 * time.Millisecond)
	var output strings.Builder
	//if len(args) == 3 {
	//	timeout_new, err := strconv.ParseFloat(args[2], 64)
	//	if err != nil {
	//		fmt.Println("timeout needs to be a number (int or float)")
	//		os.Exit(-100)
	//	}
	//	timeout = timeout_new
	//}
	result, debug, err := map[string]interface{}{}, "", 0
	if mode == "ntpv1" {
		result, debug, err = performNTPv1Measurement(host, *timeout) //very unlikely to receive an answer as nobody supports ntpv1 anymore
	} else if mode == "ntpv2" {
		result, debug, err = performNTPv3Measurement(host, *timeout, 2) //same code as in 3 basically
	} else if mode == "ntpv3" {
		result, debug, err = performNTPv3Measurement(host, *timeout, 3)
	} else if mode == "ntpv4" {
		result, debug, err = performNTPv4Measurement(host, *timeout)
	} else if mode == "ntpv5" {
		result, debug, err = performNTPv5Measurement(host, *timeout, "")
	} else if mode == "draft_ntpv5" {
		result, debug, err = performNTPv5Measurement(host, *timeout, *draft)
	} else {
		fmt.Println("unknown command\n")
		fmt.Println(usage_info)
		os.Exit(-100)
	}

	if *debugArg {
		fmt.Println(debug + "\nFinal result:\n")
	}
	output.WriteString("\n")
	if err != 0 { //measurement failed. Show the error message
		fmt.Println(result["error"])
	} else {
		jsonToString(result, &output)
		fmt.Print(output.String())
	}
	os.Exit(err)
}
