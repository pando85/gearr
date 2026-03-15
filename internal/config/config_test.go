package config

import (
	"testing"
)

type QueueConfig struct {
	QueueType       string `mapstructure:"queueType"`
	PostgresConnStr string `mapstructure:"postgresConnStr"`
}

func ValidateConfig(config QueueConfig) error {
	if config.QueueType == "postgres" && config.PostgresConnStr == "" {
		return &ValidationError{
			Field:   "postgresConnStr",
			Message: "PostgreSQL connection string is required when queue type is 'postgres'",
		}
	}
	return nil
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    QueueConfig
		wantErr   bool
		errString string
	}{
		{
			name: "valid postgres config",
			config: QueueConfig{
				QueueType:       "postgres",
				PostgresConnStr: "postgres://user:pass@localhost:5432/db",
			},
			wantErr: false,
		},
		{
			name: "missing postgres connection string",
			config: QueueConfig{
				QueueType:       "postgres",
				PostgresConnStr: "",
			},
			wantErr:   true,
			errString: "PostgreSQL connection string is required when queue type is 'postgres'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errString != "" {
				if err.Error() != tt.errString {
					t.Errorf("ValidateConfig() error = %v, want %v", err.Error(), tt.errString)
				}
			}
		})
	}
}
