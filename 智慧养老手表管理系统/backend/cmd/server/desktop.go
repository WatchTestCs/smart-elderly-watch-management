// 桌面端（Electron）内嵌模式协议：与 Java 版 DesktopReady 对齐。
//
// main.cjs 以环境变量启动后端：
//
//	SUINING_PORT / SUINING_BIND_ADDRESS  监听地址（PORT=0 表示随机端口）
//	SUINING_READY_FILE                   就绪后原子写入 {"port":N,"pid":M}
//	SUINING_PARENT_PID                   父进程退出时后端自行退出
//	SUINING_DATA_DIR                     数据目录（图片默认落到其下 images/）
//	SUINING_DESKTOP_TOKEN                 桌面端 bootstrap token（= DESKTOP_TOKEN）
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// applyDesktopEnv 把桌面端环境变量折叠进监听地址与图片目录，返回就绪文件与父 PID。
func applyDesktopEnv(addr, imageDir string) (newAddr, newImageDir, readyFile string, parentPid int) {
	newAddr, newImageDir = addr, imageDir
	if p := os.Getenv("SUINING_PORT"); p != "" {
		bind := os.Getenv("SUINING_BIND_ADDRESS")
		if bind == "" {
			bind = "127.0.0.1"
		}
		newAddr = bind + ":" + p
	}
	if d := os.Getenv("SUINING_DATA_DIR"); d != "" && os.Getenv("IMAGE_DIR") == "" {
		newImageDir = filepath.Join(d, "images")
	}
	readyFile = os.Getenv("SUINING_READY_FILE")
	if s := os.Getenv("SUINING_PARENT_PID"); s != "" {
		fmt.Sscanf(s, "%d", &parentPid)
	}
	return newAddr, newImageDir, readyFile, parentPid
}

// writeReadyFile 原子写入就绪文件（同目录临时文件 + rename），格式与 Java 端一致。
func writeReadyFile(path string, port int) {
	if path == "" {
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("readyfile: mkdir %s: %v", dir, err)
		return
	}
	tmp, err := os.CreateTemp(dir, "ready-*.tmp")
	if err != nil {
		log.Printf("readyfile: %v", err)
		return
	}
	tmpName := tmp.Name()
	fmt.Fprintf(tmp, "{\"port\":%d,\"pid\":%d}", port, os.Getpid())
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		log.Printf("readyfile: %v", err)
		return
	}
	if err := os.Rename(tmpName, path); err != nil {
		os.Remove(tmpName)
		log.Printf("readyfile: %v", err)
	}
}

// watchParent 父进程退出时关闭服务器（对齐 Java DesktopReady 的 daemon 看护线程）。
func watchParent(parentPid int, shutdown func()) {
	if parentPid <= 0 || parentPid == os.Getpid() {
		return
	}
	go func() {
		t := time.NewTicker(2 * time.Second)
		defer t.Stop()
		for range t.C {
			if !parentAlive(parentPid) {
				log.Printf("desktop parent %d exited, shutting down", parentPid)
				shutdown()
				return
			}
		}
	}()
}
