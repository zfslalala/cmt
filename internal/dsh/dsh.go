// Package dsh 通过 DSH Desktop 的 qg-dsh-open 桥接插件创建并打开工作区。
package dsh

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const bridgeVersion = 1

// ErrBridgeUnavailable 表示没有找到 DSH 桥接插件，或插件未在运行。
var ErrBridgeUnavailable = errors.New("未检测到 DSH 工作区桥接插件 qg-dsh-open")

// Result 是一次打开工作区请求的结果。
type Result struct {
	WorkspaceID string
	Created     bool
}

// RemoveResult 是一次删除工作区请求的结果。
type RemoveResult struct {
	Removed     bool
	WorkspaceID string
}

type bridgeInfo struct {
	Version int    `json:"version"`
	Port    int    `json:"port"`
	Token   string `json:"token"`
	PID     int    `json:"pid"`
}

func bridgePath() (string, error) {
	if path := os.Getenv("QG_DSH_BRIDGE"); path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户目录失败: %w", err)
	}
	return filepath.Join(home, ".qg", "dsh-bridge.json"), nil
}

func readBridge() (*bridgeInfo, error) {
	path, err := bridgePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var info bridgeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("解析 DSH 桥接文件失败: %w", err)
	}
	if info.Version != bridgeVersion || info.Port <= 0 || info.Port > 65535 || info.Token == "" {
		return nil, fmt.Errorf("DSH 桥接文件格式不支持，请更新 qg-dsh-open 插件")
	}
	return &info, nil
}

// OpenWorkspace 在 DSH Desktop 中创建/复用 dir 对应的工作区并打开它。
// DSH 未运行时先启动 Desktop，再等待桥接插件就绪。
func OpenWorkspace(dir string) (*Result, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("解析目录失败: %w", err)
	}
	if st, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("工作树目录不可用: %w", err)
	} else if !st.IsDir() {
		return nil, fmt.Errorf("工作树路径不是目录: %s", dir)
	}

	info, err := readBridge()
	if err != nil {
		if launchErr := launchDSH(); launchErr != nil {
			return nil, fmt.Errorf("%w；另外启动 DeepSeek Harness 失败: %v", ErrBridgeUnavailable, launchErr)
		}
		info, err = waitForBridge(15 * time.Second)
		if err != nil {
			return nil, fmt.Errorf("%w。请确认 DeepSeek Harness 已安装 dsh-plugin 并重启", ErrBridgeUnavailable)
		}
	}

	result, err := callOpen(info, dir)
	if err == nil {
		return result, nil
	}

	// 桥接文件可能来自已退出的宿主（端口/令牌已失效）。聚焦 DSH 后
	// 等一次新的桥接文件，再重试一次。
	var rejected *rejectedError
	if errors.As(err, &rejected) {
		return nil, err
	}
	if launchErr := launchDSH(); launchErr != nil {
		return nil, err
	}
	nextInfo, waitErr := waitForChangedBridge(info, 10*time.Second)
	if waitErr != nil {
		return nil, err
	}
	return callOpen(nextInfo, dir)
}

func waitForBridge(timeout time.Duration) (*bridgeInfo, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		info, err := readBridge()
		if err == nil {
			return info, nil
		}
		lastErr = err
		if time.Now().After(deadline) {
			return nil, lastErr
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// waitForChangedBridge 等待持有新令牌/端口的宿主重新发布桥接文件。
// 旧宿主崩溃后桥接文件可能残留，不能直接复用。
func waitForChangedBridge(previous *bridgeInfo, timeout time.Duration) (*bridgeInfo, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		info, err := readBridge()
		if err == nil && (info.Token != previous.Token || info.Port != previous.Port) {
			return info, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("DSH 桥接未更新，请重启 DeepSeek Harness 后重试")
		}
		if time.Now().After(deadline) {
			return nil, lastErr
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func callOpen(info *bridgeInfo, dir string) (*Result, error) {
	body, err := json.Marshal(map[string]string{"path": dir})
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/open", info.Port)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+info.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("连接 DSH 桥接失败: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		OK          bool   `json:"ok"`
		WorkspaceID string `json:"workspaceId"`
		Created     bool   `json:"created"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&out); err != nil {
		return nil, fmt.Errorf("读取 DSH 桥接响应失败: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("DSH 桥接令牌已失效，请重启 DeepSeek Harness 后重试")
	}
	if resp.StatusCode != http.StatusOK || !out.OK {
		if out.Error == "" {
			out.Error = resp.Status
		}
		return nil, &rejectedError{message: out.Error}
	}
	return &Result{WorkspaceID: out.WorkspaceID, Created: out.Created}, nil
}

// rejectedError 表示宿主明确拒绝了请求（路径不合法、插件内部错误等），
// 重试没有意义。
type rejectedError struct {
	message string
}

func (e *rejectedError) Error() string {
	return "DSH 桥接返回错误: " + e.message
}

// RemoveWorkspace 删除 dir 对应的 DSH 工作区。
// 桥接文件不存在表示 DSH 没有在运行（或插件未安装），此时静默跳过；
// 该函数不会为了删除而启动 DSH。
func RemoveWorkspace(dir string) (RemoveResult, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return RemoveResult{}, fmt.Errorf("解析目录失败: %w", err)
	}
	info, err := readBridge()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return RemoveResult{}, nil
		}
		return RemoveResult{}, err
	}
	return callRemove(info, dir)
}

func callRemove(info *bridgeInfo, dir string) (RemoveResult, error) {
	body, err := json.Marshal(map[string]string{"path": dir})
	if err != nil {
		return RemoveResult{}, err
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/remove", info.Port)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return RemoveResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+info.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return RemoveResult{}, fmt.Errorf("连接 DSH 桥接失败: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		OK          bool   `json:"ok"`
		Removed     bool   `json:"removed"`
		WorkspaceID string `json:"workspaceId"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&out); err != nil {
		return RemoveResult{}, fmt.Errorf("读取 DSH 桥接响应失败: %w", err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return RemoveResult{}, errors.New("DSH 桥接令牌已失效，请重启 DeepSeek Harness 后重试")
	}
	if resp.StatusCode != http.StatusOK || !out.OK {
		if out.Error == "" {
			out.Error = resp.Status
		}
		return RemoveResult{}, &rejectedError{message: out.Error}
	}
	return RemoveResult{Removed: out.Removed, WorkspaceID: out.WorkspaceID}, nil
}

func launchDSH() error {
	if os.Getenv("QG_DSH_NO_LAUNCH") == "1" {
		return nil
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-a", "DeepSeek Harness")
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", "dsh://open")
	default:
		cmd = exec.Command("xdg-open", "dsh://open")
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("无法启动 DeepSeek Harness: %w", err)
	}
	return nil
}
