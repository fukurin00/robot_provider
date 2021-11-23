package msg

import (
	"math"
	"time"
)

type ROS_header struct {
	Seq      uint32    `json:"seq"`
	Stamp    TimeStamp `json:"stamp"`
	Frame_id string    `json:"frame_id"`
}

type TimeStamp struct {
	Secs  uint32 `json:"secs"`
	Nsecs uint32 `json:"nsecs"`
}

func (t TimeStamp) CalcTime() time.Time {
	o := time.Unix(int64(t.Secs), int64(t.Nsecs))
	return o
}

func (t TimeStamp) ToF() float64 {
	return float64(t.Secs) + float64(t.Nsecs*uint32(math.Pow10(-9)))
}

func CalcTimeUnix(uni float64) time.Time {
	sec, dec := math.Modf(uni)
	t := time.Unix(int64(sec), int64(dec*1e9))
	return t
}

func FtoStamp(f float64) TimeStamp {
	sec, dec := math.Modf(f)
	t := TimeStamp{
		Secs:  uint32(sec),
		Nsecs: uint32(dec * 1e9),
	}
	return t
}

func CalcStamp(t time.Time) TimeStamp {
	o := TimeStamp{
		Secs:  uint32(t.Unix()),
		Nsecs: uint32(t.UnixNano()),
	}
	return o
}

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func (p Point) Distance(o Point) float64 {
	return math.Hypot(p.X-o.X, p.Y-o.Y)
}

type Quaternion struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	W float64 `json:"w"`
}

type Pose struct {
	Position    Point      `json:"position"`
	Orientation Quaternion `json:"orientation"`
}

type ROS_PoseStamped struct {
	Header ROS_header `json:"header"`
	Pose   Pose       `json:"pose"`
}

type Path struct {
	Header ROS_header        `json:"header"`
	Poses  []ROS_PoseStamped `json:"poses"`
}

type Odometry struct {
	Header         ROS_header          `json:"header"`
	Child_Frame_ID string              `json:"child_frame_id"`
	Pose           PoseWithCovariance  `json:"pose"`
	Twist          TwistWithCovariance `json:"twist"`
}

type PoseWithCovariance struct {
	Pose       Pose        `json:"pose"`
	Covariance [36]float64 `json:"covariance"`
}

type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}
type Twist struct {
	Linear  Vector3 `json:"linear"`
	Angular Vector3 `json:"angular"`
}

type TwistWithCovariance struct {
	Twist      Twist       `json:"twist"`
	Covariance [36]float64 `json:"covariance"`
}

func Yaw2Quaternion(yaw float64) Quaternion {
	cy := math.Cos(yaw / 2)
	sy := math.Sin(yaw / 2)

	var q Quaternion
	q.W = cy
	q.X = 0
	q.Y = 0
	q.Z = sy
	return q
}
