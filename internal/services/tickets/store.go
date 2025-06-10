package tickets

import (
	"bytes"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/diegob0/rspv_backend/internal/types"
	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
	"golang.org/x/text/encoding/charmap"
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

	weddingDate := os.Getenv("WEDDING_DATE")
	weddingPlace := os.Getenv("WEDDING_PLACE")

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

		var insertErr error
		if idx == 0 {
			insertErr = s.insertTicketIntoDB(code, "named", &guest.ID)
		} else {
			insertErr = s.insertTicketIntoDB(code, "named", nil)
		}

		if insertErr != nil {
			return nil, fmt.Errorf("DB insert failed: %w", insertErr)
		}

		imgOpts := gofpdf.ImageOptions{
			ImageType: "PNG",
			ReadDpi:   false,
		}
		imageAlias := fmt.Sprintf("qr%d", idx)
		pdf.RegisterImageOptionsReader(imageAlias, imgOpts, bytes.NewReader(qrBytes))

		pdf.AddPage()

		// Layout proportions
		rightWidth := 60.0
		leftWidth := 200.0 - rightWidth

		// Left background
		pdf.SetFillColor(123, 46, 46)
		pdf.Rect(0, 0, leftWidth, 80, "F")

		// Title "C & V"
		pdf.SetTextColor(212, 175, 55)
		pdf.SetFont("Arial", "I", 28)
		pdf.SetXY(0, 10)
		pdf.CellFormat(leftWidth, 15, "C & V", "", 0, "C", false, 0, "")

		// Guest Info (white text)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Arial", "", 12)

		labelX := 10.0

		startY := 35.0
		lineSpacing := 7.0

		pdf.SetXY(labelX, startY)
		pdf.CellFormat(0, 6, toLatin1(fmt.Sprintf("Invitado: %s", name)), "", 0, "L", false, 0, "")

		pdf.SetXY(labelX, startY+lineSpacing)
		pdf.CellFormat(0, 6, fmt.Sprintf("Fecha: %s", weddingDate), "", 0, "L", false, 0, "")

		pdf.SetXY(labelX, startY+lineSpacing*2)
		pdf.CellFormat(0, 6, fmt.Sprintf("Lugar: %s", weddingPlace), "", 0, "L", false, 0, "")

		qrSize := 40.0
		ticketHeight := 80.0

		qrX := leftWidth + (rightWidth-qrSize)/2
		qrY := (ticketHeight - qrSize) / 2
		pdf.ImageOptions(imageAlias, qrX, qrY, qrSize, 0, false, imgOpts, 0, "")

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

func (s *Store) insertTicketIntoDB(code string, ticketType string, guestID *int) error {
	var err error
	if guestID == nil {
		_, err = s.db.Exec(`
			INSERT INTO tickets (code, type, guest_id, created_at)
			VALUES ($1, $2, NULL, $3)
		`, code, ticketType, time.Now())
	} else {
		_, err = s.db.Exec(`
			INSERT INTO tickets (code, type, guest_id, created_at)
			VALUES ($1, $2, $3, $4)
		`, code, ticketType, *guestID, time.Now())
	}
	return err
}

// Convert UTF strings to Latin format
func toLatin1(input string) string {
	encoder := charmap.ISO8859_1.NewEncoder()
	output, err := encoder.String(input)
	if err != nil {
		// Fallback to original in case of error
		return input
	}

	return output
}
