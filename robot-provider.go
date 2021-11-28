package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	// synerex "github.com/fukurin00/provider_api"

	msg "github.com/fukurin00/robot_provider/msg"
	robot "github.com/fukurin00/robot_provider/robot"

	sxcav "github.com/synerex/proto_cav"
	sxmqtt "github.com/synerex/proto_mqtt"
	api "github.com/synerex/synerex_api"
	pbase "github.com/synerex/synerex_proto"
	sxutil "github.com/synerex/synerex_sxutil"
	"google.golang.org/protobuf/proto"
)

var (
	pubPose *bool = flag.Bool("pubPose", false, "publish pose for objmap")

	nodesrv = flag.String("nodesrv", "127.0.0.1:9990", "node serv address")

	mu sync.Mutex

	syMqttClient *sxutil.SXServiceClient
	routeClient  *sxutil.SXServiceClient

	robotList       map[int]*robot.RobotStatus
	sxServerAddress string
)

func mqttCallback(clt *sxutil.SXServiceClient, sp *api.Supply) {
	//from MQTT broker
	if sp.SenderId == uint64(clt.ClientID) {
		// ignore my message.
		return
	}
	rcd := sxmqtt.MQTTRecord{}
	err := proto.Unmarshal(sp.Cdata.Entity, &rcd)
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
					if d == nil {
						return
					}

					// stopping other robot
					objects := []*sxcav.Point{}
					for _, r := range robotList {
						if r.Ros.ID == id {
							continue
						}
						if !r.HavePath {
							objects = append(objects, r.Current)
						} else {
							objects = append(objects, r.Goal)
						}
					}
					d.Objects = objects

					out, err := proto.Marshal(d)
					if err != nil {
						log.Print(err)
					}
					cout := api.Content{Entity: out}
					smo := sxutil.SupplyOpts{
						Name:  "RouteDemand",
						Cdata: &cout,
					}
					_, err = routeClient.NotifySupply(&smo)
					if err != nil {
						log.Print(err)
						reconnectClient(clt)
					} else {
						log.Printf("send dest request robot%d from (%f, %f) to (%f, %f)", id, d.Start.X, d.Start.Y, d.Goal.X, d.Goal.Y)
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
				robot := robotList[id]
				robot.UpdatePose(&rcd)

				if *pubPose {
					var pose msg.Pose
					var odom msg.Odometry
					err := json.Unmarshal(rcd.Record, &odom)
					if err != nil {
						log.Print(err)
					}
					pose = odom.Pose.Pose
					out := robotList[id].NewPoseMQTT(pose)
					sout, err := proto.Marshal(out)
					if err != nil {
						log.Print(sout)
					}
					cout := api.Content{Entity: sout}
					smo := sxutil.SupplyOpts{
						Name:  "robotPosition",
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
}

func routeCallback(clt *sxutil.SXServiceClient, sp *api.Supply) {
	if sp.SenderId == uint64(clt.ClientID) {
		// ignore my message.
		return
	}
	if sp.SupplyName == "SupplyRoute" {
		rcd := &sxcav.Path{}
		err := proto.Unmarshal(sp.Cdata.Entity, rcd)
		if err != nil {
			log.Print(err)
		}
		log.Printf("[INFO/robot] receive route robot%d", rcd.RobotId)
		id := int(rcd.RobotId)
		if rob, ok := robotList[id]; ok {
			rob.SetPath(rcd)
			jsonbyte, err := robot.MakePathMsg(rcd)
			if err != nil {
				log.Print(err)
			} else {
				robot.SendPath(id, jsonbyte, syMqttClient)
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

func subsclibeRouteSupply(client *sxutil.SXServiceClient) {
	ctx := context.Background()
	for {
		client.SubscribeSupply(ctx, routeCallback)
		reconnectClient(client)
	}
}

func LoggingSettings(logFile string) {
	if _, err := os.Stat("log/"); os.IsNotExist(err) {
		os.Mkdir("log/", 0777)
	}
	if _, err := os.Stat("log/pose"); os.IsNotExist(err) {
		os.Mkdir("log/pose", 0777)
	}
	logfile, _ := os.OpenFile(logFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	multiLogFile := io.MultiWriter(os.Stdout, logfile)
	log.SetFlags(log.Ldate | log.Ltime)
	log.SetOutput(multiLogFile)
}

func main() {
	//logging configuration
	now := time.Now()
	LoggingSettings("log/" + now.Format("2006-01-02-15") + ".log")

	robotList = make(map[int]*robot.RobotStatus)

	robotList[1] = robot.NewRobot(1)
	robot1 := robotList[1]
	robot1.Radius = 0.3
	robotList[2] = robot.NewRobot(2)
	robot2 := robotList[2]
	robot2.Radius = 0.3
	robotList[3] = robot.NewRobot(3)
	robot3 := robotList[3]
	robot3.Radius = 0.3

	go sxutil.HandleSigInt()
	wg := sync.WaitGroup{}
	flag.Parse()
	sxutil.RegisterDeferFunction(sxutil.UnRegisterNode)

	//mqttClient = mqttApi.ConnectMqttBroker(*broker, *port, connectHandler, connectLostHandler, messagePubHandler)
	channels := []uint32{pbase.MQTT_GATEWAY_SVC, pbase.ROUTING_SERVICE}
	srv, err := sxutil.RegisterNode(*nodesrv, "RobotProvider", channels, nil)
	if err != nil {
		log.Fatal("can not registar node")
	}
	log.Printf("connectiong server [%s]", srv)

	sxServerAddress = srv

	synerexClient := sxutil.GrpcConnectServer(srv)
	argJson1 := "{Client: RobotMQTT}"
	syMqttClient = sxutil.NewSXServiceClient(synerexClient, pbase.MQTT_GATEWAY_SVC, argJson1)
	argJson2 := "{Client: RobotRoute}"
	routeClient = sxutil.NewSXServiceClient(synerexClient, pbase.ROUTING_SERVICE, argJson2)
	// names := []string{"ROBOT_MQTT", "ROBOT_ROUTING"

	log.Print("start subscribing")
	go subsclibeMQTTSupply(syMqttClient)
	go subsclibeRouteSupply(routeClient)

	wg.Add(1)
	wg.Wait()
}
