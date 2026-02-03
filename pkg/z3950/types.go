package z3950

import "encoding/asn1"

// PDU Tags
const (
	TagInitializeRequest  = 20
	TagInitializeResponse = 21
	TagSearchRequest      = 22
	TagSearchResponse     = 23
	TagPresentRequest     = 24
	TagPresentResponse    = 25
	TagScanRequest        = 35
	TagScanResponse       = 36
	TagClose              = 48
)

// --- Structs for Structured Search ---

// Bib1UseAttributes maps common Z39.50 Use attributes to their integer values.
const (
	UseAttributePersonalName = 1
	UseAttributeCorporateName = 2
	UseAttributeTitle  = 4
	UseAttributeTitleSeries = 5
	UseAttributeISBN   = 7
	UseAttributeISSN   = 8
	UseAttributeSubject = 21
	UseAttributeDatePub = 31
	UseAttributeAuthor = 1003 // Generic Author
	UseAttributeAny    = 1016
)

// QueryNode is the interface for nodes in the query tree (Leaf or Complex).
type QueryNode interface {
	isQueryNode()
}

// QueryClause represents a leaf node (a single search term).
type QueryClause struct {
	Attribute int
	Term      string
}
func (QueryClause) isQueryNode() {}

// QueryComplex represents a branch node (boolean operation).
type QueryComplex struct {
	Operator string // "AND", "OR", "AND-NOT"
	Left     QueryNode
	Right    QueryNode
}
func (QueryComplex) isQueryNode() {}

// StructuredQuery represents a parsed Z39.50 query as a Tree.
type StructuredQuery struct {
	Root     QueryNode
	Limit    int
	Offset   int
	SortKeys []SortKey
}

type SortKey struct {
	Attribute int // Use Attribute
	Relation  int // 0=Ascending, 1=Descending
}


// --- Unused ASN.1 struct definitions are removed for clarity ---
// The project uses manual BER packet construction/parsing, so these
// high-level structs were not being used directly.

type PDU struct {
	InitializeRequest  *InitializeRequest  `asn1:"optional,tag:20"`
	InitializeResponse *InitializeResponse `asn1:"optional,tag:21"`
	SearchRequest      *SearchRequest      `asn1:"optional,tag:22"`
}

type InitializeRequest struct {
	ReferenceId      []byte `asn1:"optional,tag:2"`
	ProtocolVersion  asn1.BitString
	Options          asn1.BitString
	PreferredMessageSize int
	MaximumRecordSize    int
}

type InitializeResponse struct {
	ReferenceId      []byte `asn1:"optional,tag:2"`
	ProtocolVersion  asn1.BitString
	Options          asn1.BitString
	PreferredMessageSize int
	MaximumRecordSize    int
	Result           bool
}

type SearchRequest struct {
	ReferenceId            []byte `asn1:"optional,tag:2"`
	SmallSetUpperBound     int
	LargeSetLowerBound     int
	MediumSetPresentNumber int
	ReplaceIndicator       bool
	ResultSetName          string
	DatabaseNames          []string
	Query                  Query
}

type Query struct {
	Type1 *RPNQuery `asn1:"optional,tag:1"`
}

type RPNQuery struct {
	AttributeSetId asn1.ObjectIdentifier
	RPN            RPNStructure
}

type RPNStructure struct {
	Op *Operand `asn1:"optional,tag:0"`
}

type Operand struct {
	AttrTerm *AttributesPlusTerm `asn1:"optional,tag:0"`
}

type AttributesPlusTerm struct {
	Attributes []AttributeElement `asn1:"tag:44"`
	Term       Term
}

type AttributeElement struct {
	AttributeType  int
	AttributeValue int
}

type Term struct {
	General []byte `asn1:"optional,tag:45"`
}
