package main

import (
	"database/sql"
	"log"

	_ "github.com/glebarez/go-sqlite"
	"golang.org/x/crypto/bcrypt"
)

var db *sql.DB

// initDB inicializa la base de datos relacional SQLite y precarga datos de prueba
func initDB() {
	var err error
	db, err = sql.Open("sqlite", "./apnea_monitoring.db")
	if err != nil {
		log.Fatal("Error al abrir base de datos SQLite: ", err)
	}

	crearTablas := `
	CREATE TABLE IF NOT EXISTS roles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nombre TEXT UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		nombre TEXT NOT NULL,
		id_rol INTEGER NOT NULL,
		FOREIGN KEY (id_rol) REFERENCES roles(id)
	);

	CREATE TABLE IF NOT EXISTS pacientes (
		id_paciente INTEGER PRIMARY KEY AUTOINCREMENT,
		nombre TEXT NOT NULL,
		edad INTEGER NOT NULL,
		diagnostico TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS devices (
		id_dispositivo TEXT PRIMARY KEY,
		api_token TEXT UNIQUE NOT NULL,
		id_paciente INTEGER UNIQUE NOT NULL,
		estado TEXT DEFAULT 'desconectado',
		ultima_conexion DATETIME,
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
		fecha_hora DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (id_dispositivo) REFERENCES devices(id_dispositivo),
		FOREIGN KEY (id_paciente) REFERENCES pacientes(id_paciente)
	);

	CREATE TABLE IF NOT EXISTS audit_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		evento TEXT NOT NULL,
		detalles TEXT,
		ip_origen TEXT,
		fecha_hora DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err = db.Exec(crearTablas); err != nil {
		log.Fatal("Error al inicializar tablas: ", err)
	}

	seedDatabase()
}

// seedDatabase precarga los roles, usuarios cifrados y la relación unívoca Paciente-Dispositivo
func seedDatabase() {
	db.Exec("INSERT OR IGNORE INTO roles (id, nombre) VALUES (1, 'medico'), (2, 'paciente');")

	hashPassword := func(pwd string) string {
		h, _ := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
		return string(h)
	}

	// 1. Usuarios con contraseñas hasheadas (nunca texto plano)
	db.Exec(`INSERT OR IGNORE INTO users (id, email, password_hash, nombre, id_rol) 
		VALUES (1, 'doctor@redmedica.com', ?, 'Dra. Méndez', 1);`, hashPassword("medico2026"))

	db.Exec(`INSERT OR IGNORE INTO users (id, email, password_hash, nombre, id_rol) 
		VALUES (2, 'pacienteA@redmedica.com', ?, 'Don Roberto Gómez', 2);`, hashPassword("pacienteA123"))

	db.Exec(`INSERT OR IGNORE INTO users (id, email, password_hash, nombre, id_rol) 
		VALUES (3, 'pacienteB@redmedica.com', ?, 'Doña Elena Ramos', 2);`, hashPassword("pacienteB123"))

	// 2. Pacientes clínicos
	db.Exec(`INSERT OR IGNORE INTO pacientes (id_paciente, nombre, edad, diagnostico) 
		VALUES (1, 'Don Roberto Gómez', 78, 'AOS Severa Posicional');`)

	db.Exec(`INSERT OR IGNORE INTO pacientes (id_paciente, nombre, edad, diagnostico) 
		VALUES (2, 'Doña Elena Ramos', 81, 'AOS Moderada con Desaturación');`)

	// 3. Dispositivos asociados de forma independiente a cada paciente
	db.Exec(`INSERT OR IGNORE INTO devices (id_dispositivo, api_token, id_paciente, estado) 
		VALUES ('ESP32_A', 'token_esp32_paciente_a_2026', 1, 'desconectado');`)

	db.Exec(`INSERT OR IGNORE INTO devices (id_dispositivo, api_token, id_paciente, estado) 
		VALUES ('ESP32_B', 'token_esp32_paciente_b_2026', 2, 'desconectado');`)
}
