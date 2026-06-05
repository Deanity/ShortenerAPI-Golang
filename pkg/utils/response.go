package utils

import (
	"github.com/gofiber/fiber/v2"
)

type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	ErrorCode string      `json:"error_code,omitempty"`
	Data      interface{} `json:"data"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

type PaginatedResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    interface{}     `json:"data"`
	Meta    PaginationMeta  `json:"meta"`
}

func Success(c *fiber.Ctx, statusCode int, message string, data interface{}) error {
	return c.Status(statusCode).JSON(APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(c *fiber.Ctx, statusCode int, message string, errorCode string) error {
	return c.Status(statusCode).JSON(APIResponse{
		Success: false,
		Message: message,
		ErrorCode: errorCode,
		Data:    nil,
	})
}

func Paginated(c *fiber.Ctx, statusCode int, message string, data interface{}, page, perPage int, total int64) error {
	totalPages := int((total + int64(perPage) - 1) / int64(perPage))
	return c.Status(statusCode).JSON(PaginatedResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta: PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}
