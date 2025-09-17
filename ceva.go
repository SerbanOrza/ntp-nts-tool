package main

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"time"
)

const (
	//NTPV5_VERSION = 5
	//HEADER_SIZE   = 48
	NTP_PORT = 123
)

// build an NTPv5 request; if draft != "" append Draft ID extension (type 0xF5FF)
func buildNTPv5RequestWithDraft(draft string) ([]byte, uint64) {
	clientCookie := rand.Uint64()
	buf := make([]byte, HEADER_SIZE)

	// Byte 0: LI(2) | VN(3) | Mode(3)  (client mode=3)
	buf[0] = (0 << 6) | (NTPV5_VERSION << 3) | 3
	// stratum/poll/precision/timescale/era/flags/rootdelay/rootdisp all zero for request
	// server cookie 0
	binary.BigEndian.PutUint64(buf[24:32], clientCookie) // client cookie
	// recv/tx timestamps = 0 for request
	// Append extension if requested
	if draft != "" {
		payload := []byte(draft) // MUST NOT be null-terminated
		extLen := 4 + len(payload)
		// pad to 4 bytes
		if rem := extLen % 4; rem != 0 {
			extLen += 4 - rem
		}
		ext := make([]byte, extLen)
		binary.BigEndian.PutUint16(ext[0:2], 0xF5FF)         // type
		binary.BigEndian.PutUint16(ext[2:4], uint16(extLen)) // length
		copy(ext[4:], payload)
		buf = append(buf, ext...)
	}
	return buf, clientCookie
}

func hexdump(b []byte) string {
	return hex.EncodeToString(b)
}

func parseV5HeaderAndExts(data []byte) (map[string]interface{}, error) {
	if len(data) < HEADER_SIZE {
		return nil, fmt.Errorf("response too short: %d", len(data))
	}
	m := make(map[string]interface{})
	m["raw_len"] = len(data)
	m["raw_hex"] = hexdump(data)

	// parse some fields via offsets (avoid struct alignment)
	vn := (data[0] >> 3) & 0x07
	mode := data[0] & 0x07
	li := (data[0] >> 6) & 0x03
	m["vn"] = vn
	m["mode"] = mode
	m["leap"] = li
	m["stratum"] = data[1]
	m["poll"] = int8(data[2])
	m["precision"] = int8(data[3])
	m["timescale"] = data[4]
	m["era"] = data[5]
	m["flags_raw"] = binary.BigEndian.Uint16(data[6:8])
	m["root_delay_raw"] = binary.BigEndian.Uint32(data[8:12])
	m["root_disp_raw"] = binary.BigEndian.Uint32(data[12:16])
	m["server_cookie"] = binary.BigEndian.Uint64(data[16:24])
	m["client_cookie"] = binary.BigEndian.Uint64(data[24:32])
	m["recv_ts"] = binary.BigEndian.Uint64(data[32:40])
	m["tx_ts"] = binary.BigEndian.Uint64(data[40:48])

	// extension parsing
	exts := []map[string]interface{}{}
	if len(data) > HEADER_SIZE {
		extdata := data[HEADER_SIZE:]
		for len(extdata) >= 4 {
			if len(extdata) < 4 {
				break
			}
			typ := binary.BigEndian.Uint16(extdata[0:2])
			length := binary.BigEndian.Uint16(extdata[2:4])
			if length < 4 || int(length) > len(extdata) {
				// malformed or truncated extension — bail
				break
			}
			payload := extdata[4:length]
			e := map[string]interface{}{
				"type":    fmt.Sprintf("0x%04x", typ),
				"length":  int(length),
				"payload": fmt.Sprintf("%x", payload),
			}
			// if draft id
			if typ == 0xF5FF {
				e["draft_str"] = string(payload)
			}
			exts = append(exts, e)
			extdata = extdata[length:]
		}
	}
	m["extensions"] = exts
	return m, nil
}

func tryOne(server string, draft string, timeout time.Duration) (map[string]interface{}, error) {
	req, clientCookie := buildNTPv5RequestWithDraft(draft)

	addr := net.JoinHostPort(server, fmt.Sprintf("%d", NTP_PORT))
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial error: %v", err)
	}
	defer conn.Close()

	// record local send time (NTP64) for debugging if needed
	sendTime := time.Now()
	_, err = conn.Write(req)
	if err != nil {
		return nil, fmt.Errorf("write error: %v", err)
	}
	conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, 1500)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, err // likely timeout or no route
	}
	recvTime := time.Now()
	resp := buf[:n]

	parsed, err := parseV5HeaderAndExts(resp)
	if err != nil {
		return nil, err
	}
	parsed["client_cookie_sent"] = clientCookie
	parsed["client_sent_time_local"] = sendTime.String()
	parsed["client_recv_time_local"] = recvTime.String()
	return parsed, nil
}

func mainn() {
	//if len(os.Args) < 2 {
	//	fmt.Printf("usage: %s <host>\n", os.Args[0])
	//	os.Exit(1)
	//}
	//server := os.Args[1]
	server := "ntpd-rs.sidnlabs.nl"

	// try variants: 1) no draft ext 2) common drafts (add or remove as you like)
	drafts := []string{
		"draft-ietf-ntp-ntpv5-05",
		"", // try no draft extension first
		"draft-ietf-ntp-ntpv5-06",
		"draft-ietf-ntp-ntpv5-04",
	}

	timeout := 3 * time.Second
	for _, d := range drafts {
		fmt.Printf("=== Trying draft: '%s' ===\n", d)
		info, err := tryOne(server, d, timeout)
		if err != nil {
			fmt.Printf("No reply (or error): %v\n\n", err)
			time.Sleep(250 * time.Millisecond)
			continue
		}
		fmt.Printf("Reply! Parsed info:\n")
		for k, v := range info {
			fmt.Printf("  %s: %v\n", k, v)
		}
		fmt.Printf("\n")
		// stop once we got something
		//break
	}

	fmt.Println("done")
}
