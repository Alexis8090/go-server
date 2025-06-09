package user

import (
	"log"
	"strings"
	"time"

	t "github.com/axuman/go-server/biz"     // Adjust import path
	G "github.com/axuman/go-server/globals" // Adjust import path
	m "github.com/axuman/go-server/models"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

var validate = validator.New()


func BuildRoutes(router fiber.Router) {
	userGroup := router.Group("/user")
	userGroup.Get("/q", q)
	userGroup.Post("/c", c)
}

func q(c *fiber.Ctx) error {
	payload := new(t.PaginatorWith[m.User])
	if err := c.QueryParser(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse query parameters: " + err.Error(),
		})
	}
	payload.SetDefaults() // Apply default pagination values

	// Business logic: if age is 100, return error
	// if payload.Age != nil && *payload.Age == 100 {
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"error": "年龄为 100 不符合业务要求", // "Age 100 is not allowed by business requirements"
	// 	})
	// }

	var queryBuilder strings.Builder
	var args []interface{}

	queryBuilder.WriteString("SELECT id, name, age, created_at FROM users WHERE deleted_at IS NULL")
	if payload.D.Age != nil {
		queryBuilder.WriteString(" AND age = ?")
		args = append(args, &payload.D.Age)
	}
	if payload.D.Name != nil {
		queryBuilder.WriteString(" AND name = ?")
		args = append(args, &payload.D.Name)
	}
	if payload.ID != nil {
		queryBuilder.WriteString(" AND id > ? LIMIT ?")
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
			"error": "Could not query users: " + err.Error(),
		})
	}
	defer rows.Close()

	users := []t.Table[m.User]{}
	for rows.Next() {
		var user t.Table[m.User]
		var createdAt time.Time // 直接使用 time.Time 接收 DATETIME

		if err := rows.Scan(&user.ID, &user.D.Name, &user.D.Age, &createdAt); err != nil {
			log.Printf("Error scanning user row: %v", err)
			continue
		}
		user.CreatedAt = createdAt // 直接赋值，无需手动解析
		users = append(users, user)
	}


	if err = rows.Err(); err != nil {
		log.Printf("Error after iterating user rows: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error processing user query results: " + err.Error(),
		})
	}

	return c.JSON(users)
}





func c(c *fiber.Ctx) error {
	payload := new(m.User)
	if err := c.BodyParser(payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON: " + err.Error(),
		})
	}

	// if err := validate.Struct(payload); err != nil {
	// 	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
	// 		"error": "Validation failed: " + err.Error(),
	// 	})
	// }

	query := `INSERT INTO users (name, age) VALUES (?, ?) RETURNING id, name, age, created_at;`
	var user t.Table[m.User]
	err := G.DmailDB.QueryRowContext(c.Context(), query, payload.Name, payload.Age).Scan(
		&user.ID,
		&user.D.Name,
		&user.D.Age,
		&user.CreatedAt,
	)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create user: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}


