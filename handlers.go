package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

var repositorio *Repo

var posturasValidas = map[string]bool{"": true, "boca_arriba": true, "boca_abajo": true, "izquierdo": true, "derecho": true}

func jerr(c echo.Context, code int, msg string) error {
	return c.JSON(code, map[string]string{"error": msg})
}

func corto(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// pacienteAutorizado valida el :id y el permiso EN EL SERVIDOR. Si devuelve false, ya respondió el error.
func pacienteAutorizado(c echo.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		_ = jerr(c, http.StatusBadRequest, "ID de paciente inválido")
		return 0, false
	}
	s := sesionDe(c)
	if !puedeVerPaciente(s, id) {
		repositorio.RegistrarAuditoria("ACCESO_DENEGADO", fmt.Sprintf("usuario %d pidió datos del paciente %d", s.UserID, id), c.RealIP())
		_ = jerr(c, http.StatusForbidden, "Sin permiso")
		return 0, false
	}
	return id, true
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func HandlerLogin(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return jerr(c, http.StatusBadRequest, "Formato de credenciales inválido")
	}
	var idUser, pid int
	var nombre, hashGuardado, rol, tipo string
	err := db.QueryRow(`SELECT u.id, u.nombre, u.password_hash, r.nombre, COALESCE(u.id_paciente, 0), COALESCE(p.tipo, '')
		FROM users u JOIN roles r ON u.id_rol = r.id
		LEFT JOIN pacientes p ON p.id_paciente = u.id_paciente WHERE u.email = ?`, req.Email).
		Scan(&idUser, &nombre, &hashGuardado, &rol, &pid, &tipo)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hashGuardado), []byte(req.Password)) != nil {
		repositorio.RegistrarAuditoria("LOGIN_FALLIDO", "Intento con correo: "+corto(req.Email, 60), c.RealIP())
		return jerr(c, http.StatusUnauthorized, "Credenciales incorrectas")
	}
	tok := nuevoToken()
	muSes.Lock()
	sesiones[tok] = Sesion{UserID: idUser, Nombre: nombre, Rol: rol, IDPaciente: pid, Exp: time.Now().Add(8 * time.Hour)}
	muSes.Unlock()
	repositorio.RegistrarAuditoria("LOGIN_EXITOSO", "Usuario "+strconv.Itoa(idUser)+" ("+rol+")", c.RealIP())
	return c.JSON(http.StatusOK, map[string]interface{}{
		"token": tok,
		"user":  map[string]interface{}{"id": idUser, "name": nombre, "role": rol, "tipo": tipo, "pid": pid},
	})
}

func HandlerLogout(c echo.Context) error {
	s := sesionDe(c)
	muSes.Lock()
	delete(sesiones, c.Get("tok").(string))
	muSes.Unlock()
	repositorio.RegistrarAuditoria("LOGOUT", "Usuario "+strconv.Itoa(s.UserID)+" ("+s.Rol+")", c.RealIP())
	return c.JSON(http.StatusOK, map[string]string{})
}

// HandlerIngestionTelemetria: datos del ESP32 con credencial propia y validación técnica.
func HandlerIngestionTelemetria(c echo.Context) error {
	token := c.Request().Header.Get("X-Device-Token")
	if token == "" {
		repositorio.RegistrarAuditoria("RECHAZO_TELEMETRIA", "Petición sin cabecera X-Device-Token", c.RealIP())
		return jerr(c, http.StatusUnauthorized, "Token de dispositivo ausente")
	}
	var data TelemetriaIn
	if err := c.Bind(&data); err != nil {
		return jerr(c, http.StatusBadRequest, "Payload JSON corrupto o incompleto")
	}
	idPaciente, tipo, err := repositorio.ValidarTokenDispositivo(token, data.DeviceID)
	if err != nil {
		repositorio.RegistrarAuditoria("TOKEN_NO_AUTORIZADO", "Dispositivo: "+corto(data.DeviceID, 40), c.RealIP())
		return jerr(c, http.StatusUnauthorized, "Dispositivo no autorizado")
	}
	if !posturasValidas[data.Posture] {
		repositorio.RegistrarAuditoria("VALOR_ANOMALO", "Postura inválida de "+data.DeviceID, c.RealIP())
		return jerr(c, http.StatusUnprocessableEntity, "Postura no válida")
	}
	if data.FingerDetected && (data.SpO2 < 50 || data.SpO2 > 100 || data.BPM < 30 || data.BPM > 230) {
		repositorio.RegistrarAuditoria("VALOR_ANOMALO", "Lectura fuera de rango de "+data.DeviceID, c.RealIP())
		return jerr(c, http.StatusUnprocessableEntity, "Valores fuera de rango fisiológico")
	}
	st, err := repositorio.GuardarTelemetria(data, idPaciente, tipo)
	if err != nil {
		return jerr(c, http.StatusInternalServerError, "Error interno al guardar telemetría")
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"status": "ok", "id_paciente": idPaciente, "st": st})
}

func HandlerListaPacientes(c echo.Context) error {
	if !soloRoles(c, "medico", "admin") {
		return jerr(c, http.StatusForbidden, "Sin permiso")
	}
	l, err := repositorio.ListarPacientes()
	if err != nil {
		return jerr(c, http.StatusInternalServerError, "Error interno")
	}
	return c.JSON(http.StatusOK, l)
}

func HandlerUltimo(c echo.Context) error {
	id, ok := pacienteAutorizado(c)
	if !ok {
		return nil
	}
	d, err := repositorio.Ultimo(id)
	if errors.Is(err, sql.ErrNoRows) {
		return jerr(c, http.StatusNotFound, "Sin dispositivo o sin datos todavía")
	}
	if err != nil {
		return jerr(c, http.StatusInternalServerError, "Error interno")
	}
	return c.JSON(http.StatusOK, d)
}

func HandlerHistorial(c echo.Context) error {
	id, ok := pacienteAutorizado(c)
	if !ok {
		return nil
	}
	h, err := repositorio.Historial(id)
	if err != nil {
		return jerr(c, http.StatusInternalServerError, "Error interno")
	}
	return c.JSON(http.StatusOK, h)
}

func HandlerAck(c echo.Context) error {
	id, ok := pacienteAutorizado(c)
	if !ok {
		return nil
	}
	if err := repositorio.AcusarAlerta(id); err != nil {
		return jerr(c, http.StatusInternalServerError, "Error interno")
	}
	return c.JSON(http.StatusOK, map[string]string{})
}

func HandlerDispositivos(c echo.Context) error {
	if !soloRoles(c, "admin") {
		return jerr(c, http.StatusForbidden, "Sin permiso")
	}
	l, err := repositorio.Dispositivos()
	if err != nil {
		return jerr(c, http.StatusInternalServerError, "Error interno")
	}
	return c.JSON(http.StatusOK, l)
}

func HandlerAsignar(c echo.Context) error {
	if !soloRoles(c, "admin") {
		return jerr(c, http.StatusForbidden, "Sin permiso")
	}
	var b struct {
		UserID int `json:"userId"`
	}
	if err := c.Bind(&b); err != nil || b.UserID <= 0 {
		return jerr(c, http.StatusUnprocessableEntity, "Datos no válidos")
	}
	dev := c.Param("id")
	err := repositorio.AsignarDispositivo(dev, b.UserID)
	if errors.Is(err, ErrConflicto) {
		return jerr(c, http.StatusConflict, "Ese paciente ya tiene otro dispositivo")
	}
	if errors.Is(err, ErrNoExiste) {
		return jerr(c, http.StatusNotFound, "Dispositivo o paciente no existe")
	}
	if err != nil {
		return jerr(c, http.StatusInternalServerError, "Error interno")
	}
	s := sesionDe(c)
	repositorio.RegistrarAuditoria("ASOCIACION_CAMBIADA", fmt.Sprintf("%s -> paciente %d (por usuario %d)", dev, b.UserID, s.UserID), c.RealIP())
	return c.JSON(http.StatusOK, map[string]string{})
}

func HandlerAuditoria(c echo.Context) error {
	if !soloRoles(c, "admin") {
		return jerr(c, http.StatusForbidden, "Sin permiso")
	}
	l, err := repositorio.ListarAuditoria()
	if err != nil {
		return jerr(c, http.StatusInternalServerError, "Error interno")
	}
	return c.JSON(http.StatusOK, l)
}
