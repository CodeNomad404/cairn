package model

import "time"

type TransferChannel struct {
	ID        int64
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	FileCount int64
	TextCount int64
}

type TransferItem struct {
	ID         int64
	ChannelID  int64
	Kind       string
	Body       string
	StorageKey string
	Filename   string
	SizeBytes  int64
	State      string
	CreatedAt  time.Time
}
