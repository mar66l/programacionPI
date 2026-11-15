
from machine import Pin, SoftI2C
import time
import math

# =====================================
# CONFIGURACION MPU6050
# =====================================

i2c = SoftI2C(
    scl=Pin(22),
    sda=Pin(21),
    freq=100000
)

MPU_ADDR = 0x68

i2c.writeto_mem(MPU_ADDR, 0x6B, b'\x00')
time.sleep_ms(200)

print("\nMONITOR DE POSTURA MPU6050")


# =====================================
# CONVERSION DE DATOS
# =====================================

def conv(high, low):

    valor = (high << 8) | low

    if valor >= 32768:
        valor -= 65536

    return valor


# =====================================
# LECTURA Y CALCULO DE ANGULOS
# =====================================

def leer_angulos():

    datos = i2c.readfrom_mem(MPU_ADDR, 0x3B, 6)

    ax = conv(datos[0], datos[1]) / 16384
    ay = conv(datos[2], datos[3]) / 16384
    az = conv(datos[4], datos[5]) / 16384

    roll = math.degrees(
        math.atan2(ay, az)
    )

    pitch = math.degrees(
        math.atan2(
            -ax,
            math.sqrt(ay**2 + az**2)
        )
    )

    return roll, pitch


# =====================================
# CLASIFICADOR DE POSTURA
# =====================================

def clasificar_postura(roll, pitch, postura_actual):

    # SUPINO
    if pitch <= -78:
        return "Supino"

    # LATERAL DERECHO
    if abs(roll) <= 35 and pitch >= -5:
        return "Lateral derecho"

    # PRONO
    if abs(roll) >= 145 and pitch >= 60:
        return "Prono"

    # LATERAL IZQUIERDO
    if abs(roll) >= 145 and -25 <= pitch <= 20:
        return "Lateral izquierdo"

    # Conservar ultima postura reconocida
    return postura_actual


# =====================================
# MONITOREO Y GRAFICA
# =====================================

postura_actual = "Sin clasificar"

print("Iniciando medicion continua...")

while True:

    try:

        roll, pitch = leer_angulos()

        postura_actual = clasificar_postura(
            roll,
            pitch,
            postura_actual
        )

        # Mostrar siempre los angulos y la postura
        print("Roll: {:.2f} | Pitch: {:.2f} | Postura: {}".format(
            roll,
            pitch,
            postura_actual
        ))

        time.sleep_ms(100)

    except OSError as e:

        print("Error I2C:", e)
        time.sleep_ms(100)


        # Datos numericos para el Plotter
        print((roll, pitch))

        time.sleep_ms(100)

    except OSError as e:

        print("Error I2C:", e)
        time.sleep_ms(100)
