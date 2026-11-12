# SueñoSeguro / HipoSentry — monitoreo de apnea del sueño

Backend en Go (Echo + SQLite) que también sirve la pantalla. Todo corre en **un solo puerto (8084)**.

```
main.go auth.go db.go repo.go handlers.go   servidor y API (sesiones, roles, validación, auditoría)
frontend/                                   pantalla (index.html + config.js), la sirve el servidor
esp32/                                      main.py + firmware.py + config.example.py (MicroPython)
herramientas/simulador_esp32.py             simula los 2 ESP32 sin hardware (datos marcados "simulado")
docs/INTEGRACION.md                         qué se encontró, qué se cambió y qué falta
legacy_no_entregar/                         archivos anteriores: BORRAR antes de entregar el ZIP final
```

## Cómo ejecutarlo en tu computadora (Windows; en Mac/Linux es igual)
1. **Instala Go** desde https://go.dev/dl (tu `go.mod` pide Go 1.26.2 o más nuevo). Verifica: abre una terminal y escribe `go version`.
2. **Configura secretos:** en la carpeta del proyecto copia `.env.example` como `.env` y cambia la contraseña y los dos tokens.
   (Windows: `copy .env.example .env`). Sin comentarios al final de las líneas.
3. **Arranca:** `go run .` — la primera vez descarga dependencias (necesita internet). Debe decir que escucha en `0.0.0.0:8084`.
4. **Abre** http://localhost:8084. Cuentas (contraseña = `SEED_PASSWORD` de tu `.env`):
   `doctor@redmedica.com` (médico) · `pacienteA@redmedica.com` (Don Roberto, paciente Tipo 1: pre-diagnóstico, ECG + oxímetro) ·
   `pacienteB@redmedica.com` (Doña Elena, paciente Tipo 2: diagnosticada, giroscopio + oxímetro) · `admin@redmedica.com` (administrador).
5. **Sin hardware:** en otra terminal, `python herramientas/simulador_esp32.py` envía datos de ambos ESP32 a la vez.
   Verás en pantalla: alerta de oxígeno bajo (A/Elena), alerta boca arriba (B/Don Roberto), "Sin dedo", y con
   `--sin-b 40` la desconexión de un ESP32 mientras el otro sigue. Para ver ambas sesiones a la vez usa dos navegadores
   (o una ventana privada).
6. **Con los ESP32 reales:** averigua la IP de tu compu (`ipconfig` → "Dirección IPv4"), deja que Windows permita el
   puerto 8084 en el firewall (aparece un aviso al arrancar), y en cada placa copia `esp32/config.example.py` como
   `config.py` con su propia red, `SERVER_URL=http://<IP>:8084/api/telemetry`, `DEVICE_ID`, `DEVICE_TOKEN` y `SENSOR`.
   Sube `main.py`, `firmware.py` y `config.py`. La compu y los ESP32 deben estar en la **misma red Wi-Fi**.
7. **Si cambias un token en `.env`**, reinicia el servidor (se actualiza solo). Si cambias el esquema, borra `sueno_seguro.db`.

## Si algo falla
- `go run .` muestra errores de compilación → copia el mensaje completo y mándamelo.
- Pantalla en blanco o "No se puede conectar" → ¿está corriendo el servidor? ¿abriste `http://localhost:8084` y no el archivo suelto?
- El ESP32 recibe 401 → el `DEVICE_ID`/`DEVICE_TOKEN` no coinciden con `.env`; 422 → lectura fuera de rango o postura inválida.
- "Sin datos" en un paciente → su ESP32 aún no ha enviado nada (o no está asociado a ese paciente).
