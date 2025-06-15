package tickets

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/diegob0/rspv_backend/internal/services/jobs/queue"
	"github.com/diegob0/rspv_backend/internal/types"
	"github.com/jung-kurt/gofpdf"
	"github.com/lib/pq"
	"github.com/skip2/go-qrcode"
	"golang.org/x/text/encoding/charmap"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) GetTicketInfo(guestName string, confirmAttendance bool) ([]types.ReturnGuestMetadata, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin the transaction %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	normalized := normalizeName(guestName)

	var guestID int
	err = s.db.QueryRow(`
		SELECT id FROM guests WHERE LOWER(full_name) = $1
	`, normalized).Scan(&guestID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("guest %s not found", guestName)
		}
		return nil, fmt.Errorf("failed to find guest ID: %w", err)
	}

	guest, err := s.getGuestByID(tx, guestID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch guest: %w", err)
	}

	if guest.ConfirmAttendance != confirmAttendance {
		_, err := tx.Exec(`UPDATE guests SET confirm_attendance = $1 WHERE id = $2`, confirmAttendance, guestID)
		if err != nil {
			return nil, fmt.Errorf("failed to update attendance confirmation: %w", err)
		}

		guest.ConfirmAttendance = confirmAttendance
	}

	if !guest.ConfirmAttendance {
		return nil, fmt.Errorf("user must confirm attendance before generating the ticket")
	}

	_, err = tx.Exec(`UPDATE guests SET ticket_generated = TRUE WHERE id = $1`, guest.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update guest status %w", err)
	}

	rows, err := tx.Query(`SELECT qr_code_urls, pdf_files FROM guests WHERE id = $1`, guestID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tickets: %w", err)
	}
	defer rows.Close()

	var qrCodes []string
	var pdfs []string

	for rows.Next() {
		var qrURLRaw, pdfURLRaw string
		if err := rows.Scan(&qrURLRaw, &pdfURLRaw); err != nil {
			return nil, fmt.Errorf("failed to scan ticket data: %w", err)
		}

		// Clean the strings no stupid {} characters
		qrURLClean := strings.Trim(qrURLRaw, "{}")
		pdfURLClean := strings.Trim(pdfURLRaw, "{}")

		qrCodes = append(qrCodes, qrURLClean)
		pdfs = append(pdfs, pdfURLClean)
	}

	var tableName *string
	if guest.TableId != nil {
		err = tx.QueryRow(`SELECT name FROM tables WHERE id = $1`, *guest.TableId).Scan(&tableName)
		if err != nil {
			if err != sql.ErrNoRows {
				return nil, fmt.Errorf("failed to fetch table name: %w", err)
			}
		}
	}

	metadata := types.ReturnGuestMetadata{
		GuestName:   guest.FullName,
		Additionals: guest.Additionals,
		TableName:   tableName,
		QRCodes:     qrCodes,
		PDFiles:     pdfs,
	}

	return []types.ReturnGuestMetadata{metadata}, nil
}

// Public function to activate the tickets(generate them and store them in s3)
func (s *Store) GenerateTickets(guestID int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin the transaction %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)

		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	guest, err := s.getGuestByID(tx, guestID)
	if err != nil {
		return fmt.Errorf("failed to fetch guest: %w", err)
	}

	if guest.TicketGenerated {
		return fmt.Errorf("ticket already generated for this guest")
	}

	qrCodes, pdfData, err := s.generateTicketsForGuest(tx, guest)
	if err != nil {
		return err
	}

	// Upload que qr code as a background job
	var base64Qrs []string
	for _, qr := range qrCodes {
		base64Qrs = append(base64Qrs, base64.StdEncoding.EncodeToString(qr))
	}

	job := queue.QrUploadJob{
		TicketID: guest.ID,
		QrCodes:  base64Qrs,
	}

	jobJson, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal QR job: %w", err)
	}

	if err := queue.EnqueueJob(context.Background(), queue.QrJobQueue, string(jobJson)); err != nil {
		return fmt.Errorf("failed to enqueue QR upload job: %w", err)
	}

	// Upload the pdf file as background job
	base64PDF := base64.StdEncoding.EncodeToString(pdfData)

	pdfJob := queue.PdfUploadJob{
		TicketID:  guest.ID,
		PDFBase64: base64PDF,
	}

	pdfJobJSON, err := json.Marshal(pdfJob)
	if err != nil {
		return fmt.Errorf("failed to marshal PDF job: %w", err)
	}

	if err := queue.EnqueueJob(context.Background(), queue.PdfJobQueue, string(pdfJobJSON)); err != nil {
		return fmt.Errorf("failed to enqueue PDF upload job: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

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

func (s *Store) generateTicketsForGuest(tx *sql.Tx, guest *types.Guest) ([][]byte, []byte, error) {
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

	bgBytes, err := os.ReadFile("assets/Pase2.png")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read background image: %w", err)
	}
	bgAlias := "bg"
	imgOpts := gofpdf.ImageOptions{
		ImageType: "PNG",
		ReadDpi:   false,
	}
	pdf.RegisterImageOptionsReader(bgAlias, imgOpts, bytes.NewReader(bgBytes))

	var qrCodes [][]byte

	for idx, name := range names {
		code := generateUniqueCode()
		qrContent := fmt.Sprintf("INVITADO: %s\nFECHA: %s", name, time.Now().Format("2006-01-02 15:04:05"))

		qrBytes, err := generateQRCode(qrContent)
		if err != nil {
			return nil, nil, fmt.Errorf("QR generation failed: %w", err)
		}

		if err := s.insertTicketIntoDB(tx, code, "named", &guest.ID); err != nil {
			return nil, nil, fmt.Errorf("db insert failed: %w", err)
		}

		imgOpts := gofpdf.ImageOptions{
			ImageType: "PNG",
			ReadDpi:   false,
		}
		imageAlias := fmt.Sprintf("qr%d", idx)
		pdf.RegisterImageOptionsReader(imageAlias, imgOpts, bytes.NewReader(qrBytes))

		pdf.AddPage()

		rightWidth := 58.0
		leftWidth := 202.0 - rightWidth

		pdf.ImageOptions(bgAlias, 0, 0, 200, 80, false, imgOpts, 0, "")

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

		qrCodes = append(qrCodes, qrBytes)

	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, nil, fmt.Errorf("PDF output failed: %w", err)
	}
	return qrCodes, buf.Bytes(), nil
}

func generateQRCode(content string) ([]byte, error) {
	return qrcode.Encode(content, qrcode.Medium, 256)
}

func generateUniqueCode() string {
	return strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.Itoa(rand.Intn(1000))
}

// Helper queries
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

// Add the URL's from aws to the tickets table
func (s *Store) UpdateQrCodeUrls(guestID int, urls []string) error {
	res, err := s.db.Exec(`
		UPDATE guests
		SET qr_code_urls = $1
		WHERE id = $2
	`, pq.Array(urls), guestID)
	if err != nil {
		return fmt.Errorf("failed to update qr_code_urls for guest %d: %w", guestID, err)
	}

	rows, _ := res.RowsAffected()

	if rows == 0 {
		log.Printf("⚠️ No rows updated for guest ID %d", guestID)
	} else {
		log.Printf("✅ Updated %d rows for guest ID %d", rows, guestID)
	}

	return nil
}

func (s *Store) UpdatePDFfileUrls(guestID int, urls []string) error {
	log.Printf("🧾 Updating pdf_files for guest %d with URLs: %v", guestID, urls)

	res, err := s.db.Exec(`
		UPDATE guests
		SET pdf_files = $1
		WHERE id = $2
	`, pq.Array(urls), guestID)
	if err != nil {
		return fmt.Errorf("failed to update pdf_files for guest %d: %w", guestID, err)
	}

	rows, _ := res.RowsAffected()

	if rows == 0 {
		log.Printf("⚠️ No rows updated for guest ID %d", guestID)
	} else {
		log.Printf("✅ Updated %d rows for guest ID %d", rows, guestID)
	}

	return nil
}

// Helpers
func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	re := regexp.MustCompile(`\s+`)
	name = re.ReplaceAllString(name, " ")
	return name
}

// Convert UTF strings to Latin format
func toLatin1(input string) string {
	encoder := charmap.ISO8859_1.NewEncoder()
	output, err := encoder.String(input)
	if err != nil {
		return input
	}

	return output
}
