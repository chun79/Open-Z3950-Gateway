package main

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	ber "github.com/go-asn1-ber/asn1-ber"
	"github.com/gin-gonic/gin"
	"github.com/yourusername/open-z3950-gateway/pkg/auth"
	"github.com/yourusername/open-z3950-gateway/pkg/provider"
	"github.com/yourusername/open-z3950-gateway/pkg/ui"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

// --- ZServer Definitions ---

const (
	TagInitializeRequest  = 20
	TagInitializeResponse = 21
	TagSearchRequest      = 22
	TagSearchResponse     = 23
	TagPresentRequest     = 24
	TagPresentResponse    = 25
	TagScanRequest        = 35
	TagScanResponse       = 36
)

type Session struct {
	ResultIDs []string
	DBName    string
}

type Server struct {
	provider    provider.Provider
	mu          sync.RWMutex
	sessions    map[string]*Session
	allowedIPs  []*net.IPNet
	allowAllIPs bool
	profile     *z3950.MARCProfile
}

func NewServer(p provider.Provider) *Server {
	s := &Server{
		provider: p,
		sessions: make(map[string]*Session),
	}
	s.loadWhitelist()
	s.profile = &z3950.ProfileMARC21
	return s
}

func (s *Server) loadWhitelist() {
	env := os.Getenv("ZSERVER_ALLOWED_IPS")
	if env == "" {
		s.allowAllIPs = true
		return
	}
	parts := strings.Split(env, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, "/") {
			if strings.Contains(part, ":") {
				part += "/128"
			} else {
				part += "/32"
			}
		}
		_, ipnet, err := net.ParseCIDR(part)
		if err != nil {
			continue
		}
		s.allowedIPs = append(s.allowedIPs, ipnet)
	}
}

func (s *Server) checkIP(addr net.Addr) bool {
	if s.allowAllIPs {
		return true
	}
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		return false
	}
	for _, ipnet := range s.allowedIPs {
		if ipnet.Contains(tcpAddr.IP) {
			return true
		}
	}
	return false
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()
	connID := conn.RemoteAddr().String()
	slog.Info("new z39.50 connection", "conn_id", connID)

	s.mu.Lock()
	s.sessions[connID] = &Session{ResultIDs: []string{}, DBName: "Default"}
	s.mu.Unlock()

	for {
		pkt, err := ber.ReadPacket(conn)
		if err != nil {
			s.mu.Lock()
			delete(s.sessions, connID)
			s.mu.Unlock()
			return
		}

		switch pkt.Tag {
		case TagInitializeRequest:
			s.handleInit(conn, connID)
		case TagSearchRequest:
			s.handleSearch(conn, connID, pkt)
		case TagPresentRequest:
			s.handlePresent(conn, connID, pkt)
		case TagScanRequest:
			s.handleScan(conn, connID, pkt)
		}
	}
}

func (s *Server) handleInit(conn net.Conn, connID string) {
	resp := ber.Encode(ber.ClassContext, ber.TypeConstructed, TagInitializeResponse, nil, "InitResp")
	resp.AppendChild(ber.Encode(ber.ClassUniversal, ber.TypePrimitive, ber.TagBitString, []byte{0x00, 0xC0}, "Ver"))
	resp.AppendChild(ber.Encode(ber.ClassUniversal, ber.TypePrimitive, ber.TagBitString, []byte{0x00, 0xF0}, "Opt"))
	resp.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, 1048576, "MsgSize"))
	resp.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, 1048576, "RecSize"))
	resp.AppendChild(ber.NewBoolean(ber.ClassUniversal, ber.TypePrimitive, ber.TagBoolean, true, "Result"))
	resp.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 110, "GoZServer", "ImpId"))
	conn.Write(resp.Bytes())
	slog.Info("init success", "conn_id", connID)
}

func parseOperand(operand *ber.Packet) (z3950.QueryClause, error) {
	var clause z3950.QueryClause
	if operand.Tag != 0 || operand.ClassType != ber.ClassContext {
		return clause, fmt.Errorf("packet is not an operand")
	}
	
	if len(operand.Children) == 0 {
		return clause, fmt.Errorf("operand has no children")
	}
	apt := operand.Children[0] // Attribute-plus-term
	if apt.Tag != 102 {
		return clause, fmt.Errorf("expected APT (tag 102) inside operand, got %d", apt.Tag)
	}

	for _, child := range apt.Children {
		if child.Tag == 44 { // AttributeList
			if len(child.Children) > 0 {
				attr := child.Children[0] // Attribute
				if attr.Tag == ber.TagSequence && len(attr.Children) >= 2 {
					attrType, _ := attr.Children[0].Value.(int64)
					attrValue, _ := attr.Children[1].Value.(int64)
					if attrType == 1 { // 1 = Use attribute
						clause.Attribute = int(attrValue)
					}
				}
			}
		} else if child.Tag == 45 { // Term
			clause.Term = string(child.Data.Bytes())
		}
	}
	
	if clause.Term == "" {
		return clause, fmt.Errorf("could not find term in operand")
	}
	
	return clause, nil
}

// recursiveParseRPN processes the RPN structure recursively
func recursiveParseRPN(p *ber.Packet) (z3950.QueryNode, error) {
	// Choice: Operand [0] or RPNRpnOp [1]
	
	if p.ClassType == ber.ClassContext && p.Tag == 0 {
		// Operand
		return parseOperand(p)
	}
	
	if p.ClassType == ber.ClassContext && p.Tag == 1 {
		// Complex (RPN1, RPN2, Op)
		if len(p.Children) < 3 {
			return nil, fmt.Errorf("complex RPN missing children")
		}
		
		left, err := recursiveParseRPN(p.Children[0])
		if err != nil { return nil, err }
		
		right, err := recursiveParseRPN(p.Children[1])
		if err != nil { return nil, err }
		
		opNode := p.Children[2]
		opStr := "AND"
		if len(opNode.Children) > 0 {
			if v, ok := opNode.Children[0].Value.(int64); ok {
				switch v {
				case 0: opStr = "AND"
				case 1: opStr = "OR"
				case 2: opStr = "AND-NOT"
				}
			}
		}
		
		return z3950.QueryComplex{
			Operator: opStr,
			Left:     left,
			Right:    right,
		}, nil
	}
	
	return nil, fmt.Errorf("unknown RPN tag: %d", p.Tag)
}

func parseRPNQuery(queryPacket *ber.Packet) (z3950.StructuredQuery, error) {
	// Locate the RPNStructure inside the Query/RPNQuery wrapper
	// Structure: Query [1] -> RPNQuery [Sequence] -> RPNStructure [Choice]
	// Caller usually passes the Query [1] packet or its child.
	// Let's assume input is Query packet.
	
	var rpnStruct *ber.Packet
	
	// Simplified logic: The RPN query usually has Bib-1 OID then the Structure.
	// The Structure is the second child of RPNQuery sequence.
	
	if len(queryPacket.Children) > 0 {
		// RPNQuery
		rpnQuery := queryPacket.Children[0]
		if len(rpnQuery.Children) >= 2 {
			rpnStruct = rpnQuery.Children[1]
		}
	}
	
	if rpnStruct == nil {
		// Fallback: Try to parse whatever we got
		// This handles cases where caller stripped layers
		if len(queryPacket.Children) > 0 {
			rpnStruct = queryPacket.Children[0]
		} else {
			return z3950.StructuredQuery{}, fmt.Errorf("empty query packet")
		}
	}

	root, err := recursiveParseRPN(rpnStruct)
	if err != nil {
		return z3950.StructuredQuery{}, err
	}
	
	return z3950.StructuredQuery{Root: root}, nil
}

func (s *Server) handleSearch(conn net.Conn, connID string, req *ber.Packet) {
	dbName := "Default"
	for _, c := range req.Children {
		if c.Tag == ber.TagSequence && len(c.Children) > 0 && c.Children[0].Tag == ber.TagVisibleString {
			dbName = string(c.Children[0].Data.Bytes())
			break
		}
	}

	var queryNode *ber.Packet
	for _, c := range req.Children {
		if c.Tag == 21 && c.ClassType == ber.ClassContext {
			queryNode = c
			break
		}
	}

	if queryNode == nil {
		slog.Error("missing query node in search request", "conn_id", connID)
		resp := ber.Encode(ber.ClassContext, ber.TypeConstructed, TagSearchResponse, nil, "SearchResp")
		resp.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, 1, "Status"))
		conn.Write(resp.Bytes())
		return
	}

	query, err := parseRPNQuery(queryNode)
	if err != nil {
		slog.Error("failed to parse RPN query", "error", err, "conn_id", connID)
		resp := ber.Encode(ber.ClassContext, ber.TypeConstructed, TagSearchResponse, nil, "SearchResp")
		resp.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, 1, "Status"))
		conn.Write(resp.Bytes())
		return
	}

	ids, err := s.provider.Search(dbName, query)
	if err != nil {
		slog.Error("provider search failed", "error", err, "conn_id", connID)
		resp := ber.Encode(ber.ClassContext, ber.TypeConstructed, TagSearchResponse, nil, "SearchResp")
		resp.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, 1, "Status"))
		conn.Write(resp.Bytes())
		return
	}

	s.mu.Lock()
	if sess, ok := s.sessions[connID]; ok {
		sess.ResultIDs = ids
		sess.DBName = dbName
	}
	s.mu.Unlock()
	
	slog.Info("search processed", "db", dbName, "found", len(ids))

	resp := ber.Encode(ber.ClassContext, ber.TypeConstructed, TagSearchResponse, nil, "SearchResp")
	resp.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 23, int64(len(ids)), "ResultCount"))
	resp.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 24, 0, "Returned"))
	resp.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 25, 0, "NextPos"))
	resp.AppendChild(ber.NewBoolean(ber.ClassContext, ber.TypePrimitive, 26, true, "Status"))
	conn.Write(resp.Bytes())
}

func (s *Server) handlePresent(conn net.Conn, connID string, req *ber.Packet) {
	reqCount, startPoint := 1, 1
	for _, c := range req.Children {
		if c.Tag == 29 { if v, ok := c.Value.(int64); ok { reqCount = int(v) } }
		if c.Tag == 30 { if v, ok := c.Value.(int64); ok { startPoint = int(v) } }
	}

	s.mu.RLock()
	sess, ok := s.sessions[connID]
	s.mu.RUnlock()
	if !ok { return }

	ids := sess.ResultIDs
	startIdx := startPoint - 1
	if startIdx < 0 { startIdx = 0 }
	endIdx := startIdx + reqCount
	if endIdx > len(ids) { endIdx = len(ids) }
	
	var subsetIDs []string
	if startIdx < len(ids) { subsetIDs = ids[startIdx:endIdx] }
	
	records, _ := s.provider.Fetch(sess.DBName, subsetIDs)
	slog.Info("present processed", "conn_id", connID, "returned", len(records))

	profile := &z3950.ProfileMARC21
	if strings.Contains(strings.ToUpper(sess.DBName), "CNMARC") { profile = &z3950.ProfileCNMARC }
	if strings.Contains(strings.ToUpper(sess.DBName), "UNIMARC") { profile = &z3950.ProfileUNIMARC }

	resp := ber.Encode(ber.ClassContext, ber.TypeConstructed, TagPresentResponse, nil, "PresentResp")
	resp.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 2, "ref", "RefId"))
	resp.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 29, int64(len(records)), "Returned"))
	resp.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 30, 0, "Next"))
	resp.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 27, 0, "Status"))
	
	recordsWrapper := ber.Encode(ber.ClassContext, ber.TypeConstructed, 28, nil, "Records")
	for _, rec := range records {
		marcData := z3950.BuildMARC(profile, "", rec.GetTitle(profile), rec.GetAuthor(profile), rec.GetISBN(profile), rec.GetPublisher(profile), "", rec.GetISSN(profile), rec.GetSubject(profile))
		
		namePlusRecord := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "Record")
		dbRecord := ber.Encode(ber.ClassContext, ber.TypeConstructed, 1, nil, "DBRecord")
		octet := ber.NewString(ber.ClassUniversal, ber.TypePrimitive, ber.TagOctetString, string(marcData), "MARC")
		dbRecord.AppendChild(octet)
		namePlusRecord.AppendChild(dbRecord)
		recordsWrapper.AppendChild(namePlusRecord)
	}
	resp.AppendChild(recordsWrapper)
	conn.Write(resp.Bytes())
}

func (s *Server) handleScan(conn net.Conn, connID string, req *ber.Packet) {
	term := ""
	var findTerm func(*ber.Packet)
	findTerm = func(p *ber.Packet) {
		if p.Tag == 45 {
			if val, ok := p.Value.([]byte); ok { term = string(val) } else { term = string(p.Data.Bytes()) }
		}
		for _, c := range p.Children { findTerm(c) }
	}
	findTerm(req)
	
	// Default field for Z39.50 scan is often Title if not specified in attributes.
	// For simplicity, we assume Title unless we parse attributes fully.
	// (Real Z39.50 scan requests carry attributes just like search)
	field := "title" 

	s.mu.RLock()
	sess, ok := s.sessions[connID]
	s.mu.RUnlock()
	dbName := "Default"
	if ok { dbName = sess.DBName }

	results, _ := s.provider.Scan(dbName, field, term)
	slog.Info("scan processed", "term", term, "found", len(results))
	
	resp := ber.Encode(ber.ClassContext, ber.TypeConstructed, TagScanResponse, nil, "ScanResp")
	resp.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, 0, "Step"))
	resp.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, 0, "Status"))
	resp.AppendChild(ber.NewInteger(ber.ClassUniversal, ber.TypePrimitive, ber.TagInteger, int64(len(results)), "Count"))
	entriesWrapper := ber.Encode(ber.ClassContext, ber.TypeConstructed, 7, nil, "Entries")
	listEntries := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "List")
	for _, res := range results {
		entry := ber.Encode(ber.ClassUniversal, ber.TypeConstructed, ber.TagSequence, nil, "Entry")
		termInfo := ber.Encode(ber.ClassContext, ber.TypeConstructed, 1, nil, "TermInfo")
		termInfo.AppendChild(ber.NewString(ber.ClassContext, ber.TypePrimitive, 45, res.Term, "Term"))
		termInfo.AppendChild(ber.NewInteger(ber.ClassContext, ber.TypePrimitive, 2, int64(res.Count), "Count"))
		entry.AppendChild(termInfo)
		listEntries.AppendChild(entry)
	}
	entriesWrapper.AppendChild(listEntries)
	resp.AppendChild(entriesWrapper)
	conn.Write(resp.Bytes())
}

// --- Gateway and Main Logic ---

func initLogger() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}

func authMiddleware() gin.HandlerFunc {
	requiredKey := os.Getenv("GATEWAY_API_KEY")
	
	return func(c *gin.Context) {
		// 1. Check for legacy API Key (header or query)
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			apiKey = c.Query("apikey")
		}
		if requiredKey != "" && apiKey == requiredKey {
			c.Set("username", "api-key-user")
			c.Set("role", "admin")
			c.Next()
			return
		}

		// 2. Check for JWT (Authorization: Bearer <token>)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := auth.ParseToken(tokenString)
			if err == nil {
				c.Set("username", claims.Username)
				c.Set("role", claims.Role)
				c.Set("userID", claims.UserID)
				c.Next()
				return
			}
		}

		// 3. Unauthorized
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized: Invalid API Key or Token",
		})
	}
}

// setupRouter initializes the Gin engine and routes
func setupRouter(dbProvider provider.Provider) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// Public Auth Routes
	r.POST("/api/auth/login", func(c *gin.Context) {
		var creds struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.BindJSON(&creds); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		user, err := dbProvider.GetUserByUsername(creds.Username)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		if !auth.CheckPassword(creds.Password, user.PasswordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		token, err := auth.GenerateToken(user)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(200, gin.H{
			"status": "success",
			"token":  token,
			"user":   gin.H{"username": user.Username, "role": user.Role},
		})
	})

	r.POST("/api/auth/register", func(c *gin.Context) {
		var req struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Hash password here in the handler
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			c.JSON(500, gin.H{"error": "Server error"})
			return
		}

		user := &provider.User{
			Username:     req.Username,
			PasswordHash: hash,
			Role:         "user",
		}

		if err := dbProvider.CreateUser(user); err != nil {
			slog.Error("failed to create user", "error", err)
			c.JSON(409, gin.H{"error": "Username already exists or create failed"})
			return
		}

		c.JSON(201, gin.H{"status": "success", "message": "User created"})
	})

	// Protected API routes
	api := r.Group("/api")
	api.Use(authMiddleware())

	api.GET("/search", func(c *gin.Context) {
		start := time.Now()
		db := c.DefaultQuery("db", "LCDB")

		// --- Parse URL parameters into a structured Tree (Left-associated) ---
		var root z3950.QueryNode

		// First term
		term1 := c.Query("term1")
		if term1 == "" {
			term1 = c.Query("query") // Fallback for simple query
		}
		if term1 == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing query"})
			return
		}
		
		attr1Str := c.Query("attr1")
		attr1 := z3950.UseAttributeAny
		if attr1Str != "" {
			attr1, _ = strconv.Atoi(attr1Str)
		}
		
		root = z3950.QueryClause{Attribute: attr1, Term: term1}

		// Subsequent terms
		for i := 2; ; i++ {
			termKey := fmt.Sprintf("term%d", i)
			term, exists := c.GetQuery(termKey)
			if !exists { break }
			
			attrKey := fmt.Sprintf("attr%d", i)
			attrStr := c.Query(attrKey)
			attr := z3950.UseAttributeAny
			if attrStr != "" {
				attr, _ = strconv.Atoi(attrStr)
			}
			
			opKey := fmt.Sprintf("op%d", i)
			operator := c.DefaultQuery(opKey, "AND")
			
			// Build tree: Complex(Root, NewClause)
			root = z3950.QueryComplex{
				Operator: operator,
				Left:     root,
				Right:    z3950.QueryClause{Attribute: attr, Term: term},
			                        }
			                }
			
			                // Parse Sort Options
			                sortAttrStr := c.Query("sortAttr")
			                sortOrderStr := c.Query("sortOrder") // "asc" or "desc"
			                
			                var sortKeys []z3950.SortKey
			                if sortAttrStr != "" {
			                        attr, _ := strconv.Atoi(sortAttrStr)
			                        relation := 0 // Ascending
			                        if sortOrderStr == "desc" {
			                                relation = 1 // Descending
			                        }
			                        sortKeys = append(sortKeys, z3950.SortKey{Attribute: attr, Relation: relation})
			                }
			
			                structuredQuery := z3950.StructuredQuery{Root: root, SortKeys: sortKeys}
			                // --- END ---
		// DIRECT CALL TO PROVIDER
		ids, err := dbProvider.Search(db, structuredQuery)
		if err != nil {
			slog.Error("provider search failed", "error", err)
			c.JSON(500, gin.H{"error": "Search: " + err.Error()})
			return
		}

		records, err := dbProvider.Fetch(db, ids)
		if err != nil {
			slog.Error("provider fetch failed", "error", err)
			c.JSON(500, gin.H{"error": "Fetch: " + err.Error()})
			return
		}

		results := make([]map[string]interface{}, 0)
		for _, rec := range records {
			if rec.Leader == "SUTRS" {
				// Special handling for text records
				txt := ""
				if len(rec.Fields) > 0 { txt = rec.Fields[0].Value }
				results = append(results, map[string]interface{}{
					"title": "Text Record",
					"raw":   txt,
					"format": "SUTRS",
				})
			} else {
				results = append(results, map[string]interface{}{
					"record_id": rec.RecordID,
					"title":     rec.Title,
					"author":    rec.Author,
					"isbn":      rec.ISBN,
					"issn":      rec.ISSN,
					"subject":   rec.Subject,
					"publisher": rec.Publisher,
					"summary":   rec.Summary,
					"toc":       rec.TOC,
					"edition":   rec.Edition,
					"physical":  rec.PhysicalDescription,
					"series":    rec.Series,
					"notes":     rec.Notes,
					"leader":    rec.Leader,
					"fields":    rec.Fields,
					"holdings":  rec.Holdings,
				})
			}
		}

		elapsed := time.Since(start)
		slog.Info("search request completed",
			"found", len(ids),
			"fetched", len(results),
			"latency_ms", elapsed.Milliseconds(),
		)

		c.JSON(200, gin.H{
			"status": "success",
			"found":  len(ids),
			"data":   results,
		})
	})

	api.GET("/books/:db/:id", func(c *gin.Context) {
		db := c.Param("db")
		id := c.Param("id")
		
		records, err := dbProvider.Fetch(db, []string{id})
		if err != nil {
			slog.Error("failed to fetch book", "db", db, "id", id, "error", err)
			c.JSON(500, gin.H{"error": "Fetch failed: " + err.Error()})
			return
		}
		
		if len(records) == 0 {
			c.JSON(404, gin.H{"error": "Book not found"})
			return
		}
		
		rec := records[0]
		// Return friendly JSON
		c.JSON(200, gin.H{
			"status": "success",
			"data": map[string]interface{}{
				"record_id": rec.RecordID,
				"title":     rec.Title,
				"author":    rec.Author,
				"isbn":      rec.ISBN,
				"issn":      rec.ISSN,
				"subject":   rec.Subject,
				"publisher": rec.Publisher,
				"summary":   rec.Summary,
				"toc":       rec.TOC,
				"edition":   rec.Edition,
				"physical":  rec.PhysicalDescription,
				"series":    rec.Series,
				"notes":     rec.Notes,
				"leader":    rec.Leader,
				"fields":    rec.Fields,
				"holdings":  rec.Holdings,
			},
		})
	})

	api.POST("/ill-requests", func(c *gin.Context) {
		var req provider.ILLRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
			return
		}

		if req.Status == "" {
			req.Status = "pending"
		}
		
		// Use the username from the context (set by authMiddleware)
		if username, exists := c.Get("username"); exists {
			req.Requestor = username.(string)
		} else {
			req.Requestor = "anonymous" // Should not happen with authMiddleware
		}

		if err := dbProvider.CreateILLRequest(req); err != nil {
			slog.Error("failed to create ILL request", "error", err)
			c.JSON(500, gin.H{"error": "Failed to create request: " + err.Error()})
			return
		}

		slog.Info("ILL request created", "isbn", req.ISBN, "title", req.Title)
		c.JSON(201, gin.H{"status": "success", "message": "ILL request created"})
	})

	api.GET("/ill-requests", func(c *gin.Context) {
		requests, err := dbProvider.ListILLRequests()
		if err != nil {
			slog.Error("failed to list ILL requests", "error", err)
			c.JSON(500, gin.H{"error": "Failed to list requests: " + err.Error()})
			return
		}

		// Filter for regular users
		role, _ := c.Get("role")
		username, _ := c.Get("username")
		
		filtered := make([]provider.ILLRequest, 0)
		if role == "admin" {
			filtered = requests
		} else {
			for _, r := range requests {
				if r.Requestor == username {
					filtered = append(filtered, r)
				}
			}
		}

		c.JSON(200, gin.H{
			"status": "success",
			"data":   filtered,
		})
	})

	api.GET("/targets", func(c *gin.Context) {
		targets, err := dbProvider.ListTargets()
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to list targets"})
			return
		}
		
		names := make([]string, 0)
		for _, t := range targets {
			names = append(names, t.Name)
		}
		
		c.JSON(200, gin.H{
			"status": "success",
			"data":   names,
		})
	})

	api.GET("/scan", func(c *gin.Context) {
		db := c.DefaultQuery("db", "LCDB")
		term := c.Query("term")
		field := c.DefaultQuery("field", "title")

		if term == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'term' parameter"})
			return
		}

		results, err := dbProvider.Scan(db, field, term)
		if err != nil {
			slog.Error("provider scan failed", "db", db, "term", term, "error", err)
			c.JSON(500, gin.H{"error": "Scan: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"status": "success",
			"db":     db,
			"data":   results,
		})
	})

	api.PUT("/ill-requests/:id/status", func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID format"})
			return
		}

		var body struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.BindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
			return
		}

		if err := dbProvider.UpdateILLRequestStatus(id, body.Status); err != nil {
			slog.Error("failed to update ILL request status", "id", id, "status", body.Status, "error", err)
			c.JSON(500, gin.H{"error": "Failed to update status: " + err.Error()})
			return
		}

		slog.Info("ILL request status updated", "id", id, "status", body.Status)
		c.JSON(200, gin.H{"status": "success", "message": "Status updated"})
	})

	// Admin Routes
	admin := api.Group("/admin")
	admin.Use(func(c *gin.Context) {
		role, _ := c.Get("role")
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		c.Next()
	})

	admin.GET("/targets", func(c *gin.Context) {
		targets, err := dbProvider.ListTargets()
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to list targets"})
			return
		}
		c.JSON(200, gin.H{"status": "success", "data": targets})
	})

	admin.POST("/targets", func(c *gin.Context) {
		var t provider.Target
		if err := c.BindJSON(&t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}
		if err := dbProvider.CreateTarget(&t); err != nil {
			c.JSON(500, gin.H{"error": "Failed to create target: " + err.Error()})
			return
		}
		c.JSON(201, gin.H{"status": "success", "message": "Target created"})
	})

	admin.POST("/targets/test", func(c *gin.Context) {
		var t struct {
			Host string `json:"host" binding:"required"`
			Port int    `json:"port" binding:"required"`
		}
		if err := c.BindJSON(&t); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		client := z3950.NewClient(t.Host, t.Port)
		if err := client.Connect(); err != nil {
			c.JSON(200, gin.H{"status": "error", "message": "Connection failed: " + err.Error()})
			return
		}
		defer client.Close()

		if err := client.Init(); err != nil {
			c.JSON(200, gin.H{"status": "error", "message": "Handshake failed: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "success", "message": "Connection and Handshake successful!"})
	})

	admin.DELETE("/targets/:id", func(c *gin.Context) {
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		if err := dbProvider.DeleteTarget(id); err != nil {
			c.JSON(500, gin.H{"error": "Failed to delete target"})
			return
		}
		c.JSON(200, gin.H{"status": "success", "message": "Target deleted"})
	})

	// Setup SPA (Single Page Application) serving
	spaHandler := ui.SPAHandler()
	r.NoRoute(func(c *gin.Context) {
		spaHandler.ServeHTTP(c.Writer, c.Request)
	})

	return r
}

func main() {
	initLogger()

	slog.Info("running MARC self-test")
	testBlob := z3950.BuildMARC(nil, "001", "Test Title", "Test Author", "123456", "Test Publisher", "2026", "1234-5678", "Test Subject")
	if parsed, err := z3950.ParseMARC(testBlob); err != nil {
		slog.Error("self-test failed", "error", err, "hex", hex.EncodeToString(testBlob))
	} else {
		slog.Info("self-test passed", "title", parsed.GetTitle(nil), "fields", len(parsed.Fields))
	}

	// 1. Initialize Provider
	var dbProvider provider.Provider
	var err error

	dbProviderType := os.Getenv("DB_PROVIDER")
	slog.Info("initializing database provider", "type", dbProviderType)

	switch dbProviderType {
	case "sqlite":
		dbPath := os.Getenv("DB_PATH")
		dbProvider, err = provider.NewSQLiteProvider(dbPath)
	case "postgres":
		dsn := os.Getenv("DB_DSN")
		dbProvider, err = provider.NewPostgresProvider(dsn)
	default:
		if dbProviderType == "" {
			dbProviderType = "memory"
		}
		slog.Info("using in-memory provider as default")
		dbProvider = provider.NewMemoryProvider()
	}

	if err != nil {
		slog.Error("failed to initialize database provider", "type", dbProviderType, "error", err)
		panic(err)
	}
	slog.Info("database provider initialized successfully", "type", dbProviderType)

	// Wrap with HybridProvider to support remote targets
	hybridProvider := provider.NewHybridProvider(dbProvider)
	// Update the main dbProvider variable to point to the hybrid one
	dbProvider = hybridProvider

	// 2. Start Z39.50 Server in a goroutine
	go func() {
		zPort := os.Getenv("ZSERVER_PORT")
		if zPort == "" {
			zPort = "2100"
		}
		srv := NewServer(dbProvider) // Share the provider
		listener, err := net.Listen("tcp", "0.0.0.0:"+zPort)
		if err != nil {
			slog.Error("failed to start Z39.50 listener", "error", err)
			return // Don't panic main thread, but log error
		}
		slog.Info("Z39.50 server starting", "port", zPort)

		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			if !srv.checkIP(conn.RemoteAddr()) {
				conn.Close()
				continue
			}
			go srv.handleConnection(conn)
		}
	}()

	// 3. Start HTTP Gateway
	r := setupRouter(dbProvider)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8899"
	}

	slog.Info("gateway starting", "addr", ":"+port)
	r.Run(":" + port)
}