package tickets

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/diegob0/rspv_backend/internal/services/jobs/queue"
	"github.com/diegob0/rspv_backend/internal/types"
	"github.com/diegob0/rspv_backend/internal/utils"
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

// Get the guest
func (s *Store) getGuestByID(tx *sql.Tx, guestID int) (*types.Guest, error) {
	var g types.Guest
	err := tx.QueryRow(`
		SELECT id, full_name, additionals, ticket_generated, confirm_attendance, ticket_sent
		FROM guests
		WHERE id = $1
	`, guestID).Scan(&g.ID, &g.FullName, &g.Additionals, &g.TicketGenerated, &g.ConfirmAttendance, &g.TicketSent)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// Regenerate tickets
func (s *Store) RegenerateTicket(guestID int) ([]byte, error) {
	var pdfURL string

	err := s.db.QueryRow(`
	SELECT pdf_files
	FROM guests
	WHERE id = $1
`, guestID).Scan(&pdfURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("guest not found")
		}
		return nil, err
	}

	if pdfURL == "" {
		return nil, errors.New("no PDF file found for guest")
	}

	pdfBytes, err := downloadPDF(pdfURL)
	if err != nil {
		return nil, err
	}

	return pdfBytes, nil
}

// Generate generals
func (s *Store) GenerateGeneral(generalID int) ([]byte, error) {
	var pdfURL string

	err := s.db.QueryRow(`
	SELECT pdf_file
	FROM generals
	WHERE id = $1
`, generalID).Scan(&pdfURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("general not found")
		}
		return nil, err
	}

	if pdfURL == "" {
		return nil, errors.New("no PDF file found for guest")
	}

	pdfBytes, err := downloadPDF(pdfURL)
	if err != nil {
		return nil, err
	}

	return pdfBytes, nil
}

// Get the tikcet info
func (s *Store) GetTicketInfo(guestName string, confirmAttendance bool, email string) ([]types.ReturnGuestMetadata, error) {
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

	if guest.TicketSent {
		return nil, fmt.Errorf("ticket already generated for this guest: %v", guestID)
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

	_, err = tx.Exec(`UPDATE guests SET ticket_sent = TRUE WHERE id = $1`, guest.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update guest status %w", err)
	}

	var qrURLArrayRaw string
	var pdfURL string

	err = tx.QueryRow(`SELECT qr_code_urls, pdf_files FROM guests WHERE id = $1`, guestID).
		Scan(&qrURLArrayRaw, &pdfURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query ticket data: %w", err)
	}

	qrCodes := strings.Split(strings.Trim(qrURLArrayRaw, "{}"), ",")
	for i := range qrCodes {
		qrCodes[i] = strings.TrimSpace(qrCodes[i])
	}

	pdfs := []string{pdfURL}

	var tableName *string
	if guest.TableId != nil {
		err = tx.QueryRow(`SELECT name FROM tables WHERE id = $1`, *guest.TableId).Scan(&tableName)
		if err != nil && err != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to fetch table name: %w", err)
		}
	}

	// Send the email
	if email != "" {
		pdfURL := pdfs[0]
		if err != nil {
			log.Printf("error downloading pdf %v", err)
		}

		job := queue.EmailSendJob{
			GuestID:   guest.ID,
			Recipient: email,
			PDFURL:    pdfURL,
		}

		jobJson, err := json.Marshal(job)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal QR job: %w", err)
		}

		if err := queue.EnqueueJob(context.Background(), queue.EmailJobQueue, string(jobJson)); err != nil {
			return nil, fmt.Errorf("failed to enqueue QR upload job: %w", err)
		}

	}

	metadata := types.ReturnGuestMetadata{
		GuestName:   guest.FullName,
		Additionals: guest.Additionals,
		TableName:   tableName,
		QRCodes:     qrCodes,
		PDFiles:     pdfURL,
	}

	return []types.ReturnGuestMetadata{metadata}, nil
}

func (s *Store) GenerateAllTickets() error {
	guests, err := s.getAllGuestsWithoutTickets()
	if err != nil {
		return fmt.Errorf("failed to fetch guests without tickets: %w", err)
	}

	if len(guests) == 0 {
		log.Println("🎉 All guests already have tickets. Nothing to do.")
		return nil
	}

	for _, guest := range guests {
		err := s.GenerateTicket(guest.ID)
		if err != nil {

			log.Printf("failed to generate ticket for guest ID %d: %v", guest.ID, err)
			continue
		}
		log.Printf("successfully generated ticket for guest ID %d", guest.ID)
	}

	return nil
}

func (s *Store) getAllGuestsWithoutTickets() ([]*types.Guest, error) {
	rows, err := s.db.Query(`SELECT id, full_name, additionals, ticket_generated FROM guests WHERE ticket_generated = FALSE`)
	if err != nil {
		return nil, fmt.Errorf("failed to query guests: %w", err)
	}
	defer rows.Close()

	var guests []*types.Guest

	for rows.Next() {
		var g types.Guest
		if err := rows.Scan(&g.ID, &g.FullName, &g.Additionals, &g.TicketGenerated); err != nil {
			return nil, fmt.Errorf("failed to scan guest row: %w", err)
		}
		guests = append(guests, &g)

	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return guests, nil
}

// Public function to activate the tickets(generate them and store them in s3)
func (s *Store) GenerateTicket(guestID int) error {
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

	var base64Qrs []string
	for _, qr := range qrCodes {
		base64Qrs = append(base64Qrs, base64.StdEncoding.EncodeToString(qr))
	}

	base64PDF := base64.StdEncoding.EncodeToString(pdfData)

	job := queue.FullUploadJob{
		TicketID:   guest.ID,
		QrCodes:    base64Qrs,
		PDFBase64:  base64PDF,
		TicketType: "named",
	}

	jobJson, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("failed to marshal FullUpload job: %w", err)
	}

	if err := queue.EnqueueJob(context.Background(), queue.FullUploadQueue, string(jobJson)); err != nil {
		return fmt.Errorf("failed to enqueue FullUpload job: %w", err)
	}

	// Upload que qr code as a background job
	// var base64Qrs []string
	// for _, qr := range qrCodes {
	// 	base64Qrs = append(base64Qrs, base64.StdEncoding.EncodeToString(qr))
	// }
	//
	// job := queue.QrUploadJob{
	// 	TicketID:   guest.ID,
	// 	QrCodes:    base64Qrs,
	// 	TicketType: "named",
	// }
	//
	// jobJson, err := json.Marshal(job)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal QR job: %w", err)
	// }
	//
	// if err := queue.EnqueueJob(context.Background(), queue.QrJobQueue, string(jobJson)); err != nil {
	// 	return fmt.Errorf("failed to enqueue QR upload job: %w", err)
	// }
	//
	// // Upload the pdf file as background job
	// base64PDF := base64.StdEncoding.EncodeToString(pdfData)
	//
	// pdfJob := queue.PdfUploadJob{
	// 	TicketID:   guest.ID,
	// 	PDFBase64:  base64PDF,
	// 	TicketType: "named",
	// }
	//
	// pdfJobJSON, err := json.Marshal(pdfJob)
	// if err != nil {
	// 	return fmt.Errorf("failed to marshal PDF job: %w", err)
	// }
	//
	// if err := queue.EnqueueJob(context.Background(), queue.PdfJobQueue, string(pdfJobJSON)); err != nil {
	// 	return fmt.Errorf("failed to enqueue PDF upload job: %w", err)
	// }

	_, err = tx.Exec(`UPDATE guests SET ticket_generated = TRUE WHERE id = $1`, guestID)
	if err != nil {
		return fmt.Errorf("failed to update guest ticket_generated status: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
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

	bgBytes, err := os.ReadFile("assets/Pase3.png")
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
		// qrContent := fmt.Sprintf("INVITADO: %s\nFECHA: %s", name, time.Now().Format("2006-01-02 15:04:05"))

		qrBytes, err := generateQRCode(code)
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

// Scan QR
func (s *Store) ScanQR(code string) (types.QRScanResult, error) {
	var ticket struct {
		ID        int
		GuestID   sql.NullInt64
		GeneralID sql.NullInt64
		Status    string
	}

	err := s.db.QueryRow(`
		SELECT id, guest_id, general_id, status FROM tickets WHERE code = $1`, code).Scan(&ticket.ID, &ticket.GuestID, &ticket.GeneralID, &ticket.Status)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid code")
	} else if err != nil {
		return nil, fmt.Errorf("error consulting ticket: %w", err)
	}

	if ticket.Status == "used" {
		return nil, fmt.Errorf("this ticket was already used")
	}

	_, err = s.db.Exec(`UPDATE tickets SET status = 'used' WHERE id = $1`, ticket.ID)
	if err != nil {
		return nil, fmt.Errorf("error updating ticket: %w", err)
	}

	status := ticket.Status

	if ticket.GuestID.Valid {

		var guest struct {
			FullName    string
			Additionals int
			TableID     *int
		}

		err = s.db.QueryRow(`
		SELECT full_name, additionals, table_id FROM guests WHERE id = $1
	`, ticket.GuestID).Scan(&guest.FullName, &guest.Additionals, &guest.TableID)
		if err != nil {
			return nil, fmt.Errorf("error consulting the guest: %w", err)
		}

		name := guest.FullName
		if guest.Additionals > 0 {
			name += " y compañía"
		}

		var tableName *string
		if guest.TableID != nil {

			var tName string
			err = s.db.QueryRow(`SELECT name FROM tables WHERE id = $1`, *guest.TableID).Scan(&tName)
			if err != nil && err != sql.ErrNoRows {
				return nil, fmt.Errorf("error consulting the table: %w", err)
			}
			tableName = &tName

		}

		return &types.ReturnScannedData{
			GuestName:    name,
			TableName:    tableName,
			TicketStatus: status,
		}, nil

	} else if ticket.GeneralID.Valid {

		var general struct {
			Folio   *int
			TableID *int
		}

		err = s.db.QueryRow(`
		SELECT folio, table_id FROM generals WHERE id = $1
	`, ticket.GeneralID).Scan(&general.Folio, &general.TableID)
		if err != nil {
			return nil, fmt.Errorf("error consulting the general: %w", err)
		}

		folio := general.Folio

		var tableName *string
		if general.TableID != nil {

			var tName string
			err = s.db.QueryRow(`SELECT name FROM tables WHERE id = $1`, *general.TableID).Scan(&tName)
			if err != nil && err != sql.ErrNoRows {
				return nil, fmt.Errorf("error consulting the table: %w", err)
			}
			tableName = &tName

		}

		return &types.ReturnGeneralScannedData{
			GeneralFolio: folio,
			TableName:    tableName,
			TicketStatus: status,
		}, nil
	} else {
		return nil, fmt.Errorf("the code does not match with any ticket")
	}
}

// --- GENERAL TICKETS

// Generate the actual ticket
func (s *Store) GenerateGeneralTicket(count int) (err error) {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
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

	// Get the next folio
	var lastFolio int
	err = tx.QueryRow(`SELECT COALESCE(MAX(folio), 0) FROM generals`).Scan(&lastFolio)
	if err != nil {
		return fmt.Errorf("failed to fetch last general folio: %w", err)
	}

	for i := 0; i < count; i++ {
		nextFolio := lastFolio + 1
		lastFolio = nextFolio

		var generalID int
		err = tx.QueryRow(`
    INSERT INTO generals (table_id, created_at, folio)
    VALUES (NULL, NOW(), $1) RETURNING id
`, nextFolio).Scan(&generalID)
		if err != nil {
			return fmt.Errorf("failed to insert general: %w", err)
		}

		// Generate QR code and PDF
		code := generateUniqueCode()
		qrBytes, err := generateQRCode(code)
		if err != nil {
			return fmt.Errorf("failed to generate QR code: %w", err)
		}

		weddingDate := os.Getenv("WEDDING_DATE")

		weddingPlace := os.Getenv("WEDDING_PLACE")

		// Build PDF
		pdf := gofpdf.NewCustom(&gofpdf.InitType{
			UnitStr: "mm",
			Size:    gofpdf.SizeType{Wd: 200, Ht: 80},
		})

		bgBytes, err := os.ReadFile("assets/Pase3.png")
		if err != nil {
			return fmt.Errorf("failed to read background image: %w", err)
		}

		bgAlias := "bg"
		imgOpts := gofpdf.ImageOptions{ImageType: "PNG"}
		pdf.RegisterImageOptionsReader(bgAlias, imgOpts, bytes.NewReader(bgBytes))

		imageAlias := "qr0"
		pdf.RegisterImageOptionsReader(imageAlias, imgOpts, bytes.NewReader(qrBytes))

		pdf.AddPage()
		pdf.ImageOptions(bgAlias, 0, 0, 200, 80, false, imgOpts, 0, "")

		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Arial", "B", 12)

		labelX := 35.0
		startY := 32.0
		lineSpacing := 7.0

		pdf.SetXY(labelX, startY)

		pdf.CellFormat(0, 6, toLatin1(fmt.Sprintf("Invitado: General #%d", nextFolio)), "", 0, "L", false, 0, "")

		pdf.SetXY(labelX, startY+lineSpacing)
		pdf.CellFormat(0, 6, fmt.Sprintf("Fecha: %s", weddingDate), "", 0, "L", false, 0, "")

		pdf.SetXY(labelX, startY+lineSpacing*2)

		pdf.CellFormat(0, 6, fmt.Sprintf("Lugar: %s", weddingPlace), "", 0, "L", false, 0, "")

		qrSize := 40.0
		rightWidth := 58.0
		leftWidth := 202.0 - rightWidth
		qrX := leftWidth + (rightWidth-qrSize)/2
		qrY := (82.0 - qrSize) / 2

		pdf.ImageOptions(imageAlias, qrX, qrY, qrSize, 0, false, imgOpts, 0, "")

		var pdfBuf bytes.Buffer
		if err := pdf.Output(&pdfBuf); err != nil {
			return fmt.Errorf("failed to write PDF: %w", err)
		}
		pdfData := pdfBuf.Bytes()

		// Insert ticket linked to general
		if err := s.insertGeneralTicketIntoDB(tx, code, "general", &generalID); err != nil {
			return fmt.Errorf("failed to insert ticket: %w", err)
		}

		fullUploadJob := queue.FullUploadJob{
			TicketID:   generalID,
			QrCodes:    []string{base64.StdEncoding.EncodeToString(qrBytes)},
			PDFBase64:  base64.StdEncoding.EncodeToString(pdfData),
			TicketType: "general",
		}

		jobJSON, err := json.Marshal(fullUploadJob)
		if err != nil {
			return fmt.Errorf("failed to marshal FullUpload job: %w", err)
		}

		if err := queue.EnqueueJob(context.Background(), queue.FullUploadQueue, string(jobJSON)); err != nil {
			return fmt.Errorf("failed to enqueue FullUpload job: %w", err)
		}

		// Enqueue QR job
		// qrJob := queue.QrUploadJob{
		// 	TicketID:   generalID,
		// 	QrCodes:    []string{base64.StdEncoding.EncodeToString(qrBytes)},
		// 	TicketType: "general",
		// }
		//
		// qrJSON, err := json.Marshal(qrJob)
		// if err != nil {
		// 	return fmt.Errorf("failed to marshal QR job: %w", err)
		// }
		// if err := queue.EnqueueJob(context.Background(), queue.QrJobQueue, string(qrJSON)); err != nil {
		// 	return fmt.Errorf("failed to enqueue QR upload job: %w", err)
		// }
		//
		// // Enqueue PDF job
		// pdfJob := queue.PdfUploadJob{
		// 	TicketID:   generalID,
		// 	PDFBase64:  base64.StdEncoding.EncodeToString(pdfData),
		// 	TicketType: "general",
		// }
		// pdfJSON, err := json.Marshal(pdfJob)
		// if err != nil {
		// 	return fmt.Errorf("failed to marshal PDF job: %w", err)
		// }
		// if err := queue.EnqueueJob(context.Background(), queue.PdfJobQueue, string(pdfJSON)); err != nil {
		// 	return fmt.Errorf("failed to enqueue PDF upload job: %w", err)
		// }
	}

	return nil
}

// --- INFO ABOUT THE TICKETS (named and generals)
func (s *Store) GetTicketsCount() (types.AllTickets, error) {
	var result types.AllTickets

	err := s.db.QueryRow(`
		SELECT
			-- Count general tickets from generals table
			(SELECT COUNT(*) FROM generals) AS general_count,

			-- Named tickets = guests + their additionals

			(SELECT COALESCE(SUM(additionals + 1), 0) FROM guests) AS named_count,

			-- Total tickets = general_count + named_count
			(
				(SELECT COUNT(*) FROM generals) +
				(SELECT COALESCE(SUM(additionals + 1), 0) FROM guests)
			) AS total_count,

			-- Guests total = guests + additionals

			(SELECT COALESCE(SUM(additionals + 1), 0) FROM guests) AS total_guest_count,

			-- Guests confirmed count (guests + additionals)
			(SELECT COALESCE(SUM(additionals + 1), 0) FROM guests WHERE confirm_attendance = true) AS confirmed_guest_count,

			-- Guests not confirmed count (guests + additionals)
			(SELECT COALESCE(SUM(additionals + 1), 0) FROM guests WHERE confirm_attendance = false) AS not_confirmed_guest_count
	`).Scan(
		&result.GeneralTickets,
		&result.NamedTickets,

		&result.TotalTickets,
		&result.GuestTotal,

		&result.GuestConfirmed,
		&result.GuestNotConfirmed,
	)
	if err != nil {
		return types.AllTickets{}, fmt.Errorf("failed to fetch ticket and guest counts: %w", err)
	}

	return result, nil
}

func (s *Store) GetNamedTicketsInfo() ([]types.NamedTicket, error) {
	rows, err := s.db.Query(`
		SELECT id, full_name, additionals, confirm_attendance, table_id,
		       ticket_generated, ticket_sent, qr_code_urls, pdf_files, created_at
		FROM guests
		ORDER BY full_name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch named tickets: %w", err)
	}
	defer rows.Close()

	var tickets []types.NamedTicket

	for rows.Next() {
		var ticket types.NamedTicket

		err := rows.Scan(
			&ticket.ID,
			&ticket.FullName,
			&ticket.Additionals,
			&ticket.ConfirmAttendance,
			&ticket.TableId,
			&ticket.TicketGenerated,
			&ticket.TicketSent,
			pq.Array(&ticket.QRCodes),
			&ticket.PDFiles,
			&ticket.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan named ticket: %w", err)
		}
		tickets = append(tickets, ticket)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return tickets, nil
}

func (s *Store) GetGeneralTicketsInfo(params types.PaginationParams) (*types.PaginatedResult[types.GeneralTicket], error) {
	var whereClause string
	var args []interface{}
	orderBy := "folio"

	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		whereClause = " WHERE CAST(folio AS TEXT) ILIKE $1"
		args = append(args, "%"+strings.TrimSpace(*params.Search)+"%")
	}

	baseQuery := `
		SELECT id, folio, table_id, qr_code_url, pdf_file, created_at
		FROM generals
	` + whereClause

	countQuery := `SELECT COUNT(*) FROM generals` + whereClause

	return utils.Paginate(s.db, baseQuery, countQuery, func(rows *sql.Rows) (types.GeneralTicket, error) {
		var ticket types.GeneralTicket
		err := rows.Scan(
			&ticket.ID,
			&ticket.Folio,
			&ticket.TableId,
			&ticket.QrCodeUrl,
			&ticket.PDFUrl,
			&ticket.CreatedAt,
		)
		if err != nil {
			return types.GeneralTicket{}, err
		}
		return ticket, nil
	}, params, orderBy, args...)
}

func (s *Store) GetUnassignedGeneralTickets(params types.PaginationParams) (*types.PaginatedResult[types.GeneralTicket], error) {
	var andWhere string
	var args []interface{}
	orderBy := "folio"
	whereClause := " WHERE table_id IS NULL"

	if params.Search != nil && strings.TrimSpace(*params.Search) != "" {
		andWhere = " AND CAST(folio AS TEXT) ILIKE $1"
		args = append(args, "%"+strings.TrimSpace(*params.Search)+"%")
	}

	baseQuery := `
		SELECT id, folio, table_id, qr_code_url, pdf_file, created_at
		FROM generals
	` + whereClause + andWhere

	countQuery := `SELECT COUNT(*) FROM generals` + whereClause + andWhere

	return utils.Paginate(s.db, baseQuery, countQuery, func(rows *sql.Rows) (types.GeneralTicket, error) {
		var ticket types.GeneralTicket
		err := rows.Scan(
			&ticket.ID,
			&ticket.Folio,
			&ticket.TableId,
			&ticket.QrCodeUrl,
			&ticket.PDFUrl,
			&ticket.CreatedAt,
		)
		if err != nil {
			return types.GeneralTicket{}, err
		}
		return ticket, nil
	}, params, orderBy, args...)
}

// Helper functions
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

func (s *Store) insertGeneralTicketIntoDB(tx *sql.Tx, code string, ticketType string, generalID *int) error {
	if generalID == nil {
		_, err := tx.Exec(`
			INSERT INTO tickets (code, type, general_id, created_at)
			VALUES ($1, $2, NULL, $3)
		`, code, ticketType, time.Now())
		return err
	}

	_, err := tx.Exec(`
        INSERT INTO tickets (code, type, general_id, created_at)
        VALUES ($1, $2, $3, $4)
    `, code, ticketType, *generalID, time.Now())

	return err
}

// This for general tickets
func (s *Store) UpdateGeneralQrCodeUrls(generalID int, urls []string) error {
	var url string
	if len(urls) > 0 {
		url = urls[0]
	} else {
		url = ""
	}

	res, err := s.db.Exec(`
		UPDATE generals
		SET qr_code_url = $1
		WHERE id = $2
	`, url, generalID)
	if err != nil {
		return fmt.Errorf("failed to update qr_code_url for general %d: %w", generalID, err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		log.Printf("⚠️ No rows updated for general ID %d", generalID)
	} else {
		log.Printf("✅ Updated %d rows for general ID %d", rows, generalID)
	}
	return nil
}

func (s *Store) UpdateGeneralPDFfileUrls(generalID int, url string) error {
	log.Printf("🧾 Updating pdf_files for general %d with URL: %s", generalID, url)

	res, err := s.db.Exec(`
		UPDATE generals
		SET pdf_file = $1
		WHERE id = $2
	`, url, generalID)
	if err != nil {
		return fmt.Errorf("failed to update pdf_files for general %d: %w", generalID, err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		log.Printf("⚠️ No rows updated for general ID %d", generalID)
	} else {
		log.Printf("✅ Updated %d rows for general ID %d", rows, generalID)
	}
	return nil
}

// This for named tickets
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

func (s *Store) UpdatePDFfileUrls(guestID int, url string) error {
	log.Printf("🧾 Updating pdf_files for guest %d with URLs: %v", guestID, url)

	res, err := s.db.Exec(`
		UPDATE guests
		SET pdf_files = $1
		WHERE id = $2
	`, url, guestID)
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

func downloadPDF(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
