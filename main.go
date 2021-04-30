package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	synerex "github.com/fukurin00/provider_api"
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

	robotList     map[int]*robot.RobotStatus
	synerexConfig *synerex.SynerexConfig
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
				var p msg.ROS_PoseStamped
				var id int

				err := json.Unmarshal(rcd.Record, &p)
				if err != nil {
					log.Print(err)
				}
				fmt.Sscanf(rcd.Topic, "robot/dest/%d", &id)

				if rob, ok := robotList[id]; ok {
					d := rob.NewDestRequest(robot.CavPoint(p), p.Header.Stamp)
					out, err := proto.Marshal(d)
					if err != nil {
						log.Print(err)
					}
					_, err = synerexConfig.NotifySupply(out, synerex.ROUTING_SERVICE, "DestDemand")
					if err != nil {
						log.Print(err)
						synerexConfig.ReconnectClient(clt)
					}
				} else {
					log.Printf("robot %d not exist", id)
				}
			} else if strings.HasPrefix(rcd.Topic, "robot/pose") {
				var pose msg.ROS_PoseStamped
				var id int

				err := json.Unmarshal(rcd.Record, &pose)
				if err != nil {
					log.Print(err)
				}
				fmt.Sscanf(rcd.Topic, "robot/pose/%d", &id)
				if _, ok := robotList[id]; !ok {
					robotList[id] = robot.NewRobot(id)
				}
				// pmsg := robotList[id].NewPoseMessage(pose)
				// out, err = proto.Marshal(pmsg)
				// if err != nil {
				// 	log.Print(err)
				// }
				// _, err = synerexConfig.NotifySupply(out, synerex.MQTT_GATEWAY_SVC, "PoseSupply")
				// if err != nil {
				// 	log.Print(err)
				// 	synerexConfig.ReconnectClient(clt)
				// }
			}
		}
	}
}

func main() {
	robotList = make(map[int]*robot.RobotStatus)
	wg := sync.WaitGroup{}
	flag.Parse()
	client = mqttApi.ConnectMqttBroker(*broker, *port, connectHandler, connectLostHandler, messagePubHandler)
	channels := []uint32{synerex.MQTT_GATEWAY_SVC, synerex.ROUTING_SERVICE}
	names := []string{"ROBOT_MQTT", "ROBOT_ROUTING"}
	synerexConfig, err := synerex.NewSynerexConfig("RobotNode", channels, names)
	if err != nil {
		log.Print(err)
	}
	synerexConfig.SubscribeSupply(synerex.MQTT_GATEWAY_SVC, mqttCallback)
	wg.Add(1)
	wg.Wait()
}
