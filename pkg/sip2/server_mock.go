package sip2

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

type MockServer struct {
	Port     int
	Listener net.Listener
}

func NewMockServer(port int) *MockServer {
	return &MockServer{Port: port}
}

func (s *MockServer) Start() error {
	addr := fmt.Sprintf(":%d", s.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.Listener = ln
	
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handle(conn)
		}
	}()
	return nil
}

func (s *MockServer) Close() {
	if s.Listener != nil {
		s.Listener.Close()
	}
}

func (s *MockServer) handle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	
	for {
		// Read until CR
		line, err := reader.ReadString('')
		if err != nil {
			return
		}
		
		line = strings.TrimSpace(line) // Remove CR and whitespace
		if len(line) < 2 { continue }
		
		// Parse
		cmd := line[:2]
		
		var resp string
		switch cmd {
		case CmdLogin: // 93
			// 94<1>...
			// Respond with OK
			resp = BuildMessage(CmdLoginResponse, "1", nil)
		case CmdPatronInfo: // 63
			// 64<PatronStatus><Language><TransactionDate><HoldCount><OverdueCount><ChargedCount><FineCount><RecallCount><UnavailableCount>...
			// PatronStatus: 14 chars blank? or meaningful?
			// Usually "              " (14 spaces) means OK.
			// Fixed: 14 + 3 + 18 + 4 + 4 + 4 + 4 + 4 + 4 = 59 chars fixed
			
			patronStatus := "              " // 14
			lang := "001"
			date := FormatDate(time.Now())
			counts := "000000000005000000000000" // 6 * 4 digits
			fixed := patronStatus + lang + date + counts
			
			fields := map[string]string{
				"AE": "John Doe", // Personal Name
				"BZ": "5",        // Limit
			}
			resp = BuildMessage(CmdPatronInfoResponse, fixed, fields)
		case CmdSCStatus: // 99
			// 98<Online><Checkin><Checkout><Renewal>...
			fixed := "YYYYYY" + FormatDate(time.Now()) + "2.00"
			fields := map[string]string{
				"AO": "MockLibrary",
			}
			resp = BuildMessage(CmdACSStatus, fixed, fields)
		}
		
		if resp != "" {
			conn.Write([]byte(resp + ""))
		}
	}
}
