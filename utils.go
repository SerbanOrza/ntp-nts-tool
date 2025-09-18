package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

func time32ToSeconds(val uint32) float64 {
	intPart := val >> 16
	fracPart := val & 0xFFFF
	return float64(intPart) + float64(fracPart)/65536.0
}

func ntp64ToTime(ntp uint64) time.Time {
	// NTP epoch starts 1900, Unix starts 1970 -> difference is 2208988800s
	const ntpEpochOffset = 2208988800
	secs := int64(ntp >> 32)
	frac := float64(ntp&0xFFFFFFFF) / (1 << 32)
	return time.Unix(secs-int64(ntpEpochOffset), int64(frac*1e9))
}
func timeToNtpUint64(t time.Time) uint64 {
	const ntpEpochOffset = 2208988800
	secs := uint64(t.Unix() + ntpEpochOffset)
	frac := uint64((float64(t.Nanosecond()) / 1e9) * (1 << 32))
	return (secs << 32) | frac
}
func nowToNtpUint64() uint64 {
	t := time.Now()
	unixSec := t.Unix() // seconds since 1970-01-01
	unixNano := t.Nanosecond()
	ntpSec := uint64(unixSec + 2208988800)
	frac := uint64((float64(unixNano) / 1e9) * (1 << 32))
	return (ntpSec << 32) | frac
}
func ntp64ToFloatSeconds(ntp uint64) float64 {
	seconds := float64(ntp >> 32)                  // integer part
	frac := float64(ntp&0xFFFFFFFF) / 4294967296.0 // fractional part
	return seconds + frac
}
func getNtpVersionInResponse(data []byte) uint8 {
	livmodeByte := uint8(data[0])
	version := uint8((livmodeByte >> 3) & 0x07)
	return version
}

func parseAccordingToRightVersion(data []byte, t1_uint uint64, t4_uint uint64, client_cookie uint64, draft string, debug_output *strings.Builder) (map[string]interface{}, error) {
	version := getNtpVersionInResponse(data)
	if version == uint8(1) {
		return parseNTPv1Response(data, t1_uint, t4_uint)
	} else if version == uint8(2) || version == uint8(3) {
		return parseNTPv3Response(data, t1_uint, t4_uint)
	} else if version == uint8(4) {
		return parseNTPv4Response(data, t1_uint, t4_uint, debug_output)
	} else if version == uint8(5) {
		return parseNTPv5Response(data, client_cookie, t1_uint, t4_uint, draft, debug_output)
	} else {
		//unknow version
		return map[string]interface{}{}, fmt.Errorf("unknow version: %v", version)
	}
}

func printJson(server string, data map[string]interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("Error converting to JSON. Showing raw data\n%v\n", data)
		return
	}
	fmt.Printf("NTPv5 response from %s:\n%s\n", server, string(jsonData))
}
func jsonToString(data map[string]interface{}, output *strings.Builder) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		output.WriteString(fmt.Sprintf("Error converting to JSON. Showing raw data\n%v\n", data))
		return
	}
	output.WriteString(string(jsonData))
}

func printHex4PerLine(data []byte, debug_output *strings.Builder) {
	wasNil := false
	if debug_output == nil {
		debug_output = new(strings.Builder)
		wasNil = true
	}
	for i := 0; i < len(data); i += 4 {
		end := i + 4
		if end > len(data) {
			end = len(data)
		}
		for _, b := range data[i:end] {
			debug_output.WriteString(fmt.Sprintf("%02X ", b))
		}
		debug_output.WriteString(fmt.Sprintf("\n"))
	}
	if wasNil {
		fmt.Printf(debug_output.String())
	}
}

//func jsonToStringWithError(data map[string]interface{}, output *strings.Builder) {
//	jsonData, err := json.MarshalIndent(data, "", "  ")
//	if err != nil {
//		output.WriteString(fmt.Sprintf("Error converting to JSON. Showing raw data\n%v\n", data))
//		return
//	}
//	output.WriteString(string(jsonData))
//}
