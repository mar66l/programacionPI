package main

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

var repositorio *Repo

func HandlerIndex(c echo.Context) error {
	return c.File(filepath.Join("htmltmpl", "index.html"))
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// HandlerLogin valida credenciales con hash en el servidor y registra auditoría
func HandlerLogin(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Formato de credenciales inválido"})
	}

	var idUser, idRol int
	var nombre, hashGuardado, rolNombre string

	query := `SELECT u.id, u.nombre, u.password_hash, u.id_rol, r.nombre 
		FROM users u JOIN roles r ON u.id_rol = r.id WHERE u.email = ?`
	err := db.QueryRow(query, req.Email).Scan(&idUser, &nombre, &hashGuardado, &idRol, &rolNombre)

	if err != nil {
		repositorio.RegistrarAuditoria("LOGIN_FALLIDO", "Email inexistente: "+req.Email, c.RealIP())
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Credenciales incorrectas"})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashGuardado), []byte(req.Password)); err != nil {
		repositorio.RegistrarAuditoria("LOGIN_FALLIDO", "Password inválido: "+req.Email, c.RealIP())
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Credenciales incorrectas"})
	}

	repositorio.RegistrarAuditoria("LOGIN_EXITOSO", "Usuario: "+req.Email+" ("+rolNombre+")", c.RealIP())

	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":     idUser,
		"nombre": nombre,
		"email":  req.Email,
		"rol":    rolNombre,
	})
}

// HandlerIngestionTelemetria procesa los datos del ESP32 con validación técnica y de seguridad
func HandlerIngestionTelemetria(c echo.Context) error {
	token := c.Request().Header.Get("X-Device-Token")
	if token == "" {
		repositorio.RegistrarAuditoria("RECHAZO_TELEMETRIA", "Petición sin cabecera X-Device-Token", c.RealIP())
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Token de dispositivo ausente"})
	}

	var data TelemetriaIn
	if err := c.Bind(&data); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Payload JSON corrupto o incompleto"})
	}

	// 1. Verificación de identidad y pertenencia del dispositivo
	idPaciente, err := repositorio.ValidarTokenDispositivo(token, data.DeviceID)
	if err != nil {
		repositorio.RegistrarAuditoria("TOKEN_NO_AUTORIZADO", "Dispositivo: "+data.DeviceID, c.RealIP())
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Dispositivo no autorizado para este paciente"})
	}

	// 2. Validación de límites técnicos fisiológicos (Rechazo controlado)
	if data.FingerDetected {
		if data.SpO2 < 50.0 || data.SpO2 > 100.0 || data.BPM < 30.0 || data.BPM > 230.0 {
			repositorio.RegistrarAuditoria("VALOR_ANOMALO", "Lectura fuera de rango para "+data.DeviceID, c.RealIP())
			return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "Valores biomédicos fuera de rango fisiológico"})
		}
	}

	// 3. Persistencia de la telemetría
	if err := repositorio.GuardarTelemetria(data, idPaciente); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Error interno al persistir telemetría"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":      "ok",
		"id_paciente": idPaciente,
	})
}

// HandlerEstadoDispositivos expone el watchdog de desconexión para la pantalla en vivo
func HandlerEstadoDispositivos(c echo.Context) error {
	estados, err := repositorio.EstadoDispositivos()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, estados)
}

// HandlerAPIPacientes lista los pacientes para el selector del médico
func HandlerAPIPacientes(c echo.Context) error {
	pacientes, err := repositorio.ListarPacientes()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, pacientes)
}

// HandlerHistorialPaciente entrega los registros cronológicos del paciente solicitado
func HandlerHistorialPaciente(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "ID de paciente inválido"})
	}

	limiteStr := c.QueryParam("limit")
	limite := 30
	if limiteStr != "" {
		if l, err := strconv.Atoi(limiteStr); err == nil && l > 0 {
			limite = l
		}
	}

	lecturas, err := repositorio.ObtenerUltimasLecturas(id, limite)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, lecturas)
}

// HandlerAdminOverview entrega el conteo de usuarios, credenciales hasheadas, sensores y tiempos de desconexión
func HandlerAdminOverview(c echo.Context) error {
	usuarios, err := repositorio.ObtenerResumenAdmin()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"total_usuarios": len(usuarios),
		"usuarios":       usuarios,
	})
}