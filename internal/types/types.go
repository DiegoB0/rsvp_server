package types

import "time"

// Create signatures for each service
type UserStore interface {
	GetUserByEmail(email string) (*User, error)
	GetUserByID(id int) (*User, error)
	CreateUser(User) error
	GetUsers() ([]User, error)
	DeleteUser(id int) error
	UpdateUser(User) error
}

type TableStore interface {
	CreateTable(Table) error
	DeleteTable(Table) error
	AssignGuest(Guest) error
	GetTableByID(id int) (*Table, error)
	GetTables()
	UpdateTable(Table)
}

type GuestStore interface {
	CreateGuest(Guest) error
	ModifyGuest(Guest) error
	GetGuestByID(id int) (*Guest, error)
	GetGuests()
}

type TicketStore interface {
	GenerateTickets(Ticket) error
	GenerateGeneralTickets(Ticket) error
	ScanQr(Ticket) error
	GetTicketsCount()
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

type Guest struct {
	ID                int       `json:"id"`
	FullName          string    `json:"fullName"`
	ConfirmAttendance bool      `json:"confirmAttendance"`
	TableId           int       `json:"tableId"`
	CreatedAt         time.Time `json:"createdAt"`
}

type Ticket struct {
	ID         int       `json:"id"`
	QrCode     string    `json:"qrCode"`
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
type RegisterUserPayload struct {
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=3,max=130"`
}

type LoginUserPayload struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type GetUsersPayload struct {
	FirstName string `json:"firstName" validate:"required"`
	LastName  string `json:"lastName" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=3,max=130"`
	CreatedAt string `json:"createdAt"`
}
