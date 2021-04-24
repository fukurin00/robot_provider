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
