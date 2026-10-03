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
.sponsor-box { margin-top: 14px; padding: 10px 12px; background: #fafafa; border-radius: 8px; border: 1px solid #e8e8e8; text-align: center; }
.sponsor-box .sponsor-title { font-size: 13px; color: #999; margin-bottom: 2px; }
.sponsor-box .sponsor-desc { font-size: 11px; color: #bbb; margin-bottom: 8px; }
.sponsor-qrcodes { display: flex; gap: 16px; justify-content: center; }
.sponsor-qr { text-align: center; }
.sponsor-qr img { width: 96px; height: 96px; border-radius: 6px; border: 2px solid #fff; box-shadow: 0 1px 4px rgba(0,0,0,.08); }
.sponsor-qr .qr-label { font-size: 11px; color: #999; margin-top: 4px; }
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
    '<div class="sponsor-box">'+
    '<div class="sponsor-title">制作不易，感谢支持</div>'+
    '<div class="sponsor-desc">觉得好用请扫码赞助 ~</div>'+
    '<div class="sponsor-qrcodes">'+
    '<div class="sponsor-qr"><img src="data:image/jpeg;base64,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" alt="支付宝"><div class="qr-label">支付宝</div></div>'+
    '<div class="sponsor-qr"><img src="data:image/jpeg;base64,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" alt="微信"><div class="qr-label">微信</div></div>'+
    '</div></div>'+
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
