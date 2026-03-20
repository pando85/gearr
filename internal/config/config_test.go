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

func TestValidatePriorityConfig(t *testing.T) {
	tests := []struct {
		name      string
		config    PriorityConfig
		wantErr   bool
		errString string
	}{
		{
			name: "valid default priority config",
			config: PriorityConfig{
				PriorityBySize:  false,
				PriorityByAge:   false,
				DefaultPriority: 5,
			},
			wantErr: false,
		},
		{
			name: "valid priority config with size priority enabled",
			config: PriorityConfig{
				PriorityBySize:  true,
				PriorityByAge:   false,
				DefaultPriority: 5,
			},
			wantErr: false,
		},
		{
			name: "valid priority config with age priority enabled",
			config: PriorityConfig{
				PriorityBySize:  false,
				PriorityByAge:   true,
				DefaultPriority: 5,
			},
			wantErr: false,
		},
		{
			name: "valid priority config with both priorities enabled",
			config: PriorityConfig{
				PriorityBySize:  true,
				PriorityByAge:   true,
				DefaultPriority: 10,
			},
			wantErr: false,
		},
		{
			name: "invalid priority below range",
			config: PriorityConfig{
				PriorityBySize:  false,
				PriorityByAge:   false,
				DefaultPriority: 0,
			},
			wantErr:   true,
			errString: "defaultPriority must be between 1 and 10",
		},
		{
			name: "invalid priority above range",
			config: PriorityConfig{
				PriorityBySize:  false,
				PriorityByAge:   false,
				DefaultPriority: 11,
			},
			wantErr:   true,
			errString: "defaultPriority must be between 1 and 10",
		},
		{
			name: "valid minimum priority",
			config: PriorityConfig{
				PriorityBySize:  false,
				PriorityByAge:   false,
				DefaultPriority: 1,
			},
			wantErr: false,
		},
		{
			name: "valid maximum priority",
			config: PriorityConfig{
				PriorityBySize:  false,
				PriorityByAge:   false,
				DefaultPriority: 10,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePriorityConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePriorityConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errString != "" {
				if err.Error() != tt.errString {
					t.Errorf("ValidatePriorityConfig() error = %v, want %v", err.Error(), tt.errString)
				}
			}
		})
	}
}

func TestDefaultPriorityConfig(t *testing.T) {
	config := DefaultPriorityConfig()
	if config.PriorityBySize != false {
		t.Errorf("DefaultPriorityConfig() PriorityBySize = %v, want false", config.PriorityBySize)
	}
	if config.PriorityByAge != false {
		t.Errorf("DefaultPriorityConfig() PriorityByAge = %v, want false", config.PriorityByAge)
	}
	if config.DefaultPriority != 5 {
		t.Errorf("DefaultPriorityConfig() DefaultPriority = %v, want 5", config.DefaultPriority)
	}
}
