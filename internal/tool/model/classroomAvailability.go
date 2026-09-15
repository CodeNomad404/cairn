package model

import "time"

// ClassroomAvailability 是教务系统中一间教室在一个节次的占用快照。
type ClassroomAvailability struct {
	Date           time.Time `json:"date"`
	Period         int       `json:"period"`
	BuildingID     string    `json:"building_id"`
	BuildingName   string    `json:"building_name"`
	RoomName       string    `json:"room_name"`
	Capacity       int       `json:"capacity"`
	OccupationType string    `json:"occupation_type"`
	SyncedAt       time.Time `json:"synced_at"`
}
