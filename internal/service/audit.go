package service

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type AuditLog struct {
	ID         uint      `json:"id,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	Operation  string    `json:"operation"`
	Table      string    `json:"table"`
	RecordID   string    `json:"record_id"`
	Actor      string    `json:"actor"`
	RequestID  string    `json:"request_id"`
	SQL        string    `json:"sql,omitempty"`
	BeforeData string    `json:"before_data,omitempty"`
	AfterData  string    `json:"after_data,omitempty"`
	Duration   int64     `json:"duration_ms"`
	Status     string    `json:"status"`
	ErrorMsg   string    `json:"error_msg,omitempty"`
}

type AuditContext struct {
	RequestID  string
	Operation  string
	Table      string
	RecordID   string
	Actor      string
	StartTime  time.Time
	BeforeData map[string]interface{}
	AfterData  map[string]interface{}
	SQL        string
}

type AuditService struct {
	mu    sync.Mutex
	file  *os.File
	enc   *json.Encoder
	table string
}

// NewAuditService creates a new AuditService that writes to the specified file path.
// If filePath is empty, defaults to "audit.log" in the current directory.
func NewAuditService(filePath string) *AuditService {
	if filePath == "" {
		filePath = "audit.log"
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		os.MkdirAll(dir, 0755)
	}

	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// Audit disabled if file cannot be opened
		return &AuditService{}
	}

	return &AuditService{
		file:  file,
		enc:   json.NewEncoder(file),
	}
}

// NewAuditServiceWithDB creates an AuditService with SQL capture enabled on the GORM DB.
// Registers a GORM logger to capture all executed SQL statements.
func NewAuditServiceWithDB(filePath string, db *gorm.DB) *AuditService {
	svc := NewAuditService(filePath)
	if db != nil && svc.file != nil {
		// Wrap GORM's default logger to capture SQL statements
		db.Config.Logger = &auditSQLLogger{svc: svc, prev: db.Config.Logger}
	}
	return svc
}

func (s *AuditService) Start(operation, table, recordID string) *AuditContext {
	return &AuditContext{
		RequestID:  generateRequestID(),
		Operation:  operation,
		Table:      table,
		RecordID:   recordID,
		StartTime:  time.Now(),
		BeforeData: make(map[string]interface{}),
	}
}

func (s *AuditService) Success(ctx *AuditContext, beforeData, afterData interface{}, affectedRows int64) {
	if s.file == nil {
		return
	}
	s.writeEntry(ctx, beforeData, afterData, "success", "")
}

func (s *AuditService) Fail(ctx *AuditContext, errMsg string) {
	if s.file == nil {
		return
	}
	s.writeEntry(ctx, ctx.BeforeData, nil, "failed", errMsg)
}

func (s *AuditService) writeEntry(ctx *AuditContext, beforeData, afterData interface{}, status, errMsg string) {
	entry := AuditLog{
		Timestamp:  ctx.StartTime,
		Operation:  ctx.Operation,
		Table:      ctx.Table,
		RecordID:   ctx.RecordID,
		Actor:      ctx.Actor,
		RequestID:  ctx.RequestID,
		SQL:        ctx.SQL,
		BeforeData: toJSON(beforeData),
		AfterData:  toJSON(afterData),
		Duration:   time.Since(ctx.StartTime).Milliseconds(),
		Status:     status,
		ErrorMsg:   errMsg,
	}
	s.write(entry)
}

func (s *AuditService) write(entry AuditLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.enc.Encode(entry)
}

// Close closes the audit log file.
func (s *AuditService) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

func toJSON(data interface{}) string {
	if data == nil {
		return ""
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	return string(bytes)
}

// auditSQLLogger wraps GORM's logger to capture executed SQL statements.
type auditSQLLogger struct {
	svc  *AuditService
	prev logger.Interface
}

// lastSQL stores the most recently executed SQL (thread-safe).
var lastSQL struct {
	mu  sync.Mutex
	val string
}

func (l *auditSQLLogger) LogMode(level logger.LogLevel) logger.Interface {
	if l.prev != nil {
		return l.prev.LogMode(level)
	}
	return l
}

func (l *auditSQLLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.prev != nil {
		l.prev.Info(ctx, msg, data...)
	}
}

func (l *auditSQLLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.prev != nil {
		l.prev.Warn(ctx, msg, data...)
	}
}

func (l *auditSQLLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.prev != nil {
		l.prev.Error(ctx, msg, data...)
	}
}

// Trace captures the SQL statement executed by GORM and stores it globally.
func (l *auditSQLLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	sql, rows := fc()
	_ = rows

	// Clean up the SQL string: remove newlines and extra spaces for readability
	cleanSQL := strings.Join(strings.Fields(sql), " ")

	lastSQL.mu.Lock()
	lastSQL.val = cleanSQL
	lastSQL.mu.Unlock()

	if l.prev != nil {
		l.prev.Trace(ctx, begin, fc, err)
	}
}

// GetLastSQL returns the most recently captured SQL statement.
func GetLastSQL() string {
	lastSQL.mu.Lock()
	defer lastSQL.mu.Unlock()
	return lastSQL.val
}

// CaptureSQLForContext sets the captured SQL on the given AuditContext.
func CaptureSQLForContext(ctx *AuditContext) {
	ctx.SQL = GetLastSQL()
}
