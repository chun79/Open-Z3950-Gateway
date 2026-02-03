package z3950

import (
	"fmt"
	"log/slog"
	"net"
	"time"

	ber "github.com/go-asn1-ber/asn1-ber"
)

const (
	OID_Bib1    = "1.2.840.10003.3.1"
	OID_MARC21  = "1.2.840.10003.5.10" // MARC 21 (USMARC)
	OID_UNIMARC = "1.2.840.10003.5.1"  // UNIMARC
	OID_SUTRS   = "1.2.840.10003.5.101" // Simple Unstructured Text
)

type Client struct {
	conn net.Conn
	host string
	port int
}

func NewClient(host string, port int) *Client {
	return &Client{host: host, port: port}
}

func (c *Client) Connect() error {
	address := fmt.Sprintf("%s:%d", c.host, c.port)
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Client) Close() {
	if c.conn != nil {
		pdu := ber.Encode(ber.ClassContext, ber.TypeConstructed, 48, nil, "Close")
		pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 211, 0, "Reason"))
		c.conn.Write(pdu.Bytes())
		c.conn.Close()
	}
}

func (c *Client) sendPDU(pdu *ber.Packet) (*ber.Packet, error) {
	data := pdu.Bytes()
	slog.Info("sending PDU", "hex", fmt.Sprintf("%X", data))
	if _, err := c.conn.Write(data); err != nil {
		return nil, err
	}
	pkt, err := ber.ReadPacket(c.conn)
	if err != nil {
		return nil, err
	}
	
	if pkt.Tag == 48 {
		reason := "unknown"
		slog.Info("Close PDU received", "children", len(pkt.Children))
		for i, c := range pkt.Children {
			val := c.Value
			if v, ok := c.Value.(int64); ok { val = v }
			slog.Info("Close Child", "index", i, "tag", c.Tag, "value", val, "data_hex", fmt.Sprintf("%X", c.Data.Bytes()))
		}
		
		if len(pkt.Children) > 0 && pkt.Children[0].Tag == 211 {
			if v, ok := pkt.Children[0].Value.(int64); ok {
				reason = fmt.Sprintf("code %d", v)
			} else {
				v := decodeInt(pkt.Children[0])
				reason = fmt.Sprintf("code %d", v)
			}
		}
		return nil, fmt.Errorf("server closed connection: %s", reason)
	}
	
	return pkt, nil
}

func (c *Client) Init() error {
	pdu := ber.Encode(ber.ClassContext, ber.TypeConstructed, 20, nil, "InitializeRequest")
	
	// ProtocolVersion [3] IMPLICIT BIT STRING
	// v3 (bit 2) set. 0010 0000 = 0x20.
	ver := ber.Encode(ber.ClassContext, ber.TypePrimitive, 3, nil, "ProtocolVersion")
	ver.Data.Write([]byte{0x00, 0x20}) 
	pdu.AppendChild(ver)

	// Options [4] IMPLICIT BIT STRING
	// search(0)|present(1) = 1100 0000 = 0xC0
	opts := ber.Encode(ber.ClassContext, ber.TypePrimitive, 4, nil, "Options")
	opts.Data.Write([]byte{0x00, 0xC0})
	pdu.AppendChild(opts)

	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 5, 65536, "PreferredMessageSize"))
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 6, 65536, "MaximumRecordSize"))
	
	// Optional fields removed for compatibility (yaz-client imitation)
	// pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 110, "yaz-client", "Id"))
	// pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 111, "YAZ", "Name"))
	// pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 112, "5.34.0", "Ver"))

	resp, err := c.sendPDU(pdu)
	if err != nil {
		return err
	}
	if resp.Tag != 21 {
		return fmt.Errorf("unexpected response tag: %d", resp.Tag)
	}

	// Check result
	accepted := false
	for _, child := range resp.Children {
		// Result is Tag 12 (IMPLICIT BOOLEAN) in standard, but some impls might use Tag 1 (Universal)
		if child.Tag == 12 || child.Tag == 1 {
			if v, ok := child.Value.(bool); ok {
				accepted = v
			} else {
				// Manual decode: non-zero byte = true
				if len(child.Data.Bytes()) > 0 && child.Data.Bytes()[0] != 0 {
					accepted = true
				}
			}
		}
	}

	if !accepted {
		return fmt.Errorf("server rejected connection (Init=False)")
	}
	return nil
}

// decodeInt manually decodes a BER integer from a packet's data
func decodeInt(p *ber.Packet) int64 {
	if v, ok := p.Value.(int64); ok {
		return v
	}
	data := p.Data.Bytes()
	if len(data) == 0 {
		return 0
	}
	var val int64
	for _, b := range data {
		val = (val << 8) | int64(b)
	}
	return val
}

// buildOperand creates a BER packet for a single search clause
func buildOperand(clause QueryClause) *ber.Packet {
	op := ber.Encode(ber.ClassContext, ber.TypeConstructed, 0, nil, "Operand")
	apt := ber.Encode(ber.ClassContext, ber.TypeConstructed, 102, nil, "AttributesPlusTerm")

	attrs := ber.Encode(ber.ClassContext, ber.TypeConstructed, 44, nil, "Attrs")
	attr := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "Attr")
	attr.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 120, 1, "Type"))
	attr.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 121, int64(clause.Attribute), "Value"))
	attrs.AppendChild(attr)
	apt.AppendChild(attrs)

	term := ber.NewString(ber.ClassContext, ber.TypePrimitive, 45, clause.Term, "Term")
	apt.AppendChild(term)

	op.AppendChild(apt)
	return op
}

func buildRPN(node QueryNode) *ber.Packet {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case QueryClause:
		return buildOperand(n)
	case QueryComplex:
		complex := ber.Encode(ber.ClassContext, ber.TypeConstructed, 1, nil, "Complex")
		complex.AppendChild(buildRPN(n.Left))
		complex.AppendChild(buildRPN(n.Right))
		
		opVal := 0 // AND
		switch n.Operator {
		case "OR": opVal = 1
		case "AND-NOT": opVal = 2
		}
		
		op := ber.Encode(ber.ClassContext, ber.TypeConstructed, 46, nil, "Operator")
		op.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, ber.Tag(opVal), 0, "OpCode"))
		complex.AppendChild(op)
		
		return complex
	}
	return nil
}

func (c *Client) StructuredSearch(dbName string, query StructuredQuery) (int, error) {
	pdu := ber.Encode(ber.ClassContext, ber.TypeConstructed, 22, nil, "SearchRequest")
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 13, 1, "SmallSetUpperBound"))
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 14, 1, "LargeSetLowerBound"))
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 15, 0, "MediumSetPresentNumber"))
	pdu.AppendChild(ber.NewBoolean(ber.ClassContext, ber.TypePrimitive, 16, true, "ReplaceIndicator"))
	pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 17, "default", "ResultSetName"))

	dbs := ber.Encode(ber.ClassContext, ber.TypeConstructed, 18, nil, "DatabaseNames")
	dbs.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 105, dbName, "DatabaseName"))
	pdu.AppendChild(dbs)

	// Add Small/Medium Set Element Set Names to hint format "B" (Brief) or "F" (Full)
	// This might help servers default to a known format without needing RecordComposition in Present
	// pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 100, "B", "SmallSetElementSetNames"))
	// pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 101, "B", "MediumSetElementSetNames"))

	searchQuery := ber.Encode(ber.ClassContext, ber.TypeConstructed, 21, nil, "SearchQuery")
	rpnQuery := ber.Encode(ber.ClassContext, ber.TypeConstructed, 1, nil, "RPNQuery")
	
	attrSetId := ber.Encode(ber.ClassUniversal, ber.TypePrimitive, ber.TagObjectIdentifier, nil, "AttributeSetId")
	attrSetId.Data.Write([]byte{0x2A, 0x86, 0x48, 0xCE, 0x13, 0x03, 0x01})
	rpnQuery.AppendChild(attrSetId)

	struct_ := buildRPN(query.Root)
	if struct_ == nil {
		struct_ = buildOperand(QueryClause{Attribute: UseAttributeAny, Term: " "})
	}
	rpnQuery.AppendChild(struct_)
	
	searchQuery.AppendChild(rpnQuery)
	pdu.AppendChild(searchQuery)

	resp, err := c.sendPDU(pdu)
	if err != nil {
		return 0, err
	}
	if resp.Tag != 23 {
		return 0, fmt.Errorf("bad tag: %d", resp.Tag)
	}

	for _, child := range resp.Children {
		if child.Tag == 23 {
			return int(decodeInt(child)), nil
		}
	}
	return 0, nil
}


func (c *Client) Search(dbName string, simpleTerm string) (int, error) {
	query := StructuredQuery{
		Root: QueryClause{Attribute: UseAttributeAny, Term: simpleTerm},
	}
	return c.StructuredSearch(dbName, query)
}

func (c *Client) Present(start int, count int, syntaxOID string) ([]*MARCRecord, error) {

	pdu := ber.Encode(ber.ClassContext, ber.TypeConstructed, 24, nil, "PresentRequest")

	pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 31, "default", "ResultSetId"))

	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 30, int64(start), "ResultSetStartPoint"))

	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 29, int64(count), "NumberOfRecordsRequested"))



	if syntaxOID != "" {

		var oidBytes []byte

		if syntaxOID == OID_MARC21 {

			oidBytes = []byte{0x2A, 0x86, 0x48, 0xCE, 0x13, 0x05, 0x0A}

		} else if syntaxOID == OID_SUTRS {

			oidBytes = []byte{0x2A, 0x86, 0x48, 0xCE, 0x13, 0x05, 0x65}

		} else if syntaxOID == OID_UNIMARC {

			oidBytes = []byte{0x2A, 0x86, 0x48, 0xCE, 0x13, 0x05, 0x01}

		} else {

			// Default to MARC21

			oidBytes = []byte{0x2A, 0x86, 0x48, 0xCE, 0x13, 0x05, 0x0A}

		}

		pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 104, string(oidBytes), "PreferredRecordSyntax"))

	}



	resp, err := c.sendPDU(pdu)

	if err != nil {

		return nil, err

	}



		if resp.Tag != 25 {



			return nil, fmt.Errorf("unexpected present response: %d", resp.Tag)



		}



	



		slog.Info("PresentResponse received", "children_count", len(resp.Children))



		for i, c := range resp.Children {



			slog.Info("PresentResponse Child", "index", i, "tag", c.Tag, "class", c.ClassType)



		}



	



		var records []*MARCRecord

		for _, child := range resp.Children {

			if child.Tag == 28 {

				for i, recSeq := range child.Children {

					octet := findOctetString(recSeq)

					if octet != nil {

						slog.Info("Found OctetString", "size", len(octet))

						if syntaxOID == OID_SUTRS {

							rec := &MARCRecord{

								Leader: "SUTRS",

								Fields: []MARCField{

									{Tag: "TXT", Value: string(octet)},

								},

							}

							records = append(records, rec)

						} else {

							marc, err := ParseMARC(octet)

							if err == nil {

								records = append(records, marc)

							} else {

								slog.Error("ParseMARC failed", "error", err, "hex_start", fmt.Sprintf("%X", octet[:min(len(octet), 20)]))

							}

						}

					} else {

						slog.Warn("No OctetString found in record", "index", i)

						logPacketStructure(recSeq, 0)

					}

				}

			}

		}

		return records, nil

	}

	

	func min(a, b int) int {

		if a < b { return a }

		return b

	}

	

	func logPacketStructure(p *ber.Packet, depth int) {

		padding := ""

		for i := 0; i < depth; i++ { padding += "  " }

		slog.Info(fmt.Sprintf("%sTag: %d, Class: %d, Len: %d, Children: %d", padding, p.Tag, p.ClassType, len(p.Data.Bytes()), len(p.Children)))

		for _, c := range p.Children {

			logPacketStructure(c, depth+1)

		}

	}

func findOctetString(p *ber.Packet) []byte {
	if p.Tag == ber.TagOctetString && p.ClassType == ber.ClassUniversal {
		return p.Data.Bytes()
	}
	// Handle EXTERNAL (Tag 8)
	// EXTERNAL ::= [UNIVERSAL 8] IMPLICIT SEQUENCE {
	//   ...
	//   encoding CHOICE {
	//     single-ASN1-type [0] ABSTRACT-SYNTAX.&Type,
	//     octet-aligned    [1] IMPLICIT OCTET STRING,
	//     arbitrary        [2] IMPLICIT BIT STRING
	//   }
	// }
	if p.Tag == ber.TagExternal && p.ClassType == ber.ClassUniversal {
		for _, child := range p.Children {
			if child.ClassType == ber.ClassContext {
				if child.Tag == 1 { // octet-aligned
					return child.Data.Bytes()
				}
				if child.Tag == 0 { // single-ASN1-type
					// Recurse to find OctetString inside
					return findOctetString(child)
				}
			}
		}
	}

	for _, child := range p.Children {
		if res := findOctetString(child); res != nil {
			return res
		}
	}
	return nil
}

func (c *Client) Scan(dbName string, startTerm string, attributes map[int]int) ([]ScanEntry, error) {
	pdu := ber.Encode(ber.ClassContext, ber.TypeConstructed, 35, nil, "ScanRequest")
	
	dbs := ber.Encode(ber.ClassContext, ber.TypeConstructed, 3, nil, "DatabaseNames")
	dbs.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 105, dbName, "DB"))
	pdu.AppendChild(dbs)
	
	apt := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "APT")
	
	attrs := ber.Encode(ber.ClassContext, ber.TypeConstructed, 44, nil, "Attrs")
	attr := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "Attr")
	
	attrType := 1
	attrVal := 4
	if attributes != nil {
		if v, ok := attributes[1]; ok {
			attrVal = v
		}
	}

	attr.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, int64(attrType), "Type"))
	attr.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, int64(attrVal), "Value")) 
	attrs.AppendChild(attr)
	apt.AppendChild(attrs)
	
	term := ber.NewString(ber.ClassContext, ber.TypePrimitive, 45, startTerm, "Term")
	apt.AppendChild(term)
	
	pdu.AppendChild(apt)
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 31, 10, "NumberOfTermsRequested"))
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 32, 0, "StepSize"))
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 33, 1, "PositionOfTerm"))

	resp, err := c.sendPDU(pdu)
	if err != nil {
		return nil, err
	}

	if resp.Tag != 36 {
		return nil, fmt.Errorf("bad scan response: %d", resp.Tag)
	}

	var entries []ScanEntry
	
	for _, child := range resp.Children {
		if child.Tag == 7 { 
			list := child.Children[0] 
			for _, entry := range list.Children {
				termStr := ""
				cnt := 0
				
				var walk func(*ber.Packet)
				walk = func(p *ber.Packet) {
					if p.Tag == 45 {
						if v, ok := p.Value.([]byte); ok { termStr = string(v) } else { termStr = string(p.Data.Bytes()) }
					}
					if p.Tag == 2 {
						if v, ok := p.Value.(int64); ok { cnt = int(v) }
					}
					for _, sub := range p.Children { walk(sub) }
				}
				walk(entry)
				
				if termStr != "" {
					entries = append(entries, ScanEntry{Term: termStr, Count: cnt})
				}
			}
		}
	}

	return entries, nil
}

type ScanEntry struct {
	Term  string
	Count int
}

func (c *Client) Sort(resultSetName string, keys []SortKey) error {
	pdu := ber.Encode(ber.ClassContext, ber.TypeConstructed, 43, nil, "SortRequest")
	
	// ReferenceID (Optional)
	
	// InputResultSetNames [3] IMPLICIT SEQUENCE OF InternationalString
	input := ber.Encode(ber.ClassContext, ber.TypeConstructed, 3, nil, "InputResultSets")
	input.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagVisibleString, resultSetName, "ResultSetName"))
	pdu.AppendChild(input)

	// SortedResultSetName [4] IMPLICIT InternationalString
	pdu.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 4, resultSetName, "SortedResultSetName"))

	// SortSequence [5] IMPLICIT SEQUENCE OF SortKeySpec
	seq := ber.Encode(ber.ClassContext, ber.TypeConstructed, 5, nil, "SortSequence")
	
	for _, k := range keys {
		spec := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "SortKeySpec")
		
		// SortKey [0] CHOICE { privateSortKey, elementSpec, sortAttributes }
		sk := ber.Encode(ber.ClassContext, ber.TypeConstructed, 0, nil, "SortKey")
		// sortAttributes [2] IMPLICIT SEQUENCE { id, list }
		sa := ber.Encode(ber.ClassContext, ber.TypeConstructed, 2, nil, "SortAttributes")
		sa.AppendChild(ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagObjectIdentifier, OID_Bib1, "AttributeSetId"))
		
		list := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "AttributeList")
		
		// Use Attribute
		useAttr := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "Attr")
		useAttr.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, 1, "Type"))
		useAttr.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, int64(k.Attribute), "Value"))
		list.AppendChild(useAttr)

		// Relation Attribute (Sort Relation: 1=Ascending, 2=Descending)
		relVal := 1 // Default Ascending
		if k.Relation == 1 { relVal = 2 } // Descending
		
		relAttr := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "Attr")
		relAttr.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, 2, "Type")) // 2 = Relation
		// Relation: 3=Equal (default?), 1=Less, 2=LE... 
		// Actually, for Sort, Bib-1 Relation values are often repurposed or specific sort attributes used.
		// Standard Sort Relation attribute (type 7?): 1=Ascending, 2=Descending.
		// But in Bib-1, standard Use attributes are used.
		// The standard way is using SortKeySpec -> SortRelation (Tag 1) INTEGER { ascending(0), descending(1), ascendingByKey(3)... }
		// Wait, SortKeySpec has `sortRelation [1] IMPLICIT INTEGER DEFAULT ascending`.
		// It is NOT inside the AttributeList.
		
		sa.AppendChild(list)
		sk.AppendChild(sa)
		spec.AppendChild(sk)

		// SortRelation [1] IMPLICIT INTEGER DEFAULT 0 (ascending)
		relation := 0 
		if k.Relation == 1 { relation = 1 } // 1 = descending
		spec.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 1, int64(relation), "SortRelation"))
		
		// CaseSensitivity [2] IMPLICIT INTEGER { caseSensitive(0), caseInsensitive(1) }
		spec.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 2, 1, "CaseInsensitive"))

		seq.AppendChild(spec)
	}
	pdu.AppendChild(seq)

	resp, err := c.sendPDU(pdu)
	if err != nil {
		return err
	}

	if resp.Tag != 44 {
		return fmt.Errorf("unexpected sort response tag: %d", resp.Tag)
	}
	
	// Check SortStatus [3] IMPLICIT INTEGER { success(0), partial-1(1), failure(2) }
	for _, child := range resp.Children {
		if child.Tag == 3 {
			status := decodeInt(child)
			if status != 0 {
				return fmt.Errorf("sort failed with status: %d", status)
			}
		}
	}

	return nil
}

func (c *Client) DeleteResultSet(resultSetName string) error {
	pdu := ber.Encode(ber.ClassContext, ber.TypeConstructed, 30, nil, "DeleteRequest")
	pdu.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 32, 1, "DeleteAll"))

	resp, err := c.sendPDU(pdu)
	if err != nil {
		return err
	}

	if resp.Tag != 31 {
		return fmt.Errorf("bad delete response: %d", resp.Tag)
	}
	return nil
}