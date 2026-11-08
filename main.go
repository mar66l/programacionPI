package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// 1. Inicializar base de datos SQLite y semillas
	initDB()
	defer db.Close()

	// 2. Asignar el repositorio
	repositorio = NuevoRepo(db)

	e := echo.New()

	// Middlewares
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, "X-Device-Token"},
	}))

	// Servir archivos estáticos del frontend
	e.Static("/assets", "./assets")

	// Vistas HTML
	e.GET("/", HandlerIndex)
	e.GET("/index.html", HandlerIndex)

	// Endpoints de la API
	e.POST("/api/auth/login", HandlerLogin)
	e.GET("/api/pacientes", HandlerAPIPacientes)
	e.POST("/api/telemetry", HandlerIngestionTelemetria)
	e.GET("/api/devices/status", HandlerEstadoDispositivos)
	e.GET("/api/patients/:id/history", HandlerHistorialPaciente)

	// Iniciar servidor local
	e.Logger.Fatal(e.Start("0.0.0.0:8084"))
}
