package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"time"

	ber "github.com/go-asn1-ber/asn1-ber"
)

func main() {
	host := "lx2.loc.gov" // Default to LCDB (usually stable)
	port := 210
	if len(os.Args) > 1 {
		host = os.Args[1]
	}

	fmt.Printf("Connecting to %s:%d...\n", host, port)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		fmt.Printf("Dial failed: %v\n", err)
		return
	}
	defer conn.Close()

	// Build Init PDU
	pdu := ber.Encode(ber.ClassContext, ber.TypeConstructed, 20, nil, "InitializeRequest")
	
			// ProtocolVersion [3] IMPLICIT BIT STRING
			// v3 (bit 2) set. 0010 0000 -> 0x20
			pv := ber.Encode(ber.ClassContext, ber.TypePrimitive, 3, nil, "ProtocolVersion")
			pv.Data.Write([]byte{0x00, 0x20}) // Unused bits 0, Value 0x20
			pdu.AppendChild(pv)		
		// Options [4] IMPLICIT BIT STRING
		// search(0), present(1) -> 1100 0000 -> 0xC0
		opts := ber.Encode(ber.ClassContext, ber.TypePrimitive, 4, nil, "Options")
		opts.Data.Write([]byte{0x00, 0xC0})
		pdu.AppendChild(opts)	
	// PreferredMessageSize [5] IMPLICIT INTEGER
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 5, 1048576, "PreferredMessageSize"))
	
	// MaximumRecordSize [6] IMPLICIT INTEGER
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 6, 1048576, "MaximumRecordSize"))
	
	// ImplementationId [110] IMPLICIT InternationalString
	pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 110, "GoZClient", "Id"))
	
	// ImplementationName [111] IMPLICIT InternationalString
	pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 111, "GoZClient", "Name"))

	fmt.Println("Sending Init PDU:")
	// Print hex
	bytes := pdu.Bytes()
	fmt.Printf("%s\n", hex.Dump(bytes))

	conn.Write(bytes)

	// Read response
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	// Use ber.ReadPacket to see if we get anything
	resp, err := ber.ReadPacket(conn)
	if err != nil {
		fmt.Printf("Read failed: %v\n", err)
		return
	}

	fmt.Printf("Response Tag: %d (Class: %d)\n", resp.Tag, resp.ClassType)
	if resp.Tag == 21 {
		fmt.Println("Init Accepted!")
	} else {
		fmt.Println("Init Rejected or Unknown response.")
	}
}
