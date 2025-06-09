package tickets

import (
	"bytes"
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/diegob0/rspv_backend/internal/types"
	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Public function to generate PDF for a given guest ID
func (s *Store) GenerateTicketsPDF(guestID int) ([]byte, error) {
	guest, err := s.getGuestByID(guestID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guest: %w", err)
	}
	return s.generateTicketsForGuest(guest)
}

// Internal function that fetches guest data by ID
func (s *Store) getGuestByID(guestID int) (*types.Guest, error) {
	var g types.Guest
	err := s.db.QueryRow(`
		SELECT id, full_name, additionals
		FROM guests
		WHERE id = $1
	`, guestID).Scan(&g.ID, &g.FullName, &g.Additionals)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// Internal function that generates tickets and builds the PDF
func (s *Store) generateTicketsForGuest(guest *types.Guest) ([]byte, error) {
	names := []string{guest.FullName}
	for i := 1; i <= guest.Additionals; i++ {
		names = append(names, fmt.Sprintf("Acompañante de %s", guest.FullName))
	}

	// Custom ticket size: 200mm wide x 80mm tall
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "mm",
		Size:    gofpdf.SizeType{Wd: 200, Ht: 80},
	})

	for idx, name := range names {
		code := generateUniqueCode()
		qrContent := fmt.Sprintf("INVITADO: %s\nFECHA: %s", name, time.Now().Format("2006-01-02 15:04:05"))

		qrBytes, err := generateQRCode(qrContent)
		if err != nil {
			return nil, fmt.Errorf("QR generation failed: %w", err)
		}

		err = s.insertTicketIntoDB(code, "named", guest.ID)
		if err != nil {
			return nil, fmt.Errorf("DB insert failed: %w", err)
		}

		imgOpts := gofpdf.ImageOptions{
			ImageType: "PNG",
			ReadDpi:   false,
		}

		imageAlias := fmt.Sprintf("qr%d", idx)
		pdf.RegisterImageOptionsReader(imageAlias, imgOpts, bytes.NewReader(qrBytes))

		pdf.AddPage()

		// Title
		pdf.SetFont("Arial", "B", 16)
		pdf.SetXY(0, 10)
		pdf.CellFormat(200, 10, "Boda de Carlos y Vane", "", 1, "C", false, 0, "")

		// Guest info (left side)
		pdf.SetFont("Arial", "", 12)
		pdf.SetXY(10, 30)
		pdf.MultiCell(100, 10, fmt.Sprintf("Invitado:\n%s", name), "", "L", false)

		// QR code (right side, centered vertically)
		pdf.ImageOptions(imageAlias, 145, 20, 40, 0, false, imgOpts, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("PDF output failed: %w", err)
	}

	return buf.Bytes(), nil
}

func generateQRCode(content string) ([]byte, error) {
	return qrcode.Encode(content, qrcode.Medium, 256)
}

func generateUniqueCode() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.Itoa(rand.Intn(1000))
}

func (s *Store) insertTicketIntoDB(code string, ticketType string, guestID int) error {
	_, err := s.db.Exec(`
		INSERT INTO tickets (code, type, guest_id, created_at)
		VALUES ($1, $2, $3, $4)
	`, code, ticketType, guestID, time.Now())
	return err
}
