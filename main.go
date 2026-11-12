package main

import (
	"bufio"
	"log"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// cargarEnv lee un archivo .env sencillo (CLAVE=valor) sin dependencias externas.
func cargarEnv(ruta string) {
	f, err := os.Open(ruta)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		k, v, ok := strings.Cut(l, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), "\"'")
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

func exigirEnv(k string) string {
	v := os.Getenv(k)
	if v == "" {
		log.Fatalf("Falta la variable %s. Copie .env.example como .env y complete los valores.", k)
	}
	return v
}

func main() {
	cargarEnv(".env")
	initDB()
	defer db.Close()
	repositorio = NuevoRepo(db)

	origen := os.Getenv("CORS_ORIGIN")
	if origen == "" {
		origen = "*"
	}
	puerto := os.Getenv("PORT")
	if puerto == "" {
		puerto = "8084"
	}

	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:  []string{origen},
		AllowHeaders:  []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-Device-Token"},
		ExposeHeaders: []string{"Date"},
	}))

	// Frontend (misma dirección y puerto que la API: no hay problemas de CORS)
	e.Static("/", "frontend")

	// Sesión
	e.POST("/api/auth/login", HandlerLogin)
	e.POST("/api/auth/logout", HandlerLogout, RequireAuth)

	// ESP32: credencial propia por dispositivo (no usa sesión de usuario)
	e.POST("/api/telemetry", HandlerIngestionTelemetria, middleware.BodyLimit("16K"))

	// Pantalla: todas exigen sesión y el servidor comprueba el permiso
	e.GET("/api/patients", HandlerListaPacientes, RequireAuth)
	e.GET("/api/patients/:id/latest", HandlerUltimo, RequireAuth)
	e.GET("/api/patients/:id/history", HandlerHistorial, RequireAuth)
	e.POST("/api/patients/:id/alert/ack", HandlerAck, RequireAuth)
	e.GET("/api/devices", HandlerDispositivos, RequireAuth)
	e.POST("/api/devices/:id/assign", HandlerAsignar, RequireAuth)
	e.GET("/api/audit", HandlerAuditoria, RequireAuth)

	e.Logger.Fatal(e.Start("0.0.0.0:" + puerto))
}
