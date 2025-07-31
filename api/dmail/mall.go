package mall

import (
	"database/sql"
	"log"
	"strconv" // Added for integer to string conversion
	"strings"
	"time"

	t "github.com/axuman/go-server/biz"
	G "github.com/axuman/go-server/globals"
	m "github.com/axuman/go-server/models"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New()

func BuildRoutes(router fiber.Router) {
	mallGroup := router.Group("/mall")
	mallGroup.Get("/q", qMall)
	mallGroup.Post("/c", cMall)
	mallGroup.Put("/u", uMall)
	mallGroup.Post("/bc", bcMall)
	mallGroup.Delete("/bd", bdMall)
}

func qMall(c *fiber.Ctx) error {
	payload := new(t.PaginatorWith[m.Mall])
	if err := c.QueryParser(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse query parameters: " + err.Error(),
		})
	}
	payload.SetDefaults()

	var queryBuilder strings.Builder
	var args []interface{}

	queryBuilder.WriteString("SELECT id, name, location, created_at, updated_at FROM malls WHERE deleted_at IS NULL")
	if payload.D.Name != nil {
		queryBuilder.WriteString(" AND name = ?")
		args = append(args, payload.D.Name)
	}
	if payload.D.Location != nil {
		queryBuilder.WriteString(" AND location = ?")
		args = append(args, payload.D.Location)
	}
	if payload.ID != nil {
		queryBuilder.WriteString(" AND id > ? ORDER BY id ASC LIMIT ?")
		args = append(args, *payload.ID)
		args = append(args, payload.PS)
	} else {
		queryBuilder.WriteString(" LIMIT ? OFFSET ?")
		args = append(args, payload.PS)
		args = append(args, payload.PN*payload.PS)
	}

	finalQuery := queryBuilder.String()

	rows, err := G.DmailDB.QueryContext(c.Context(), finalQuery, args...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not query malls: " + err.Error(),
		})
	}
	defer rows.Close()

	malls := []t.Table[m.Mall]{}
	for rows.Next() {
		var mall t.Table[m.Mall]
		var createdAt, updatedAt time.Time

		if err := rows.Scan(&mall.ID, &mall.D.Name, &mall.D.Location, &createdAt, &updatedAt); err != nil {
			log.Printf("Error scanning mall row: %v", err)
			continue
		}
		mall.CreatedAt = createdAt
		mall.UpdatedAt = sql.NullTime{Time: updatedAt, Valid: true}
		malls = append(malls, mall)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating mall rows: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error processing mall query results: " + err.Error(),
		})
	}

	return c.JSON(malls)
}

func cMall(c *fiber.Ctx) error {
	payload := new(m.Mall)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	if err := validate.Struct(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Validation failed: " + err.Error(),
		})
	}

	query := `INSERT INTO malls (name, location) VALUES (?, ?) RETURNING id, name, location, created_at, updated_at;`
	var mall t.Table[m.Mall]
	var createdAt, updatedAt time.Time
	err := G.DmailDB.QueryRowContext(c.Context(), query, payload.Name, payload.Location).Scan(
		&mall.ID,
		&mall.D.Name,
		&mall.D.Location,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		log.Printf("Error creating mall: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create mall: " + err.Error(),
		})
	}
	mall.CreatedAt = createdAt
	mall.UpdatedAt = sql.NullTime{Time: updatedAt, Valid: true}

	return c.Status(fiber.StatusCreated).JSON(mall)
}

func uMall(c *fiber.Ctx) error {
	payload := new(t.Table[m.Mall]) // Expecting ID and Data for update
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	if payload.ID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Mall ID is required for update",
		})
	}

	if err := validate.Struct(payload.D); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Validation failed: " + err.Error(),
		})
	}

	query := `UPDATE malls SET name = ?, location = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND deleted_at IS NULL RETURNING id, name, location, created_at, updated_at;`
	var mall t.Table[m.Mall]
	var createdAt, updatedAt time.Time
	err := G.DmailDB.QueryRowContext(c.Context(), query, payload.D.Name, payload.D.Location, *payload.ID).Scan(
		&mall.ID,
		&mall.D.Name,
		&mall.D.Location,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Mall not found or already deleted",
			})
		}
		log.Printf("Error updating mall: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update mall: " + err.Error(),
		})
	}
	mall.CreatedAt = createdAt
	mall.UpdatedAt = sql.NullTime{Time: updatedAt, Valid: true}

	return c.JSON(mall)
}

func bcMall(c *fiber.Ctx) error {
	var payloads []m.Mall
	if err := c.BodyParser(&payloads); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON array: " + err.Error(),
		})
	}

	if len(payloads) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No malls provided for batch creation",
		})
	}

	// Build query for batch insert
	var queryBuilder strings.Builder
	queryBuilder.WriteString("INSERT INTO malls (name, location) VALUES ")
	var args []interface{}
	for i, p := range payloads {
		if err := validate.Struct(p); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Validation failed for item " + strconv.Itoa(i) + ": " + err.Error(),
			})
		}
		queryBuilder.WriteString("(?, ?)")
		args = append(args, p.Name, p.Location)
		if i < len(payloads)-1 {
			queryBuilder.WriteString(", ")
		}
	}
	queryBuilder.WriteString(" RETURNING id, name, location, created_at, updated_at;")

	rows, err := G.DmailDB.QueryContext(c.Context(), queryBuilder.String(), args...)
	if err != nil {
		log.Printf("Error batch creating malls: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not batch create malls: " + err.Error(),
		})
	}
	defer rows.Close()

	createdMalls := []t.Table[m.Mall]{}
	for rows.Next() {
		var mall t.Table[m.Mall]
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&mall.ID, &mall.D.Name, &mall.D.Location, &createdAt, &updatedAt); err != nil {
			log.Printf("Error scanning created mall row: %v", err)
			continue
		}
		mall.CreatedAt = createdAt
		mall.UpdatedAt = sql.NullTime{Time: updatedAt, Valid: true}
		createdMalls = append(createdMalls, mall)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating batch created mall rows: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error processing batch create results: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(createdMalls)
}

func bdMall(c *fiber.Ctx) error {
	var payload struct {
		IDs []int64 `json:"ids"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	if len(payload.IDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No IDs provided for deletion",
		})
	}

	placeholders := strings.Repeat("?,", len(payload.IDs))
	placeholders = placeholders[:len(placeholders)-1]

	query := "UPDATE malls SET deleted_at = CURRENT_TIMESTAMP WHERE id IN (" + placeholders + ")"

	args := make([]interface{}, len(payload.IDs))
	for i, id := range payload.IDs {
		args[i] = id
	}

	result, err := G.DmailDB.ExecContext(c.Context(), query, args...)
	if err != nil {
		log.Printf("Error batch deleting malls: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not delete malls: " + err.Error(),
		})
	}

	affected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting affected rows: %v", err)
	}

	return c.JSON(fiber.Map{
		"deleted": affected,
	})
}
