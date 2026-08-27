package bootstrap

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	DefaultUsername = "admin"
	metaKey         = "singleton"
)

var (
	ErrNotInitialized     = errors.New("bootstrap: platform not initialized")
	ErrInvalidCredentials = errors.New("bootstrap: invalid credentials")
	ErrWeakPassword       = errors.New("bootstrap: password too short")
	ErrPasswordAlreadySet = errors.New("bootstrap: password already set")
)

// Credentials is returned only when a new default password is minted (first Open).
type Credentials struct {
	Username string
	Password string // empty on subsequent Open
}

// Store persists first-boot state in a local SQLite file.
type Store struct {
	mu sync.Mutex
	db *gorm.DB
}

type metaRow struct {
	ID                 string `gorm:"primaryKey;size:32"`
	Initialized        bool   `gorm:"not null"`
	AdminUsername      string `gorm:"column:admin_username;size:128;not null"`
	AdminPasswordHash  string `gorm:"column:admin_password_hash;type:text;not null"`
	MustChangePassword bool   `gorm:"column:must_change_password;not null"`
	AgentTokenHash     string `gorm:"column:agent_token_hash;type:text"`
	AppDBDriver        string `gorm:"column:app_db_driver;size:32"`
	AppDBDSN           string `gorm:"column:app_db_dsn;type:text"`
	WizardStep         string `gorm:"column:wizard_step;size:64"`
	EncKeyB64          string `gorm:"column:enc_key_b64;type:text"`
	RestartRequired    bool   `gorm:"column:restart_required;not null"`
}

func (metaRow) TableName() string { return "bootstrap_meta" }

// Open opens or creates the bootstrap DB at path.
// On first create it mints a default admin password and returns it in Credentials.Password.
func Open(path string) (*Store, Credentials, error) {
	if strings.TrimSpace(path) == "" {
		return nil, Credentials{}, errors.New("bootstrap: empty path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, Credentials{}, fmt.Errorf("bootstrap: mkdir: %w", err)
	}
	fresh := false
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fresh = true
	}
	gdb, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, Credentials{}, fmt.Errorf("bootstrap: open: %w", err)
	}
	if err := gdb.AutoMigrate(&metaRow{}); err != nil {
		_ = closeDB(gdb)
		return nil, Credentials{}, fmt.Errorf("bootstrap: migrate: %w", err)
	}

	st := &Store{db: gdb}
	var row metaRow
	err = gdb.First(&row, "id = ?", metaKey).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		pass, hash, err := mintPassword()
		if err != nil {
			_ = st.Close()
			return nil, Credentials{}, err
		}
		if err := writePlaintextPassword(path, pass); err != nil {
			_ = st.Close()
			return nil, Credentials{}, err
		}
		encB64, err := mintEncKeyB64()
		if err != nil {
			_ = st.Close()
			return nil, Credentials{}, err
		}
		row = metaRow{
			ID:                 metaKey,
			Initialized:        false,
			AdminUsername:      DefaultUsername,
			AdminPasswordHash:  hash,
			MustChangePassword: true,
			EncKeyB64:          encB64,
		}
		if err := gdb.Create(&row).Error; err != nil {
			_ = st.Close()
			return nil, Credentials{}, fmt.Errorf("bootstrap: create meta: %w", err)
		}
		return st, Credentials{Username: DefaultUsername, Password: pass}, nil
	}
	if err != nil {
		_ = st.Close()
		return nil, Credentials{}, fmt.Errorf("bootstrap: load meta: %w", err)
	}
	if strings.TrimSpace(row.EncKeyB64) == "" {
		encB64, err := mintEncKeyB64()
		if err != nil {
			_ = st.Close()
			return nil, Credentials{}, err
		}
		if err := gdb.Model(&metaRow{}).Where("id = ?", metaKey).Update("enc_key_b64", encB64).Error; err != nil {
			_ = st.Close()
			return nil, Credentials{}, fmt.Errorf("bootstrap: store enc key: %w", err)
		}
	}
	_ = fresh
	return st, Credentials{Username: row.AdminUsername}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return closeDB(s.db)
}

func (s *Store) Initialized() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return false
	}
	return row.Initialized
}

func (s *Store) MustChangePassword() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return true
	}
	return row.MustChangePassword
}

// AdminAccount returns the bootstrap admin credentials needed to seed the
// console-users table. passwordHash is empty when no admin has been written.
func (s *Store) AdminAccount() (username, passwordHash string, mustChangePassword bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return "", "", true
	}
	return row.AdminUsername, row.AdminPasswordHash, row.MustChangePassword
}

func (s *Store) EnsureReadyForBusiness() error {
	if !s.Initialized() {
		return ErrNotInitialized
	}
	return nil
}

func (s *Store) MarkInitialized() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Model(&metaRow{}).Where("id = ?", metaKey).Update("initialized", true).Error
}

func (s *Store) VerifyPassword(username, password string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return false, err
	}
	if username != row.AdminUsername {
		return false, nil
	}
	err = bcrypt.CompareHashAndPassword([]byte(row.AdminPasswordHash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) ChangePassword(username, oldPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return err
	}
	if username != row.AdminUsername {
		return ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(row.AdminPasswordHash), []byte(oldPassword)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return ErrInvalidCredentials
		}
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.Model(&metaRow{}).Where("id = ?", metaKey).Updates(map[string]any{
		"admin_password_hash":  string(hash),
		"must_change_password": false,
	}).Error
}

// SetPassword sets the admin password after first login. It only works while
// MustChangePassword is true — the session already proved the bootstrap secret.
func (s *Store) SetPassword(username, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return err
	}
	if username != row.AdminUsername {
		return ErrInvalidCredentials
	}
	if !row.MustChangePassword {
		return ErrPasswordAlreadySet
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.Model(&metaRow{}).Where("id = ?", metaKey).Updates(map[string]any{
		"admin_password_hash":  string(hash),
		"must_change_password": false,
	}).Error
}

// EnsureAgentToken returns a newly minted agent token once. Subsequent calls
// return ("", false, nil) — plaintext is only available at mint time.
func (s *Store) EnsureAgentToken() (plain string, minted bool, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(row.AgentTokenHash) != "" {
		return "", false, nil
	}
	plain, hash, err := mintPassword()
	if err != nil {
		return "", false, err
	}
	if err := s.db.Model(&metaRow{}).Where("id = ?", metaKey).Update("agent_token_hash", hash).Error; err != nil {
		return "", false, err
	}
	return plain, true, nil
}

// VerifyAgentToken checks a bearer agent token against the stored hash.
func (s *Store) VerifyAgentToken(token string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(row.AgentTokenHash) == "" || strings.TrimSpace(token) == "" {
		return false, nil
	}
	err = bcrypt.CompareHashAndPassword([]byte(row.AgentTokenHash), []byte(token))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) load() (metaRow, error) {
	var row metaRow
	err := s.db.First(&row, "id = ?", metaKey).Error
	return row, err
}

// EncKey returns the 32-byte AES key stored in bootstrap (local trust boundary).
func (s *Store) EncKey() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return nil, err
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(row.EncKeyB64))
	if err != nil {
		return nil, fmt.Errorf("bootstrap: decode enc key: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("bootstrap: enc key length %d", len(raw))
	}
	return raw, nil
}

func (s *Store) SetAppDB(driver, dsn string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Model(&metaRow{}).Where("id = ?", metaKey).Updates(map[string]any{
		"app_db_driver": driver,
		"app_db_dsn":    dsn,
	}).Error
}

func (s *Store) AppDB() (driver, dsn string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return "", "", err
	}
	return row.AppDBDriver, row.AppDBDSN, nil
}

func (s *Store) SetWizardStep(step string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Model(&metaRow{}).Where("id = ?", metaKey).Update("wizard_step", step).Error
}

func (s *Store) WizardStep() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return ""
	}
	return row.WizardStep
}

func (s *Store) RestartRequired() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	row, err := s.load()
	if err != nil {
		return false
	}
	return row.RestartRequired
}

func (s *Store) SetRestartRequired(v bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.db.Model(&metaRow{}).Where("id = ?", metaKey).Update("restart_required", v).Error
}

func mintEncKeyB64() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(b[:]), nil
}

func mintPassword() (plain, hash string, err error) {
	var b [18]byte
	if _, err = rand.Read(b[:]); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b[:])
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return plain, string(h), nil
}

// passwordFilePath derives the plaintext admin-password file path from the
// bootstrap DB path (e.g. data/bootstrap.db -> data/bootstrap.admin-password).
func passwordFilePath(bootPath string) string {
	return strings.TrimSuffix(bootPath, ".db") + ".admin-password"
}

// writePlaintextPassword stores the freshly minted default admin password so it
// can be re-printed after a restart before initialization completes. It is
// written 0600 (owner-only). A failure here aborts bootstrap: without a
// recoverable password the admin would be locked out on restart.
func writePlaintextPassword(bootPath, pass string) error {
	p := passwordFilePath(bootPath)
	if err := os.WriteFile(p, []byte(pass+"\n"), 0o600); err != nil {
		return fmt.Errorf("bootstrap: store plaintext password: %w", err)
	}
	return nil
}

// ReadStoredPassword returns the plaintext default admin password if it was
// persisted by writePlaintextPassword and still applies (initialization not
// finished). It returns os.ErrNotExist when no stored password exists.
func ReadStoredPassword(bootPath string) (string, error) {
	b, err := os.ReadFile(passwordFilePath(bootPath))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// RemoveStoredPassword deletes the persisted plaintext default admin password.
// It is called after the admin changes the password so the old initial secret
// is no longer recoverable. Missing file is not an error.
func RemoveStoredPassword(bootPath string) error {
	err := os.Remove(passwordFilePath(bootPath))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func closeDB(gdb *gorm.DB) error {
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
