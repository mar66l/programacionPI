package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	snoozeMs = 5 * 60 * 1000      // tras "Entendido", no se vuelve a alertar durante 5 min
	diaMs    = 24 * 60 * 60 * 1000
)

var (
	ErrNoExiste  = errors.New("no existe")
	ErrConflicto = errors.New("conflicto de asociación")
)

// TelemetriaIn es el JSON que envía cada ESP32.
type TelemetriaIn struct {
	DeviceID       string  `json:"device_id"`
	SpO2           float64 `json:"spo2"`
	BPM            float64 `json:"bpm"`
	Posture        string  `json:"posture"`
	FingerDetected bool    `json:"finger_detected"`
	Simulated      bool    `json:"simulated"`
}

type PacienteLista struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Tipo     string `json:"tipo"`
	DeviceID string `json:"deviceId"`
	St       string `json:"st"`
	Ts       int64  `json:"ts"`
}

type Lectura struct {
	T      int64   `json:"t"`
	SpO2   float64 `json:"spo2"`
	BPM    float64 `json:"bpm"`
	Pos    string  `json:"pos"`
	Finger bool    `json:"finger"`
}

type Evento struct {
	T int64  `json:"t"`
	M string `json:"m"`
}

type Repo struct {
	db *sql.DB
}

func NuevoRepo(db *sql.DB) *Repo { return &Repo{db: db} }

func ahoraMs() int64 { return time.Now().UnixMilli() }

func etiquetaPostura(p string) string {
	switch p {
	case "boca_arriba":
		return "Boca arriba"
	case "boca_abajo":
		return "Boca abajo"
	case "izquierdo":
		return "De lado (izquierdo)"
	case "derecho":
		return "De lado (derecho)"
	}
	return ""
}

// ValidarTokenDispositivo: el dueño del dato sale de la tabla devices, nunca del cuerpo enviado.
func (r *Repo) ValidarTokenDispositivo(token, deviceID string) (int, string, error) {
	var id int
	var tipo string
	err := r.db.QueryRow(`SELECT d.id_paciente, p.tipo FROM devices d
		JOIN pacientes p ON p.id_paciente = d.id_paciente
		WHERE d.token_hash = ? AND d.id_dispositivo = ?`, hashToken(token), deviceID).Scan(&id, &tipo)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", errors.New("dispositivo o token no autorizado")
		}
		return 0, "", err
	}
	return id, tipo, nil
}

// GuardarTelemetria guarda la lectura, decide el estado (ok/alerta) y registra el evento al cambiar.
func (r *Repo) GuardarTelemetria(data TelemetriaIn, idPaciente int, tipo string) (string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	now := ahoraMs()
	var prev string
	var ackMs int64
	if err := tx.QueryRow(`SELECT st, ack_ms FROM devices WHERE id_dispositivo = ?`, data.DeviceID).Scan(&prev, &ackMs); err != nil {
		return "", err
	}

	// Reglas de alerta (REGLAS DE EJEMPLO: validarlas y documentarlas en el informe)
	alerta := false
	msg := ""
	if tipo == "dx" && data.Posture == "boca_arriba" {
		alerta = true
		msg = "ALERTA: boca arriba (giroscopio)"
	}
	if tipo == "pre" && data.FingerDetected && data.SpO2 < 90 {
		alerta = true
		msg = fmt.Sprintf("ALERTA: oxígeno bajo (SpO₂ %.0f %%)", data.SpO2)
	}
	if alerta && now-ackMs < snoozeMs {
		alerta = false
	}
	st := "ok"
	if alerta {
		st = "alerta"
	}

	// Sin dedo no hay lectura válida: se guarda 0 y la pantalla no muestra valores.
	finger := 0
	spo2, bpm := 0.0, 0.0
	if data.FingerDetected {
		finger = 1
		spo2, bpm = data.SpO2, data.BPM
	}
	fuente := "real"
	if data.Simulated {
		fuente = "simulado"
	}

	if _, err := tx.Exec(`INSERT INTO telemetria_apnea
		(id_dispositivo, id_paciente, spo2, bpm, postura, finger_detected, fuente, ts_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, data.DeviceID, idPaciente, spo2, bpm, data.Posture, finger, fuente, now); err != nil {
		return "", err
	}
	if _, err := tx.Exec(`UPDATE devices SET st = ?, ultima_ms = ?, estado = 'conectado' WHERE id_dispositivo = ?`, st, now, data.DeviceID); err != nil {
		return "", err
	}
	if alerta && prev != "alerta" {
		if _, err := tx.Exec(`INSERT INTO eventos (id_paciente, ts_ms, mensaje) VALUES (?, ?, ?)`, idPaciente, now, msg); err != nil {
			return "", err
		}
	}
	return st, tx.Commit()
}

func (r *Repo) ListarPacientes() ([]PacienteLista, error) {
	rows, err := r.db.Query(`SELECT p.id_paciente, p.nombre, p.tipo, COALESCE(d.id_dispositivo, ''), COALESCE(d.st, 'ok'), COALESCE(d.ultima_ms, 0)
		FROM pacientes p LEFT JOIN devices d ON d.id_paciente = p.id_paciente ORDER BY p.id_paciente`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lista := []PacienteLista{}
	for rows.Next() {
		var p PacienteLista
		if err := rows.Scan(&p.ID, &p.Name, &p.Tipo, &p.DeviceID, &p.St, &p.Ts); err != nil {
			return nil, err
		}
		lista = append(lista, p)
	}
	return lista, nil
}

// Ultimo devuelve la última lectura del paciente; sql.ErrNoRows si no hay dispositivo o datos.
func (r *Repo) Ultimo(id int) (map[string]interface{}, error) {
	var dev, st, tipo, postura, fuente string
	var ts int64
	var spo2, bpm float64
	var finger int
	err := r.db.QueryRow(`SELECT d.id_dispositivo, d.st, d.ultima_ms, p.tipo,
		COALESCE(t.spo2, 0), COALESCE(t.bpm, 0), COALESCE(t.postura, ''), COALESCE(t.finger_detected, 0), COALESCE(t.fuente, 'real')
		FROM devices d JOIN pacientes p ON p.id_paciente = d.id_paciente
		LEFT JOIN telemetria_apnea t ON t.id = (SELECT MAX(id) FROM telemetria_apnea WHERE id_paciente = d.id_paciente)
		WHERE d.id_paciente = ?`, id).Scan(&dev, &st, &ts, &tipo, &spo2, &bpm, &postura, &finger, &fuente)
	if err != nil {
		return nil, err
	}
	if ts == 0 {
		return nil, sql.ErrNoRows
	}
	return map[string]interface{}{
		"userId": id, "deviceId": dev, "tipo": tipo, "st": st,
		"spo2": spo2, "bpm": bpm, "pos": etiquetaPostura(postura), "finger": finger == 1,
		"ts": ts, "source": fuente,
	}, nil
}

// Historial: últimas 30 lecturas (orden cronológico), últimos 10 eventos y alertas por noche (7 días).
func (r *Repo) Historial(id int) (map[string]interface{}, error) {
	rows, err := r.db.Query(`SELECT ts_ms, spo2, bpm, postura, finger_detected FROM telemetria_apnea
		WHERE id_paciente = ? ORDER BY id DESC LIMIT 30`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	lect := []Lectura{}
	for rows.Next() {
		var l Lectura
		var p string
		var f int
		if err := rows.Scan(&l.T, &l.SpO2, &l.BPM, &p, &f); err != nil {
			return nil, err
		}
		l.Pos = etiquetaPostura(p)
		l.Finger = f == 1
		lect = append([]Lectura{l}, lect...)
	}

	rows2, err := r.db.Query(`SELECT ts_ms, mensaje FROM eventos WHERE id_paciente = ? ORDER BY id DESC LIMIT 10`, id)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	ev := []Evento{}
	for rows2.Next() {
		var e Evento
		if err := rows2.Scan(&e.T, &e.M); err != nil {
			return nil, err
		}
		ev = append(ev, e)
	}

	now := ahoraMs()
	noches := make([]int, 7)
	rows3, err := r.db.Query(`SELECT ts_ms FROM eventos WHERE id_paciente = ? AND mensaje LIKE 'ALERTA%' AND ts_ms > ?`, id, now-7*diaMs)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()
	for rows3.Next() {
		var t int64
		if err := rows3.Scan(&t); err != nil {
			return nil, err
		}
		i := 6 - int((now-t)/diaMs)
		if i >= 0 && i < 7 {
			noches[i]++
		}
	}
	return map[string]interface{}{"nights": noches, "events": ev, "readings": lect}, nil
}

// AcusarAlerta: el paciente confirma; se silencia 5 min aunque la condición continúe.
func (r *Repo) AcusarAlerta(idPaciente int) error {
	now := ahoraMs()
	if _, err := r.db.Exec(`UPDATE devices SET st = 'ok', ack_ms = ? WHERE id_paciente = ?`, now, idPaciente); err != nil {
		return err
	}
	_, err := r.db.Exec(`INSERT INTO eventos (id_paciente, ts_ms, mensaje) VALUES (?, ?, 'Alerta confirmada por el paciente')`, idPaciente, now)
	return err
}

func (r *Repo) Dispositivos() ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`SELECT d.id_dispositivo, d.id_paciente, d.ultima_ms, d.sensores
		FROM devices d ORDER BY d.id_dispositivo`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]interface{}{}
	for rows.Next() {
		var id, sensor string
		var pid int
		var ts int64
		if err := rows.Scan(&id, &pid, &ts, &sensor); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{"id": id, "sensor": sensor, "user": pid, "ts": ts})
	}
	return out, nil
}

func (r *Repo) AsignarDispositivo(devID string, pid int) error {
	var n int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM pacientes WHERE id_paciente = ?`, pid).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrNoExiste
	}
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM devices WHERE id_paciente = ? AND id_dispositivo <> ?`, pid, devID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return ErrConflicto
	}
	res, err := r.db.Exec(`UPDATE devices SET id_paciente = ? WHERE id_dispositivo = ?`, pid, devID)
	if err != nil {
		return err
	}
	if c, _ := res.RowsAffected(); c == 0 {
		return ErrNoExiste
	}
	return nil
}

// RegistrarAuditoria: bitácora sin contraseñas ni tokens.
func (r *Repo) RegistrarAuditoria(evento, detalles, ip string) {
	r.db.Exec(`INSERT INTO audit_logs (evento, detalles, ip_origen, ts_ms) VALUES (?, ?, ?, ?)`, evento, detalles, ip, ahoraMs())
}

func (r *Repo) ListarAuditoria() ([]Evento, error) {
	rows, err := r.db.Query(`SELECT ts_ms, evento, COALESCE(detalles, '') FROM audit_logs ORDER BY id DESC LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Evento{}
	for rows.Next() {
		var t int64
		var ev, det string
		if err := rows.Scan(&t, &ev, &det); err != nil {
			return nil, err
		}
		m := ev
		if det != "" {
			m = ev + ": " + det
		}
		out = append(out, Evento{T: t, M: m})
	}
	return out, nil
}
