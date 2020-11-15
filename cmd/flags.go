package cmd

import (
	"github.com/spf13/pflag"
)

func BrokerFlags(){
	pflag.String("broker.host","localhost" ,"Broker Host")
	pflag.Int("broker.port",5672 ,"WebServer Port")
	pflag.String("broker.user","broker" ,"Broker User")
	pflag.String("broker.password","broker" ,"Broker User")
	pflag.String("broker.taskEncodeQueue","tasks" ,"Broker tasks queue name")
	pflag.String("broker.taskPGSQueue","tasks_pgstosrt" ,"Broker tasks pgstosrt queue name")
	pflag.String("broker.eventQueue","task_events" ,"Broker tasks events queue name")


}