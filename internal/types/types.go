package types

import (
	"time"
)

// Create signatures for each service
type UserStore interface {
	CreateUser(User) error
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id int) (*User, error)
	GetUsers() ([]User, error)
	DeleteUser(id int) error
	UpdateUser(*User) error
}

type TableStore interface {
	CreateTable(Table) error
	GetTableByName(name string) (*Table, error)
	GetTableByID(id int) (*Table, error)
	GetTables() ([]Table, error)
	DeleteTable(id int) error
	UpdateTable(*Table) error
	GetTableWithGuestsByID(tableID int) (*TableAndGuests, error)
	GetTablesWithGuests() ([]TableAndGuests, error)
	// BatchInsert([]Table) error
}

type GuestStore interface {
	CreateGuest(Guest) error
	GetGuestByID(id int) (*Guest, error)
	GetGuests() ([]Guest, error)
	GetGuestByName(name string) (*Guest, error)
	DeleteGuest(id int) error
	UpdateGuest(*Guest) error
	AssignGuest(guestID int, tableID int) error
	UnassignGuest(guestID int) error
	GetTicketsPerGuest(guestID int) ([]GuestWithTickets, error)
	// BatchInsert([]Guest) error
}

type TicketStore interface {
	GenerateTicket(guestID int) error
	GetTicketInfo(guestName string, confirmAttendance bool, email string) ([]ReturnGuestMetadata, error)
	RegenerateTicket(guestID int) ([]byte, error)
	ScanQR(code string) (*ReturnScanedData, error)

	GenerateAllTickets() error
	GenerateGeneralTicket(count int) (err error)
	GenerateGeneral(generalID int) ([]byte, error)

	GetGeneralTicketsInfo() ([]GeneralTicket, error)
	// GetNamedTicketsInfo() ([]NamedTicket, error)
	GetTicketsCount() (AllTickets, error)
}

type GeneralStore interface {
	DeleteGeneral(id int) error
	AssignGeneral(generalID int, tableID int) error
	UnassignGeneral(generalID int) error
}

type PhotoStore interface {
	UploadPhoto(Photo) error
	DeletePhoto(Photo) error
}

type NotificationStore interface {
	SendNotifications(Notification) error
}

// Structures for each table
type User struct {
	ID        int       `json:"id"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"emal"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"createdAt"`
}

type Table struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Capacity  int       `json:"capacity"`
	CreatedAt time.Time `json:"createdAt"`
}

type TableAndGuests struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Capacity  int       `json:"capacity"`
	CreatedAt time.Time `json:"createdAt"`
	Guests    []Guest   `json:"guests"`
}

type GuestWithTickets struct {
	ID                int       `json:"id"`
	FullName          string    `json:"fullName"`
	Additionals       int       `json:"additionals"`
	ConfirmAttendance bool      `json:"confirmAttendance"`
	TableId           *int      `json:"tableId"`
	QrCodeUrls        []string  `json:"qrCodeUrls"`
	TicketGenerated   bool      `json:"ticketGenerated"`
	CreatedAt         time.Time `json:"createdAt"`
}

type Guest struct {
	ID                int       `json:"id"`
	FullName          string    `json:"fullName"`
	Additionals       int       `json:"additionals"`
	ConfirmAttendance bool      `json:"confirmAttendance"`
	TableId           *int      `json:"tableId"`
	TicketGenerated   bool      `json:"ticketGenerated"`
	TicketSent        bool      `json:"ticketSent"`
	CreatedAt         time.Time `json:"createdAt"`
}

type General struct {
	ID        int       `json:"id"`
	Folio     int       `json:"folio"`
	TableId   *int      `json:"tableId"`
	QrCodeUrl string    `json:"qrCodeUrl"`
	PDFUrl    string    `json:"pdfUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

type GeneralTicket struct {
	ID        int       `json:"id"`
	Folio     int       `json:"folio"`
	TableId   *int      `json:"tableId"`
	QrCodeUrl string    `json:"qrCodeUrl"`
	PDFUrl    string    `json:"pdfUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

type NamedTicket struct {
	ID                int       `json:"id"`
	FullName          string    `json:"fullName"`
	Additionals       int       `json:"additionals"`
	ConfirmAttendance bool      `json:"confirmAttendance"`
	TableId           *int      `json:"tableId"`
	TicketGenerated   bool      `json:"ticketGenerated"`
	TicketSent        bool      `json:"ticketSent"`
	QRCodes           []string  `json:"qrCodes"`
	PDFiles           string    `json:"pdfiles"`
	CreatedAt         time.Time `json:"createdAt"`
}

type AllTickets struct {
	NamedTickets      int `json:"namedTickets"`
	GeneralTickets    int `json:"generalTickets"`
	TotalTickets      int `json:"totalTickets"`
	GuestTotal        int `json:"guestTotal"`
	GuestConfirmed    int `json:"guestConfirmed"`
	GuestNotConfirmed int `json:"guestNotConfirmed"`
}

type Ticket struct {
	ID         int       `json:"id"`
	QrCode     string    `json:"qrCode"`
	QrCodeUrls []string  `json:"qrCodeUrls"`
	TicketType string    `json:"ticketType"`
	GuestId    int       `json:"guestId"`
	CreatedAt  time.Time `json:"createdAt"`
}

type Notification struct {
	ID int `json:"id"`
}

type Photo struct {
	ID        int    `json:"id"`
	Photo_URL string `json:"photo_url"`
}

// JSON Payloads

// Payloads for the users
type RegisterUserPayload struct {
	FirstName string `json:"firstName" validate:"required" example:"Uri"`
	LastName  string `json:"lastName" validate:"required" example:"La creatura de la noche"`
	Email     string `json:"email" validate:"required,email" example:"uri@uri.com"`
	Password  string `json:"password" validate:"required,min=3,max=130" example:"1234"`
}

type LoginUserPayload struct {
	Email    string `json:"email" validate:"required,email" example:"me@me.com"`
	Password string `json:"password" validate:"required" example:"cum"`
}

type GetUsersPayload struct {
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=3,max=130"`
	CreatedAt string `json:"createdAt"`
}

type UpdateUserPayload struct {
	FirstName *string `json:"firstName,omitempty" example:"Uri"`
	LastName  *string `json:"lastName,omitempty" example:"La creatura de la noche"`
	Email     *string `json:"email,omitempty" validate:"omitempty,email" example:"uri@uri.com"`
	Password  *string `json:"password,omitempty" validate:"omitempty,min=3,max=130" example:"123"`
}

// Payloads for the tables
type CreateTablePayload struct {
	Name     string `json:"name" validate:"required" example:"Mesa 1"`
	Capacity int    `json:"capacity,omitempty" example:"10"`
}

type UpdateTablePayload struct {
	Name     *string `json:"name,omitempty" example:"Mesa 1"`
	Capacity *int    `json:"capacity,omitempty" example:"10"`
}

// Payloads for the guests
type CreateGuestPayload struct {
	FullName          string `json:"fullName" validate:"required" example:"Juan Perez"`
	Additionals       *int   `json:"additionals" validate:"required" example:"0"`
	ConfirmAttendance *bool  `json:"confirmAttendance" validate:"required" example:"false"`
}

type UpdateGuestPayload struct {
	FullName          *string `json:"fullName,omitempty" example:"Eduardo Garcia"`
	Additionals       *int    `json:"additionals,omitempty" example:"0"`
	ConfirmAttendance *bool   `json:"confirmAttendance,omitempty" example:"false"`
}

// Payloads for the tickets
type ReturnGuestMetadata struct {
	GuestName   string   `json:"guestName"`
	Additionals int      `json:"additionals"`
	TableName   *string  `json:"tableName,omitempty"`
	QRCodes     []string `json:"qrCodes"`
	PDFiles     string   `json:"pdfiles"`
}

// Return payload after scan ticket
type ReturnScanedData struct {
	GuestName    string  `json:"guestName"`
	TableName    *string `json:"tableName,omitempty"`
	TicketStatus string  `json:"ticketStatus"`
}

type ReturnPDFile struct {
	PDFiles []string `json:"pdfiles"`
}

// Responses
type ErrorResponse struct {
	Error string `json:"error"`
}

type LoginSuccessResponse struct {
	Token string `json:"token"`
}
