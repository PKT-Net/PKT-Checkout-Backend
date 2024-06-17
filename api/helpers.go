package api

import "github.com/gofiber/fiber/v2"

func craftApiError(code string, message string) ApiError {
	return ApiError{
		Code:    code,
		Message: message,
	}
}

func (s *Server) preflightPublicView(c *fiber.Ctx) error {
	c.Response().Header.Add("Access-Control-Allow-Origin", s.CorsOrigin)
	c.Response().Header.Add("Access-Control-Allow-Headers", "X-VIEW-KEY")
	return nil
}
