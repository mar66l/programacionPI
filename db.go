package main

import (
	"database/sql"
	"log"

	_ "github.com/glebarez/go-sqlite"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

// initDB crea la base SQLite (archivo nuevo sueno_seguro.db) y precarga cuentas de prueba.
func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./sueno_seguro.db")
	if err != nil {
		log.Fatal("Error al abrir base de datos SQLite: ", err)
	}

	crearTablas := `
	CREATE TABLE IF NOT EXISTS roles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nombre TEXT UNIQUE NOT NULL
	);
	CREATE TABLE IF NOT EXISTS pacientes (
		id_paciente INTEGER PRIMARY KEY AUTOINCREMENT,
		nombre TEXT NOT NULL,
		edad INTEGER NOT NULL,
		diagnostico TEXT NOT NULL,
		tipo TEXT NOT NULL CHECK (tipo IN ('pre','dx'))
	);
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		nombre TEXT NOT NULL,
		id_rol INTEGER NOT NULL,
		id_paciente INTEGER,
		FOREIGN KEY (id_rol) REFERENCES roles(id),
		FOREIGN KEY (id_paciente) REFERENCES pacientes(id_paciente)
	);
	CREATE TABLE IF NOT EXISTS devices (
		id_dispositivo TEXT PRIMARY KEY,
		token_hash TEXT UNIQUE NOT NULL,
		id_paciente INTEGER UNIQUE NOT NULL,
		sensores TEXT NOT NULL DEFAULT '',
		estado TEXT NOT NULL DEFAULT 'desconectado',
		st TEXT NOT NULL DEFAULT 'ok',
		ultima_ms INTEGER NOT NULL DEFAULT 0,
		ack_ms INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY (id_paciente) REFERENCES pacientes(id_paciente)
	);
	CREATE TABLE IF NOT EXISTS telemetria_apnea (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		id_dispositivo TEXT NOT NULL,
		id_paciente INTEGER NOT NULL,
		spo2 REAL NOT NULL,
		bpm REAL NOT NULL,
		postura TEXT NOT NULL,
		finger_detected INTEGER NOT NULL,
		fuente TEXT NOT NULL DEFAULT 'real',
		ts_ms INTEGER NOT NULL,
		FOREIGN KEY (id_dispositivo) REFERENCES devices(id_dispositivo),
		FOREIGN KEY (id_paciente) REFERENCES pacientes(id_paciente)
	);
	CREATE TABLE IF NOT EXISTS eventos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		id_paciente INTEGER NOT NULL,
		ts_ms INTEGER NOT NULL,
		mensaje TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		evento TEXT NOT NULL,
		detalles TEXT,
		ip_origen TEXT,
		ts_ms INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS ix_tel_pac ON telemetria_apnea(id_paciente, id);`

	if _, err = db.Exec(crearTablas); err != nil {
		log.Fatal("Error al inicializar tablas: ", err)
	}
	seedDatabase()
}

// seedDatabase: contraseña y tokens salen del .env (nunca del código). Los tokens se guardan como hash.
func seedDatabase() {
	pwd := exigirEnv("SEED_PASSWORD")
	tokA := exigirEnv("DEVICE_TOKEN_A")
	tokB := exigirEnv("DEVICE_TOKEN_B")

	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("No se pudo generar el hash: ", err)
	}
	ph := string(h)

	db.Exec("INSERT OR IGNORE INTO roles (id, nombre) VALUES (1,'medico'),(2,'paciente'),(3,'admin');")

	db.Exec(`INSERT OR IGNORE INTO pacientes (id_paciente, nombre, edad, diagnostico, tipo) VALUES
		(1, 'Don Roberto Gómez', 78, 'Pre-diagnóstico: sospecha de AOS', 'pre'),
		(2, 'Doña Elena Ramos', 81, 'AOS severa posicional (diagnosticado)', 'dx');`)

	db.Exec(`INSERT OR IGNORE INTO users (id, email, password_hash, nombre, id_rol, id_paciente) VALUES
		(1, 'doctor@redmedica.com', ?, 'Dra. Méndez', 1, NULL),
		(2, 'pacienteA@redmedica.com', ?, 'Don Roberto Gómez', 2, 1),
		(3, 'pacienteB@redmedica.com', ?, 'Doña Elena Ramos', 2, 2),
		(4, 'admin@redmedica.com', ?, 'Administración', 3, NULL);`, ph, ph, ph, ph)

	db.Exec(`INSERT OR IGNORE INTO devices (id_dispositivo, token_hash, id_paciente) VALUES ('ESP32_A', ?, 1), ('ESP32_B', ?, 2);`,
		hashToken(tokA), hashToken(tokB))
	// El .env es la fuente de verdad: si cambia un token o la contraseña de prueba, se actualiza aquí.
	db.Exec(`UPDATE users SET password_hash = ? WHERE id IN (1, 2, 3, 4)`, ph)
	db.Exec(`UPDATE devices SET token_hash = ? WHERE id_dispositivo = 'ESP32_A'`, hashToken(tokA))
	db.Exec(`UPDATE devices SET token_hash = ? WHERE id_dispositivo = 'ESP32_B'`, hashToken(tokB))
	db.Exec(`UPDATE devices SET sensores = 'MAX30102 (Oxímetro) + AD8232 (ECG)' WHERE id_dispositivo = 'ESP32_A'`)
	db.Exec(`UPDATE devices SET sensores = 'MAX30102 (Oxímetro) + MPU6050 (Giroscopio I2C)' WHERE id_dispositivo = 'ESP32_B'`)
}
