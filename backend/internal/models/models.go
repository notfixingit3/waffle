package models

import (
	"time"

	"github.com/google/uuid"
)

type WaffleStatus string

const (
	WaffleStatusActive    WaffleStatus = "active"
	WaffleStatusCompleted WaffleStatus = "completed"
)

type SpotStatus string

const (
	SpotStatusAvailable SpotStatus = "available"
	SpotStatusPending   SpotStatus = "pending"
	SpotStatusPaid      SpotStatus = "paid"
	SpotStatusWinner    SpotStatus = "winner"
	SpotStatusLoser     SpotStatus = "loser"
)

// Role constants (string constants, not enum type)
const (
	RoleSuperAdmin    = "super_admin"
	RoleAdmin         = "admin"
	RoleWaffleManager = "waffle_manager"
)

// Payment method type constants
const (
	PaymentMethodTypeVenmo   = "venmo"
	PaymentMethodTypePayPal  = "paypal"
	PaymentMethodTypeCashApp = "cashapp"
	PaymentMethodTypeZelle   = "zelle"
)

type Waffle struct {
	ID                     uuid.UUID    `json:"id" db:"id"`
	Slug                   string       `json:"slug" db:"slug"`
	Title                  string       `json:"title" db:"title"`
	Description            *string      `json:"description,omitempty" db:"description"`
	ImageURL               *string      `json:"image_url,omitempty" db:"image_url"`
	TotalSpots             int          `json:"total_spots" db:"total_spots"`
	SpotPrice              int          `json:"spot_price" db:"spot_price"`
	// Deprecated: Use PaymentMethods instead. Kept for backward compatibility with existing waffles.
	PaymentInfo            *string        `json:"payment_info,omitempty" db:"payment_info"`
	PaymentMethods         []PaymentMethod `json:"payment_methods,omitempty"`
	Status                 WaffleStatus    `json:"status" db:"status"`
	WinningSpotNumber      *int         `json:"winning_spot_number,omitempty" db:"winning_spot_number"`
	WinningInstagramHandle *string      `json:"winning_instagram_handle,omitempty" db:"winning_instagram_handle"`
	ItemCount              int          `json:"item_count" db:"item_count"`
	WinningSpotNumbers     []int        `json:"winning_spot_numbers,omitempty" db:"winning_spot_numbers"`
	WinningInstagramHandles []string     `json:"winning_instagram_handles,omitempty" db:"winning_instagram_handles"`
	InstagramMediaLinks    []string     `json:"instagram_media_links,omitempty" db:"instagram_media_links"`
	Archived               bool         `json:"archived" db:"archived"`
	CreatedAt              time.Time    `json:"created_at" db:"created_at"`
	CompletedAt            *time.Time   `json:"completed_at,omitempty" db:"completed_at"`
}

type WinnerInfo struct {
	SpotNumber      int
	InstagramHandle string
}

func (w Waffle) Winners() []WinnerInfo {
	var winners []WinnerInfo
	for i := 0; i < len(w.WinningSpotNumbers); i++ {
		handle := ""
		if i < len(w.WinningInstagramHandles) {
			handle = w.WinningInstagramHandles[i]
		}
		winners = append(winners, WinnerInfo{
			SpotNumber:      w.WinningSpotNumbers[i],
			InstagramHandle: handle,
		})
	}
	return winners
}

type Spot struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	WaffleID        uuid.UUID  `json:"waffle_id" db:"waffle_id"`
	Number          int        `json:"number" db:"number"`
	Status          SpotStatus `json:"status" db:"status"`
	ClaimedByHandle *string    `json:"claimed_by_handle,omitempty" db:"claimed_by_handle"`
	ClaimedAt       *time.Time `json:"claimed_at,omitempty" db:"claimed_at"`
	PaidAt          *time.Time `json:"paid_at,omitempty" db:"paid_at"`
}

type ActivityEvent struct {
	ID              uuid.UUID `json:"id" db:"id"`
	WaffleID        uuid.UUID `json:"waffle_id" db:"waffle_id"`
	EventType       string    `json:"event_type" db:"event_type"`
	Message         string    `json:"message" db:"message"`
	InstagramHandle *string   `json:"instagram_handle,omitempty" db:"instagram_handle"`
	SpotNumbers     []int     `json:"spot_numbers,omitempty" db:"spot_numbers"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type BuyerStats struct {
	InstagramHandle     string    `json:"instagram_handle" db:"instagram_handle"`
	TotalWafflesEntered int       `json:"total_waffles_entered" db:"total_waffles_entered"`
	TotalWins           int       `json:"total_wins" db:"total_wins"`
	TotalLosses         int       `json:"total_losses" db:"total_losses"`
	TotalSpotsClaimed   int       `json:"total_spots_claimed" db:"total_spots_claimed"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

type BuyerStatsWithRank struct {
	BuyerStats
	Rank int `json:"rank"`
}

type BuyerWaffleHistory struct {
	WaffleID               uuid.UUID  `json:"waffle_id"`
	Slug                   string     `json:"slug"`
	Title                  string     `json:"title"`
	SpotPrice              int        `json:"spot_price"`
	Status                 string     `json:"status"`
	WinningSpotNumber      *int       `json:"winning_spot_number,omitempty"`
	WinningInstagramHandle *string    `json:"winning_instagram_handle,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	CompletedAt            *time.Time `json:"completed_at,omitempty"`
	SpotNumbers            []int      `json:"spot_numbers"`
	IsWinner               bool       `json:"is_winner"`
}

type CreateWaffleRequest struct {
	Title               string      `json:"title"`
	Description         *string     `json:"description,omitempty"`
	ImageURL            *string     `json:"image_url,omitempty"`
	TotalSpots          int         `json:"total_spots"`
	SpotPrice           int         `json:"spot_price"`
	PaymentInfo         *string     `json:"payment_info,omitempty"`
	ItemCount           int         `json:"item_count"`
	InstagramMediaLinks []string    `json:"instagram_media_links,omitempty"`
	PaymentMethodIDs    []uuid.UUID `json:"payment_method_ids,omitempty"`
}

type UpdateWaffleRequest struct {
	Title               string      `json:"title"`
	Description         *string     `json:"description,omitempty"`
	ImageURL            *string     `json:"image_url,omitempty"`
	SpotPrice           int         `json:"spot_price"`
	PaymentInfo         *string     `json:"payment_info,omitempty"`
	ItemCount           int         `json:"item_count"`
	InstagramMediaLinks []string    `json:"instagram_media_links,omitempty"`
	Archived            *bool       `json:"archived,omitempty"`
	PaymentMethodIDs    []uuid.UUID `json:"payment_method_ids,omitempty"`
}

type CreateClaimRequest struct {
	WaffleID        string `json:"waffle_id"`
	Spots           []int  `json:"spots"`
	InstagramHandle string `json:"instagram_handle"`
}

type RandomClaimRequest struct {
	WaffleID        string `json:"waffle_id"`
	Count           int    `json:"count"`
	InstagramHandle string `json:"instagram_handle"`
}

type SocialLink struct {
	Platform string `json:"platform"`
	Handle   string `json:"handle"`
}

var ValidSocialPlatforms = []string{"instagram", "tiktok", "x", "facebook", "youtube", "discord"}

type Admin struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	Username    string       `json:"username" db:"username"`
	Email       string       `json:"email" db:"email"`
	DisplayName *string      `json:"display_name,omitempty" db:"display_name"`
	FirstName   *string      `json:"first_name,omitempty" db:"first_name"`
	LastName    *string      `json:"last_name,omitempty" db:"last_name"`
	SocialLinks []SocialLink `json:"social_links,omitempty" db:"social_links"`
	Role        string       `json:"role" db:"role"`
	Timezone    string       `json:"timezone" db:"timezone"`
	Active      bool         `json:"active" db:"active"`
	LastLoginAt *time.Time   `json:"last_login_at,omitempty" db:"last_login_at"`
	LastLoginIP *string      `json:"last_login_ip,omitempty" db:"last_login_ip"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

type AuditLog struct {
	ID         uuid.UUID `json:"id" db:"id"`
	AdminID    uuid.UUID `json:"admin_id" db:"admin_id"`
	Action     string    `json:"action" db:"action"`
	TargetType string    `json:"target_type" db:"target_type"`
	TargetID   string    `json:"target_id" db:"target_id"`
	Details    string    `json:"details" db:"details"`
	IPAddress  string    `json:"ip_address" db:"ip_address"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type SystemSetting struct {
	Key       string     `json:"key" db:"key"`
	Value     string     `json:"value" db:"value"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	UpdatedBy *uuid.UUID `json:"updated_by,omitempty" db:"updated_by"`
}

type LoginHistory struct {
	ID          uuid.UUID `json:"id" db:"id"`
	AdminID     uuid.UUID `json:"admin_id" db:"admin_id"`
	IPAddress   string    `json:"ip_address" db:"ip_address"`
	UserAgent   string    `json:"user_agent" db:"user_agent"`
	Browser     string    `json:"browser" db:"browser"`
	OS          string    `json:"os" db:"os"`
	DeviceType  string    `json:"device_type" db:"device_type"`
	IPOrg       string    `json:"ip_org" db:"ip_org"`
	IPCountry   string    `json:"ip_country" db:"ip_country"`
	IPCity      string    `json:"ip_city" db:"ip_city"`
	IPASN       string    `json:"ip_asn" db:"ip_asn"`
	WhoisServer string    `json:"whois_server" db:"whois_server"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type AdminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type User struct {
	ID              uuid.UUID `json:"id" db:"id"`
	InstagramHandle string    `json:"instagram_handle" db:"instagram_handle"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type CreateAdminRequest struct {
	Username    string  `json:"username"`
	Email       *string `json:"email,omitempty"`
	Password    string  `json:"password"`
	FirstName   *string `json:"first_name,omitempty"`
	LastName    *string `json:"last_name,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Role        string  `json:"role"`
}

type UpdateAdminRequest struct {
	DisplayName *string `json:"display_name,omitempty"`
	Role        *string `json:"role,omitempty"`
	Active      *bool   `json:"active,omitempty"`
}

type UpdateAdminProfileRequest struct {
	FirstName   *string      `json:"first_name,omitempty"`
	LastName    *string      `json:"last_name,omitempty"`
	Email       *string      `json:"email,omitempty"`
	SocialLinks []SocialLink `json:"social_links,omitempty"`
}

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type PasswordResetConfirm struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type SetWinnerRequest struct {
	WinningSpotNumber  int   `json:"winning_spot_number"`
	WinningSpotNumbers []int `json:"winning_spot_numbers"`
}

type DroughtEntry struct {
	InstagramHandle string    `json:"instagram_handle"`
	TotalEntries    int       `json:"total_entries"`
	LastEntryDate   time.Time `json:"last_entry_date"`
	LongestDrought  int       `json:"longest_drought"`
}

type PowerBuyerEntry struct {
	InstagramHandle string  `json:"instagram_handle"`
	TotalSpots      int     `json:"total_spots_claimed"`
	TotalSpent      int     `json:"total_spent"`
	WinRate         float64 `json:"win_rate"`
}

type MonthlyActivity struct {
	Month        string `json:"month"`
	Waffles      int    `json:"waffles"`
	SpotsClaimed int    `json:"spots_claimed"`
	Revenue      int    `json:"revenue"`
}

type SpotVelocity struct {
	Status             string  `json:"status"`
	WaffleCount        int     `json:"waffle_count"`
	AvgFirstClaimHours float64 `json:"avg_first_claim_hours"`
	AvgCompletionHours float64 `json:"avg_completion_hours"`
}

type WHOISResult struct {
	Organization *string `json:"organization,omitempty"`
	Country      *string `json:"country,omitempty"`
	City         *string `json:"city,omitempty"`
	ASN          *string `json:"asn,omitempty"`
	Raw          string  `json:"-"`
}

type PaymentMethod struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Type        string    `json:"type" db:"type"`
	DisplayName string    `json:"display_name" db:"display_name"`
	HandleOrURL string    `json:"handle_or_url" db:"handle_or_url"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type CreatePaymentMethodRequest struct {
	Type        string `json:"type" binding:"required"`
	DisplayName string `json:"display_name" binding:"required"`
	HandleOrURL string `json:"handle_or_url" binding:"required"`
}

type UpdatePaymentMethodRequest struct {
	DisplayName *string `json:"display_name"`
	HandleOrURL *string `json:"handle_or_url"`
}
