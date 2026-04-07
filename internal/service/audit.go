package service

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID         uint      `gorm:"primaryKey"`
	Timestamp  time.Time `gorm:"index"`
	Operation  string    `gorm:"size:20;index"`
	Table      string    `gorm:"size:100;index"`
	RecordID   string    `gorm:"size:100;index"`
	Actor      string    `gorm:"size:100"`
	RequestID  string    `gorm:"size:50;index"`
	BeforeData string    `gorm:"type:text"`
	AfterData  string    `gorm:"type:text"`
	Duration   int64
	Status     string    `gorm:"size:20;index"`
	ErrorMsg   string    `gorm:"type:text"`
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
}

type AuditService struct {
	repo  interface{}
	Table string
}

func NewAuditService(repo interface{}, table string) *AuditService {
	return &AuditService{
		repo:  repo,
		Table: table,
	}
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
	duration := time.Since(ctx.StartTime).Milliseconds()

	_ = AuditLog{
		Timestamp:  ctx.StartTime,
		Operation:  ctx.Operation,
		Table:      ctx.Table,
		RecordID:   ctx.RecordID,
		Actor:      ctx.Actor,
		RequestID:  ctx.RequestID,
		BeforeData: toJSON(ctx.BeforeData),
		AfterData:  toJSON(ctx.AfterData),
		Duration:   duration,
		Status:     "success",
	}
}

func (s *AuditService) Fail(ctx *AuditContext, errMsg string) {
	duration := time.Since(ctx.StartTime).Milliseconds()

	_ = AuditLog{
		Timestamp:  ctx.StartTime,
		Operation:  ctx.Operation,
		Table:      ctx.Table,
		RecordID:   ctx.RecordID,
		Actor:      ctx.Actor,
		RequestID:  ctx.RequestID,
		BeforeData: toJSON(ctx.BeforeData),
		Duration:   duration,
		Status:     "failed",
		ErrorMsg:   errMsg,
	}
}

func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
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
