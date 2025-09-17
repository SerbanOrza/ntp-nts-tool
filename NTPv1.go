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

type NTPv1Header struct {
	LIStatus      uint8  // Leap Indicator (2 bits) + Status (6 bits)
	Type          uint8  // Request, response, etc.
	Precision     uint16 // 16-bit precision
	EstError      uint32 // Estimated error
	EstDriftRate  uint32 // Estimated drift rate
	RefID         uint32 // Reference clock ID
	RefTimestamp  uint64
	OrigTimestamp uint64
	RecvTimestamp uint64
	TxTimestamp   uint64
}

func buildNTPv1Request() ([]byte, float64) {
	req := make([]byte, 48)

	// LI=0, Status=0 -> LIStatus = 0
	// Type=1 (client request)
	binary.BigEndian.PutUint16(req[0:2], uint16(0<<8|1))

	// Precision = 0
	binary.BigEndian.PutUint16(req[2:4], 0)

	// Estimated error, drift, refID = 0
	// Timestamps mostly zero except TxTimestamp

	t1 := nowToNtpUint64()
	binary.BigEndian.PutUint64(req[40:], t1)

	return req, ntp64ToFloatSeconds(t1)
}
func parseNTPv1Response(data []byte, t1, t4 float64) (map[string]interface{}, error) {
	if len(data) < 48 {
		return nil, fmt.Errorf("response too short: %d bytes", len(data))
	}

	h := NTPv1Header{}
	buf := bytes.NewReader(data[:48])
	if err := binary.Read(buf, binary.BigEndian, &h); err != nil {
		return nil, err
	}

	t2 := ntp64ToFloatSeconds(h.RecvTimestamp)
	t3 := ntp64ToFloatSeconds(h.TxTimestamp)

	rtt := (t4 - t1) - (t3 - t2)
	offset := ((t2 - t1) + (t3 - t4)) / 2

	info := map[string]interface{}{
		"li_status":      h.LIStatus,
		"type":           h.Type,
		"precision":      h.Precision,
		"est_error":      h.EstError,
		"est_drift_rate": h.EstDriftRate,
		"ref_id":         h.RefID,
		"ref_timestamp":  h.RefTimestamp,
		"orig_timestamp": h.OrigTimestamp,
		"recv_timestamp": h.RecvTimestamp,
		"tx_timestamp":   h.TxTimestamp,
		"rtt":            rtt,
		"offset":         offset,
	}

	return info, nil
}

func performNTPv1Measurement(server string, timeout float64) (map[string]interface{}, string, int) {

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

	req, t1 := buildNTPv1Request()
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

	t4 := ntp64ToFloatSeconds(nowToNtpUint64())
	result, err := parseNTPv1Response(resp[:n], t1, t4)

	if err != nil {
		m := fmt.Sprintf("error parsing response: %v\n", err)
		output.WriteString(m)
		error_message["error"] = m
		return error_message, output.String(), 4
	}
	return result, output.String(), 0
}
