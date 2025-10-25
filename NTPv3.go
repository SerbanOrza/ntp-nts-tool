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

type NTPv3Header struct { //same as NTPv4
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

func buildNTPv3or2Request(ntpVersion int) ([]byte, uint64) {
	req := make([]byte, NTP_PACKET_SIZE)

	// LI = 0 (no warning), VN = 2 or 3, Mode = 3 (client)
	req[0] = (0 << 6) | (3 << 3) | 3
	if ntpVersion == 2 {
		req[0] = (0 << 6) | (2 << 3) | 3
	}

	t1 := nowToNtpUint64() //timeToNtp64(time.Now())
	binary.BigEndian.PutUint64(req[40:], t1)

	return req, t1
}

func parseNTPv3Response(data []byte, t1_uint uint64, t4_uint uint64) (map[string]interface{}, error) {
	if len(data) < NTP_PACKET_SIZE {
		return nil, fmt.Errorf("response too short: %d bytes", len(data))
	}

	h := NTPv3Header{}
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

	info := map[string]interface{}{
		"leap":             (h.LIVNMode >> 6) & 0x03,
		"version":          (h.LIVNMode >> 3) & 0x07,
		"mode":             h.LIVNMode & 0x07,
		"stratum":          h.Stratum,
		"poll":             h.Poll,
		"precision":        h.Precision,
		"root_delay":       time32ToSeconds(h.RootDelay),
		"root_disp":        time32ToSeconds(h.RootDispersion),
		"ref_id":           h.RefID, //this may have a different meaning in NTPv4
		"ref_timestamp":    h.RefTimestamp,
		"orig_timestamp":   h.OrigTimestamp,
		"recv_timestamp":   h.RecvTimestamp,
		"tx_timestamp":     h.TxTimestamp,
		"client_recv_time": t4_uint, //we add this field to be shown in the results
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
	return info, nil
}

func performNTPv3Measurement(server string, timeout float64, ntpVersion int) (map[string]interface{}, string, int) {

	var output strings.Builder
	error_message := map[string]interface{}{}
	addr := net.JoinHostPort(server, strconv.Itoa(123))

	conn, err := net.Dial("udp", addr)
	if err != nil {
		m := fmt.Sprintf("error connecting: %v\n", err)
		output.WriteString(m)
		error_message["error"] = m
		return error_message, output.String(), 1
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			return
		}
	}(conn)

	req, t1 := buildNTPv3or2Request(ntpVersion)
	_, err = conn.Write(req)
	if err != nil {
		m := fmt.Sprintf("could not send data: %v\n", err)
		output.WriteString(m)
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
		error_message["error"] = m
		return error_message, output.String(), 3
	}

	t4_uint := nowToNtpUint64()

	// Get the server IP we actually measured
	remoteAddr := conn.RemoteAddr().(*net.UDPAddr)
	measuredIP := remoteAddr.IP.String()
	//IMPORTANT. Check if the returned version is NTPv5, otherwise, parse according to the right NTP version
	result, err := parseAccordingToRightVersion(resp[:n], t1, t4_uint, 0, "", &output) //parseNTPv3Response(resp[:n], t1, t4)

	if err != nil {
		m := fmt.Sprintf("error parsing response: %v\n", err)
		output.WriteString(m)
		error_message["error"] = m
		return error_message, output.String(), 4
	}

	if result != nil {
		result["Host"] = server
		result["Measured server IP"] = measuredIP
	}
	return result, output.String(), 0
}
