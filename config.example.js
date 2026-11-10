// Copie este archivo como config.js y ajuste los valores. NO incluya contraseñas ni tokens aquí.
window.APP_CONFIG = {
  MODE: "live",                       // "mock" = datos simulados | "live" = backend real
  BASE: "http://192.168.1.10:8000",   // IP del equipo que ejecuta el servidor en la red local
  POLL_MS: 2000,                      // cada cuánto se actualiza la pantalla (ms)
  STALE_S: 10                         // segundos sin datos para marcar "Sin conexión"
};
