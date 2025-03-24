package server

import (
	"fmt"
	"github.com/certainty/oneapi/internal/storage"
	"github.com/gofiber/fiber/v2"
	"strconv"
	"strings"
)

type JSONAPIHandler struct {
	repo storage.Repository
}

func NewJSONAPIHandler(repo storage.Repository) *JSONAPIHandler {
	return &JSONAPIHandler{
		repo: repo,
	}
}

// List handles GET requests for listing entities
func (h *JSONAPIHandler) List(c *fiber.Ctx) error {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.Query("page[number]", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page[size]", "10"))

	// Get data from repository
	data, total, err := h.repo.List(page, pageSize)
	if err != nil {
		return err
	}

	// Format as JSONapi response
	result := map[string]interface{}{
		"data": formatJSONapiData(data),
		"meta": map[string]interface{}{
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": (total + pageSize - 1) / pageSize,
		},
	}

	return c.JSON(result)
}

func (h *JSONAPIHandler) Get(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ID format")
	}

	data, err := h.repo.FindByID(id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	return c.JSON(map[string]interface{}{
		"data": formatJSONapiResource(data),
	})
}

// Create handles POST requests to create entities
func (h *JSONAPIHandler) Create(c *fiber.Ctx) error {
	var request struct {
		Data struct {
			Attributes map[string]interface{} `json:"attributes"`
		} `json:"data"`
	}

	if err := c.BodyParser(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Extract attributes
	data := request.Data.Attributes

	// Create in repository
	id, err := h.repo.Create(data)
	if err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "validation failed") {
			return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	// Fetch the created entity
	entity, err := h.repo.FindByID(id)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(map[string]interface{}{
		"data": formatJSONapiResource(entity),
	})
}

// Update handles PATCH requests to update entities
func (h *JSONAPIHandler) Update(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ID format")
	}

	var request struct {
		Data struct {
			Attributes map[string]interface{} `json:"attributes"`
		} `json:"data"`
	}

	if err := c.BodyParser(&request); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid request body")
	}

	// Extract attributes
	data := request.Data.Attributes

	// Update in repository
	if err := h.repo.Update(id, data); err != nil {
		// Check if it's a validation error
		if strings.Contains(err.Error(), "validation failed") {
			return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
		}
		return err
	}

	// Fetch the updated entity
	entity, err := h.repo.FindByID(id)
	if err != nil {
		return err
	}

	return c.JSON(map[string]interface{}{
		"data": formatJSONapiResource(entity),
	})
}

// Delete handles DELETE requests to remove entities
func (h *JSONAPIHandler) Delete(c *fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid ID format")
	}

	if err := h.repo.Delete(id); err != nil {
		return err
	}

	return c.Status(fiber.StatusNoContent).Send(nil)
}

// Helper functions to format data in JSONapi format
func formatJSONapiData(data []map[string]interface{}) []map[string]interface{} {
	result := make([]map[string]interface{}, len(data))
	for i, item := range data {
		result[i] = formatJSONapiResource(item)
	}
	return result
}

func formatJSONapiResource(data map[string]interface{}) map[string]interface{} {
	id, _ := data["id"].(int64)
	idStr := fmt.Sprintf("%d", id)

	// Remove id from attributes
	attributes := make(map[string]interface{})
	for k, v := range data {
		if k != "id" {
			attributes[k] = v
		}
	}

	return map[string]interface{}{
		"id":         idStr,
		"type":       "entity", // This should be dynamically determined based on entity type
		"attributes": attributes,
	}
}
