package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// NTPv4 constants
const (
	NTPV4_VERSION   = 4
	MODE_CLIENT     = 3
	NTP_PACKET_SIZE = 48
)

type NTPv4Header struct {
	LIVNMode       uint8 // LI (2) | VN (3) | Mode (3)
	Stratum        uint8
	Poll           int8
	Precision      int8
	RootDelay      uint32
	RootDispersion uint32
	RefID          uint32
	RefTimestamp   uint64
	OrigTimestamp  uint64
	RecvTimestamp  uint64
	TxTimestamp    uint64
}

func buildNTPv4Request() ([]byte, uint64) {
	req := make([]byte, NTP_PACKET_SIZE)

	// LI = 0 (no warning), VN = 4, Mode = 3 (client)
	req[0] = (0 << 6) | (NTPV4_VERSION << 3) | MODE_CLIENT

	t1 := nowToNtpUint64() //timeToNtp64(time.Now())
	binary.BigEndian.PutUint64(req[40:], t1)

	return req, t1
}

func parseNTPv4Response(data []byte, t1_uint uint64, t4_uint uint64, debug_output *strings.Builder) (map[string]interface{}, error) {
	if len(data) < NTP_PACKET_SIZE {
		return nil, fmt.Errorf("response too short: %d bytes", len(data))
	}

	debug_output.WriteString(fmt.Sprintf("received %v\n", len(data)))
	h := NTPv4Header{}
	buf := bytes.NewReader(data[:NTP_PACKET_SIZE])
	if err := binary.Read(buf, binary.BigEndian, &h); err != nil {
		return nil, err
	}

	t1 := ntp64ToFloatSeconds(t1_uint)
	t2 := ntp64ToFloatSeconds(h.RecvTimestamp)
	t3 := ntp64ToFloatSeconds(h.TxTimestamp)
	t4 := ntp64ToFloatSeconds(t4_uint)

	rtt := (t4 - t1) - (t3 - t2)
	offset := ((t2 - t1) + (t3 - t4)) / 2

	info := map[string]interface{}{ //same as NTPv3
		"leap":             (h.LIVNMode >> 6) & 0x03,
		"version":          (h.LIVNMode >> 3) & 0x07,
		"mode":             h.LIVNMode & 0x07,
		"stratum":          h.Stratum,
		"poll":             h.Poll,
		"precision":        h.Precision,
		"root_delay":       time32ToSeconds(h.RootDelay),
		"root_disp":        time32ToSeconds(h.RootDispersion),
		"ref_id":           h.RefID,
		"ref_timestamp":    h.RefTimestamp,
		"orig_timestamp":   h.OrigTimestamp, //t1
		"recv_timestamp":   h.RecvTimestamp, //t2
		"tx_timestamp":     h.TxTimestamp,   //t3
		"client_recv_time": t4_uint,         //we add this field to be shown in the results
		"rtt":              rtt,
		"offset":           offset,
	}
	if info["orig_timestamp"] == uint64(0) {
		info["anomaly"] = "timestamps are invalid, orig_timestamp (t1) is 0"
	} else if info["recv_timestamp"] == uint64(0) {
		info["anomaly"] = "timestamps are invalid, recv_timestamp (t2) is 0"
	} else if info["tx_timestamp"] == uint64(0) {
		info["anomaly"] = "timestamps are invalid, tx_timestamp (t3) is 0"
	}

	// Check if there are extension fields after the 48-byte header
	if len(data) > NTP_PACKET_SIZE {
		//fmt.Println("EXTRA FIELDDDDD")
		exts := []map[string]interface{}{}
		extData := data[NTP_PACKET_SIZE:]
		debug_output.WriteString(fmt.Sprintf("extension(s) detected: %v", extData))
		for len(extData) >= 4 {
			typ := binary.BigEndian.Uint16(extData[0:2])
			length := binary.BigEndian.Uint16(extData[2:4])
			if int(length) > len(extData) || length < 4 {
				break
			}
			payload := extData[4:length]
			exts = append(exts, map[string]interface{}{
				"type": typ,
				"data": payload,
			})
			extData = extData[length:]
		}
		info["extensions"] = exts
	}

	return info, nil
}

func performNTPv4Measurement(server string, timeout float64) (map[string]interface{}, string, int) {

	var output strings.Builder
	error_message := map[string]interface{}{}
	//addr := fmt.Sprintf("%s:%d", server, 123)
	addr := net.JoinHostPort(server, strconv.Itoa(123))

	conn, err := net.Dial("udp", addr)
	if err != nil {
		//fmt.Printf("error connecting: %v\n", err)
		m := fmt.Sprintf("error connecting: %v\n", err)
		output.WriteString(m)
		//os.Exit(1)
		error_message["error"] = m
		return error_message, output.String(), 1
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			return
		}
	}(conn)

	req, t1 := buildNTPv4Request()
	output.WriteString(fmt.Sprintf("Packet ntpv4 size sent: %d bytes\n", len(req)))
	_, err = conn.Write(req)
	if err != nil {
		m := fmt.Sprintf("could not send request: %v\n", err)
		output.WriteString(m)
		//os.Exit(2)
		error_message["error"] = m
		return error_message, output.String(), 2
	}

	err = conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	if err != nil {
		m := fmt.Sprintf("error reading bytes: %v\n", err)
		output.WriteString(m)
		error_message["error"] = m
		return error_message, output.String(), 3
	}
	resp := make([]byte, 1024)
	n, err := conn.Read(resp)
	if err != nil {
		m := fmt.Sprintf("measurement timeout: %v\n", err)
		output.WriteString(m)
		//os.Exit(3)
		error_message["error"] = m
		return error_message, output.String(), 3
	}

	t4_uint := nowToNtpUint64()

	//IMPORTANT. Check if the returned version is NTPv5, otherwise, parse according to the right NTP version
	result, err := parseAccordingToRightVersion(resp[:n], t1, t4_uint, 0, "", &output) //parseNTPv4Response(resp[:n], t1, t4, &output)

	if err != nil {
		m := fmt.Sprintf("error reading/parsing response: %v\n", err)
		output.WriteString(m)
		//os.Exit(4)
		error_message["error"] = m
		return error_message, output.String(), 4
	}
	return result, output.String(), 0
}
