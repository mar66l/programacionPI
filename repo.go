package main

import (
	"database/sql"
	"errors"
)

// TelemetriaIn representa el payload JSON enviado desde los ESP32
type TelemetriaIn struct {
	DeviceID       string  `json:"device_id"`
	SpO2           float64 `json:"spo2"`
	BPM            float64 `json:"bpm"`
	Posture        string  `json:"posture"`
	FingerDetected bool    `json:"finger_detected"`
}

// RegistroTelemetria representa el dato persistido para el historial del paciente
type RegistroTelemetria struct {
	ID             int     `json:"id"`
	DeviceID       string  `json:"device_id"`
	IDPaciente     int     `json:"id_paciente"`
	SpO2           float64 `json:"spo2"`
	BPM            float64 `json:"bpm"`
	Posture        string  `json:"posture"`
	FingerDetected bool    `json:"finger_detected"`
	FechaHora      string  `json:"fecha_hora"`
}

// DeviceStatus expone el estado de presencia de cada microcontrolador
type DeviceStatus struct {
	IDDispositivo  string `json:"id_dispositivo"`
	IDPaciente     int    `json:"id_paciente"`
	NombrePaciente string `json:"nombre_paciente"`
	Estado         string `json:"estado"`
	UltimaConexion string `json:"ultima_conexion"`
	Activo         bool   `json:"activo"`
}

// PacienteDTO información básica para el dashboard
type PacienteDTO struct {
	ID          int    `json:"id_paciente"`
	Nombre      string `json:"nombre"`
	Edad        int    `json:"edad"`
	Diagnostico string `json:"diagnostico"`
}

type Repo struct {
	db *sql.DB
}

func NuevoRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// ValidarTokenDispositivo comprueba que la API Key corresponda al dispositivo
func (r *Repo) ValidarTokenDispositivo(token string, deviceID string) (int, error) {
	var idPaciente int
	query := `SELECT id_paciente FROM devices WHERE api_token = ? AND id_dispositivo = ?`
	err := r.db.QueryRow(query, token, deviceID).Scan(&idPaciente)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.New("dispositivo o token no autorizado")
		}
		return 0, err
	}
	return idPaciente, nil
}

// GuardarTelemetria registra la medición y actualiza la marca temporal de conexión
func (r *Repo) GuardarTelemetria(data TelemetriaIn, idPaciente int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	insertQ := `INSERT INTO telemetria_apnea 
		(id_dispositivo, id_paciente, spo2, bpm, postura, finger_detected, fecha_hora) 
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	fingerInt := 0
	if data.FingerDetected {
		fingerInt = 1
	}

	_, err = tx.Exec(insertQ, data.DeviceID, idPaciente, data.SpO2, data.BPM, data.Posture, fingerInt)
	if err != nil {
		return err
	}

	updateDev := `UPDATE devices SET estado = 'conectado', ultima_conexion = CURRENT_TIMESTAMP WHERE id_dispositivo = ?`
	_, err = tx.Exec(updateDev, data.DeviceID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// ListarPacientes obtiene el catálogo de pacientes para el panel médico
func (r *Repo) ListarPacientes() ([]PacienteDTO, error) {
	rows, err := r.db.Query("SELECT id_paciente, nombre, edad, diagnostico FROM pacientes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []PacienteDTO
	for rows.Next() {
		var p PacienteDTO
		if err := rows.Scan(&p.ID, &p.Nombre, &p.Edad, &p.Diagnostico); err != nil {
			return nil, err
		}
		lista = append(lista, p)
	}
	return lista, nil
}

// ObtenerUltimasLecturas consulta el historial de un paciente específico respetando aislamiento de datos
func (r *Repo) ObtenerUltimasLecturas(idPaciente int, limit int) ([]RegistroTelemetria, error) {
	query := `SELECT id, id_dispositivo, id_paciente, spo2, bpm, postura, finger_detected, fecha_hora 
		FROM telemetria_apnea 
		WHERE id_paciente = ? 
		ORDER BY id DESC LIMIT ?`

	rows, err := r.db.Query(query, idPaciente, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lecturas []RegistroTelemetria
	for rows.Next() {
		var reg RegistroTelemetria
		var fingerInt int
		if err := rows.Scan(&reg.ID, &reg.DeviceID, &reg.IDPaciente, &reg.SpO2, &reg.BPM, &reg.Posture, &fingerInt, &reg.FechaHora); err != nil {
			return nil, err
		}
		reg.FingerDetected = (fingerInt == 1)
		lecturas = append(lecturas, reg)
	}
	return lecturas, nil
}

// EstadoDispositivos evalúa la última conexión para determinar si la placa sigue viva o se desconectó
func (r *Repo) EstadoDispositivos() ([]DeviceStatus, error) {
	query := `SELECT d.id_dispositivo, d.id_paciente, p.nombre, 
		COALESCE(d.ultima_conexion, ''),
		CASE 
			WHEN d.ultima_conexion IS NOT NULL AND (strftime('%s', 'now') - strftime('%s', d.ultima_conexion)) <= 6 THEN 'conectado'
			ELSE 'desconectado'
		END AS estado_actual
		FROM devices d
		JOIN pacientes p ON d.id_paciente = p.id_paciente`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var estados []DeviceStatus
	for rows.Next() {
		var st DeviceStatus
		if err := rows.Scan(&st.IDDispositivo, &st.IDPaciente, &st.NombrePaciente, &st.UltimaConexion, &st.Estado); err != nil {
			return nil, err
		}
		st.Activo = (st.Estado == "conectado")
		estados = append(estados, st)
	}
	return estados, nil
}

// RegistrarAuditoria guarda bitácora de eventos relevantes sin exponer contraseñas ni tokens
func (r *Repo) RegistrarAuditoria(evento, detalles, ip string) {
	r.db.Exec(`INSERT INTO audit_logs (evento, detalles, ip_origen) VALUES (?, ?, ?)`, evento, detalles, ip)
}
