package model

import "time"

type Transfer struct {
	Common
	CreatedAt time.Time `gorm:"index:idx_transfer_server_id;<-:create"`
	ServerID  uint64    `gorm:"index:idx_transfer_server_id;index"`
	In        uint64    `gorm:"index:idx_transfer_server_id"`
	Out       uint64    `gorm:"index:idx_transfer_server_id"`
}
