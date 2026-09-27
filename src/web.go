package main

// indexHTML is the embedded web UI
const indexHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>自动解压 AutoUnpack</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, "Microsoft YaHei", sans-serif; background: #f5f6fa; color: #2d3436; padding: 20px; }
.container { max-width: 900px; margin: 0 auto; }
h1 { font-size: 22px; margin-bottom: 16px; display: flex; align-items: center; gap: 8px; }
.card { background: #fff; border-radius: 10px; padding: 20px; margin-bottom: 16px; box-shadow: 0 1px 3px rgba(0,0,0,.08); }
.card h2 { font-size: 16px; margin-bottom: 14px; color: #0984e3; }
.row { display: flex; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.field { flex: 1; min-width: 180px; }
.field label { display: block; font-size: 13px; color: #636e72; margin-bottom: 4px; }
.field input { width: 100%; padding: 8px 10px; border: 1px solid #dfe6e9; border-radius: 6px; font-size: 14px; }
.field input:focus { outline: none; border-color: #0984e3; }
.btn { display: inline-block; padding: 8px 18px; border: none; border-radius: 6px; cursor: pointer; font-size: 14px; margin-right: 8px; }
.btn-primary { background: #0984e3; color: #fff; }
.btn-primary:hover { background: #0873c4; }
.btn-warn { background: #fdcb6e; color: #2d3436; }
.btn-green { background: #00b894; color: #fff; }
.btn-sm { padding: 3px 10px; font-size: 12px; margin-right: 4px; }
.stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(110px, 1fr)); gap: 12px; margin-bottom: 0; }
.stat { background: #f8f9fa; border-radius: 8px; padding: 12px; text-align: center; cursor: pointer; transition: background .2s; border: 2px solid transparent; }
.stat:hover { background: #eef1f5; }
.stat.active { border-color: #0984e3; background: #e8f4fd; }
.stat .num { font-size: 24px; font-weight: 700; color: #0984e3; }
.stat .lbl { font-size: 12px; color: #636e72; margin-top: 4px; }
.stat.ok .num { color: #00b894; }
.stat.fail .num { color: #d63031; }
.stat.run .num { color: #e17055; }
.panel { margin-top: 0; border-top: 1px solid #eee; display: none; }
.panel.show { display: block; }
.panel-inner { padding: 16px 4px 4px; }
.jobitem { display: flex; align-items: center; padding: 8px 10px; border-radius: 6px; margin-bottom: 4px; background: #f8f9fa; font-size: 13px; gap: 8px; flex-wrap: wrap; }
.jobitem:hover { background: #eef1f5; }
.jobitem .jname { flex: 1; min-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.jobitem .jsize { color: #b2bec3; font-size: 12px; white-space: nowrap; }
.jobitem .jtime { color: #b2bec3; font-size: 11px; white-space: nowrap; }
.jobitem .jerr { color: #d63031; font-size: 11px; width: 100%; }
.logbox { background: #2d3436; color: #dfe6e9; border-radius: 8px; padding: 12px; font-family: Consolas, monospace; font-size: 12px; height: 280px; overflow-y: auto; line-height: 1.6; }
.logbox .e { color: #ff7675; } .logbox .w { color: #fdcb6e; } .logbox .i { color: #dfe6e9; }
.badge { display: inline-block; padding: 2px 8px; border-radius: 10px; font-size: 11px; }
.badge-on { background: #00b894; color: #fff; }
.badge-off { background: #b2bec3; color: #fff; }
.badge-sleep { background: #6c5ce7; color: #fff; }
.switch { display: flex; align-items: center; gap: 8px; }
.switch input { width: 18px; height: 18px; }
.path-hint { font-size: 12px; color: #b2bec3; margin-top: 2px; }
.toast { position: fixed; top: 20px; right: 20px; padding: 10px 20px; border-radius: 6px; color: #fff; font-size: 14px; z-index: 999; opacity: 0; transition: opacity .3s; }
.toast.show { opacity: 1; }
.toast.ok { background: #00b894; } .toast.err { background: #d63031; }
.empty-hint { text-align: center; color: #b2bec3; font-size: 13px; padding: 16px; }
</style>
</head>
<body>
<div class="container">
<h1>📦 自动解压 AutoUnpack</h1>

<div class="card">
  <h2>运行状态 <span id="pauseBadge" class="badge badge-off">已暂停</span></h2>
  <div class="stats">
    <div class="stat run" id="cardRunning" onclick="togglePanel('running')"><div class="num" id="stRunning">0</div><div class="lbl">正在解压</div></div>
    <div class="stat" id="cardQueued" onclick="togglePanel('queued')"><div class="num" id="stQueued">0</div><div class="lbl">队列等待</div></div>
    <div class="stat ok" id="cardDone" onclick="togglePanel('done')"><div class="num" id="stOK">0</div><div class="lbl">成功</div></div>
    <div class="stat fail" id="cardFail" onclick="togglePanel('failed')"><div class="num" id="stFail">0</div><div class="lbl">失败</div></div>
    <div class="stat" id="cardCpu" onclick="togglePanel('settings')"><div class="num" id="stCPU">0</div><div class="lbl">设置</div></div>
  </div>
  <div class="panel" id="panel"></div>
  <div style="margin-top:12px;">
    <button class="btn btn-primary" onclick="doScan()">立即扫描</button>
    <button class="btn btn-warn" onclick="doPause()" id="pauseBtn">暂停监控</button>
  </div>
  <div style="margin-top:8px;font-size:13px;color:#636e72;">当前任务: <span id="stCurrent">-</span></div>
</div>

<div class="card">
  <h2>路径配置</h2>
  <div class="row">
    <div class="field">
      <label>监控目录（投放压缩包）</label>
      <input type="text" id="watchDir" placeholder="/vol1/待解压">
      <div class="path-hint">在此目录投放压缩包后自动检测</div>
    </div>
  </div>
  <div class="row">
    <div class="field">
      <label>解压输出目录</label>
      <input type="text" id="outputDir" placeholder="/vol1/已解压">
      <div class="path-hint">每个压缩包解压到独立子目录</div>
    </div>
    <div class="field">
      <label>已处理归档目录</label>
      <input type="text" id="archiveDir" placeholder="/vol1/解压完成">
      <div class="path-hint">解压成功的压缩包直接放这里，失败的在 failed/ 子目录</div>
    </div>
  </div>
  <button class="btn btn-primary" onclick="savePaths()">保存路径</button>
</div>

<div class="card">
  <h2>运行日志</h2>
  <div class="logbox" id="logbox"></div>
</div>

</div>
<div class="toast" id="toast"></div>
<script>
var API = '/app/autounpack/api';
var activePanel = '';

function showToast(msg, ok){
  var t = document.getElementById('toast');
  t.textContent = msg; t.className = 'toast show ' + (ok?'ok':'err');
  setTimeout(function(){t.className='toast';}, 2000);
}
function api(path, opts){
  return fetch(API+path, opts).then(function(r){
    return r.text().then(function(t){
      try { return JSON.parse(t); } catch(e) { return {ok:false, error:t}; }
    });
  });
}
function fmtSize(s){
  if(!s) return '';
  if(s>1048576) return (s/1048576).toFixed(1)+'MB';
  if(s>1024) return (s/1024).toFixed(0)+'KB';
  return s+'B';
}

// ---- panel management ----
var cardMap = {
  running: 'cardRunning', queued: 'cardQueued',
  done: 'cardDone', failed: 'cardFail',
  settings: 'cardCpu'
};
function togglePanel(which){
  var p = document.getElementById('panel');
  if(activePanel === which){
    activePanel = '';
    p.classList.remove('show');
    p.innerHTML = '';
    clearActiveCards();
    return;
  }
  activePanel = which;
  p.classList.add('show');
  clearActiveCards();
  var el = document.getElementById(cardMap[which]);
  if(el) el.classList.add('active');
  renderPanel();
}
function clearActiveCards(){
  Object.values(cardMap).forEach(function(id){
    document.getElementById(id).classList.remove('active');
  });
}
function renderPanel(){
  var p = document.getElementById('panel');
  if(!activePanel) return;
  if(activePanel === 'settings'){
    renderSettings(p);
  } else {
    loadJobs(p);
  }
}

// ---- settings panel ----
function renderSettings(p){
  p.innerHTML = '<div class="panel-inner">'+
    '<div class="row">'+
    '<div class="field"><label>并行解压任务数 (0=自动=CPU核心数)</label><input type="number" id="maxWorkers" min="0" max="64"></div>'+
    '<div class="field"><label>每个任务 7z 线程数 (0=自动=CPU核心数)</label><input type="number" id="threadsPerJob" min="0" max="64"></div>'+
    '</div>'+
    '<div class="row">'+
    '<div class="field"><label>扫描间隔（秒）</label><input type="number" id="checkInterval" min="3" max="300"></div>'+
    '<div class="field"><label>空闲自动休眠（分钟，0=关闭）</label><input type="number" id="idleTimeout" min="0" max="1440"></div>'+
    '</div>'+
    '<div class="row">'+
    '<div class="field"><label>文件稳定检测次数（2=等两次扫描大小不变才解压，1=立即解压）</label><input type="number" id="stableChecks" min="1" max="5"></div>'+
    '<div class="field"><label>磁盘空间预留（MB，解压前检查剩余空间）</label><input type="number" id="diskMargin" min="100" max="102400" step="100"></div>'+
    '</div>'+
    '<div class="row">'+
    '<div class="field"><label>密码列表（每行一个，自动尝试）</label><textarea id="passwords" rows="4" style="width:100%;padding:8px 10px;border:1px solid #dfe6e9;border-radius:6px;font-size:14px;font-family:monospace;" placeholder="password1&#10;password2"></textarea></div>'+
    '</div>'+
    '<div class="row"><div class="field switch"><input type="checkbox" id="overwrite"><label for="overwrite" style="margin:0">覆盖已存在的文件（不勾选则跳过已存在文件）</label></div></div>'+
    '<button class="btn btn-primary" onclick="savePerf()">保存设置</button>'+
    '</div>';
  api('/config').then(function(c){
    document.getElementById('maxWorkers').value = c.max_workers||0;
    document.getElementById('threadsPerJob').value = c.threads_per_job||0;
    document.getElementById('checkInterval').value = c.check_interval||10;
    document.getElementById('idleTimeout').value = c.idle_timeout_min!=null?c.idle_timeout_min:30;
    document.getElementById('stableChecks').value = c.stable_checks||2;
    document.getElementById('diskMargin').value = c.disk_margin_mb||500;
    document.getElementById('overwrite').checked = c.overwrite||false;
    document.getElementById('passwords').value = (c.passwords||[]).join('\n');
  });
}
function savePerf(){
  var pwText = document.getElementById('passwords').value;
  var pwList = pwText.split('\n').map(function(s){return s.trim();}).filter(function(s){return s;});
  api('/config').then(function(c){
    c.max_workers = parseInt(document.getElementById('maxWorkers').value)||0;
    c.threads_per_job = parseInt(document.getElementById('threadsPerJob').value)||0;
    c.check_interval = parseInt(document.getElementById('checkInterval').value)||10;
    c.idle_timeout_min = parseInt(document.getElementById('idleTimeout').value)||0;
    c.stable_checks = parseInt(document.getElementById('stableChecks').value)||2;
    c.disk_margin_mb = parseInt(document.getElementById('diskMargin').value)||500;
    c.overwrite = document.getElementById('overwrite').checked;
    c.passwords = pwList;
    api('/config', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(c)}).then(function(){
      showToast('设置已保存', true);
    });
  });
}

// ---- job lists ----
function loadJobs(p){
  if(!p) p = document.getElementById('panel');
  api('/jobs?status='+activePanel).then(function(d){
    var jobs = d.jobs||[];
    if(jobs.length === 0){
      p.innerHTML = '<div class="empty-hint">暂无任务</div>';
      return;
    }
    var html = '';
    for(var i=0;i<jobs.length;i++){
      var j = jobs[i];
      var btns = '';
      if(j.status === 'running' || j.status === 'queued'){
        btns += '<button class="btn btn-sm btn-warn" onclick="jobAction(\''+encodeURIComponent(j.path)+'\',\'cancel\')">取消</button>';
      }
      if(j.status === 'failed'){
        btns += '<button class="btn btn-sm btn-green" onclick="jobAction(\''+encodeURIComponent(j.path)+'\',\'retry\')">重试</button>';
      }
      if(j.status === 'done' || j.status === 'failed' || j.status === 'cancelled'){
        btns += '<button class="btn btn-sm btn-primary" onclick="jobAction(\''+encodeURIComponent(j.path)+'\',\'remove\')">删除</button>';
      }
      var errHtml = j.error ? '<div class="jerr">错误: '+j.error+'</div>' : '';
      var statusHtml = j.status === 'running' ? '<span class="jsize" style="color:#0984e3">解压中</span>' : '';
      html += '<div class="jobitem">'+
        '<span class="jname" title="'+j.path+'">'+j.name+'</span>'+
        '<span class="jsize">'+fmtSize(j.size)+'</span>'+
        '<span class="jtime">'+(j.start_time||'')+'</span>'+
        statusHtml + btns + errHtml +
        '</div>';
    }
    p.innerHTML = html;
  });
}
function jobAction(pathEnc, action){
  var path = decodeURIComponent(pathEnc);
  api('/job', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify({path:path, action:action})}).then(function(d){
    if(d.ok){
      showToast(action==='cancel'?'已取消':(action==='retry'?'已重试':'已删除'), true);
      refresh();
    } else {
      showToast(d.error||'操作失败', false);
    }
  });
}

// ---- config / paths ----
function loadPaths(){
  api('/config').then(function(c){
    document.getElementById('watchDir').value = c.watch_dir||'';
    document.getElementById('outputDir').value = c.output_dir||'';
    document.getElementById('archiveDir').value = c.archive_dir||'';
  });
}
function savePaths(){
  api('/config').then(function(c){
    c.watch_dir = document.getElementById('watchDir').value.trim();
    c.output_dir = document.getElementById('outputDir').value.trim();
    c.archive_dir = document.getElementById('archiveDir').value.trim();
    api('/config', {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(c)}).then(function(){
      showToast('路径已保存', true);
    });
  });
}

// ---- status / logs ----
function refresh(){
  api('/status').then(function(s){
    document.getElementById('stRunning').textContent = s.running;
    document.getElementById('stQueued').textContent = s.queued;
    document.getElementById('stOK').textContent = s.total_ok;
    document.getElementById('stFail').textContent = s.total_fail;
    document.getElementById('stCPU').textContent = s.cpu_count;
    document.getElementById('stCurrent').textContent = s.current_job||'-';
    var badge = document.getElementById('pauseBadge');
    var btn = document.getElementById('pauseBtn');
    if(s.pause){ badge.textContent='已休眠'; badge.className='badge badge-sleep'; btn.textContent='恢复监控'; }
    else { badge.textContent='运行中'; badge.className='badge badge-on'; btn.textContent='暂停监控'; }
    var lb = document.getElementById('logbox');
    var logs = s.logs||[];
    var html = '';
    for(var i=0;i<logs.length;i++){
      var l = logs[i];
      var cls = l.level==='ERROR'?'e':(l.level==='WARN'?'w':'i');
      html += '<div class="'+cls+'">'+l.time+' ['+l.level+'] '+l.message+'</div>';
    }
    lb.innerHTML = html;
    lb.scrollTop = 0;
    if(activePanel && activePanel !== 'settings') loadJobs();
  }).catch(function(){});
}
function doScan(){ api('/scan', {method:'POST'}).then(function(){showToast('已触发扫描',true);}); }
function doPause(){ api('/pause', {method:'POST'}).then(refresh); }
loadPaths(); refresh(); setInterval(refresh, 3000);
</script>
</body>
</html>`
