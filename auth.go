package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// Sesion vive solo en memoria del servidor: al reiniciarlo hay que volver a entrar.
type Sesion struct {
	UserID     int
	Nombre     string
	Rol        string
	IDPaciente int
	Exp        time.Time
}

var (
	sesiones = map[string]Sesion{}
	muSes    sync.Mutex
)

func hashToken(t string) string {
	s := sha256.Sum256([]byte(t))
	return hex.EncodeToString(s[:])
}

func nuevoToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// RequireAuth exige "Authorization: Bearer <token>" válido y no vencido.
func RequireAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		tok := strings.TrimPrefix(c.Request().Header.Get("Authorization"), "Bearer ")
		muSes.Lock()
		s, ok := sesiones[tok]
		muSes.Unlock()
		if tok == "" || !ok || time.Now().After(s.Exp) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Sesión inválida o vencida"})
		}
		c.Set("ses", s)
		c.Set("tok", tok)
		return next(c)
	}
}

func sesionDe(c echo.Context) Sesion { return c.Get("ses").(Sesion) }

func soloRoles(c echo.Context, roles ...string) bool {
	r := sesionDe(c).Rol
	for _, x := range roles {
		if x == r {
			return true
		}
	}
	return false
}

// puedeVerPaciente: el paciente solo ve su propio registro; médico y admin ven todos.
func puedeVerPaciente(s Sesion, id int) bool {
	if s.Rol == "medico" || s.Rol == "admin" {
		return true
	}
	return s.Rol == "paciente" && s.IDPaciente == id
}
