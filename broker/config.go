package broker

type Config struct {
	Type                   string `mapstructure:"type"`
	Host                   string `mapstructure:"host"`
	Port                   int    `mapstructure:"port"`
	User                   string `mapstructure:"user"`
	Password               string `mapstructure:"password"`
	TaskEncodeQueueName    string `mapstructure:"taskEncodeQueue"`
	DeleteSourceOnComplete bool   `mapstructure:"deleteSourceOnComplete"`
	TaskPGSToSrtQueueName  string `mapstructure:"taskPGSQueue"`
	TaskEventQueueName     string `mapstructure:"eventQueue"`
}
