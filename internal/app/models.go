package app

import (
	"context"
	"github.com/coder/websocket"
	"net"
	"remote-desktop/internal/wol"
	"time"
)

const Prefix = "/app/remote-desktop"

type User struct {
	UID   string `json:"uid"`
	Name  string `json:"name"`
	Admin bool   `json:"admin"`
	CSRF  string `json:"csrfToken"`
}
type Settings struct {
	FileClipboard     *bool  `json:"fileClipboard,omitempty"`
	Wallpaper         *bool  `json:"wallpaper,omitempty"`
	Icon              string `json:"icon,omitempty"`
	Security          string `json:"security"`
	Clipboard         string `json:"clipboard"`
	CertificatePolicy string `json:"certificatePolicy"`
	AuthMode          string `json:"authMode"`
}
type Connection struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	Protocol     string     `json:"protocol"`
	TargetOS     string     `json:"targetOs"`
	Host         string     `json:"host"`
	Port         int        `json:"port"`
	GroupID      string     `json:"groupId"`
	CredentialID string     `json:"credentialId"`
	Settings     Settings   `json:"settings"`
	WOL          wol.Config `json:"wol"`
	Version      int        `json:"version"`
}
type Credential struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
	Username  string    `json:"username,omitempty"`
	Password  string    `json:"password,omitempty"`
	Domain    string    `json:"domain,omitempty"`
}
type Group struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version int    `json:"version"`
}
type Display struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	DPI    int `json:"dpi"`
}
type sessionRequest struct {
	Wallpaper        *bool   `json:"wallpaper"`
	RenderMode       string  `json:"renderMode"`
	CertificateCheck string  `json:"certificateCheck"`
	ConnectionID     string  `json:"connectionId"`
	Display          Display `json:"display"`
	Timezone         string  `json:"timezone"`
	Ref              string  `json:"ephemeralCredentialRef"`
}
type Session struct {
	Wallpaper              *bool              `json:"wallpaper,omitempty"`
	RenderMode             string             `json:"renderMode,omitempty"`
	CertificateFingerprint string             `json:"-"`
	Origin                 string             `json:"-"`
	ID                     string             `json:"id"`
	ConnectionID           string             `json:"connectionId"`
	State                  string             `json:"state"`
	CreatedAt              time.Time          `json:"createdAt"`
	ExpiresAt              time.Time          `json:"expiresAt"`
	EndedAt                *time.Time         `json:"endedAt,omitempty"`
	Reason                 string             `json:"endReason,omitempty"`
	Owner                  string             `json:"-"`
	Config                 Connection         `json:"-"`
	Login                  Credential         `json:"-"`
	Display                Display            `json:"-"`
	Timezone               string             `json:"-"`
	Ticket                 [32]byte           `json:"-"`
	TicketExpiry           time.Time          `json:"-"`
	WS                     *websocket.Conn    `json:"-"`
	Upstream               net.Conn           `json:"-"`
	Cancel                 context.CancelFunc `json:"-"`
	Timer                  *time.Timer        `json:"-"`
	LastInput              time.Time          `json:"-"`
}
type ephemeral struct {
	Owner, ConnectionID string
	Login               Credential
	Expires             time.Time
}
