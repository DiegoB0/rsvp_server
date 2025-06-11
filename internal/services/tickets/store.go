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
func (s *Store) GenerateTickets(guestID int, confirmAttendance bool) ([]byte, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin the transaction %w", err)
	}

	// Rollback if anything happens
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after rollback

		} else if err != nil {
			tx.Rollback() // err is named return value
		} else {
			err = tx.Commit()
		}
	}()

	guest, err := s.getGuestByID(tx, guestID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guest: %w", err)
	}

	// Update Confirm Attendance in case is true.
	if guest.ConfirmAttendance != confirmAttendance {
		_, err := tx.Exec(`UPDATE guests SET confirm_attendance = $1 WHERE id = $2`, confirmAttendance, guestID)
		if err != nil {
			return nil, fmt.Errorf("failed to update attendance confirmation: %w", err)
		}

		guest.ConfirmAttendance = confirmAttendance
	}

	// Check if they have confirmed assistence yet
	if !guest.ConfirmAttendance {
		return nil, fmt.Errorf("user must confirm attendace before generating the ticket")
	}

	// Check if the ticket has been generated yet
	if guest.TicketGenerated {
		return nil, fmt.Errorf("ticket already generated for this guest")
	}

	pdfData, err := s.generateTicketsForGuest(tx, guest)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(`UPDATE guests SET ticket_generated = TRUE WHERE id = $1`, guest.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update guest status %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return pdfData, nil
}

// Internal function that fetches guest data by ID
func (s *Store) getGuestByID(tx *sql.Tx, guestID int) (*types.Guest, error) {
	var g types.Guest
	err := tx.QueryRow(`
		SELECT id, full_name, additionals, ticket_generated, confirm_attendance
		FROM guests
		WHERE id = $1
	`, guestID).Scan(&g.ID, &g.FullName, &g.Additionals, &g.TicketGenerated, &g.ConfirmAttendance)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// Internal function that generates tickets and builds the PDF
func (s *Store) generateTicketsForGuest(tx *sql.Tx, guest *types.Guest) ([]byte, error) {
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

	// Load the background image
	bgBytes, err := os.ReadFile("assets/Pase2.png")
	if err != nil {
		return nil, fmt.Errorf("failed to read background image: %w", err)
	}
	bgAlias := "bg"
	imgOpts := gofpdf.ImageOptions{
		ImageType: "PNG",
		ReadDpi:   false,
	}
	pdf.RegisterImageOptionsReader(bgAlias, imgOpts, bytes.NewReader(bgBytes))

	for idx, name := range names {
		code := generateUniqueCode()
		qrContent := fmt.Sprintf("INVITADO: %s\nFECHA: %s", name, time.Now().Format("2006-01-02 15:04:05"))

		qrBytes, err := generateQRCode(qrContent)
		if err != nil {
			return nil, fmt.Errorf("QR generation failed: %w", err)
		}

		if err := s.insertTicketIntoDB(tx, code, "named", &guest.ID); err != nil {
			return nil, fmt.Errorf("db insert failed: %w", err)
		}

		imgOpts := gofpdf.ImageOptions{
			ImageType: "PNG",
			ReadDpi:   false,
		}
		imageAlias := fmt.Sprintf("qr%d", idx)
		pdf.RegisterImageOptionsReader(imageAlias, imgOpts, bytes.NewReader(qrBytes))

		pdf.AddPage()

		// Layout proportions
		rightWidth := 58.0
		leftWidth := 202.0 - rightWidth

		pdf.ImageOptions(bgAlias, 0, 0, 200, 80, false, imgOpts, 0, "")

		// Guest Info (white text)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Arial", "B", 12)

		labelX := 35.0

		startY := 32.0
		lineSpacing := 7.0

		pdf.SetXY(labelX, startY)
		pdf.CellFormat(0, 6, toLatin1(fmt.Sprintf("Invitado: %s", name)), "", 0, "L", false, 0, "")

		pdf.SetXY(labelX, startY+lineSpacing)
		pdf.CellFormat(0, 6, fmt.Sprintf("Fecha: %s", weddingDate), "", 0, "L", false, 0, "")

		pdf.SetXY(labelX, startY+lineSpacing*2)
		pdf.CellFormat(0, 6, fmt.Sprintf("Lugar: %s", weddingPlace), "", 0, "L", false, 0, "")

		qrSize := 40.0
		ticketHeight := 82.0

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

func (s *Store) insertTicketIntoDB(tx *sql.Tx, code string, ticketType string, guestID *int) error {
	if guestID == nil {
		_, err := tx.Exec(`
			INSERT INTO tickets (code, type, guest_id, created_at)
			VALUES ($1, $2, NULL, $3)
		`, code, ticketType, time.Now())
		return err
	}

	_, err := tx.Exec(`
        INSERT INTO tickets (code, type, guest_id, created_at)
        VALUES ($1, $2, $3, $4)
    `, code, ticketType, *guestID, time.Now())

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
