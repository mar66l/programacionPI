// CONFIGURACION BASICA
#include <Servo.h>  // Incluir librería del servo

const int sensorPin = A0;      // Pin del sensor AD8232
const int servoPin1 = 8;       // Pin del servo 1
const int servoPin2 = 7;       // Pin del servo 2
const int servoPin3 = 6;       // Pin del servo 3
const float umbral = 0.60;     // AJUSTAR ESTE VALOR

// Variables de deteccion
unsigned long ultimaDeteccion = 0;
const int tiempoMinimo = 1000;  // Aumentado para movimiento servo

// Objeto servo
Servo servo1, servo2, servo3;
int posicionServo = 0;         // Posición actual del servo
bool manoAbierta = true;       // Estado de la mano

void setup() {
  Serial.begin(9600);
  
  // Configurar servos
  servo1.attach(servoPin1);
  servo2.attach(servoPin2);
  servo3.attach(servoPin3);

  // Posición inicial (mano abierta)
  servo1.write(0);
  servo2.write(0);
  servo3.write(0);  
  
  Serial.println("Sistema EMG con 3 Servos - Iniciado");
  Serial.println("Cierra la mano para activar los servos 180°");
  delay(1000);
}

void loop() {
  // 1. Leer y procesar senal EMG
  float senalEMG = leerSenalEMG();
  
  // 2. Detectar cierre de mano
  bool deteccion = detectarCierreMano(senalEMG);
  
  // 3. Controlar servo si se detecta cierre
  if (deteccion) {
    controlarServos();
  }
  
  // 4. Monitoreo de valores
  Serial.println(senalEMG);
  
  delay(50);
}

// Funcion para leer senal EMG
float leerSenalEMG() {
  unsigned long inicio = millis();
  int maximo = 0;
  int minimo = 1024;
  
  // Muestrear por 50ms
  while (millis() - inicio < 50) {
    int lectura = analogRead(sensorPin);
    if (lectura > maximo) maximo = lectura;
    if (lectura < minimo) minimo = lectura;
  }
  
  // Convertir a voltaje
  float voltajePico = (maximo - minimo) * 3.3 / 1024;
  return voltajePico;
}

// Deteccion de cierre de mano
bool detectarCierreMano(float senal) {
  unsigned long ahora = millis();
  
  // Evitar detecciones multiples muy seguidas
  if (ahora - ultimaDeteccion < tiempoMinimo) {
    return false;
  }
  
  // Solo detectar si la senal supera el umbral
  if (senal > umbral) {
    ultimaDeteccion = ahora;
    return true;
  }
  
  return false;
}

// Control de los 3 servos - Movimiento sincronizado
void controlarServos() {
  Serial.println("CIERRE DETECTADO - Moviendo 3 servos...");
  
  if (manoAbierta) {
    // Cerrar mano → Servos a 180°
    Serial.println("Servos: 0° → 180°");
    
    // Movimiento suave y sincronizado
    for (posicionServo = 0; posicionServo <= 180; posicionServo += 2) {
      servo1.write(posicionServo);
      servo2.write(posicionServo);
      servo3.write(posicionServo);
      delay(15);  // Velocidad del movimiento
    }
    
    manoAbierta = false;
    Serial.println("3 Servos en posición CERRADO (180°)");
    
  } else {
    // Abrir mano → Servo a 0°
    Serial.println("Servos: 180° → 0°");
    
    // Movimiento suave y sincronizado
    for (posicionServo = 180; posicionServo >= 0; posicionServo -= 2) {
      servo1.write(posicionServo);
      servo2.write(posicionServo);
      servo3.write(posicionServo);
      delay(15);  // Velocidad del movimiento
    }
    
    manoAbierta = true;
    Serial.println("3 Servos en posición ABIERTO (0°)");
  }
}