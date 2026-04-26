package services

import "mcp-overwatch/internal/logging"

// LogService exposes log reading to the frontend.
type LogService struct {
	logger *logging.Logger
}

// NewLogService creates a LogService backed by the given logger.
func NewLogService(logger *logging.Logger) *LogService {
	return &LogService{logger: logger}
}

// GetRecentLogs returns the most recent log entries.
func (s *LogService) GetRecentLogs(count int) []logging.Entry {
	return s.logger.Recent(count)
}
