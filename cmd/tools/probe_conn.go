package main

import (
	"fmt"
	"net"
	"time"

	ber "github.com/go-asn1-ber/asn1-ber"
)

func main() {
	targets := []struct{host string; port int}{
		{"lx2.loc.gov", 210},
		{"hollis.harvard.edu", 210},
	}

	// Test Cases
	versions := map[string][]byte{
		"v3 (0x20)": {0x00, 0x20},
		"v2 (0x40)": {0x00, 0x40},
	}
	
	options := map[string][]byte{
		"S|P (0xC0)": {0x00, 0xC0},
	}

	for _, t := range targets {
		fmt.Printf("--- Target: %s ---\n", t.host)
		for vName, vBytes := range versions {
			for oName, oBytes := range options {
				fmt.Printf("Testing Ver=%s, Opt=%s... ", vName, oName)
				if test(t.host, t.port, vBytes, oBytes) {
					fmt.Println("SUCCESS!")
				} else {
					fmt.Println("FAIL")
				}
			}
		}
	}
}

func test(host string, port int, ver []byte, opt []byte) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 3*time.Second)
	if err != nil {
		fmt.Printf("DialErr: %v ", err)
		return false
	}
	defer conn.Close()

	pdu := ber.Encode(ber.ClassContext, ber.TypeConstructed, 20, nil, "InitializeRequest")
	
	// ProtocolVersion
	pv := ber.Encode(ber.ClassContext, ber.TypePrimitive, 3, nil, "ProtocolVersion")
	pv.Data.Write(ver)
	pdu.AppendChild(pv)

	// Options
	opts := ber.Encode(ber.ClassContext, ber.TypePrimitive, 4, nil, "Options")
	opts.Data.Write(opt)
	pdu.AppendChild(opts)

	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 5, 65536, "PreferredMessageSize"))
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 6, 65536, "MaximumRecordSize"))
	pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 110, "GoZClient", "Id"))
	pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 111, "GoZClient", "Name"))

	conn.Write(pdu.Bytes())
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	
	resp, err := ber.ReadPacket(conn)
	if err != nil {
		// fmt.Printf("ReadErr: %v ", err)
		return false
	}
	
	if resp.Tag == 21 {
		return true
	}
	return false
}
