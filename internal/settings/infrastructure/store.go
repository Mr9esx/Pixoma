package infrastructure

import (
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/Mr9esx/Pixoma/internal/settings/domain"
)

const rowID = "singleton"

// Store persists domain.Settings in the business database.
type Store struct {
	db  *gorm.DB
	key []byte
}

type row struct {
	ID                string `gorm:"primaryKey;size:32"`
	Placement         string `gorm:"size:32;not null"`
	DBDriver          string `gorm:"column:db_driver;size:32;not null"`
	DBDSN             string `gorm:"column:db_dsn;type:text;not null"`
	BlobDriver        string `gorm:"column:blob_driver;size:32;not null"`
	BlobRoot          string `gorm:"column:blob_root;type:text"`
	BlobEndpoint      string `gorm:"column:blob_endpoint;type:text"`
	BlobRegion        string `gorm:"column:blob_region;size:64"`
	BlobBucket        string `gorm:"column:blob_bucket;size:256"`
	BlobAccessCipher  string `gorm:"column:blob_access_cipher;type:text"`
	BlobSecretCipher  string `gorm:"column:blob_secret_cipher;type:text"`
	ComfyUIBaseURL    string `gorm:"column:comfyui_base_url;type:text"`
	ClaimWaitMS       int    `gorm:"column:claim_wait_ms"`
	LeaseSeconds      int    `gorm:"column:lease_seconds"`
	ProxyKind         string `gorm:"column:proxy_kind;size:16"`
	ProxyHost         string `gorm:"column:proxy_host;type:text"`
	ProxyPort         int    `gorm:"column:proxy_port"`
	MediaMaxBytes     int64  `gorm:"column:media_max_bytes;not null;default:0"`
	AllowSelfReg      bool   `gorm:"column:allow_self_registration;not null;default:false"`
	DefaultUserAccess string `gorm:"column:default_user_access;size:32;not null;default:'denied'"`
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

func (s *Store) Save(in domain.Settings) error {
	if err := in.Validate(); err != nil {
		return err
	}
	var existing row
	hasExisting := s.db.First(&existing, "id = ?", rowID).Error == nil
	ak, err := domain.EncryptString(s.key, in.BlobAccessKey)
	if err != nil {
		return err
	}
	sk, err := domain.EncryptString(s.key, in.BlobSecretKey)
	if err != nil {
		return err
	}
	if hasExisting {
		if in.BlobAccessKey == "" {
			ak = existing.BlobAccessCipher
		}
		if in.BlobSecretKey == "" {
			sk = existing.BlobSecretCipher
		}
	}
	r := row{
		ID:                rowID,
		Placement:         in.Placement,
		DBDriver:          in.DBDriver,
		DBDSN:             in.DBDSN,
		BlobDriver:        in.BlobDriver,
		BlobRoot:          in.BlobRoot,
		BlobEndpoint:      in.BlobEndpoint,
		BlobRegion:        in.BlobRegion,
		BlobBucket:        in.BlobBucket,
		BlobAccessCipher:  ak,
		BlobSecretCipher:  sk,
		ComfyUIBaseURL:    in.ComfyUIBaseURL,
		ClaimWaitMS:       in.ClaimWaitMS,
		LeaseSeconds:      in.LeaseSeconds,
		ProxyKind:         in.ProxyKind,
		ProxyHost:         in.ProxyHost,
		ProxyPort:         in.ProxyPort,
		MediaMaxBytes:     in.MediaMaxBytes,
		AllowSelfReg:      in.AllowSelfRegistration,
		DefaultUserAccess: domain.NormalizeDefaultUserAccess(in.DefaultUserAccess),
	}
	return s.db.Save(&r).Error
}

func (s *Store) Load() (domain.Settings, error) {
	var r row
	err := s.db.First(&r, "id = ?", rowID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Settings{}, err
	}
	if err != nil {
		return domain.Settings{}, err
	}
	ak, err := domain.DecryptString(s.key, r.BlobAccessCipher)
	if err != nil {
		return domain.Settings{}, err
	}
	sk, err := domain.DecryptString(s.key, r.BlobSecretCipher)
	if err != nil {
		return domain.Settings{}, err
	}
	return domain.Settings{
		Placement:             r.Placement,
		DBDriver:              r.DBDriver,
		DBDSN:                 r.DBDSN,
		BlobDriver:            r.BlobDriver,
		BlobRoot:              r.BlobRoot,
		BlobEndpoint:          r.BlobEndpoint,
		BlobRegion:            r.BlobRegion,
		BlobBucket:            r.BlobBucket,
		BlobAccessKey:         ak,
		BlobSecretKey:         sk,
		ComfyUIBaseURL:        r.ComfyUIBaseURL,
		ClaimWaitMS:           r.ClaimWaitMS,
		LeaseSeconds:          r.LeaseSeconds,
		ProxyKind:             r.ProxyKind,
		ProxyHost:             r.ProxyHost,
		ProxyPort:             r.ProxyPort,
		MediaMaxBytes:         r.MediaMaxBytes,
		AllowSelfRegistration: r.AllowSelfReg,
		DefaultUserAccess:     domain.NormalizeDefaultUserAccess(r.DefaultUserAccess),
	}, nil
}
