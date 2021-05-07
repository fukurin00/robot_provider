package robot

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	msg "github.com/fukurin00/robot_provider/msg"
	cav "github.com/synerex/proto_cav"
	sxmqtt "github.com/synerex/proto_mqtt"
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

type RobotStatus struct {
	Ros   RosMeta //meta information in ROS
	Pose  msg.Pose
	Point *cav.Point

	Dest       *cav.Point
	DestId     int
	HaveDest   bool
	DestUpdate time.Time

	Radius      float64
	Velocity    float64
	RotVelocity float64 //velocity of rotation

	RequestSeq int64
	Update     time.Time
}

func NewRobot(id int) *RobotStatus {
	r := new(RobotStatus)
	r.Ros = RosMeta{}
	r.Ros.ID = id
	r.Ros.RobotName = fmt.Sprintf("robot%d", id)
	r.Ros.FrameID = fmt.Sprintf("map/%s", r.Ros.RobotName)
	r.Ros.Orgin = new(cav.Point)
	r.Ros.Orgin.X = 0
	r.Ros.Orgin.Y = 0
	r.Radius = 0.5
	r.Velocity = 1.0
	r.RotVelocity = 1.0
	r.HaveDest = false
	r.Update = time.Now()
	r.DestUpdate = time.Now()
	return r
}

func CavPoint(poseStamp msg.ROS_PoseStamped) *cav.Point {
	p := new(cav.Point)
	p.X = float32(poseStamp.Pose.Position.X)
	p.Y = float32(poseStamp.Pose.Position.Y)
	return p
}

func NewCavPoint(x, y float64) *cav.Point {
	p := new(cav.Point)
	p.X = float32(x)
	p.Y = float32(y)
	return p
}

func (r *RobotStatus) NewDestRequest(dest *cav.Point, stamp msg.TimeStamp) *cav.DestinationRequest {
	if r.Point == nil {
		log.Printf("not recieve robot%d pose", r.Ros.ID)
		return nil
	}
	r.Dest = dest
	r.HaveDest = true
	req := new(cav.DestinationRequest)
	req.RobotId = int64(r.Ros.ID)
	r.RequestSeq += 1
	req.Seq = r.RequestSeq
	req.Origin = r.Ros.Orgin
	req.Current = r.Point
	req.Destination = dest
	req.Ts = timestamppb.New(time.Unix(int64(stamp.Secs), int64(stamp.Nsecs)))
	r.DestUpdate = time.Now()
	return req
}

func (r *RobotStatus) NewPoseMessage(pose msg.ROS_PoseStamped) *cav.Position {
	p := new(cav.Position)
	return p
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
	r.Point = &cav.Point{X: float32(pose.Position.X), Y: float32(pose.Position.Y)}
}

func (r *RobotStatus) IsArriveDest(arriveThresh float64) bool {
	if !r.HaveDest {
		return false
	}
	if distance(r.Dest, r.Point) <= arriveThresh {
		return true
	}
	return false
}

func distance(c, d *cav.Point) float64 {
	return math.Hypot(float64(c.X)-float64(d.X), float64(c.Y)-float64(d.Y))
}
