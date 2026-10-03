package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ---------- Config ----------

type Config struct {
	WatchDir       string   `json:"watch_dir"`
	OutputDir      string   `json:"output_dir"`
	ArchiveDir     string   `json:"archive_dir"`
	MaxWorkers     int      `json:"max_workers"`
	ThreadsPerJob  int      `json:"threads_per_job"`
	Patterns       []string `json:"patterns"`
	Overwrite      bool     `json:"overwrite"`
	CheckInterval  int      `json:"check_interval"`
	Pause          bool     `json:"pause"`
	IdleTimeoutMin int      `json:"idle_timeout_min"`
	Passwords      []string `json:"passwords"`
	StableChecks   int      `json:"stable_checks"`
	DiskMarginMB   int64    `json:"disk_margin_mb"`
}

func defaultConfig() *Config {
	n := runtime.NumCPU()
	return &Config{
		WatchDir:      "",
		OutputDir:     "",
		ArchiveDir:    "",
		MaxWorkers:    n,
		ThreadsPerJob: n,
		Patterns:      []string{".zip", ".rar", ".7z", ".tar", ".gz", ".tgz", ".bz2", ".tbz2", ".xz", ".txz", ".zst", ".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst", ".iso", ".cab", ".lzh", ".lha", ".arj"},
		Overwrite:     false,
		CheckInterval:  10,
		IdleTimeoutMin: 30,
		StableChecks:  2,
		DiskMarginMB:   500,
		Pause:         false,
	}
}

var (
	cfgPath       string
	backupCfgPath string
	cfg           *Config
	cfgMu         sync.RWMutex
)

func loadConfig() *Config {
	c := defaultConfig()
	data, err := os.ReadFile(cfgPath)
	if err == nil {
		_ = json.Unmarshal(data, c)
	}
	// If local config missing or has no watch dir, try restore from persistent backup
	if c.WatchDir == "" && backupCfgPath != "" {
		if bdata, berr := os.ReadFile(backupCfgPath); berr == nil {
			var bc Config
			if json.Unmarshal(bdata, &bc) == nil && bc.WatchDir != "" {
				c = &bc
				addLog("INFO", "从备份恢复配置")
				// Write back to local
				_ = os.WriteFile(cfgPath, bdata, 0644)
			}
		}
	}
	if c.DiskMarginMB <= 0 {
		c.DiskMarginMB = 500
	}
	return c
}

func saveConfig(c *Config) error {
	data, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(cfgPath, data, 0644); err != nil {
		return err
	}
	// Persistent backup survives app reinstall
	if backupCfgPath != "" {
		_ = os.MkdirAll(filepath.Dir(backupCfgPath), 0755)
		_ = os.WriteFile(backupCfgPath, data, 0644)
	}
	return nil
}

func getCfg() *Config {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return cfg
}

func getCfgCopy() Config {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return *cfg
}

func setCfg(c *Config) {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	cfg = c
}

// ---------- Log ----------

type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

var (
	logMu  sync.Mutex
	logBuf []LogEntry
)

func addLog(level, msg string) {
	entry := LogEntry{
		Time:    time.Now().Format("2006-01-02 15:04:05"),
		Level:   level,
		Message: msg,
	}
	logMu.Lock()
	logBuf = append(logBuf, entry)
	if len(logBuf) > 500 {
		logBuf = logBuf[len(logBuf)-500:]
	}
	logMu.Unlock()
	log.Printf("[%s] %s", level, msg)
}

// ---------- Job Registry ----------

type JobInfo struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	Status    string `json:"status"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Error     string `json:"error"`
}

const maxRetainedJobs = 200

var (
	jobReg   = make(map[string]*JobInfo)
	jobRegMu sync.Mutex
	procMap  = make(map[string]*exec.Cmd)
	procMu   sync.Mutex
)

func regJob(path, name string, size int64, status string) {
	jobRegMu.Lock()
	defer jobRegMu.Unlock()
	j := &JobInfo{Path: path, Name: name, Size: size, Status: status}
	if status == "running" {
		j.StartTime = time.Now().Format("2006-01-02 15:04:05")
	}
	jobReg[path] = j
	pruneFinishedJobs()
}

func updateJob(path, status, errMsg string) {
	jobRegMu.Lock()
	defer jobRegMu.Unlock()
	if j, ok := jobReg[path]; ok {
		j.Status = status
		j.Error = errMsg
		j.EndTime = time.Now().Format("2006-01-02 15:04:05")
	}
}

func removeJob(path string) {
	jobRegMu.Lock()
	defer jobRegMu.Unlock()
	delete(jobReg, path)
}

func listJobs(status string) []*JobInfo {
	jobRegMu.Lock()
	defer jobRegMu.Unlock()
	var result []*JobInfo
	for _, j := range jobReg {
		if status == "" || j.Status == status {
			result = append(result, j)
		}
	}
	return result
}

func pruneFinishedJobs() {
	if len(jobReg) <= maxRetainedJobs {
		return
	}
	type finished struct {
		path string
		end  string
	}
	var done []finished
	for p, j := range jobReg {
		if j.Status == "done" || j.Status == "failed" || j.Status == "cancelled" {
			done = append(done, finished{p, j.EndTime})
		}
	}
	toRemove := len(jobReg) - maxRetainedJobs
	for i := 0; i < toRemove && i < len(done); i++ {
		delete(jobReg, done[i].path)
	}
}

// ---------- Activity tracking ----------

var lastActivityNano atomic.Int64

func touchActivity() {
	lastActivityNano.Store(time.Now().UnixNano())
}

func idleDuration() time.Duration {
	return time.Since(time.Unix(0, lastActivityNano.Load()))
}

// ---------- Disk space ----------

// availableBytes is implemented in disk_linux.go / disk_other.go

// ---------- Extraction ----------

type Job struct {
	Path string
	Size int64
}

var (
	jobQueue   = make(chan Job, 256)
	activeSet  = make(map[string]bool)
	activeMu   sync.Mutex
	pendingMap = make(map[string]int64)
	pendingMu  sync.Mutex
)

var cancelledSet sync.Map

func isArchiveExt(name string) bool {
	c := getCfg()
	lower := strings.ToLower(name)
	for _, p := range c.Patterns {
		if strings.HasSuffix(lower, p) {
			return true
		}
	}
	return false
}

func extractName(path string) string {
	base := filepath.Base(path)
	lower := strings.ToLower(base)
	compound := []string{".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst"}
	for _, ext := range compound {
		if strings.HasSuffix(lower, ext) {
			return strings.TrimSuffix(base, ext)
		}
	}
	ext := filepath.Ext(base)
	if ext != "" {
		return strings.TrimSuffix(base, ext)
	}
	return base
}

func isSecondaryVolume(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".rar") {
		idx := strings.LastIndex(lower, ".part")
		if idx > 0 {
			partNumStr := lower[idx+5 : len(lower)-4]
			if partNum, err := strconv.Atoi(partNumStr); err == nil && partNum > 1 {
				return true
			}
		}
		return false
	}
	if strings.HasSuffix(lower, ".7z.") {
		idx := strings.LastIndex(lower, ".7z.")
		numStr := lower[idx+4:]
		if num, err := strconv.Atoi(numStr); err == nil && num > 1 {
			return true
		}
	}
	if strings.Contains(lower, ".zip.") {
		idx := strings.LastIndex(lower, ".zip.")
		numStr := lower[idx+5:]
		if num, err := strconv.Atoi(numStr); err == nil && num > 1 {
			return true
		}
	}
	if matched, _ := filepath.Match("*.z[0-9][0-9]", lower); matched {
		return true
	}
	if matched, _ := filepath.Match("*.r[0-9][0-9]", lower); matched {
		return true
	}
	return false
}

func uniqueDest(dir, name string) string {
	dest := filepath.Join(dir, name)
	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return dest
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	for i := 1; i < 1000; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_%d%s", base, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
	return filepath.Join(dir, fmt.Sprintf("%s_dup%s", base, ext))
}

func moveFile(src, dest string) error {
	if err := os.Rename(src, dest); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		os.Remove(dest)
		return err
	}
	return os.Remove(src)
}

func extractArchive(archivePath, outDir string, passwords []string, jobPath string) error {
	c := getCfg()
	sevenZZ := filepath.Join(os.Getenv("TRIM_APPDEST"), "bin", "7zz")
	if _, err := os.Stat(sevenZZ); err != nil {
		if p, err := exec.LookPath("7zz"); err == nil {
			sevenZZ = p
		} else if p, err := exec.LookPath("7z"); err == nil {
			sevenZZ = p
		} else {
			return fmt.Errorf("7zz binary not found")
		}
	}

	threads := c.ThreadsPerJob
	if threads <= 0 {
		threads = runtime.NumCPU()
	}

	tryExtract := func(password string) error {
		args := []string{"x", archivePath, fmt.Sprintf("-o%s", outDir), "-y",
			fmt.Sprintf("-mmt%d", threads), "-bso0", "-bsp0", "-bb0"}
		if c.Overwrite {
			args = append(args, "-aoa")
		} else {
			args = append(args, "-aos")
		}
		if password != "" {
			args = append(args, "-p"+password)
		}
		cmd := exec.Command(sevenZZ, args...)
		var stderr strings.Builder
		cmd.Stderr = &stderr
		if err := cmd.Start(); err != nil {
			return err
		}
		procMu.Lock()
		procMap[jobPath] = cmd
		procMu.Unlock()
		err := cmd.Wait()
		procMu.Lock()
		delete(procMap, jobPath)
		procMu.Unlock()
		if err != nil {
			return fmt.Errorf("%s: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil
	}

	if err := tryExtract(""); err == nil {
		return nil
	} else {
		errStr := strings.ToLower(err.Error())
		isPwdIssue := strings.Contains(errStr, "wrong password") ||
			strings.Contains(errStr, "can not open encrypted") ||
			strings.Contains(errStr, "enter password") ||
			strings.Contains(errStr, "encrypted data")
		if !isPwdIssue || len(passwords) == 0 {
			return err
		}
	}

	for i, pwd := range passwords {
		if pwd == "" {
			continue
		}
		if e := tryExtract(pwd); e == nil {
			addLog("INFO", fmt.Sprintf("密码正确（第%d个）", i+1))
			return nil
		}
	}
	return fmt.Errorf("密码错误（尝试了%d个密码）", len(passwords))
}

func processJob(job Job) {
	c := getCfg()
	name := filepath.Base(job.Path)

	jobRegMu.Lock()
	j, exists := jobReg[job.Path]
	if exists && j.Status == "cancelled" {
		jobRegMu.Unlock()
		cleanupActive(job.Path)
		return
	}
	jobRegMu.Unlock()

	relPath, err := filepath.Rel(c.WatchDir, job.Path)
	if err != nil {
		relPath = name
	}
	outRel := extractName(relPath)
	outDir := filepath.Join(c.OutputDir, outRel)

	// Disk space pre-check
	avail, derr := availableBytes(outDir)
	if derr == nil {
		need := uint64(job.Size) * 2
		margin := uint64(c.DiskMarginMB) * 1024 * 1024
		if avail < need+margin {
			errMsg := fmt.Sprintf("磁盘空间不足: 需要约 %.1f GB，可用 %.1f GB",
				float64(need+margin)/1024/1024/1024, float64(avail)/1024/1024/1024)
			addLog("ERROR", errMsg+" ("+relPath+")")
			updateJob(job.Path, "failed", errMsg)
			failedDir := filepath.Join(c.ArchiveDir, "failed")
			_ = os.MkdirAll(failedDir, 0755)
			dest := uniqueDest(failedDir, name)
			if e2 := moveFile(job.Path, dest); e2 != nil {
				addLog("WARN", fmt.Sprintf("移动失败压缩包到 %s 失败: %v", dest, e2))
			}
			cleanupActive(job.Path)
			return
		}
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		addLog("ERROR", fmt.Sprintf("创建输出目录失败 %s: %v", outDir, err))
		updateJob(job.Path, "failed", err.Error())
		cleanupActive(job.Path)
		return
	}

	updateJob(job.Path, "running", "")
	addLog("INFO", fmt.Sprintf("开始解压: %s (%.1f MB)", relPath, float64(job.Size)/1024/1024))

	err = extractArchive(job.Path, outDir, c.Passwords, job.Path)

	_, userCancelled := cancelledSet.LoadAndDelete(job.Path)

	if userCancelled {
		addLog("WARN", fmt.Sprintf("用户取消解压: %s", relPath))
		updateJob(job.Path, "cancelled", "用户取消")
		failedDir := filepath.Join(c.ArchiveDir, "failed")
		_ = os.MkdirAll(failedDir, 0755)
		dest := uniqueDest(failedDir, name)
		if e2 := moveFile(job.Path, dest); e2 != nil {
			addLog("WARN", fmt.Sprintf("移动已取消压缩包失败: %v", e2))
		}
	} else if err != nil {
		errMsg := err.Error()
		addLog("ERROR", fmt.Sprintf("解压失败: %s, 错误: %v", relPath, err))
		updateJob(job.Path, "failed", errMsg)
		failedDir := filepath.Join(c.ArchiveDir, "failed")
		_ = os.MkdirAll(failedDir, 0755)
		dest := uniqueDest(failedDir, name)
		if e2 := moveFile(job.Path, dest); e2 != nil {
			addLog("WARN", fmt.Sprintf("移动失败压缩包到 %s 失败: %v", dest, e2))
		}
	} else {
		addLog("INFO", fmt.Sprintf("解压完成: %s -> %s", relPath, outDir))
		updateJob(job.Path, "done", "")
		_ = os.MkdirAll(c.ArchiveDir, 0755)
		dest := uniqueDest(c.ArchiveDir, name)
		if e2 := moveFile(job.Path, dest); e2 != nil {
			addLog("WARN", fmt.Sprintf("移动归档失败: %v", e2))
		}
	}

	cleanupActive(job.Path)
}

func cleanupActive(path string) {
	activeMu.Lock()
	delete(activeSet, path)
	activeMu.Unlock()
}

func worker() {
	for job := range jobQueue {
		processJob(job)
	}
}

func scanAndEnqueue() {
	c := getCfg()
	if c.Pause {
		return
	}
	if c.WatchDir == "" || c.OutputDir == "" || c.ArchiveDir == "" {
		addLog("WARN", fmt.Sprintf("扫描跳过：路径未配置 watch=%q output=%q archive=%q", c.WatchDir, c.OutputDir, c.ArchiveDir))
		return
	}

	archiveAbs, _ := filepath.Abs(c.ArchiveDir)
	outputAbs, _ := filepath.Abs(c.OutputDir)

	totalFiles := 0
	matchedFiles := 0
	skippedActive := 0
	skippedDirs := 0
	skippedVolumes := 0
	pendingWait := 0

	err := filepath.WalkDir(c.WatchDir, func(full string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			absPath, _ := filepath.Abs(full)
			if absPath == archiveAbs || absPath == outputAbs {
				skippedDirs++
				return filepath.SkipDir
			}
			if len(d.Name()) > 0 && d.Name()[0] == '.' {
				skippedDirs++
				return filepath.SkipDir
			}
			return nil
		}
		totalFiles++
		name := d.Name()
		if !isArchiveExt(name) {
			return nil
		}
		if isSecondaryVolume(name) {
			skippedVolumes++
			return nil
		}
		matchedFiles++

		info, err := d.Info()
		if err != nil {
			return nil
		}

		if c.StableChecks >= 2 {
			pendingMu.Lock()
			lastSize, seen := pendingMap[full]
			if !seen || lastSize != info.Size() {
				pendingMap[full] = info.Size()
				pendingMu.Unlock()
				pendingWait++
				return nil
			}
			delete(pendingMap, full)
			pendingMu.Unlock()
		}

		activeMu.Lock()
		if activeSet[full] {
			activeMu.Unlock()
			skippedActive++
			return nil
		}
		activeSet[full] = true
		activeMu.Unlock()

		select {
		case jobQueue <- Job{Path: full, Size: info.Size()}:
			touchActivity()
			regJob(full, name, info.Size(), "queued")
			addLog("INFO", fmt.Sprintf("发现压缩包并加入队列: %s (%.1f MB)", full, float64(info.Size())/1024/1024))
		default:
			cleanupActive(full)
		}
		return nil
	})

	if err != nil {
		addLog("ERROR", fmt.Sprintf("扫描目录 %s 出错: %v", c.WatchDir, err))
	}
	addLog("DEBUG", fmt.Sprintf("扫描完成: 文件=%d 压缩包=%d 分卷跳过=%d 等待稳定=%d 处理中跳过=%d 队列=%d",
		totalFiles, matchedFiles, skippedVolumes, pendingWait, skippedActive, len(jobQueue)))
}

func scannerLoop() {
	for {
		c := getCfg()
		interval := c.CheckInterval
		if interval < 3 {
			interval = 3
		}
		time.Sleep(time.Duration(interval) * time.Second)
		scanAndEnqueue()
		if c.IdleTimeoutMin > 0 && !c.Pause && c.WatchDir != "" {
			if idleDuration() > time.Duration(c.IdleTimeoutMin)*time.Minute {
				addLog("INFO", fmt.Sprintf("空闲 %d 分钟无新压缩包，自动进入休眠", c.IdleTimeoutMin))
				cfgCopy := getCfgCopy()
				cfgCopy.Pause = true
				saveConfig(&cfgCopy)
				setCfg(&cfgCopy)
			}
		}
	}
}

func startWorkers(n int) {
	if n <= 0 {
		n = runtime.NumCPU()
	}
	for i := 0; i < n; i++ {
		go worker()
	}
	addLog("INFO", fmt.Sprintf("启动 %d 个解压工作协程，每任务 %d 线程", n, getCfg().ThreadsPerJob))
}

var gwPrefix = "/app/autounpack"
const version = "1.5.5"

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != gwPrefix+"/" && r.URL.Path != gwPrefix {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(indexHTML))
}

func handleGetConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, getCfg())
}

func handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	c := getCfgCopy()
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if c.MaxWorkers <= 0 {
		c.MaxWorkers = runtime.NumCPU()
	}
	if c.ThreadsPerJob <= 0 {
		c.ThreadsPerJob = runtime.NumCPU()
	}
	if c.CheckInterval < 3 {
		c.CheckInterval = 3
	}
	if len(c.Patterns) == 0 {
		c.Patterns = defaultConfig().Patterns
	}
	if err := saveConfig(&c); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	setCfg(&c)
	_ = os.MkdirAll(c.OutputDir, 0755)
	_ = os.MkdirAll(c.ArchiveDir, 0755)
	_ = os.MkdirAll(filepath.Join(c.ArchiveDir, "failed"), 0755)
	addLog("INFO", fmt.Sprintf("配置已更新: 监控=%s 输出=%s 归档=%s 并发=%d", c.WatchDir, c.OutputDir, c.ArchiveDir, c.MaxWorkers))
	writeJSON(w, map[string]bool{"ok": true})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	c := getCfg()
	logMu.Lock()
	entries := make([]LogEntry, len(logBuf))
	copy(entries, logBuf)
	logMu.Unlock()
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	jobRegMu.Lock()
	runningCount, queuedCount, doneCount, failCount := 0, 0, 0, 0
	var currentJob string
	for _, j := range jobReg {
		switch j.Status {
		case "running":
			runningCount++
			if currentJob == "" {
				currentJob = j.Name
			}
		case "queued":
			queuedCount++
		case "done":
			doneCount++
		case "failed":
			failCount++
		}
	}
	jobRegMu.Unlock()

	logs := entries
	if len(logs) > 100 {
		logs = logs[:100]
	}

	writeJSON(w, map[string]interface{}{
		"running":     runningCount,
		"queued":      queuedCount,
		"total_ok":    doneCount,
		"total_fail":  failCount,
		"current_job": currentJob,
		"cpu_count":   runtime.NumCPU(),
		"pause":       c.Pause,
		"logs":        logs,
	})
}

func handleJobs(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	jobs := listJobs(status)
	writeJSON(w, map[string]interface{}{"jobs": jobs, "count": len(jobs)})
}

func handleJobAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST only", 405)
		return
	}
	var body struct {
		Path   string `json:"path"`
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	jobRegMu.Lock()
	j, exists := jobReg[body.Path]
	jobRegMu.Unlock()
	if !exists {
		writeJSON(w, map[string]interface{}{"ok": false, "error": "任务不存在"})
		return
	}

	switch body.Action {
	case "cancel":
		if j.Status == "running" {
			cancelledSet.Store(body.Path, true)
			procMu.Lock()
			cmd, ok := procMap[body.Path]
			procMu.Unlock()
			if ok && cmd != nil && cmd.Process != nil {
				cmd.Process.Kill()
			}
		} else if j.Status == "queued" {
			updateJob(body.Path, "cancelled", "用户取消")
			addLog("INFO", fmt.Sprintf("从队列移除: %s", j.Name))
		}
	case "remove":
		removeJob(body.Path)
		addLog("INFO", fmt.Sprintf("从列表删除: %s", j.Name))
	case "retry":
		if j.Status != "failed" {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "只能重试失败的任务"})
			return
		}
		c := getCfg()
		srcPath := filepath.Join(c.ArchiveDir, "failed", j.Name)
		destPath := filepath.Join(c.WatchDir, j.Name)
		if err := moveFile(srcPath, destPath); err != nil {
			writeJSON(w, map[string]interface{}{"ok": false, "error": "移动文件失败: " + err.Error()})
			return
		}
		removeJob(body.Path)
		touchActivity()
		addLog("INFO", fmt.Sprintf("重试: %s 已放回监控目录", j.Name))
		cfgCopy := getCfgCopy()
		if cfgCopy.Pause {
			cfgCopy.Pause = false
			saveConfig(&cfgCopy)
			setCfg(&cfgCopy)
			addLog("INFO", "重试触发，自动恢复监控")
		}
		go scanAndEnqueue()
	default:
		writeJSON(w, map[string]interface{}{"ok": false, "error": "未知操作"})
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "POST only", 405); return }
	scanAndEnqueue()
	writeJSON(w, map[string]bool{"ok": true})
}

func handlePause(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "POST only", 405); return }
	c := getCfgCopy()
	c.Pause = !c.Pause
	saveConfig(&c)
	setCfg(&c)
	if c.Pause {
		addLog("WARN", "自动监控已暂停")
	} else {
		touchActivity()
		addLog("INFO", "自动监控已恢复")
	}
	writeJSON(w, map[string]bool{"ok": true, "pause": c.Pause})
}

func handleTestPath(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, map[string]string{"error": "path parameter required"})
		return
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		writeJSON(w, map[string]string{"path": path, "error": err.Error()})
		return
	}
	files := []string{}
	dirs := []string{}
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		} else {
			files = append(files, e.Name())
		}
	}
	writeJSON(w, map[string]interface{}{"path": path, "dirs": dirs, "files": files, "file_count": len(files)})
}

func main() {
	touchActivity()

	socketPath := flag.String("socket", "", "unix socket path")
	workDir := flag.String("workdir", "", "data directory")
	port := flag.Int("port", 8080, "http port (when no socket)")
	flag.Parse()

	if *workDir != "" {
		os.MkdirAll(*workDir, 0755)
		os.Chdir(*workDir)
	}

	cfgPath = filepath.Join(*workDir, "config.json")
	if *workDir == "" {
		cfgPath = "config.json"
	}
	backupCfgPath = "/etc/autounpack/config.json"
	cfg = loadConfig()

	startWorkers(cfg.MaxWorkers)
	go scannerLoop()
	addLog("INFO", fmt.Sprintf("AutoUnpack v%s 启动", version))

	mux := http.NewServeMux()
	mux.HandleFunc(gwPrefix+"/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			handleGetConfig(w, r)
		} else if r.Method == "POST" {
			handleSaveConfig(w, r)
		}
	})
	mux.HandleFunc(gwPrefix+"/api/status", handleStatus)
	mux.HandleFunc(gwPrefix+"/api/jobs", handleJobs)
	mux.HandleFunc(gwPrefix+"/api/job", handleJobAction)
	mux.HandleFunc(gwPrefix+"/api/scan", handleScan)
	mux.HandleFunc(gwPrefix+"/api/pause", handlePause)
	mux.HandleFunc(gwPrefix+"/api/testpath", handleTestPath)
	mux.HandleFunc(gwPrefix+"/", handleIndex)

	if *socketPath != "" {
		os.Remove(*socketPath)
		l, err := net.Listen("unix", *socketPath)
		if err != nil {
			log.Fatalf("listen unix %s: %v", *socketPath, err)
		}
		addLog("INFO", fmt.Sprintf("listening on unix:%s", *socketPath))
		if err := http.Serve(l, mux); err != nil {
			log.Fatalf("http serve: %v", err)
		}
	} else {
		addLog("INFO", fmt.Sprintf("listening on http://0.0.0.0:%d", *port))
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), mux); err != nil {
			log.Fatalf("http listen: %v", err)
		}
	}
}
