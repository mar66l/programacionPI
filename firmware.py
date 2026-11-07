from machine import Pin, I2C
import time
import ustruct
import math

# ==============================================================================
# CONFIGURACIÓN I2C Y MPU6050
# ==============================================================================
i2c = I2C(0, scl=Pin(22), sda=Pin(21), freq=100000)
MPU_ADDR = 0x68

def inicializar_mpu():
    try:
        devices = i2c.scan()
        if MPU_ADDR in devices:
            i2c.writeto_mem(MPU_ADDR, 0x6B, b'\x00')
            time.sleep(0.05)
            return True
    except Exception:
        pass
    return False

def leer_muestra_cruda():
    """Lee el sensor con auto-recuperación si se mueve un cable físico"""
    intentos = 0
    while intentos < 3:
        try:
            raw = i2c.readfrom_mem(MPU_ADDR, 0x3B, 14)
            ax_raw, ay_raw, az_raw, temp_raw, gx_raw, gy_raw, gz_raw = ustruct.unpack('>hhhhhhh', raw)
            
            # Escala a unidades 'g' (±2g) y °/s (±250°/s)
            ax = ax_raw / 16384.0
            ay = ay_raw / 16384.0
            az = az_raw / 16384.0
            
            gx = gx_raw / 131.0
            gy = gy_raw / 131.0
            gz = gz_raw / 131.0
            
            temp_c = (temp_raw / 340.0) + 36.53
            return ax, ay, az, gx, gy, gz, temp_c
        except OSError:
            intentos += 1
            time.sleep(0.02)
            inicializar_mpu()
            
    return 0.0, 0.0, 1.0, 0.0, 0.0, 0.0, 36.5

# ==============================================================================
# CALIBRACIÓN Y CÁLCULO DE ÁNGULOS EN REPOSO (DECÚBITO / CAMA)
# ==============================================================================
def leer_angulos_sueno():
    """
    Filtro de media móvil (5 muestras) adaptado para posición acostada.
    - angulo_giro_lateral: Giro del cuerpo sobre la cama (Roll respecto a Z).
      0° = Boca arriba plano.
      +90° = De lado derecho.
      -90° = De lado izquierdo.
      ±180° = Boca abajo.
    - angulo_elevacion: Inclinación del torso o cabecera (Pitch respecto a Z).
    """
    muestras = 5
    suma_giro, suma_elev = 0.0, 0.0
    
    for _ in range(muestras):
        ax, ay, az, gx, gy, gz, tc = leer_muestra_cruda()
        
        # En reposo horizontal, la gravedad apunta principalmente a Z (+1g boca arriba)
        # Giro lateral (rotación entre Y y Z)
        giro = math.atan2(ay, az) * (180.0 / math.pi)
        
        # Elevación longitudinal del torso (entre X y la resultante Y-Z)
        elev = math.atan2(ax, math.sqrt(ay**2 + az**2)) * (180.0 / math.pi)
        
        suma_giro += giro
        suma_elev += elev
        time.sleep(0.005)
        
    ang_giro = suma_giro / muestras
    ang_elev = suma_elev / muestras
    
    # Umbral de zona muerta para evitar oscilaciones por temblor en reposo
    if abs(ang_giro) < 0.5: ang_giro = 0.0
    if abs(ang_elev) < 0.5: ang_elev = 0.0
        
    return ang_giro, ang_elev