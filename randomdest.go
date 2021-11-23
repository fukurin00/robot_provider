package main

// import (
// 	"log"
// 	"math/rand"
// 	"time"

// 	msg "github.com/fukurin00/robot_provider/msg"
// 	robot "github.com/fukurin00/robot_provider/robot"
// 	cav "github.com/synerex/proto_cav"
// 	api "github.com/synerex/synerex_api"
// 	sxutil "github.com/synerex/synerex_sxutil"
// 	"google.golang.org/protobuf/proto"
// )

// var (
// 	destList = [9][2]float64{{-2, 0}, {6, 0}, {22, 5}, {31, 4}, {25, -5}, {25.5, -10}, {10, -15}, {21, -23}, {-6, -26}}
// 	dests    map[int]*cav.Point

// 	freeDest []int

// 	destRobot = 0
// )

// func init() {
// 	dests = make(map[int]*cav.Point)

// 	for i, d := range destList {
// 		dests[i] = robot.NewCavPoint(d[0], d[1])
// 		freeDest = append(freeDest, i)
// 	}
// }

// func randomDestManager() {
// 	timer := time.NewTicker(3 * time.Second)
// 	time.Sleep(3 * time.Second)
// 	for {
// 		select {
// 		case <-timer.C:
// 			destRobot += 1
// 			if destRobot > 5 {
// 				destRobot = 1
// 			}
// 			if _, ok := robotList[destRobot]; ok {
// 				robot := robotList[destRobot]
// 				id := robot.Ros.ID
// 				if robot.IsArriveDest(arriveThresh) || time.Since(robot.DestUpdate).Seconds() > 40 || !robot.HaveDest {
// 					var randnum int
// 					if len(freeDest) == 0 {
// 						randnum = 0
// 					} else {
// 						randnum = rand.Intn(len(freeDest) - 1)
// 					}
// 					var r *cav.Point
// 					if v, ok := dests[randnum]; !ok {
// 						r = &cav.Point{X: 0, Y: 0}
// 					} else {
// 						r = v
// 					}
// 					freeDest = append(freeDest[:randnum], freeDest[randnum+1:]...)
// 					if robot.HaveDest {
// 						freeDest = append(freeDest, robot.DestId)
// 					}
// 					robot.DestId = randnum
// 					d := robot.NewDestRequest(r, msg.CalcStamp(time.Now()))
// 					if d != nil {
// 						out, err := proto.Marshal(d)
// 						if err != nil {
// 							log.Print(err)
// 						}
// 						cout := api.Content{Entity: out}
// 						smo := sxutil.SupplyOpts{
// 							Name:  "DestDemand",
// 							Cdata: &cout,
// 						}
// 						_, err = routeClient.NotifySupply(&smo)
// 						if err != nil {
// 							log.Print(err)
// 							reconnectClient(syMqttClient)
// 						} else {
// 							log.Printf("send dest request robot%d from (%f, %f) to (%f, %f)", id, d.Start.X, d.Start.Y, d.Goal.X, d.Goal.Y)
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}
// }
