package types

import "time"

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
}

type GuestStore interface {
	CreateGuest(Guest) error
	GetGuestByID(id int) (*Guest, error)
	GetGuests()
	GetTicketPerGuest()
	GetGuestAndTable()
	AssignGuest(Guest) error
	DeleteGuest(id int) error
	UpdateGuest(Guest) error
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

// Payloads for the tickets

// Responses
type ErrorResponse struct {
	Error string `json:"error"`
}

type LoginSuccessResponse struct {
	Token string `json:"token"`
}
