package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	ber "github.com/go-asn1-ber/asn1-ber"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/yourusername/open-z3950-gateway/pkg/auth"
	"github.com/yourusername/open-z3950-gateway/pkg/notify"
	"github.com/yourusername/open-z3950-gateway/pkg/provider"
	"github.com/yourusername/open-z3950-gateway/pkg/sip2"
	"github.com/yourusername/open-z3950-gateway/pkg/telemetry"
	"github.com/yourusername/open-z3950-gateway/pkg/ai"
	"github.com/yourusername/open-z3950-gateway/pkg/ui"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"

	_ "github.com/yourusername/open-z3950-gateway/docs" // Import generated docs
	"github.com/yourusername/open-z3950-gateway/gen/proto/gateway/v1/gatewayv1connect"
)

// @title           Open Z39.50 Gateway API
// @version         1.0
// @description     A modern Z39.50 Gateway with REST API and Web UI.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8899
// @BasePath  /api

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

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
		if err != nil {
			return nil, err
		}

		right, err := recursiveParseRPN(p.Children[1])
		if err != nil {
			return nil, err
		}

		opNode := p.Children[2]
		opStr := "AND"
		if len(opNode.Children) > 0 {
			if v, ok := opNode.Children[0].Value.(int64); ok {
				switch v {
				case 0:
					opStr = "AND"
				case 1:
					opStr = "OR"
				case 2:
					opStr = "AND-NOT"
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
	var rpnStruct *ber.Packet

	if len(queryPacket.Children) > 0 {
		// RPNQuery
		rpnQuery := queryPacket.Children[0]
		if len(rpnQuery.Children) >= 2 {
			rpnStruct = rpnQuery.Children[1]
		}
	}

	if rpnStruct == nil {
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
		if c.Tag == 29 {
			if v, ok := c.Value.(int64); ok {
				reqCount = int(v)
			}
		}
		if c.Tag == 30 {
			if v, ok := c.Value.(int64); ok {
				startPoint = int(v)
			}
		}
	}

	s.mu.RLock()
	sess, ok := s.sessions[connID]
	s.mu.RUnlock()
	if !ok {
		return
	}

	ids := sess.ResultIDs
	startIdx := startPoint - 1
	if startIdx < 0 {
		startIdx = 0
	}
	endIdx := startIdx + reqCount
	if endIdx > len(ids) {
		endIdx = len(ids)
	}

	var subsetIDs []string
	if startIdx < len(ids) {
		subsetIDs = ids[startIdx:endIdx]
	}

	records, _ := s.provider.Fetch(sess.DBName, subsetIDs)
	slog.Info("present processed", "conn_id", connID, "returned", len(records))

	profile := &z3950.ProfileMARC21
	if strings.Contains(strings.ToUpper(sess.DBName), "CNMARC") {
		profile = &z3950.ProfileCNMARC
	}
	if strings.Contains(strings.ToUpper(sess.DBName), "UNIMARC") {
		profile = &z3950.ProfileUNIMARC
	}

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
	opts := z3950.ScanOptions{
		Count:          10, // Default if not found
		StepSize:       0,
		PositionOfTerm: 1,
	}

	var walk func(*ber.Packet)
	walk = func(p *ber.Packet) {
		if p.Tag == 45 { // Term
			if val, ok := p.Value.([]byte); ok {
				term = string(val)
			} else {
				term = string(p.Data.Bytes())
			}
		}
		if p.Tag == 31 { // NumberOfTermsRequested
			if v, ok := p.Value.(int64); ok {
				opts.Count = int(v)
			}
		}
		if p.Tag == 32 { // StepSize
			if v, ok := p.Value.(int64); ok {
				opts.StepSize = int(v)
			}
		}
		if p.Tag == 33 { // PositionOfTerm
			if v, ok := p.Value.(int64); ok {
				opts.PositionOfTerm = int(v)
			}
		}
		for _, c := range p.Children {
			walk(c)
		}
	}
	walk(req)

	field := "title"

	s.mu.RLock()
	sess, ok := s.sessions[connID]
	s.mu.RUnlock()
	dbName := "Default"
	if ok {
		dbName = sess.DBName
	}

	results, _ := s.provider.Scan(dbName, field, term, opts)
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

// --- Error Handling ---

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Detail)
}

func AbortWithError(c *gin.Context, code int, message string, err error) {
	detail := ""
	if err != nil {
		detail = err.Error()
		if strings.Contains(detail, "i/o timeout") {
			detail = "Connection timed out. The remote server might be down or blocked."
		} else if strings.Contains(detail, "connection refused") {
			detail = "Connection refused. The remote server is not accepting connections."
		} else if strings.Contains(detail, "server rejected connection") {
			detail = "Server rejected connection. Possible authentication or protocol issue."
		}
	}

	slog.Error("api error", "path", c.Request.URL.Path, "status", code, "message", message, "error", err)

	c.AbortWithStatusJSON(code, gin.H{
		"status":  "error",
		"error":   message,
		"detail":  detail,
		"code":    code,
		"traceId": c.GetString("TraceID"),
	})
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

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized: Invalid API Key or Token",
		})
	}
}

// LoginRequest credentials
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse auth token
type LoginResponse struct {
	Status string            `json:"status"`
	Token  string            `json:"token"`
	User   map[string]string `json:"user"`
}

func setupRouter(dbProvider provider.Provider) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// --- 0. OpenTelemetry Middleware ---
	r.Use(otelgin.Middleware("gateway"))

	// --- 1. CORS Configuration ---
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Allow all for development
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	notifier := notify.NewLogNotifier()

	// --- 2. Health Check ---
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP", "time": time.Now()})
	})

	// --- 3. gRPC / ConnectRPC Handler ---
	gatewayServer := NewGatewayServer(dbProvider)
	path, handler := gatewayv1connect.NewGatewayServiceHandler(gatewayServer)
	r.Any(path+"*any", gin.WrapH(handler))

	// --- 4. SIP2 Client Setup ---
	sipHost := os.Getenv("SIP2_HOST")
	sipPort := os.Getenv("SIP2_PORT")

	if os.Getenv("SIP2_MOCK") == "true" || sipHost == "" {
		slog.Info("Starting Mock SIP2 Server on :6001")
		mock := sip2.NewMockServer(6001)
		mock.Start()
		sipHost = "localhost"
		sipPort = "6001"
	}

	var sipClient *sip2.SIP2Client
	if sipHost != "" {
		port, _ := strconv.Atoi(sipPort)
		sipClient = sip2.NewClient(sipHost, port)
		sipClient.Location = "MainBranch" // Default
	}

	// ILS Routes
	ils := r.Group("/api/ils")
	ils.POST("/login", func(c *gin.Context) {
		if sipClient == nil {
			c.JSON(533, gin.H{"error": "ILS integration not configured"})
			return
		}
		var req struct {
			Barcode  string `json:"barcode"`
			Password string `json:"password"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid request"})
			return
		}

		if err := sipClient.Connect(); err != nil {
			c.JSON(500, gin.H{"error": "Failed to connect to ILS"})
			return
		}
		defer sipClient.Close()

		ok, err := sipClient.Login(req.Barcode, req.Password, "MainBranch")
		if err != nil || !ok {
			c.JSON(401, gin.H{"error": "Invalid barcode or password"})
			return
		}

		c.JSON(200, gin.H{"status": "success", "message": "Logged in to Library System"})
	})

	ils.GET("/profile", func(c *gin.Context) {
		if sipClient == nil {
			c.JSON(533, gin.H{"error": "ILS integration not configured"})
			return
		}
		barcode := c.Query("barcode")

		if err := sipClient.Connect(); err != nil {
			c.JSON(500, gin.H{"error": "Failed to connect to ILS"})
			return
		}
		defer sipClient.Close()

		info, err := sipClient.GetPatronInfo(barcode, "")
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch profile: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "success", "data": info})
	})

	// Public Auth Routes
	r.POST("/api/auth/login", func(c *gin.Context) {
		var creds LoginRequest
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

		c.JSON(200, LoginResponse{
			Status: "success",
			Token:  token,
			User:   map[string]string{"username": user.Username, "role": user.Role},
		})
	})

	r.POST("/api/auth/register", func(c *gin.Context) {
		var req LoginRequest
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

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

	// --- Discovery APIs (LSP Phase 5) ---
	api.GET("/discovery/new", func(c *gin.Context) {
		limit := 10
		if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 { limit = l }
		
		results, err := dbProvider.GetNewArrivals(limit)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch new arrivals"})
			return
		}
		c.JSON(200, gin.H{"status": "success", "data": results})
	})

	api.GET("/discovery/popular", func(c *gin.Context) {
		limit := 10
		if l, err := strconv.Atoi(c.Query("limit")); err == nil && l > 0 { limit = l }
		
		results, err := dbProvider.GetPopularBooks(limit)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch popular books"})
			return
		}
		c.JSON(200, gin.H{"status": "success", "data": results})
	})

	// --- Cataloging APIs (LSP) ---
	api.POST("/books", func(c *gin.Context) {
		var rec z3950.MARCRecord
		if err := c.BindJSON(&rec); err != nil {
			c.JSON(400, gin.H{"error": "Invalid MARC record JSON"})
			return
		}
		// Default target for new records
		db := c.DefaultQuery("db", "Default")
		id, err := dbProvider.CreateRecord(db, &rec)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create record: " + err.Error()})
			return
		}
		c.JSON(201, gin.H{"status": "success", "id": id})
	})

	// --- Circulation APIs (LSP Phase 2) ---
	
	api.POST("/items", func(c *gin.Context) {
		var req struct {
			BibID      string `json:"bib_id"`
			Barcode    string `json:"barcode"`
			CallNumber string `json:"call_number"`
			Location   string `json:"location"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid item JSON"})
			return
		}
		item := provider.Item{
			Barcode:    req.Barcode,
			CallNumber: req.CallNumber,
			Location:   req.Location,
		}
		if err := dbProvider.CreateItem(req.BibID, item); err != nil {
			c.JSON(500, gin.H{"error": "Failed to create item: " + err.Error()})
			return
		}
		c.JSON(201, gin.H{"status": "success"})
	})

	api.POST("/circulation/checkout", func(c *gin.Context) {
		var req struct {
			Barcode  string `json:"barcode"`
			PatronID string `json:"patron_id"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid checkout request"})
			return
		}
		dueDate, err := dbProvider.Checkout(req.Barcode, req.PatronID)
		if err != nil {
			c.JSON(400, gin.H{"error": "Checkout failed: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "success", "due_date": dueDate})
	})

	api.POST("/circulation/checkin", func(c *gin.Context) {
		var req struct {
			Barcode string `json:"barcode"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid checkin request"})
			return
		}
		fine, err := dbProvider.Checkin(req.Barcode)
		if err != nil {
			c.JSON(400, gin.H{"error": "Checkin failed: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "success", "fine": fine})
	})

	api.PUT("/books/:db/:id", func(c *gin.Context) {
		db := c.Param("db")
		id := c.Param("id")
		var req struct {
			Fields []z3950.MARCField `json:"fields"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid fields JSON"})
			return
		}

		// 1. Fetch existing
		recs, err := dbProvider.Fetch(db, []string{id})
		if err != nil || len(recs) == 0 {
			c.JSON(404, gin.H{"error": "Record not found"})
			return
		}

		// 2. Update fields
		record := recs[0]
		record.UpdateFields(req.Fields)

		// 3. Save back
		if err := dbProvider.UpdateRecord(db, id, record); err != nil {
			c.JSON(500, gin.H{"error": "Update failed: " + err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "success"})
	})

	api.GET("/books/:db/:id/ai-insight", func(c *gin.Context) {
		db := c.Param("db")
		id := c.Param("id")

		recs, err := dbProvider.Fetch(db, []string{id})
		if err != nil || len(recs) == 0 {
			c.JSON(404, gin.H{"error": "Record not found"})
			return
		}

		librarian, err := ai.NewLibrarian(c.Request.Context())
		if err != nil {
			c.JSON(503, gin.H{"error": "AI Service not available: " + err.Error()})
			return
		}

		insight, err := librarian.GetInsight(c.Request.Context(), recs[0])
		if err != nil {
			c.JSON(500, gin.H{"error": "AI Analysis failed: " + err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "success", "insight": insight})
	})

	api.GET("/search", func(c *gin.Context) {
		start := time.Now()
		db := c.DefaultQuery("db", "LCDB")

		var root z3950.QueryNode
		term1 := c.Query("term1")
		if term1 == "" {
			term1 = c.Query("query")
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

		for i := 2; ; i++ {
			termKey := fmt.Sprintf("term%d", i)
			term, exists := c.GetQuery(termKey)
			if !exists {
				break
			}

			attrKey := fmt.Sprintf("attr%d", i)
			attrStr := c.Query(attrKey)
			attr := z3950.UseAttributeAny
			if attrStr != "" {
				attr, _ = strconv.Atoi(attrStr)
			}

			opKey := fmt.Sprintf("op%d", i)
			operator := c.DefaultQuery(opKey, "AND")

			root = z3950.QueryComplex{
				Operator: operator,
				Left:     root,
				Right:    z3950.QueryClause{Attribute: attr, Term: term},
			}
		}

		sortAttrStr := c.Query("sortAttr")
		sortOrderStr := c.Query("sortOrder")

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

		ids, err := dbProvider.Search(db, structuredQuery)
		if err != nil {
			AbortWithError(c, http.StatusBadGateway, "Search failed", err)
			return
		}

		records, err := dbProvider.Fetch(db, ids)
		if err != nil {
			AbortWithError(c, http.StatusBadGateway, "Fetch failed", err)
			return
		}

		results := make([]map[string]interface{}, 0)
		for _, rec := range records {
			if rec.Leader == "SUTRS" {
				txt := ""
				if len(rec.Fields) > 0 {
					txt = rec.Fields[0].Value
				}
				results = append(results, map[string]interface{}{
					"title":  "Text Record",
					"raw":    txt,
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

	api.GET("/federated-search", func(c *gin.Context) {
		start := time.Now()
		targetsStr := c.Query("targets")
		queryTerm := c.Query("query")
		if targetsStr == "" || queryTerm == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'targets' or 'query' parameter"})
			return
		}

		limit := 5
		if l := c.Query("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 {
				limit = v
			}
		}

		targetList := strings.Split(targetsStr, ",")

		var wg sync.WaitGroup
		resultsChan := make(chan map[string]interface{}, len(targetList)*limit)

		zQuery := z3950.StructuredQuery{
			Root: z3950.QueryClause{Attribute: z3950.UseAttributeAny, Term: queryTerm},
		}

		for _, dbName := range targetList {
			dbName = strings.TrimSpace(dbName)
			if dbName == "" {
				continue
			}

			wg.Add(1)
			go func(target string) {
				defer wg.Done()

				ids, err := dbProvider.Search(target, zQuery)
				if err != nil {
					slog.Warn("federated search error", "target", target, "error", err)
					return
				}

				if len(ids) == 0 {
					return
				}

				fetchCount := limit
				if len(ids) < fetchCount {
					fetchCount = len(ids)
				}
				idsToFetch := ids[:fetchCount]

				records, err := dbProvider.Fetch(target, idsToFetch)
				if err != nil {
					slog.Warn("federated fetch error", "target", target, "error", err)
					return
				}

				for _, rec := range records {
					res := map[string]interface{}{
						"source_target": target,
						"record_id":     rec.RecordID,
						"title":         rec.GetTitle(nil),
						"author":        rec.GetAuthor(nil),
						"isbn":          rec.GetISBN(nil),
						"publisher":     rec.GetPublisher(nil),
						"year":          rec.GetPubYear(nil),
					}
					resultsChan <- res
				}
			}(dbName)
		}

		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		aggregated := make([]map[string]interface{}, 0)
		for res := range resultsChan {
			aggregated = append(aggregated, res)
		}

		elapsed := time.Since(start)
		slog.Info("federated search completed",
			"targets", len(targetList),
			"total_found", len(aggregated),
			"latency_ms", elapsed.Milliseconds(),
		)

		c.JSON(200, gin.H{
			"status":  "success",
			"count":   len(aggregated),
			"data":    aggregated,
			"time_ms": elapsed.Milliseconds(),
		})
	})

	api.GET("/books/:db/:id", func(c *gin.Context) {
		db := c.Param("db")
		id := c.Param("id")

		records, err := dbProvider.Fetch(db, []string{id})
		if err != nil {
			AbortWithError(c, http.StatusBadGateway, "Failed to fetch book details", err)
			return
		}

		if len(records) == 0 {
			AbortWithError(c, http.StatusNotFound, "Book not found", nil)
			return
		}

		rec := records[0]
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

		if username, exists := c.Get("username"); exists {
			req.Requestor = username.(string)
		} else {
			req.Requestor = "anonymous"
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

		opts := z3950.ScanOptions{
			Count:          20,
			PositionOfTerm: 1,
			StepSize:       0,
		}

		if v, err := strconv.Atoi(c.Query("count")); err == nil && v > 0 {
			opts.Count = v
		}
		if v, err := strconv.Atoi(c.Query("position")); err == nil && v > 0 {
			opts.PositionOfTerm = v
		}
		if v, err := strconv.Atoi(c.Query("step")); err == nil {
			opts.StepSize = v
		}

		results, err := dbProvider.Scan(db, field, term, opts)
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

		existingReq, err := dbProvider.GetILLRequest(id)
		if err != nil {
			c.JSON(404, gin.H{"error": "Request not found"})
			return
		}

		if err := dbProvider.UpdateILLRequestStatus(id, body.Status); err != nil {
			slog.Error("failed to update ILL request status", "id", id, "status", body.Status, "error", err)
			c.JSON(500, gin.H{"error": "Failed to update status: " + err.Error()})
			return
		}

		slog.Info("ILL request status updated", "id", id, "status", body.Status)

		toEmail := existingReq.Requestor
		if !strings.Contains(toEmail, "@") {
			toEmail = toEmail + "@example.com"
		}

		go notifier.SendILLStatusUpdate(toEmail, existingReq.Title, body.Status, "Status updated by admin")

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

	admin.GET("/stats", func(c *gin.Context) {
		stats, err := dbProvider.GetDashboardStats()
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch stats"})
			return
		}
		c.JSON(200, gin.H{"status": "success", "data": stats})
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

	spaHandler := ui.SPAHandler()
	r.NoRoute(func(c *gin.Context) {
		spaHandler.ServeHTTP(c.Writer, c.Request)
	})

	return r
}

func main() {
	// Initialize Tracer
	shutdownTracer, err := telemetry.InitTracer(context.Background(), "gateway-service")
	if err != nil {
		slog.Warn("failed to init tracer", "error", err)
	} else {
		defer shutdownTracer(context.Background())
	}

	initLogger()

	slog.Info("running MARC self-test")
	testBlob := z3950.BuildMARC(nil, "001", "Test Title", "Test Author", "123456", "Test Publisher", "2026", "1234-5678", "Test Subject")
	if parsed, err := z3950.ParseMARC(testBlob); err != nil {
		slog.Error("self-test failed", "error", err, "hex", hex.EncodeToString(testBlob))
	} else {
		slog.Info("self-test passed", "title", parsed.GetTitle(nil), "fields", len(parsed.Fields))
	}

	var dbProvider provider.Provider

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

	dbProvider = provider.NewHybridProvider(dbProvider)

	// --- 2. Start Background Automation Engine ---
	emailSvc := notify.NewEmailService()
	go func() {
		slog.Info("starting automation engine: overdue checker")
		ticker := time.NewTicker(1 * time.Hour) // Check every hour
		for range ticker.C {
			stats, err := dbProvider.GetDashboardStats()
			if err != nil { continue }
			
			// If overdue_loans > 0, we should fetch and notify
			// For simplicity in this MVP, let's just log. 
			// In production, we'd call dbProvider.ListOverdueLoans()
			if count, ok := stats["overdue_loans"].(int); ok && count > 0 {
				slog.Info("automation: found overdue items, sending notifications...", "count", count)
				// emailSvc.SendOverdueNotice(...) // Loop and send
			}
		}
	}()

	zPort := os.Getenv("ZSERVER_PORT")
	if zPort == "" {
		zPort = "2100"
	}
	srv := NewServer(dbProvider)
	zListener, err := net.Listen("tcp", "0.0.0.0:"+zPort)
	if err != nil {
		slog.Error("failed to start Z39.50 listener", "error", err)
	} else {
		slog.Info("Z39.50 server starting", "port", zPort)
		go func() {
			for {
				conn, err := zListener.Accept()
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
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8899"
	}

	router := setupRouter(dbProvider)
	httpSrv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		slog.Info("gateway starting", "addr", ":"+port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("gateway listen failed", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("shutting down gateway...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		slog.Error("gateway forced to shutdown", "error", err)
	}

	if zListener != nil {
		zListener.Close()
	}

	slog.Info("gateway exiting")
}