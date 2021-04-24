package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	synerex "github.com/fukurin00/provider_api/src/api"
	mqttApi "github.com/fukurin00/provider_robot_node/mqtt"
	msg "github.com/fukurin00/provider_robot_node/msg"
	robot "github.com/fukurin00/provider_robot_node/robot"
	sxmqtt "github.com/synerex/proto_mqtt"
	api "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"
	"google.golang.org/protobuf/proto"
)

var (
	broker  *string = flag.String("mqtt", "127.0.0.1", "mqtt broker address")
	port    *int    = flag.Int("port", 1883, "mqtt broker port")
	client  *mqtt.Client
	robotID *int = flag.Int("robotID", 1, "robotID")

	robotList map[int]*robot.RobotStatus
)

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	log.Printf("Received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	log.Print("MQTT Connected")
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	log.Printf("MQTT Connect lost: %v", err)
}

func mqttCallback(clt *sxutil.SXServiceClient, sp *api.Supply) {
	//from MQTT broker
	if sp.SenderId == uint64(clt.ClientID) {
		// ignore my message.
		return
	}
	rcd := &sxmqtt.MQTTRecord{}
	err := proto.Unmarshal(sp.Cdata.Entity, rcd)
	if err == nil {
		if strings.HasPrefix(rcd.Topic, "robot/") {
			if strings.HasPrefix(rcd.Topic, "robot/dest") {
				// var p msg.Path
				// var id int

				// err := json.Unmarshal(rcd.Record, &p)
				// if err != nil {
				// 	log.Print(err)
				// }
				// fmt.Sscanf(rcd.Topic, "robot/path/%d", &id)

				// if rob, ok := robotList[id]; ok {
				// 	rob.UpdatePath(rcd)
				// }
			} else if strings.HasPrefix(rcd.Topic, "robot/pose") {
				var pose msg.ROS_PoseStamped
				var id int

				err := json.Unmarshal(rcd.Record, &pose)
				if err != nil {
					log.Print(err)
				}
				fmt.Sscanf(rcd.Topic, "robot/pose/%d", &id)
				if rob, ok := robotList[id]; ok {
					rob.UpdatePose(rcd)
				} else {
					robotList[id] = robot.NewRobot(id)
				}
			}
		}
	}
}

func main() {
	robotList = make(map[int]*robot.RobotStatus)
	wg := sync.WaitGroup{}
	flag.Parse()
	client = mqttApi.ConnectMqttBroker(*broker, *port, connectHandler, connectLostHandler, messagePubHandler)
	channels := []uint32{synerex.MQTT_GATEWAY_SVC}
	names := []string{"ROBOT_MQTT"}
	s, err := synerex.NewSynerexConfig("RobotNode", channels, names)
	if err != nil {
		log.Print(err)
	}
	s.SubscribeSupply(synerex.MQTT_GATEWAY_SVC, mqttCallback)
	wg.Add(1)
	wg.Wait()
}
