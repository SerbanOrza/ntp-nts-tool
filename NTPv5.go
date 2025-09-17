package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"strings"

	//"os"
	"strconv"
	"time"
)

const (
	NTPV5_VERSION = 5
	TIMESCALE_UTC = 0
	HEADER_SIZE   = 48
)

type NTPv5Header struct {
	LIVNMode       uint8 // LI(2) | VN(3) | Mode(3)
	Stratum        uint8
	Poll           int8
	Precision      int8
	Timescale      uint8
	Era            uint8
	Flags          uint16
	RootDelay      uint32
	RootDispersion uint32
	ServerCookie   uint64
	ClientCookie   uint64
	RecvTimestamp  uint64
	TxTimestamp    uint64
}
type NTPv5Header_draft_05 struct {
	LIVNMode       uint8 // LI(2 bits) | VN(3 bits) | Mode(3 bits)
	Stratum        uint8
	Poll           int8
	Precision      int8
	Timescale      uint8
	Era            uint8
	Flags          uint16
	RootDelay      uint32
	RootDispersion uint32
	ServerCookie   uint64
	ClientCookie   uint64
	RecvTimestamp  uint64
	TxTimestamp    uint64
}

type NTPv5Header_draft_06 struct {
	LIVNMode       uint8 // LI(2 bits) | VN(3 bits) | Mode(3 bits)
	Stratum        uint8
	Poll           int8
	Precision      int8
	RootDelay      uint32 //you can see it has a different place than in draft_05
	RootDispersion uint32 //same
	Timescale      uint8
	Era            uint8
	Flags          uint16
	ServerCookie   uint64
	ClientCookie   uint64
	RecvTimestamp  uint64
	TxTimestamp    uint64
}

func decodeFlags(flags uint16) map[string]bool {
	return map[string]bool{
		"synchronized": flags&0x1 != 0,
		"interleaved":  flags&0x2 != 0,
		"auth_nak":     flags&0x4 != 0,
	}
}

//	func buildDraftIdentificationExtension(draftName string) []byte {
//		payload := []byte(draftName)
//		payloadLen := len(payload)
//		totalLen := 4 + payloadLen
//		if pad := totalLen % 4; pad != 0 {
//			totalLen += 4 - pad
//		}
//		ext := make([]byte, totalLen)
//		binary.BigEndian.PutUint16(ext[0:2], 0xF5FF)
//		binary.BigEndian.PutUint16(ext[2:4], uint16(totalLen))
//		copy(ext[4:], payload)
//		// Remaining padding bytes are already zero-initialized by make()
//		return ext
//	}
//
//	func buildNTPv5Request3() ([]byte, uint64) {
//		clientCookie := rand.Uint64()
//		fmt.Print(len("draft-ietf-ntp-ntpv5-05"))
//		buf := make([]byte, HEADER_SIZE)
//
//		// LI|VN|Mode
//		buf[0] = (0 << 6) | (NTPV5_VERSION << 3) | 3
//		// rest already zero-initialized, just fill client cookie
//		binary.BigEndian.PutUint64(buf[24:32], clientCookie)
//
//		draftField := buildDraftIdentificationExtension("draft-ietf-ntp-ntpv5-05")
//		buf = append(buf, draftField...)
//		return buf, clientCookie
//	}
func buildNTPv5Request(draft string, debug_output *strings.Builder) ([]byte, uint64) {
	clientCookie := rand.Uint64()
	buf := make([]byte, 48)

	// Byte 0: LI(2 bits) | VN(3 bits) | Mode(3 bits)
	buf[0] = (0 << 6) | (NTPV5_VERSION << 3) | 3

	// Byte 1: Stratum = 0
	buf[1] = 0
	// Byte 2: Poll = 0
	buf[2] = 0
	// Byte 3: Precision = 0
	buf[3] = 0
	// Byte 4: Timescale (UTC=0)
	buf[4] = 0
	// Byte 5: Era
	buf[5] = 0
	// Bytes 6-7: Flags (uint16)
	binary.BigEndian.PutUint16(buf[6:8], 0)
	// Bytes 8-11: Root Delay (uint32)
	binary.BigEndian.PutUint32(buf[8:12], 0)
	// Bytes 12-15: Root Dispersion (uint32)
	binary.BigEndian.PutUint32(buf[12:16], 0)

	// Bytes 16-23: Server Cookie (uint64)
	binary.BigEndian.PutUint64(buf[16:24], 0)
	// Bytes 24-31: Client Cookie (uint64)
	binary.BigEndian.PutUint64(buf[24:32], clientCookie)

	// Bytes 32-39: Recv Timestamp (uint64)
	binary.BigEndian.PutUint64(buf[32:40], 0)
	// Bytes 40-47: Tx Timestamp (uint64)
	binary.BigEndian.PutUint64(buf[40:48], 0)
	//draft := "draft-ietf-ntp-ntpv5-05"
	if draft != "" {
		payload := []byte(draft)
		//fmt.Printf("len draft name %v\n", len(payload))
		//extLen := 1 + len(payload) // type + length + body
		//len of draft is 23
		//total length for this ext field: 2+2+23+1=27 but we need 28 to be multiple of 4
		ext := make([]byte, 28)
		binary.BigEndian.PutUint16(ext[0:2], 0xF5FF) // type
		binary.BigEndian.PutUint16(ext[2:4], 28)     //I tried with 27, but the server would not respond
		copy(ext[4:], payload)
		debug_output.WriteString(fmt.Sprintf("len draft (sent) ext field: %v, content: %v\n", len(ext), ext))
		buf = append(buf, ext...)

	}
	//draftField := buildDraftIdentificationExtension("draft-ietf-ntp-ntpv5-05")
	//buf = append(buf, draftField...)
	return buf, clientCookie
}

func parseNTPv5Response(data []byte, clientCookie uint64, clientSentTime float64, draft string, debug_output *strings.Builder) (map[string]interface{}, error) {
	//data received
	debug_output.WriteString(fmt.Sprintf("received response: %v bytes\n", len(data)))
	printHex4PerLine(data, debug_output)

	if len(data) < HEADER_SIZE {
		return nil, fmt.Errorf("response too short")
	}
	if len(data) > HEADER_SIZE {
		tail := data[HEADER_SIZE:]
		debug_output.WriteString(fmt.Sprintf("Extension part (%d bytes): % X\n", len(tail), tail))
	} //11 101 100   0000 0011

	//header := NTPv5Header_draft_05{}
	buf := bytes.NewReader(data[:HEADER_SIZE])
	info := map[string]interface{}{}
	t2, t3 := -1.0, -1.0
	//in draft 06 the order of fields changed
	if draft == "draft-ietf-ntp-ntpv5-06" {
		header := NTPv5Header_draft_06{}
		// here it is important what header format we use
		if err := binary.Read(buf, binary.BigEndian, &header); err != nil {
			return nil, err
		}
		// the order for info is not important as it is a map.
		info = map[string]interface{}{
			"leap":                (header.LIVNMode >> 6) & 0x03,
			"version":             (header.LIVNMode >> 3) & 0x07,
			"mode":                header.LIVNMode & 0x07,
			"stratum":             header.Stratum,
			"poll":                header.Poll,
			"precision":           header.Precision,
			"root_delay":          time32ToSeconds(header.RootDelay),      //in seconds
			"root_disp":           time32ToSeconds(header.RootDispersion), //in seconds
			"timescale":           header.Timescale,
			"era":                 header.Era,
			"flags_raw":           header.Flags,
			"flags_decoded":       decodeFlags(header.Flags),
			"server_cookie":       header.ServerCookie,
			"client_cookie":       header.ClientCookie,
			"recv_timestamp":      header.RecvTimestamp,
			"tx_timestamp":        header.TxTimestamp,
			"client_cookie_valid": header.ClientCookie == clientCookie,
		}
		t2 = ntp64ToFloatSeconds(header.RecvTimestamp)
		t3 = ntp64ToFloatSeconds(header.TxTimestamp)
	} else {
		header := NTPv5Header_draft_05{}
		// here it is important what header format we use
		if err := binary.Read(buf, binary.BigEndian, &header); err != nil {
			return nil, err
		}
		info = map[string]interface{}{
			"leap":                (header.LIVNMode >> 6) & 0x03,
			"version":             (header.LIVNMode >> 3) & 0x07,
			"mode":                header.LIVNMode & 0x07,
			"stratum":             header.Stratum,
			"poll":                header.Poll,
			"precision":           header.Precision,
			"timescale":           header.Timescale,
			"era":                 header.Era,
			"flags_raw":           header.Flags,
			"flags_decoded":       decodeFlags(header.Flags),
			"root_delay":          time32ToSeconds(header.RootDelay),      //in seconds
			"root_disp":           time32ToSeconds(header.RootDispersion), //in seconds
			"server_cookie":       header.ServerCookie,
			"client_cookie":       header.ClientCookie,
			"recv_timestamp":      header.RecvTimestamp,
			"tx_timestamp":        header.TxTimestamp,
			"client_cookie_valid": header.ClientCookie == clientCookie,
		}
		t2 = ntp64ToFloatSeconds(header.RecvTimestamp)
		t3 = ntp64ToFloatSeconds(header.TxTimestamp)
	}

	t4_uint := nowToNtpUint64()
	t4 := ntp64ToFloatSeconds(t4_uint)
	info["client_recv_time"] = t4_uint

	if draft != "" {
		info["draft"] = draft
	} else {
		info["draft"] = "did not use an extension field for draft"
	}

	// Parse extension fields (if any)
	if len(data) > HEADER_SIZE {
		exts := []map[string]interface{}{}
		extData := data[HEADER_SIZE:]
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
	//add offset and rtt
	t1 := clientSentTime
	//t2 := ntp64ToFloatSeconds(header.RecvTimestamp)
	//t3 := ntp64ToFloatSeconds(header.TxTimestamp)

	info["offset_s"] = ((t2 - t1) + (t3 - t4)) / 2 //in seconds
	info["rtt_s"] = (t4 - t1) - (t3 - t2)          //in seconds
	return info, nil
}

// This method tries to perform an NTPv5 measurement. It supports draft options
// In case of a success measurement, the result can be seen in the "map[string]interface{}". Otherwise, if it failed, then
// you can see in "map[string]interface{}" exactly the error, in the second value you see debug messages and the error, and
// third value has the error code. This is done such that in case you do not want debug messages, you can get exactly the output
// or the error message. (only them will be printed on screen)
func performNTPv5Measurement(server string, timeout float64, draft string) (map[string]interface{}, string, int) {

	var output strings.Builder
	error_message := map[string]interface{}{}
	//addr := fmt.Sprintf("%s:%d", server, NTP_PORT)
	addr := net.JoinHostPort(server, strconv.Itoa(123))

	conn, err := net.DialTimeout("udp", addr, time.Duration(timeout)*time.Second)
	if err != nil {
		m := fmt.Sprintf("error connecting: %v\n", err)
		output.WriteString(m)
		//fmt.Printf("error connecting: %v\n", err)
		//os.Exit(2)
		error_message["error"] = m
		return error_message, output.String(), 2
	}
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			return
		}
	}(conn)
	output.WriteString(fmt.Sprintf("connected to %v\n", addr))

	t1 := ntp64ToFloatSeconds(nowToNtpUint64())
	req, client_cookie := buildNTPv5Request(draft, &output)
	output.WriteString(fmt.Sprintf("Packet ntpv5 size sent: %d bytes\n", len(req)))
	_, err = conn.Write(req)
	if err != nil {
		m := fmt.Sprintf("error sending ntpv5 request: %v\n", err)
		output.WriteString(m)
		//fmt.Printf("error sending ntpv5 request: %v\n", err)
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
		//fmt.Printf("measurement timeout: %v\n", err)
		//os.Exit(3)
		error_message["error"] = m
		return error_message, output.String(), 3
	}

	result, err := parseNTPv5Response(resp[:n], client_cookie, t1, draft, &output)
	if err != nil {
		m := fmt.Sprintf("error parsing response: %v\n", err)
		output.WriteString(m)
		//fmt.Printf("error parsing response: %v\n", err)
		//os.Exit(4)
		error_message["error"] = m
		return error_message, output.String(), 4
	}
	return result, output.String(), 0
	//jsonToString(result, &output)
	//fmt.Print(output.String())
	//os.Exit(0)
}
