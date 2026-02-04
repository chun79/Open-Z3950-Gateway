package sip2

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

type SIP2Client struct {
	Host     string
	Port     int
	Location string // CP
	User     string // CN (SIP2 Login User, not Patron)
	Pass     string // CO
	
	conn     net.Conn
	reader   *bufio.Reader
	timeout  time.Duration
}

func NewClient(host string, port int) *SIP2Client {
	return &SIP2Client{
		Host:    host,
		Port:    port,
		timeout: 10 * time.Second,
	}
}

func (c *SIP2Client) Connect() error {
	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	conn, err := net.DialTimeout("tcp", addr, c.timeout)
	if err != nil {
		return err
	}
	c.conn = conn
	c.reader = bufio.NewReader(conn)
	return nil
}

func (c *SIP2Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *SIP2Client) Send(cmd string, fixed string, fields map[string]string) (string, error) {
	msg := BuildMessage(cmd, fixed, fields)
	// SIP2 typically ends with CR (0x0D)
	msg += ""
	
	c.conn.SetDeadline(time.Now().Add(c.timeout))
	if _, err := c.conn.Write([]byte(msg)); err != nil {
		return "", err
	}
	
	// Read response
	// Responses also end with CR
	resp, err := c.reader.ReadString('')
	if err != nil {
		return "", err
	}
	
	// Strip trailing CR
	resp = resp[:len(resp)-1]
	
	// Optional: Verify Checksum
	if !VerifyChecksum(resp) {
		return resp, fmt.Errorf("checksum mismatch")
	}
	
	return resp, nil
}

func (c *SIP2Client) Login(user, pass, location string) (bool, error) {
	// 93: Login
	// Fixed: 00 (UID algo), 00 (PWD algo)
	// Fields: CN (User), CO (Pass), CP (Location)
	fields := map[string]string{
		"CN": user,
		"CO": pass,
		"CP": location,
	}
	
	resp, err := c.Send(CmdLogin, "0000", fields)
	if err != nil {
		return false, err
	}
	
	// Parse 94
	// Format: 94<1 char Ok><Fixed>...
	if len(resp) < 3 || resp[:2] != CmdLoginResponse {
		return false, fmt.Errorf("unexpected response: %s", resp)
	}
	
	ok := resp[2] == '1'
	return ok, nil
}

// GetPatronInfo sends a 63 command
// summary: allows to get summary of holds, overdues, etc.
func (c *SIP2Client) GetPatronInfo(patronID string, password string) (map[string]string, error) {
	// 63: Patron Information
	// Fixed: Language (3), Timestamp (18), Summary (10)
	// Summary: 
	//   Hold Items (Y/N)
	//   Overdue Items (Y/N)
	//   Charged Items (Y/N)
	//   Fine Items (Y/N)
	//   Recall Items (Y/N)
	//   Unavailable Holds (Y/N)
	//   ... (10 chars) -> "YYYYYYYYYY" to get everything
	
	fixed := "001" + FormatDate(time.Now()) + "YYYYYYYYYY"
	fields := map[string]string{
		"AO": c.Location, // Institution ID
		"AA": patronID,   // Patron Identifier
	}
	if password != "" {
		fields["AD"] = password
	}
	
	resp, err := c.Send(CmdPatronInfo, fixed, fields)
	if err != nil {
		return nil, err
	}
	
	// Parse response
	parsed, err := Parse(resp)
	if err != nil { return nil, err }
	
	return parsed.Fields, nil
}
