package robot

import (
	"encoding/json"
	"fmt"
	"log"

	msg "github.com/fukurin00/provider_robot_node/msg"
	cav "github.com/synerex/proto_cav"
	sxmqtt "github.com/synerex/proto_mqtt"
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
	Ros       RosMeta //meta information in ROS
	PoseStamp msg.ROS_PoseStamped
	Point     *cav.Point

	Radius      float64
	Velocity    float64
	RotVelocity float64 //velocity of rotation

	RequestSeq int64
}

func NewRobot(id int) *RobotStatus {
	r := new(RobotStatus)
	r.Ros = RosMeta{}
	r.Ros.ID = id
	r.Ros.RobotName = fmt.Sprintf("robot%d", id)
	r.Ros.FrameID = fmt.Sprintf("map/%s", r.Ros.RobotName)
	r.Ros.Orgin.X = 0
	r.Ros.Orgin.Y = 0
	r.Radius = 1
	r.Velocity = 1.0
	r.RotVelocity = 1.0
	return r
}

func CavStamp(stamp msg.TimeStamp) *cav.Stamp {
	s := new(cav.Stamp)
	s.Secs = uint64(stamp.Secs)
	s.Nsecs = uint64(stamp.Nsecs)
	return s
}

func (r *RobotStatus) NewDestRequest(dest *cav.Point, stamp msg.TimeStamp) *cav.DestinationRequest {
	req := new(cav.DestinationRequest)
	req.RobotId = int64(r.Ros.ID)
	r.RequestSeq += 1
	req.Seq = r.RequestSeq
	req.Origin = r.Ros.Orgin
	req.Current = r.Point
	req.Destination = dest
	req.Stamp = CavStamp(stamp)
	return req
}

func (r *RobotStatus) UpdatePose(rcd *sxmqtt.MQTTRecord) {
	var pose msg.ROS_PoseStamped
	var id uint32

	err := json.Unmarshal(rcd.Record, &pose)
	if err != nil {
		log.Print(err)
	}
	fmt.Sscanf(rcd.Topic, "robot/pose/%d", &id)
	r.PoseStamp = pose
	r.Point = &cav.Point{X: float32(pose.Pose.Position.X), Y: float32(pose.Pose.Position.Y)}
}
