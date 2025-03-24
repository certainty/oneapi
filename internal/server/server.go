package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/certainty/oneapi/internal/spec"
	"github.com/certainty/oneapi/internal/storage"
	"github.com/gofiber/fiber/v2"
)

type Options struct {
	APIName         string
	Port            int
	PathPrefix      string
	HealthCheckPath string
	APIDocsPrefix   string
	APIDocsUIPath   string
}

func NewOptions() *Options {
	return &Options{
		APIName:         "OneAPI",
		Port:            9090,
		PathPrefix:      "/api",
		APIDocsUIPath:   "/api/docs",
		HealthCheckPath: "/_oneapi/health",
		APIDocsPrefix:   "/_oneapi/docs",
	}
}

func OptionsFromManifest(manifest spec.Manifest) (*Options, error) {
	opts := NewOptions()
	if manifest.Server != nil {
		if manifest.Server.Port != nil {
			opts.Port = *manifest.Server.Port
		}
		if manifest.Server.HealthCheck != nil {
			opts.HealthCheckPath = *manifest.Server.HealthCheck
		}
		if manifest.Server.APIDocsPrefix != nil {
			opts.APIDocsPrefix = *manifest.Server.APIDocsPrefix
		}
		if manifest.Server.APIDocsUIPath != nil {
			opts.APIDocsUIPath = *manifest.Server.APIDocsUIPath
		}
	}
	return opts, nil
}

type Server struct {
	options      Options
	jsonAPI      *JSONAPIHandler
	repositories map[string]storage.Repository
}

func NewServer(options Options, repositories map[string]storage.Repository) *Server {

	return &Server{
		options:      options,
		repositories: repositories,
	}
}

func (s *Server) Start() {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// Handle jsonapi errors
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(map[string]any{
				"errors": []map[string]any{
					{
						"status": fmt.Sprintf("%d", code),
						"title":  "Error",
						"detail": err.Error(),
					},
				},
			})
		},
	})

	// Register API routes based on manifest
	for entityName, repo := range s.repositories {
		handler := NewJSONAPIHandler(repo)
		entityGroup := app.Group(fmt.Sprintf("/%s", entityName))

		entityGroup.Get("/", handler.List)
		entityGroup.Get("/:id", handler.Get)
		entityGroup.Post("/", handler.Create)
		entityGroup.Patch("/:id", handler.Update)
		entityGroup.Delete("/:id", handler.Delete)
	}

	log.Printf("Starting mock API server on port %d", s.options.Port)
	log.Fatal(app.Listen(fmt.Sprintf(":%d", s.options.Port)))
}

func (s *Server) serveSwaggerUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
    <title>API Docs</title>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.20.1/swagger-ui.min.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.20.1/swagger-ui-bundle.min.js"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/swagger-ui/5.20.1/swagger-ui-standalone-preset.min.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: "` + s.options.APIDocsPrefix + `/swagger.json",
                dom_id: '#swagger-ui',
                presets: [SwaggerUIBundle.presets.apis, SwaggerUIStandalonePreset],
                layout: "StandaloneLayout"
            });
        };
    </script>
</body>
</html>`))
}

func (s *Server) serveOpenAPIDocs(w http.ResponseWriter, r *http.Request) {
	openAPI := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]string{
			"title":   s.options.APIName,
			"version": "1.0.0",
		},
	}
	json.NewEncoder(w).Encode(openAPI)
}
