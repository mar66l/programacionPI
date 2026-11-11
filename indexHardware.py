<!doctype html>
<html lang="es">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>Sensor Monitor</title>
  <style>
    :root{font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#0a1020;color:#e7eefb;font-synthesis:none}*{box-sizing:border-box}body{margin:0;background:radial-gradient(ellipse at 15% 0,#15274a 0,transparent 55%),#0a1020;min-height:100vh}.wrap{max-width:1050px;margin:auto;padding:30px 20px 52px}.header{display:flex;align-items:center;justify-content:space-between;gap:16px;margin-bottom:27px}.brand{display:flex;align-items:center;gap:13px}.logo{display:grid;place-items:center;width:46px;height:46px;background:#122b43;border:1px solid #2c536c;border-radius:14px;color:#6be3d2;font-size:25px}h1{font-size:25px;letter-spacing:-.7px;margin:0}h2{margin:0;font-size:17px;font-weight:650}.muted{color:#91a3bd;font-size:13px;margin:5px 0 0}.status{display:flex;align-items:center;gap:9px;font-size:12px;color:#a7b9d2;background:#142136;border:1px solid #304059;padding:10px 13px;border-radius:30px}.dot{width:9px;height:9px;border-radius:100%;background:#7589a6}.dot.measuring{background:#47d6a0;box-shadow:0 0 10px #47d6a06b}.dot.finished{background:#6ed9ec}.dot.sensor,.dot.offline{background:#f0b25e}.tiles{display:grid;grid-template-columns:repeat(4,1fr);gap:16px;margin-bottom:17px}.panel{background:#111d31d9;border:1px solid #263952;border-radius:17px;padding:21px;box-shadow:0 14px 30px #0309142d}.eyebrow{text-transform:uppercase;letter-spacing:1px;font-weight:650;font-size:11px;color:#90a5c3;margin:0 0 18px}.value{font-size:36px;font-weight:730;letter-spacing:-1.5px;font-variant-numeric:tabular-nums}.unit{color:#9eb0c7;font-size:13px;margin-left:5px}.sub{color:#93a6be;font-size:12px;margin:11px 0 0}.red{color:#ff8590}.cyan{color:#6ed9ec}.green{color:#6ee4aa}.timer{color:#f3ca7a}.chartbox{margin-top:17px}.charthead{display:flex;align-items:center;justify-content:space-between;gap:12px;margin-bottom:14px;flex-wrap:wrap}.controls{display:flex;align-items:center;gap:9px;flex-wrap:wrap}.btn,.select{border:1px solid #38526e;background:#172a45;color:#dce8f9;padding:10px 12px;border-radius:10px;font:inherit;font-size:12px}.btn{cursor:pointer}.btn:hover:not(:disabled){background:#254063}.btn.primary{background:#176452;border-color:#339a7d}.btn.stop{background:#5a3140;border-color:#9b5065}.btn:disabled{cursor:not-allowed;opacity:.48}.select{background:#101d30}.control-label{color:#9fb2ca;font-size:12px}canvas{display:block;width:100%;height:260px;border-radius:10px;background:linear-gradient(180deg,#14263c,#101d30)}.legend{display:flex;gap:18px;margin-top:12px;color:#9fb2ca;font-size:12px}.legend b{display:inline-block;width:10px;height:10px;border-radius:3px;margin-right:6px}.footer{display:flex;justify-content:space-between;gap:10px;flex-wrap:wrap;margin-top:16px;color:#7087a6;font-size:12px}.warn{font-size:12px;color:#9fb2c9;margin:12px 0 0}.error{color:#f0b25e}@media(max-width:820px){.tiles{grid-template-columns:repeat(2,1fr)}}@media(max-width:650px){.tiles{grid-template-columns:1fr}.header{align-items:flex-start}.value{font-size:34px}.wrap{padding:22px 12px}.status{font-size:11px}}
  </style>
</head>
<body>
  <main class="wrap">
    <header class="header">
      <div class="brand">
        <div class="logo">∿</div>
        <div>
          <h1>Sensor Monitor</h1>
          <p class="muted">ESP32 <span aria-hidden="true">·</span> MAX30102 <span aria-hidden="true">·</span> Live Signal</p>
        </div>
      </div>
      <div class="status"><span id="dot" class="dot"></span><span id="status">Sin conexión</span></div>
    </header>

<section class="tiles">
  <article class="panel"><p class="eyebrow">Canal infrarrojo · IR</p><div><span id="ir" class="value cyan">—</span><span class="unit">ADC</span></div><p class="sub">Lectura óptica real</p></article>
  <article class="panel"><p class="eyebrow">Canal rojo · RED</p><div><span id="red" class="value red">—</span><span class="unit">ADC</span></div><p class="sub">Lectura óptica real</p></article>
  <article class="panel"><p class="eyebrow">Adquisición</p><div><span id="count" class="value green">0</span><span class="unit">muestras</span></div><p class="sub" id="quality">Esperando al dispositivo</p></article>
  <article class="panel"><p class="eyebrow">Temporizador</p><div><span id="timer" class="value timer">0</span><span class="unit">s</span></div><p class="sub">de <span id="duration">30</span> s configurados</p></article>
  <article class="panel"><p class="eyebrow">Contacto con el sensor</p><div><span id="finger" class="value">—</span></div><p class="sub" id="fingerQuality">Esperando lectura</p></article>
  <article class="panel"><p class="eyebrow">Frecuencia cardiaca</p><div><span id="bpm" class="value green">—</span><span class="unit">BPM</span></div><p class="sub">Estimación experimental</p></article>
</section>

<article class="panel">
  <p class="eyebrow">Contacto con el sensor</p>
  <div>
    <span id="finger" class="value">—</span>
  </div>
  <p class="sub" id="fingerQuality">Esperando lectura</p>
</article>

    <section class="panel chartbox">
      <div class="charthead">
        <div><h2>Señales ópticas en tiempo real</h2><p class="muted">Intensidad relativa · últimas muestras de la toma</p></div>
        <div class="controls">
          <label class="control-label" for="durationSelect">Duración</label>
          <select id="durationSelect" class="select" aria-label="Duración de la medición">
            <option value="15">15 s</option><option value="30" selected>30 s</option><option value="60">60 s</option><option value="120">120 s</option>
          </select>
          <button id="start" class="btn primary" type="button">Iniciar medición</button>
          <button id="stop" class="btn stop" type="button">Detener medición</button>
          <button id="reset" class="btn" type="button">Nueva medición</button>
        </div>
      </div>
      <canvas id="plot" role="img" aria-label="Gráfica de canales rojo e infrarrojo"></canvas>
      <div class="legend"><span><b style="background:#6ed9ec"></b>Infrarrojo</span><span><b style="background:#ff8590"></b>Rojo</span></div>
      <p class="warn">Monitor de señales ópticas para desarrollo y pruebas. No es un dispositivo médico.</p>
    </section>
    <footer class="footer"><span>Actualización automática cada 500 ms</span><span id="last">Sin datos recibidos</span></footer>
  </main>

  <script>
    const $=id=>document.getElementById(id);
    const cv=$('plot'),ctx=cv.getContext('2d');
    let oldSamples=[],online=false,busy=false;

    function draw(samples){
      const ratio=window.devicePixelRatio||1,w=cv.clientWidth,h=cv.clientHeight;
      cv.width=w*ratio;cv.height=h*ratio;ctx.setTransform(ratio,0,0,ratio,0,0);ctx.clearRect(0,0,w,h);
      ctx.strokeStyle='#31435c';ctx.lineWidth=1;
      for(let i=1;i<5;i++){const y=i*h/5;ctx.beginPath();ctx.moveTo(0,y);ctx.lineTo(w,y);ctx.stroke()}
      if(samples.length<2)return;
      for(const channel of [{key:'ir',color:'#6ed9ec'},{key:'red',color:'#ff8590'}]){
        const values=samples.map(sample=>Number(sample[channel])).filter(Number.isFinite);
        if(values.length<2)continue;
        const low=Math.min(...values),span=Math.max(Math.max(...values)-low,1);
        ctx.strokeStyle=channel.color;ctx.lineWidth=2;ctx.beginPath();
        samples.forEach((sample,index)=>{const x=index/(samples.length-1)*w;const y=h-25-(Number(sample[channel])-low)/span*(h-50);if(index===0)ctx.moveTo(x,y);else ctx.lineTo(x,y)});
        ctx.stroke();
      }
    }

    function format(value){return value==null?'—':Number(value).toLocaleString('es-MX')}
    function setState(state,detection){
      const offline=!online;
      const shown=offline?'Sin conexión':state;
      const style=offline?'offline':shown==='Midiendo'?'measuring':shown==='Finalizada'?'finished':shown==='Sin sensor'?'sensor':'';
      $('status').textContent=shown;$('dot').className='dot '+style;
      $('start').disabled=!online||!detection||shown==='Midiendo'||busy;
      $('stop').disabled=!online||shown!=='Midiendo'||busy;
      $('reset').disabled=!online||busy;
      $('durationSelect').disabled=shown==='Midiendo'||busy;
    }

    function render(data){
      $('ir').textContent=format(data.ir);$('red').textContent=format(data.red);
      $('count').textContent=format(data.count||0);$('timer').textContent=String(data.elapsed_seconds||0);
      $('duration').textContent=String(data.duration_seconds||30);
      $('bpm').textContent = data.bpm > 0 ? data.bpm : '—';
      
      const hasFinger = data.finger_detected === true;

$('finger').textContent = hasFinger ? 'Detectado' : 'Sin dedo';
$('finger').className = 'value ' + (hasFinger ? 'green' : 'error');

$('fingerQuality').textContent = hasFinger
  ? 'Contacto óptico detectado'
  : 'Coloca tu dedo sobre el sensor';
      
      $('durationSelect').value=String(data.duration_seconds||30);
      $('quality').textContent=data.detection?'Sensor detectado · '+data.state:'Sin sensor detectado';
      if(data.last_error){$('last').textContent='Aviso: '+data.last_error;$('last').className='error'}else{$('last').textContent='Última consulta: '+new Date().toLocaleTimeString('es-MX');$('last').className=''}
      if(Array.isArray(data.samples)){oldSamples=data.samples;draw(oldSamples)}
      setState(data.state,data.detection);
    }

    async function request(path,options){
      const response=await fetch(path,Object.assign({cache:'no-store'},options||{}));
      if(!response.ok)throw new Error('HTTP '+response.status);
      return response.json();
    }

    async function command(route){
      if(busy)return;
      busy=true;setState($('status').textContent,true);
      try{
        const suffix=route==='/api/reset'?'?duration='+encodeURIComponent($('durationSelect').value):'';
        const data=await request(route+suffix,{method:'POST'});
        online=true;render(data);
      }catch(error){
        online=false;setState('Sin conexión',false);$('quality').textContent='Verifica la red ESP32-Sensor';
      }finally{busy=false;setState($('status').textContent,$('status').textContent!=='Sin sensor')}
    }

    $('start').onclick=()=>command('/api/start');
    $('stop').onclick=()=>command('/api/stop');
    $('reset').onclick=()=>command('/api/reset');
    $('durationSelect').onchange=()=>{if(!busy)$('quality').textContent='La duración se aplicará en Nueva medición'};

    async function tick(){
      try{const data=await request('/api/data');online=true;render(data)}
      catch(error){online=false;setState('Sin conexión',false);$('quality').textContent='Verifica la red ESP32-Sensor'}
    }
    setInterval(tick,500);window.addEventListener('resize',()=>draw(oldSamples));draw([]);tick();
  </script>
</body>
</html>
