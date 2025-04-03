package controller

import (
	"github.com/AntoniadisCorp/deploy4scrap/domain"
	"github.com/gofiber/fiber/v2"
)

// This function wraps a controller and formats its response for Fiber
func MakeFiberCallback(controller func(*fiber.Ctx) (*domain.HTTPResponse, error)) fiber.Handler {

	return func(c *fiber.Ctx) error {
		// Call the controller function

		// logger.NewLogger("Info").Info("Headers: ", c.GetRespHeaders())
		// logger.NewLogger("Info").Info("Headers: ", c.GetReqHeaders())

		httpResponse, err := controller(c)

		// Set content type to JSON
		c.Type("json")

		// Handle some exclusive errors like 'FAILED TO GET THE SESSION'
		// ExclusiveErrorsHandler(c, httpResponse, err)

		if err != nil {
			// Handle error
			httpResponse.Errors.Detail = "An unknown error occurred on controller "
			return c.Status(httpResponse.Code).JSON(httpResponse)
		}

		// Set headers from the response

		/* // set cache time header to 600 as default
		c.Response().Header.Add("Cache-Time", "600") */
		// Send the standardized response
		return c.Status(httpResponse.Code).JSON(httpResponse)
	}
}

func MakeServerCallback(controller func(*fiber.Ctx) (*domain.HTTPResponse, error)) fiber.Handler {

	// tracer := otel.GetTracerProvider().Tracer("auth.cookto.online/")

	return func(c *fiber.Ctx) error {
		// Call the controller function
		httpResponse, err := controller(c)

		// httpResponse.DomainHost =   cfg.AllowedOrigins[0]
		httpResponse.SubDomainHost = string(c.Context().Host())

		// Set content type to HTML
		c.Type("html")

		if err != nil {

			// Handle error
			return c.Status(fiber.StatusInternalServerError).
				Render(httpResponse.View, fiber.Map{
					"countdownSecs": 5,
					"message":       "An unknown error",
					"errorMessage":  c.Query("error", "Unknown error"),
					"errorDesc":     c.Query("error_description", "An unknown error occurred on controller"),
					// "Code":          traceID,
					"SubDomainHost": httpResponse.SubDomainHost,
				})
		}

		// Set headers from the response
		/* logger.NewLogger("Info").Info("Headers: ", c.GetRespHeaders())
		logger.NewLogger("Info").Info("Headers: ", c.GetReqHeaders()) */

		// Render the view with the response data
		return c.Status(httpResponse.Code).Render(httpResponse.View, httpResponse)
	}
}
