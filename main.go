package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	// synerex "github.com/fukurin00/provider_api"
	mqttApi "github.com/fukurin00/provider_robot_node/mqtt"
	msg "github.com/fukurin00/provider_robot_node/msg"
	robot "github.com/fukurin00/provider_robot_node/robot"

	sxmqtt "github.com/synerex/proto_mqtt"
	api "github.com/synerex/synerex_api"
	pbase "github.com/synerex/synerex_proto"
	sxutil "github.com/synerex/synerex_sxutil"
	"google.golang.org/protobuf/proto"
)

var (
	broker     *string = flag.String("mqtt", "127.0.0.1", "mqtt broker address")
	port       *int    = flag.Int("port", 1883, "mqtt broker port")
	mqttClient *mqtt.Client

	nodesrv      = flag.String("nodesrv", "127.0.0.1:9990", "node serv address")
	robotID *int = flag.Int("robotID", 1, "robotID")

	mu sync.Mutex

	syMqttClient *sxutil.SXServiceClient
	routeClient  *sxutil.SXServiceClient

	robotList       map[int]*robot.RobotStatus
	sxServerAddress string
	// synerexConfig *synerex.SynerexConfig
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
					cout := api.Content{Entity: out}
					smo := sxutil.SupplyOpts{
						Name:  "robotRoute",
						Cdata: &cout,
					}
					_, err = routeClient.NotifySupply(&smo)
					if err != nil {
						log.Print(err)
						reconnectClient(clt)
					}
				} else {
					log.Printf("robot %d not exist", id)
				}
			} else if strings.HasPrefix(rcd.Topic, "robot/pose") {
				var id int

				fmt.Sscanf(rcd.Topic, "robot/pose/%d", &id)
				if _, ok := robotList[id]; !ok {
					robotList[id] = robot.NewRobot(id)
				}
				robotList[id].UpdatePose(rcd)

				var pose msg.Pose
				var odom msg.Odometry
				err := json.Unmarshal(rcd.Record, &odom)
				pose = odom.Pose.Pose
				out := robotList[id].NewPoseMQTT(pose)
				sout, err := proto.Marshal(out)
				cout := api.Content{Entity: sout}
				smo := sxutil.SupplyOpts{
					Name:  "robotRoute",
					Cdata: &cout,
				}
				_, err = syMqttClient.NotifySupply(&smo)
				if err != nil {
					log.Print(err)
					reconnectClient(clt)
				}
			}
		}
	}
}

func reconnectClient(client *sxutil.SXServiceClient) {
	mu.Lock()
	if client.SXClient != nil {
		client.SXClient = nil
		log.Printf("Client reset \n")
	}
	mu.Unlock()
	time.Sleep(5 * time.Second) // wait 5 seconds to reconnect
	mu.Lock()
	if client.SXClient == nil {
		newClt := sxutil.GrpcConnectServer(sxServerAddress)
		if newClt != nil {
			// log.Printf("Reconnect server [%s]\n", s.SxServerAddress)
			client.SXClient = newClt
		}
	}
	mu.Unlock()
}

func subsclibeMQTTSupply(client *sxutil.SXServiceClient) {
	ctx := context.Background()
	for {
		client.SubscribeSupply(ctx, mqttCallback)
		reconnectClient(client)
	}
}

func main() {
	robotList = make(map[int]*robot.RobotStatus)

	robotList[1] = robot.NewRobot(1)
	robot1 := robotList[1]
	robot1.Radius = 0.3
	robotList[2] = robot.NewRobot(2)
	robot2 := robotList[2]
	robot2.Radius = 0.3
	robotList[3] = robot.NewRobot(3)
	robot3 := robotList[3]
	robot3.Radius = 0.25

	go sxutil.HandleSigInt()
	wg := sync.WaitGroup{}
	flag.Parse()
	sxutil.RegisterDeferFunction(sxutil.UnRegisterNode)

	mqttClient = mqttApi.ConnectMqttBroker(*broker, *port, connectHandler, connectLostHandler, messagePubHandler)
	channels := []uint32{pbase.MQTT_GATEWAY_SVC, pbase.ROUTING_SERVICE}
	srv, err := sxutil.RegisterNode(*nodesrv, "RobotProvider", channels, nil)
	if err != nil {
		log.Fatal("can not registar node")
	}
	log.Printf("connectiong server [%s]", srv)

	sxServerAddress = srv

	synerexClient := sxutil.GrpcConnectServer(srv)
	argJson1 := fmt.Sprintf("{Client: RobotMQTT}")
	syMqttClient = sxutil.NewSXServiceClient(synerexClient, pbase.MQTT_GATEWAY_SVC, argJson1)
	argJson2 := fmt.Sprintf("{Client: RobotRoute}")
	routeClient = sxutil.NewSXServiceClient(synerexClient, pbase.ROUTING_SERVICE, argJson2)
	// names := []string{"ROBOT_MQTT", "ROBOT_ROUTING"

	go subsclibeMQTTSupply(syMqttClient)

	wg.Add(1)
	wg.Wait()
}
