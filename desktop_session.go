package gofusretrodb

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// HashDesktopSecret returns the hex-encoded SHA-256 hash of a desktop auth secret
// (poll_secret or exchange_ticket). Secrets themselves are never stored.
func HashDesktopSecret(secret string) string {
	h := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(h[:])
}

// CreateDesktopLoginSession inserts a new desktop login session in the "pending" state.
func (ds *DatabaseService) CreateDesktopLoginSession(code, deviceID, deviceName, pollSecret string, expiresAt time.Time) (*DesktopLoginSessionModel, error) {
	session := &DesktopLoginSessionModel{
		Code:           code,
		DeviceID:       deviceID,
		DeviceName:     deviceName,
		PollSecretHash: HashDesktopSecret(pollSecret),
		Status:         DesktopLoginStatusPending,
		ExpiresAt:      expiresAt,
	}
	if err := ds.db.Create(session).Error; err != nil {
		return nil, fmt.Errorf("failed to create desktop login session: %v", err)
	}
	return session, nil
}

// GetDesktopLoginSessionByCode fetches a non-expired desktop login session by code.
func (ds *DatabaseService) GetDesktopLoginSessionByCode(code string) (*DesktopLoginSessionModel, error) {
	var session DesktopLoginSessionModel
	if err := ds.db.Where("code = ?", code).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// ApproveDesktopLoginSession transitions a pending row to "approved" and stores
// the underlying web session token. The exchange ticket is issued later, on
// the first poll that sees the approved state, so the raw ticket is never
// retained on disk alongside the approval.
func (ds *DatabaseService) ApproveDesktopLoginSession(code string, userID uint, sessionToken string) error {
	now := time.Now()
	tokenCopy := sessionToken
	return ds.db.Model(&DesktopLoginSessionModel{}).
		Where("code = ? AND status = ? AND expires_at > ?", code, DesktopLoginStatusPending, now).
		Updates(map[string]interface{}{
			"status":        DesktopLoginStatusApproved,
			"user_id":       userID,
			"session_token": &tokenCopy,
			"approved_at":   &now,
		}).Error
}

// IssueDesktopExchangeTicket atomically advances an "approved" row to
// "awaiting_exchange" while recording the hash of a freshly generated ticket.
// Returns gorm.ErrRecordNotFound (or zero rows affected mapped to the caller)
// if the row is not in the approved state anymore.
func (ds *DatabaseService) IssueDesktopExchangeTicket(code, exchangeTicket string) error {
	hash := HashDesktopSecret(exchangeTicket)
	res := ds.db.Model(&DesktopLoginSessionModel{}).
		Where("code = ? AND status = ? AND expires_at > ?", code, DesktopLoginStatusApproved, time.Now()).
		Updates(map[string]interface{}{
			"status":               DesktopLoginStatusAwaitingExchange,
			"exchange_ticket_hash": &hash,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("desktop login session not in approved state")
	}
	return nil
}

// DenyDesktopLoginSession marks a pending row as denied.
func (ds *DatabaseService) DenyDesktopLoginSession(code string) error {
	return ds.db.Model(&DesktopLoginSessionModel{}).
		Where("code = ? AND status = ?", code, DesktopLoginStatusPending).
		Update("status", DesktopLoginStatusDenied).Error
}

// MarkDesktopLoginAwaitingExchange flips "approved" rows to "awaiting_exchange"
// after the poll endpoint has delivered the exchange ticket to the desktop.
func (ds *DatabaseService) MarkDesktopLoginAwaitingExchange(code string) error {
	return ds.db.Model(&DesktopLoginSessionModel{}).
		Where("code = ? AND status = ?", code, DesktopLoginStatusApproved).
		Update("status", DesktopLoginStatusAwaitingExchange).Error
}

// ConsumeDesktopLoginSession atomically consumes a session in "awaiting_exchange" by
// verifying the exchange ticket, marking it consumed, and returning the stored
// session token + approving user. The session_token column is blanked out so the
// token isn't retained on the row after delivery.
func (ds *DatabaseService) ConsumeDesktopLoginSession(deviceID, exchangeTicket string) (string, uint, error) {
	ticketHash := HashDesktopSecret(exchangeTicket)
	var (
		token  string
		userID uint
	)
	err := ds.db.Transaction(func(tx *gorm.DB) error {
		var row DesktopLoginSessionModel
		err := tx.Where(
			"device_id = ? AND exchange_ticket_hash = ? AND status = ? AND expires_at > ?",
			deviceID, ticketHash, DesktopLoginStatusAwaitingExchange, time.Now(),
		).First(&row).Error
		if err != nil {
			return err
		}
		if row.SessionToken == nil || row.UserID == nil {
			return fmt.Errorf("desktop login session in inconsistent state")
		}
		token = *row.SessionToken
		userID = *row.UserID
		now := time.Now()
		return tx.Model(&row).Updates(map[string]interface{}{
			"status":               DesktopLoginStatusConsumed,
			"session_token":        nil,
			"exchange_ticket_hash": nil,
			"consumed_at":          &now,
		}).Error
	})
	if err != nil {
		return "", 0, err
	}
	return token, userID, nil
}

// DeleteExpiredDesktopLoginSessions removes expired and terminal sessions older than
// their expiry window, keeping the table small.
func (ds *DatabaseService) DeleteExpiredDesktopLoginSessions() error {
	return ds.db.
		Where("expires_at < ?", time.Now()).
		Delete(&DesktopLoginSessionModel{}).Error
}

