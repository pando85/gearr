package config

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type PriorityConfig struct {
	PriorityBySize  bool `mapstructure:"priorityBySize"`
	PriorityByAge   bool `mapstructure:"priorityByAge"`
	DefaultPriority int  `mapstructure:"defaultPriority"`
}

func DefaultPriorityConfig() PriorityConfig {
	return PriorityConfig{
		PriorityBySize:  false,
		PriorityByAge:   false,
		DefaultPriority: 5,
	}
}

func ValidatePriorityConfig(config PriorityConfig) error {
	if config.DefaultPriority < 1 || config.DefaultPriority > 10 {
		return &ValidationError{
			Field:   "defaultPriority",
			Message: "defaultPriority must be between 1 and 10",
		}
	}
	return nil
}
