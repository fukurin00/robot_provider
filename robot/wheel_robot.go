package robot

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	msg "github.com/fukurin00/robot_provider/msg"
	cav "github.com/synerex/proto_cav"
	sxmqtt "github.com/synerex/proto_mqtt"
	api "github.com/synerex/synerex_api"
	sxutil "github.com/synerex/synerex_sxutil"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ()

type RosMeta struct {
	ID          int
	RobotName   string
	Orgin       *cav.Point
	FrameID     string //global frameID in ROS
	UpdateStamp msg.TimeStamp
}

type PoseInfo struct {
	Stamp int64
	X     float64
	Y     float64
}

type RobotStatus struct {
	Ros     RosMeta //meta information in ROS
	Pose    msg.Pose
	Current *cav.Point
	Points  []PoseInfo //for log

	Start *cav.Point
	Goal  *cav.Point
	Path  *cav.Path

	RequestDest bool
	HavePath    bool

	Radius      float64
	Velocity    float64
	RotVelocity float64 //velocity of rotation

	RequestSeq int64

	Update time.Time
}

func NewRobot(id int) *RobotStatus {
	r := new(RobotStatus)
	r.Ros = RosMeta{}
	r.Ros.ID = id
	r.Radius = 0.5
	r.Velocity = 1.0
	r.RotVelocity = 1.0

	r.HavePath = false
	r.RequestDest = false

	r.Update = time.Now()
	return r
}

func CavPoint(poseStamp msg.ROS_PoseStamped) *cav.Point {
	p := new(cav.Point)
	p.X = float32(poseStamp.Pose.Position.X)
	p.Y = float32(poseStamp.Pose.Position.Y)
	p.Z = float32(poseStamp.Pose.Position.Z)
	return p
}

func NewCavPoint(x, y float64) *cav.Point {
	p := new(cav.Point)
	p.X = float32(x)
	p.Y = float32(y)
	p.Z = 0
	return p
}

func (r *RobotStatus) NewDestRequest(dest *cav.Point, stamp msg.TimeStamp) *cav.PathRequest {
	if r.Current == nil {
		log.Printf("not recieve robot%d pose", r.Ros.ID)
		return nil
	}
	r.Goal = dest
	r.RequestDest = true
	req := new(cav.PathRequest)
	req.RobotId = int64(r.Ros.ID)
	r.RequestSeq += 1
	req.Seq = r.RequestSeq
	req.Start = r.Current
	req.Goal = dest
	req.Ts = timestamppb.New(time.Now())
	return req
}

func (r *RobotStatus) NewPoseMQTT(pose msg.Pose) *sxmqtt.MQTTRecord {
	topic := fmt.Sprintf("robot/position/%d", r.Ros.ID)
	jout, err := json.Marshal(pose)
	if err != nil {
		log.Print(err)
	}
	out := sxmqtt.MQTTRecord{
		Topic:  topic,
		Record: jout,
	}
	return &out
}

func (r *RobotStatus) UpdatePose(rcd *sxmqtt.MQTTRecord) {
	var odom msg.Odometry
	err := json.Unmarshal(rcd.Record, &odom)
	if err != nil {
		log.Print(err)
	}

	var pose msg.Pose = odom.Pose.Pose
	var id uint32

	fmt.Sscanf(rcd.Topic, "robot/pose/%d", &id)
	r.Pose = pose

	r.Current = &cav.Point{X: float32(pose.Position.X), Y: float32(pose.Position.Y), Z: float32(pose.Position.Z)}

	if r.HavePath {
		if r.IsArriveDest(1) {
			r.HavePath = false
		}
	}

	//for log
	r.Points = append(r.Points, PoseInfo{Stamp: time.Now().UnixNano(), X: pose.Position.X, Y: pose.Position.Y})
	if time.Since(r.Update).Seconds() > 0.5 {
		if len(r.Points) > 100 {
			r.AddCsvPos(fmt.Sprintf("log/pose/robot%d_%s", id, time.Now().Format("2006-01-02-15.csv")))
		}
		r.Update = time.Now()
	}

}

func (r *RobotStatus) SetPath(path *cav.Path) {
	r.Path = path
	r.HavePath = true
	r.RequestDest = false
}

func SendPath(id int, msg []byte, client *sxutil.SXServiceClient) {
	topic := fmt.Sprintf("robot/path/%d", id)
	mqttProt := sxmqtt.MQTTRecord{
		Topic:  topic,
		Record: msg,
	}
	out, err := proto.Marshal(&mqttProt)
	if err != nil {
		log.Print(err)
	}
	cout := api.Content{Entity: out}
	smo := sxutil.SupplyOpts{
		Name:  "robotRoute",
		Cdata: &cout,
	}
	_, err = client.NotifySupply(&smo)
	if err != nil {
		log.Print(err)
	} else {
		log.Printf("send path robot%d topic:%s", id, topic)
	}
}

func MakePathMsg(route *cav.Path) ([]byte, error) {
	var poses []msg.ROS_PoseStamped

	for i := 1; i < len(route.Path); i++ {
		x := float64(route.Path[i].Pose.X)
		y := float64(route.Path[i].Pose.Y)
		z := float64(route.Path[i].Pose.Z)
		prevX := float64(route.Path[i-1].Pose.X)
		prevY := float64(route.Path[i-1].Pose.Y)

		yaw := math.Atan2(float64(y-prevY), float64((x - prevX)))

		pos := msg.ROS_PoseStamped{
			Header: msg.ROS_header{
				Seq: uint32(i),
				Stamp: msg.TimeStamp{
					Secs:  uint32(route.Path[i].Ts.Seconds),
					Nsecs: uint32(route.Path[i].Ts.Nanos),
				},
				Frame_id: "map",
			},
			Pose: msg.Pose{
				Position:    msg.Point{X: x, Y: y, Z: z},
				Orientation: msg.Yaw2Quaternion(yaw),
			},
		}
		poses = append(poses, pos)
	}

	planm := msg.Path{
		Header: msg.ROS_header{Frame_id: "map"},
		Poses:  poses,
	}

	jm, err := json.MarshalIndent(planm, "", " ")
	if err != nil {
		return jm, err
	}
	return jm, err

}

func (r *RobotStatus) IsArriveDest(arriveThresh float64) bool {
	if distance(r.Goal, r.Current) <= arriveThresh {
		return true
	}
	return false
}

func distance(c, d *cav.Point) float64 {
	return math.Hypot(float64(c.X)-float64(d.X), float64(c.Y)-float64(d.Y))
}

// for log
func (r *RobotStatus) AddCsvPos(fname string) {
	//log.Print(fname)
	file, err := os.OpenFile(fname, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		log.Print(err)
	}
	for _, pos := range r.Points {
		fmt.Fprintf(file, "%f,%f,%f\n", pos.Stamp, pos.X, pos.Y)
	}

	r.Points = nil
	defer file.Close()
}
