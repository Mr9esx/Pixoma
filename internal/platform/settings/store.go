package settings

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

const rowID = "singleton"

// Store persists Settings in the business database.
type Store struct {
	db  *gorm.DB
	key []byte
}

type row struct {
	ID                  string `gorm:"primaryKey;size:32"`
	Placement           string `gorm:"size:32;not null"`
	DBDriver            string `gorm:"column:db_driver;size:32;not null"`
	DBDSN               string `gorm:"column:db_dsn;type:text;not null"`
	BlobDriver          string `gorm:"column:blob_driver;size:32;not null"`
	BlobRoot            string `gorm:"column:blob_root;type:text"`
	BlobEndpoint        string `gorm:"column:blob_endpoint;type:text"`
	BlobRegion          string `gorm:"column:blob_region;size:64"`
	BlobBucket          string `gorm:"column:blob_bucket;size:256"`
	BlobAccessCipher    string `gorm:"column:blob_access_cipher;type:text"`
	BlobSecretCipher    string `gorm:"column:blob_secret_cipher;type:text"`
	ComfyMock           bool   `gorm:"column:comfy_mock;not null"`
	ComfyUIBaseURL      string `gorm:"column:comfyui_base_url;type:text"`
	DefaultInstanceID   string `gorm:"column:default_instance_id;size:128"`
	AutoSpawnEdge       bool   `gorm:"column:auto_spawn_edge;not null"`
	ClaimWaitMS         int    `gorm:"column:claim_wait_ms"`
	LeaseSeconds        int    `gorm:"column:lease_seconds"`
	TelegramTokenCipher string `gorm:"column:telegram_token_cipher;type:text"`
}

func (row) TableName() string { return "platform_settings" }

func NewStore(gdb *gorm.DB, encKey []byte) (*Store, error) {
	if gdb == nil {
		return nil, fmt.Errorf("settings: nil db")
	}
	if len(encKey) != 32 {
		return nil, fmt.Errorf("settings: enc key must be 32 bytes")
	}
	if err := gdb.AutoMigrate(&row{}); err != nil {
		return nil, fmt.Errorf("settings: migrate: %w", err)
	}
	key := make([]byte, 32)
	copy(key, encKey)
	return &Store{db: gdb, key: key}, nil
}

func (s *Store) Save(in Settings) error {
	if err := in.Validate(); err != nil {
		return err
	}
	tg, err := encryptString(s.key, in.TelegramBotToken)
	if err != nil {
		return err
	}
	ak, err := encryptString(s.key, in.BlobAccessKey)
	if err != nil {
		return err
	}
	sk, err := encryptString(s.key, in.BlobSecretKey)
	if err != nil {
		return err
	}
	r := row{
		ID:                  rowID,
		Placement:           in.Placement,
		DBDriver:            in.DBDriver,
		DBDSN:               in.DBDSN,
		BlobDriver:          in.BlobDriver,
		BlobRoot:            in.BlobRoot,
		BlobEndpoint:        in.BlobEndpoint,
		BlobRegion:          in.BlobRegion,
		BlobBucket:          in.BlobBucket,
		BlobAccessCipher:    ak,
		BlobSecretCipher:    sk,
		ComfyMock:           in.ComfyMock,
		ComfyUIBaseURL:      in.ComfyUIBaseURL,
		DefaultInstanceID:   in.DefaultInstanceID,
		AutoSpawnEdge:       in.AutoSpawnEdge,
		ClaimWaitMS:         in.ClaimWaitMS,
		LeaseSeconds:        in.LeaseSeconds,
		TelegramTokenCipher: tg,
	}
	return s.db.Save(&r).Error
}

func (s *Store) Load() (Settings, error) {
	var r row
	err := s.db.First(&r, "id = ?", rowID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Settings{}, err
	}
	if err != nil {
		return Settings{}, err
	}
	tg, err := decryptString(s.key, r.TelegramTokenCipher)
	if err != nil {
		return Settings{}, err
	}
	ak, err := decryptString(s.key, r.BlobAccessCipher)
	if err != nil {
		return Settings{}, err
	}
	sk, err := decryptString(s.key, r.BlobSecretCipher)
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		Placement:         r.Placement,
		DBDriver:          r.DBDriver,
		DBDSN:             r.DBDSN,
		BlobDriver:        r.BlobDriver,
		BlobRoot:          r.BlobRoot,
		BlobEndpoint:      r.BlobEndpoint,
		BlobRegion:        r.BlobRegion,
		BlobBucket:        r.BlobBucket,
		BlobAccessKey:     ak,
		BlobSecretKey:     sk,
		ComfyMock:         r.ComfyMock,
		ComfyUIBaseURL:    r.ComfyUIBaseURL,
		DefaultInstanceID: r.DefaultInstanceID,
		AutoSpawnEdge:     r.AutoSpawnEdge,
		ClaimWaitMS:       r.ClaimWaitMS,
		LeaseSeconds:      r.LeaseSeconds,
		TelegramBotToken:  tg,
	}, nil
}
