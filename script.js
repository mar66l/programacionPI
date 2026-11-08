// ==============================================================================
// GESTIÓN DE ESTADO Y GRÁFICAS (APNEA DEL SUEÑO)
// ==============================================================================
let miGrafica = null;
let usuarioSesion = null;
let pacienteSeleccionadoId = 1;
let intervaloPolling = null;

// ==============================================================================
// 1. AUTENTICACIÓN REAL MEDIANTE BACKEND GO (RBAC + HASH BCRYPT)
// ==============================================================================
const formAuth = document.getElementById('main-form');

if (formAuth) {
    formAuth.onsubmit = async function(e) {
        e.preventDefault();

        const inputEmail = document.getElementById('auth-email');
        const inputPass = document.getElementById('auth-pass');

        if (!inputEmail || !inputPass) return;

        const email = inputEmail.value.trim();
        const password = inputPass.value.trim();

        try {
            const respuesta = await fetch('/api/auth/login', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ email: email, password: password })
            });

            if (!respuesta.ok) {
                const errorData = await respuesta.json();
                alert(`⚠️ Error de acceso: ${errorData.error || 'Credenciales inválidas'}`);
                return;
            }

            usuarioSesion = await respuesta.json();
            iniciarDashboard(usuarioSesion);

        } catch (err) {
            console.error("Error al conectar con la API de autenticación:", err);
            alert("❌ No se pudo conectar con el servidor Go (:8084). Verifica que esté encendido.");
        }
    };
}

function iniciarDashboard(user) {
    const pantallaAuth = document.getElementById('auth-screen');
    const panelDashboard = document.getElementById('dashboard');

    if (pantallaAuth) pantallaAuth.classList.add('hidden');
    if (panelDashboard) panelDashboard.classList.remove('hidden');

    const displayName = document.getElementById('user-display-name');
    const displayRole = document.getElementById('user-display-role');

    if (displayName) displayName.innerText = user.nombre;
    if (displayRole) displayRole.innerText = `Rol: ${user.rol.toUpperCase()}`;

    if (user.rol === 'medico') {
        const panelDoc = document.getElementById('content-doctor');
        if (panelDoc) panelDoc.classList.remove('hidden');
        cargarListaPacientes();
    } else {
        const panelPac = document.getElementById('content-paciente');
        if (panelPac) panelPac.classList.remove('hidden');
        // Si entra como paciente, se enlaza a su propio registro
        pacienteSeleccionadoId = (user.id === 2) ? 1 : 2; 
    }

    inicializarGraficaApnea();

    if (intervaloPolling) clearInterval(intervaloPolling);
    intervaloPolling = setInterval(actualizarPanelEnVivo, 2000);
    actualizarPanelEnVivo();
}

function cerrarSesion() {
    if (intervaloPolling) clearInterval(intervaloPolling);
    usuarioSesion = null;

    const panelDashboard = document.getElementById('dashboard');
    const pantallaAuth = document.getElementById('auth-screen');
    const inputPass = document.getElementById('auth-pass');

    if (panelDashboard) panelDashboard.classList.add('hidden');
    if (pantallaAuth) pantallaAuth.classList.remove('hidden');
    if (inputPass) inputPass.value = "";
}

// ==============================================================================
// 2. CARGA DE PACIENTES DESDE LA BASE DE DATOS
// ==============================================================================
async function cargarListaPacientes() {
    const tablaCuerpo = document.getElementById('tabla-pacientes-cuerpo');
    if (!tablaCuerpo) return;

    try {
        const res = await fetch('/api/pacientes');
        if (!res.ok) return;
        const pacientes = await res.json();

        tablaCuerpo.innerHTML = "";
        pacientes.forEach(p => {
            const fila = document.createElement('tr');
            fila.className = "border-b border-slate-100 hover:bg-slate-50/80 transition-all";
            
            const celdaNombre = `<td class="py-4 px-6 font-bold text-slate-800">${p.nombre}</td>`;
            const celdaEdad = `<td class="py-4 px-6 text-slate-500 font-medium">${p.edad} años</td>`;
            const celdaDiag = `<td class="py-4 px-6 text-indigo-600 font-bold">${p.diagnostico}</td>`;
            const celdaAccion = `
                <td class="py-4 px-6 text-right">
                    <button onclick="seleccionarPaciente(${p.id_paciente}, '${p.nombre}')" class="bg-indigo-600 text-white px-3 py-1.5 rounded-xl text-xs font-black hover:bg-indigo-700 shadow-sm transition-all">
                        Ver Monitoreo
                    </button>
                </td>`;

            fila.innerHTML = celdaNombre + celdaEdad + celdaDiag + celdaAccion;
            tablaCuerpo.appendChild(fila);
        });
    } catch (e) {
        console.error("Error al cargar pacientes:", e);
    }
}

function seleccionarPaciente(id, nombre) {
    pacienteSeleccionadoId = id;
    const nombreHeader = document.getElementById('doctor-paciente-nombre');
    if (nombreHeader) nombreHeader.innerText = nombre;
    actualizarPanelEnVivo();
}

// ==============================================================================
// 3. WATCHDOG DE CONEXIÓN (2 ESP32 CONCURRENTES)
// ==============================================================================
async function actualizarPanelEnVivo() {
    await verificarEstadoDispositivos();
    await cargarHistorialYGrafica(pacienteSeleccionadoId);
}

async function verificarEstadoDispositivos() {
    try {
        const res = await fetch('/api/devices/status');
        if (!res.ok) return;
        const dispositivos = await res.json();

        dispositivos.forEach(dev => {
            const devKey = dev.id_dispositivo.toLowerCase();
            const badgeDev = document.getElementById(`status-badge-${devKey}`);
            const lastSeenText = document.getElementById(`last-seen-${devKey}`);

            if (badgeDev) {
                if (dev.activo) {
                    badgeDev.className = "px-3 py-1 bg-emerald-100 text-emerald-700 text-xs font-black rounded-full uppercase tracking-wider";
                    badgeDev.innerText = "En Línea (Transmitiendo)";
                } else {
                    badgeDev.className = "px-3 py-1 bg-rose-100 text-rose-700 text-xs font-black rounded-full uppercase tracking-wider";
                    badgeDev.innerText = "Desconectado (Sin Señal)";
                }
            }

            if (lastSeenText) {
                lastSeenText.innerText = dev.ultima_conexion 
                    ? `Última recepción: ${dev.ultima_conexion}` 
                    : "Sin registros previos";
            }
        });
    } catch (e) {
        console.warn("Fallo al consultar watchdog de dispositivos:", e);
    }
}

// ==============================================================================
// 4. HISTORIAL Y GRÁFICAS DE APNEA (SpO2, BPM Y POSTURA)
// ==============================================================================
function inicializarGraficaApnea() {
    const ctx = document.getElementById('progresoGrafica');
    if (!ctx) return;

    if (miGrafica) {
        miGrafica.destroy();
    }

    miGrafica = new Chart(ctx.getContext('2d'), {
        type: 'line',
        data: {
            labels: [],
            datasets: [
                {
                    label: 'SpO2 (%)',
                    data: [],
                    borderColor: '#2563EB',
                    backgroundColor: 'rgba(37, 99, 235, 0.1)',
                    yAxisID: 'ySpO2',
                    tension: 0.3,
                    fill: true
                },
                {
                    label: 'Pulso (BPM)',
                    data: [],
                    borderColor: '#DC2626',
                    backgroundColor: 'transparent',
                    yAxisID: 'yBPM',
                    tension: 0.3,
                    borderDash: [4, 4]
                }
            ]
        },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            scales: {
                ySpO2: {
                    type: 'linear',
                    position: 'left',
                    min: 70,
                    max: 100,
                    title: { display: true, text: 'SpO2 (%)' }
                },
                yBPM: {
                    type: 'linear',
                    position: 'right',
                    min: 40,
                    max: 140,
                    grid: { drawOnChartArea: false },
                    title: { display: true, text: 'Frecuencia Cardíaca (BPM)' }
                },
                x: {
                    grid: { display: false }
                }
            }
        }
    });
}

async function cargarHistorialYGrafica(pacienteId) {
    try {
        const res = await fetch(`/api/patients/${pacienteId}/history?limit=15`);
        if (!res.ok) return;
        const lecturas = await res.json();

        if (!lecturas || lecturas.length === 0) return;

        const ultima = lecturas[0];

        const elemSpO2 = document.getElementById('val-flexion');
        const elemBPM = document.getElementById('val-bpm');
        const elemPostura = document.getElementById('val-grid-seguridad');

        if (elemSpO2) {
            if (ultima.finger_detected) {
                elemSpO2.innerText = `${ultima.spo2.toFixed(1)}%`;
                elemSpO2.className = (ultima.spo2 < 90) 
                    ? "text-8xl font-black text-rose-600 animate-pulse" 
                    : "text-8xl font-black text-indigo-600";
            } else {
                elemSpO2.innerText = "-- %";
                elemSpO2.className = "text-6xl font-bold text-slate-400";
            }
        }

        if (elemBPM) {
            elemBPM.innerText = ultima.finger_detected ? `${Math.round(ultima.bpm)} BPM` : "-- BPM";
        }

        if (elemPostura) {
            if (!ultima.finger_detected) {
                elemPostura.innerText = "SENSOR SUELTO / SIN DEDO";
                elemPostura.className = "text-2xl font-bold text-amber-500";
            } else if (ultima.spo2 < 90 && ultima.postura === "supino") {
                elemPostura.innerText = "ALERTA: APNEA POSICIONAL (BOCA ARRIBA)";
                elemPostura.className = "text-2xl font-black text-rose-600 animate-pulse";
            } else {
                elemPostura.innerText = `POSTURA: ${ultima.postura.toUpperCase()}`;
                elemPostura.className = "text-2xl font-black text-emerald-600";
            }
        }

        if (miGrafica) {
            const lecturasCronologicas = [...lecturas].reverse();
            miGrafica.data.labels = lecturasCronologicas.map(l => l.fecha_hora.split(" ")[1] || l.fecha_hora);
            miGrafica.data.datasets[0].data = lecturasCronologicas.map(l => l.finger_detected ? l.spo2 : null);
            miGrafica.data.datasets[1].data = lecturasCronologicas.map(l => l.finger_detected ? l.bpm : null);
            miGrafica.update();
        }

        const listaSensor = document.getElementById('sensor-telemetria-lista');
        if (listaSensor) {
            listaSensor.innerHTML = "";
            lecturas.slice(0, 7).forEach(r => {
                const renglon = document.createElement('div');
                const badgeColor = (r.spo2 < 90) ? "bg-rose-100 text-rose-700" : "bg-emerald-100 text-emerald-700";
                const estadoTxt = !r.finger_detected ? "SIN DEDO" : ((r.spo2 < 90) ? "DESATURACIÓN" : "NORMAL");

                renglon.className = "flex justify-between items-center bg-white p-3 rounded-xl border border-slate-100 shadow-sm mb-1 text-xs font-semibold";
                
                const spanHora = `<span class="text-slate-400">${r.fecha_hora.split(" ")[1] || r.fecha_hora}</span>`;
                const spanPos = `<span class="text-slate-700 font-bold">${r.postura.toUpperCase()}</span>`;
                const spanVal = `<span class="text-indigo-600 font-black">${r.finger_detected ? r.spo2 + '%' : '--'}</span>`;
                const spanBadge = `<span class="px-2 py-0.5 rounded-full ${badgeColor}">${estadoTxt}</span>`;

                renglon.innerHTML = spanHora + spanPos + spanVal + spanBadge;
                listaSensor.appendChild(renglon);
            });
        }

    } catch (e) {
        console.error("Error al actualizar telemetría del paciente:", e);
    }
}