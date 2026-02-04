package sip2

import (
	"bufio"
	"fmt"
	"strings"
	"time"
)

// SIP2 Command Codes
const (
	CmdPatronStatusRequest  = "23"
	CmdPatronStatusResponse = "24"
	CmdLogin                = "93"
	CmdLoginResponse        = "94"
	CmdPatronInfo           = "63"
	CmdPatronInfoResponse   = "64"
	CmdSCStatus             = "99"
	CmdACSStatus            = "98"
	CmdRequestResend        = "96"
	CmdEndSession           = "35"
	CmdEndSessionResponse   = "36"
)

// Message represents a parsed SIP2 message
type Message struct {
	Command    string
	FixedFields map[string]string // Fixed-width fields
	Fields     map[string]string // Variable fields (e.g. AO|...|)
	Raw        string
}

// ComputeChecksum calculates the SIP2 checksum
// The checksum is the 2's complement of the sum of all bytes
func ComputeChecksum(data string) string {
	sum := 0
	for _, r := range data {
		sum += int(r)
	}
	// 2's complement: (sum * -1) & 0xFFFF
	check := (-sum) & 0xFFFF
	return fmt.Sprintf("%04X", check)
}

// VerifyChecksum checks if the message ends with a valid checksum
func VerifyChecksum(raw string) bool {
	if len(raw) < 4 {
		return false
	}
	// Expected format: data...AZxxxx
	// The checksum is the last 4 chars. The 'AZ' sequence marker is optional but common.
	// Actually, SIP2 packet usually ends with "AZxxxx".
	// The checksum is calculated over everything BEFORE the checksum value itself.
	
	// Assuming raw string DOES NOT include the trailing CR/LF
	if len(raw) < 4 { return false }
	
	receivedSum := raw[len(raw)-4:]
	dataToSum := raw[:len(raw)-4]
	
	calcSum := ComputeChecksum(dataToSum)
	return receivedSum == calcSum
}

// BuildMessage constructs a SIP2 message string with checksum
func BuildMessage(command string, fixed string, fields map[string]string) string {
	var sb strings.Builder
	sb.WriteString(command)
	sb.WriteString(fixed)
	
	// Fields are sorted? No, but typically required order.
	// For simplicity in this helper, caller might need to be careful or we use a slice.
	// Here we just iterate.
	for k, v := range fields {
		sb.WriteString(k)
		sb.WriteString(v)
		sb.WriteString("|")
	}
	
	// Add Sequence Number (AY) if not present? Usually managed by session.
	// We'll append Checksum (AZ) automatically.
	
	// Checksum includes the "AZ" tag but not the value
	data := sb.String()
	data += "AZ"
	
	sum := ComputeChecksum(data)
	return data + sum
}

// GetFixedLength returns the length of the fixed data segment for a command
func GetFixedLength(cmd string) int {
	switch cmd {
	case CmdLoginResponse: return 1 // Ok/Fail
	case CmdPatronInfoResponse: return 59 // Status(14)+Lang(3)+Date(18)+Counts(24)
	case CmdACSStatus: return 24 // Online(1)+Checkin(1)+...+Date(18)+Version(4)
	default: return 0
	}
}

// Parse parses a raw SIP2 string
func Parse(raw string) (*Message, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 2 {
		return nil, fmt.Errorf("message too short")
	}
	
	cmd := raw[0:2]
	fixedLen := GetFixedLength(cmd)
	
	// Remove checksum if present (last 4 chars)
	// Usually raw includes Checksum.
	// Checksum is part of the "AZ" field which is variable.
	// But AZ is always last.
	// Let's just split by '|'
	
	// Remove Command
	body := raw[2:]
	
	msg := &Message{
		Command: cmd,
		Fields:  make(map[string]string),
		Raw:     raw,
	}
	
	// Extract Fixed Fields
	if len(body) >= fixedLen {
		// Store raw fixed string? Or parse?
		// For now we just skip it to get to variable fields
		body = body[fixedLen:]
	} else {
		// Maybe short message or error?
		body = "" 
	}
	
	// Now body should be "AEJohn Doe|BZ5|AZxxxx"
	parts := strings.Split(body, "|")
	for _, part := range parts {
		if len(part) > 2 {
			tag := part[:2]
			val := part[2:]
			msg.Fields[tag] = val
		}
	}
	
	return msg, nil
}

// FormatDate returns SIP2 date format YYYYMMDDZZZZHHMMSS
func FormatDate(t time.Time) string {
	return t.Format("20060102    150405")
}

// ParseDate parses SIP2 date format
func ParseDate(s string) (time.Time, error) {
	// 18 chars: YYYYMMDDZZZZHHMMSS
	// ZZZZ is space-padded?
	if len(s) < 18 { return time.Time{}, fmt.Errorf("date too short") }
	
	// Handle ZZZZ being spaces or whatever
	layout := "20060102    150405"
	// Sometimes it's without spaces?
	return time.Parse(layout, s)
}

// Client represents a SIP2 connection
type Client struct {
	conn     *bufio.ReadWriter // Abstract connection
	Host     string
	Port     int
	Location string // CP field (Location Code)
	User     string // CN field
	Pass     string // CO field
}
