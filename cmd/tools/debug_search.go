package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"time"

	ber "github.com/go-asn1-ber/asn1-ber"
)

func main() {
	host := "lx2.loc.gov"
	port := 210

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// 1. Init
	pdu := ber.Encode(ber.ClassContext, ber.TypeConstructed, 20, nil, "InitializeRequest")
	
	ver := ber.Encode(ber.ClassContext, ber.TypePrimitive, 3, nil, "ProtocolVersion")
	ver.Data.Write([]byte{0x00, 0x20})
	pdu.AppendChild(ver)

	opts := ber.Encode(ber.ClassContext, ber.TypePrimitive, 4, nil, "Options")
	opts.Data.Write([]byte{0x00, 0xC0})
	pdu.AppendChild(opts)

	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 5, 65536, "PreferredMessageSize"))
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 6, 65536, "MaximumRecordSize"))
	pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 110, "GoZClient", "Id"))
	pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 111, "GoZClient", "Name"))

	conn.Write(pdu.Bytes())
	ber.ReadPacket(conn) // Skip Init Response

	// 2. Search
	search := ber.Encode(ber.ClassContext, ber.TypeConstructed, 22, nil, "SearchRequest")
	search.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 0, 0, "SmallSetUpperBound"))
	search.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 1, 1, "LargeSetLowerBound"))
	search.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 2, 0, "MediumSetPresentNumber"))
	search.AppendChild(ber.NewBoolean(ber.ClassContext, ber.TypePrimitive, 3, true, "ReplaceIndicator"))
	search.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 4, "default", "ResultSetName"))

	dbs := ber.Encode(ber.ClassContext, ber.TypeConstructed, 5, nil, "DatabaseNames")
	dbs.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagVisibleString, "LCDB", "DatabaseName"))
	search.AppendChild(dbs)

	query := ber.Encode(ber.ClassContext, ber.TypeConstructed, 1, nil, "Query")
	rpn := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "RPNQuery")
	rpn.AppendChild(ber.Encode(ber.ClassUniversal, ber.TypePrimitive, ber.TagObjectIdentifier, []byte{0x2A, 0x86, 0x48, 0x86, 0xF7, 0x12, 0x03, 0x03, 0x01}, "Bib1"))

	// RPN Structure: Term "Go" (General)
	op := ber.Encode(ber.ClassContext, ber.TypeConstructed, 0, nil, "Op")
	apt := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "APT")
	// Attributes
	attrs := ber.Encode(ber.ClassContext, ber.TypeConstructed, 44, nil, "Attrs")
	attr := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "Attr")
	attr.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, 1, "Type"))
	attr.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, 1016, "Value")) // Any
	attrs.AppendChild(attr)
	apt.AppendChild(attrs)
	// Term
	apt.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 45, "Go", "Term"))
	op.AppendChild(apt)
	rpn.AppendChild(op)

	query.AppendChild(rpn)
	search.AppendChild(query)

	fmt.Println("Sending Search PDU:")
	fmt.Printf("%s\n", hex.Dump(search.Bytes()))
	conn.Write(search.Bytes())

	resp, err := ber.ReadPacket(conn)
	if err != nil {
		fmt.Println("Search Read Failed:", err)
		return
	}
	fmt.Printf("Search Response Tag: %d\n", resp.Tag)
	if resp.Tag == 23 {
		// Count is child 1?
		if len(resp.Children) > 1 {
			fmt.Printf("Search Status: %v\n", resp.Children[0].Value)
			fmt.Printf("Result Count: %v\n", resp.Children[1].Value)
		}
	}
}
