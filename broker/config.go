package broker

type Config struct {
	Host                  string `mapstructure:"host", envconfig:"BROKER_HOST"`
	Port                  int `mapstructure:"port", envconfig:"BROKER_PORT"`
	User                  string `mapstructure:"user", envconfig:"BROKER_USER"`
	Password              string `mapstructure:"password", envconfig:"BROKER_PASSWORD"`
	TaskEncodeQueueName   string `mapstructure:"taskEncodeQueue", envconfig:"BROKER_TASK_ENCODE_QUEUE"`
	TaskPGSToSrtQueueName string `mapstructure:"taskPGSQueue", envconfig:"BROKER_TASK_PGS_QUEUE"`
	TaskEventQueueName    string `mapstructure:"eventQueue", envconfig:"BROKER_EVENT_QUEUE"`
}