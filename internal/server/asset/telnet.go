package asset

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	assetbiz "github.com/ydcloud-dy/mom/internal/biz/asset"
	appLogger "github.com/ydcloud-dy/mom/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

// Telnet IAC 常量
const (
	telnetIAC  = 255 // Interpret As Command
	telnetDONT = 254 // 拒绝对方请求
	telnetDO   = 253 // 请求对方执行
	telnetWONT = 252 // 拒绝执行
	telnetWILL = 251 // 同意执行
	telnetSB   = 250 // 子协商开始
	telnetSE   = 240 // 子协商结束
	telnetNAWS = 31  // 窗口大小协商
	telnetECHO = 1   // 回显
	telnetSGA  = 3   // 抑制继续进行
	telnetTTYP = 24  // 终端类型
)

// NetworkTerminalManager 网络设备终端管理器
type NetworkTerminalManager struct {
	sessions        map[string]*NetworkTerminalSession
	mu              sync.RWMutex
	deviceUseCase   *assetbiz.NetworkDeviceUseCase
	db              *gorm.DB
}

// NetworkTerminalSession 网络设备终端会话
type NetworkTerminalSession struct {
	ID         string
	DeviceID   uint
	DeviceName string
	DeviceIP   string
	Protocol   string
	UserID     uint
	Username   string
	Conn       net.Conn       // Telnet TCP 连接
	SSHClient  *ssh.Client    // SSH 连接
	SSHSession *ssh.Session   // SSH 会话
	StdinPipe  interface{}    // stdin 管道 (io.WriteCloser for SSH)
	Recorder   *AsciinemaRecorder
	CreatedAt  time.Time
}

// NewNetworkTerminalManager 创建网络设备终端管理器
func NewNetworkTerminalManager(deviceUseCase *assetbiz.NetworkDeviceUseCase, db *gorm.DB) *NetworkTerminalManager {
	return &NetworkTerminalManager{
		sessions:      make(map[string]*NetworkTerminalSession),
		deviceUseCase: deviceUseCase,
		db:            db,
	}
}

// HandleNetworkTerminalConnection 处理网络设备终端 WebSocket 连接
func (ntm *NetworkTerminalManager) HandleNetworkTerminalConnection(c *gin.Context) {
	deviceIDStr := c.Param("id")
	deviceID, err := strconv.Atoi(deviceIDStr)
	if err != nil {
		appLogger.Error("无效的设备ID", zap.String("deviceId", deviceIDStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的设备ID"})
		return
	}

	// 获取用户信息
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")
	var uid uint
	var uname string = "unknown"
	if userID != nil {
		if id, ok := userID.(uint); ok {
			uid = id
		} else if id, ok := userID.(float64); ok {
			uid = uint(id)
		}
	}
	if username != nil {
		if name, ok := username.(string); ok {
			uname = name
		}
	}

	// 获取设备及凭证
	device, credential, err := ntm.deviceUseCase.GetByIDForConnection(c.Request.Context(), uint(deviceID))
	if err != nil {
		appLogger.Error("获取设备信息失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取终端尺寸
	colsStr := c.DefaultQuery("cols", "80")
	rowsStr := c.DefaultQuery("rows", "24")
	cols, _ := strconv.ParseUint(colsStr, 10, 16)
	rows, _ := strconv.ParseUint(rowsStr, 10, 16)
	if cols == 0 {
		cols = 80
	}
	if rows == 0 {
		rows = 24
	}

	// 升级到 WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		appLogger.Error("WebSocket升级失败", zap.Error(err))
		return
	}
	defer conn.Close()

	appLogger.Info("网络设备终端连接",
		zap.Uint("deviceId", device.ID),
		zap.String("protocol", device.Protocol),
		zap.String("ip", device.IP),
		zap.Int("port", device.Port))

	// 根据协议类型连接
	switch device.Protocol {
	case "telnet":
		ntm.handleTelnetSession(conn, device, credential, uid, uname, uint16(cols), uint16(rows))
	case "ssh":
		ntm.handleSSHSession(conn, device, credential, uid, uname, uint16(cols), uint16(rows))
	default:
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("不支持的协议: %s\r\n", device.Protocol)))
	}
}

// handleTelnetSession 处理 Telnet 会话
func (ntm *NetworkTerminalManager) handleTelnetSession(wsConn *websocket.Conn, device *assetbiz.NetworkDevice, credential *assetbiz.Credential, userID uint, username string, cols, rows uint16) {
	addr := fmt.Sprintf("%s:%d", device.IP, device.Port)

	// 建立 TCP 连接
	tcpConn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		_ = ntm.deviceUseCase.UpdateStatus(context.Background(), device.ID, 0)
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Telnet连接失败: %s\r\n", err.Error())))
		return
	}
	defer tcpConn.Close()

	// 连接成功，更新设备状态为在线
	_ = ntm.deviceUseCase.UpdateStatus(context.Background(), device.ID, 1)

	// 创建录制器
	recordingDir := "./data/terminal-recordings"
	recorder, err := NewAsciinemaRecorder(recordingDir, int(cols), int(rows))
	if err != nil {
		recorder = nil
	}

	sessionID := fmt.Sprintf("net-%d-%d", device.ID, time.Now().Unix())

	session := &NetworkTerminalSession{
		ID:         sessionID,
		DeviceID:   device.ID,
		DeviceName: device.Name,
		DeviceIP:   device.IP,
		Protocol:   "telnet",
		UserID:     userID,
		Username:   username,
		Conn:       tcpConn,
		Recorder:   recorder,
		CreatedAt:  time.Now(),
	}

	ntm.mu.Lock()
	ntm.sessions[sessionID] = session
	ntm.mu.Unlock()

	defer func() {
		ntm.closeNetworkSession(sessionID)
	}()

	appLogger.Info("Telnet会话已创建",
		zap.String("sessionID", sessionID),
		zap.String("addr", addr))

	// 设置连接超时
	tcpConn.SetReadDeadline(time.Now().Add(5 * time.Minute))

	var wg sync.WaitGroup
	wg.Add(1)

	// 从 Telnet 读取并发送到 WebSocket
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			tcpConn.SetReadDeadline(time.Now().Add(5 * time.Minute))
			n, err := tcpConn.Read(buf)
			if n > 0 {
				// 处理 Telnet IAC 命令，过滤并响应协商
				cleaned := ntm.processTelnetData(tcpConn, buf[:n], cols, rows)
				if len(cleaned) > 0 {
					if recorder != nil {
						recorder.RecordOutput(cleaned)
					}
					wsConn.WriteMessage(websocket.BinaryMessage, cleaned)
				}
			}
			if err != nil {
				appLogger.Info("Telnet连接关闭", zap.String("sessionID", sessionID), zap.Error(err))
				return
			}
		}
	}()

	// 从 WebSocket 读取并发送到 Telnet
	for {
		wsConn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		messageType, data, err := wsConn.ReadMessage()
		if err != nil {
			appLogger.Info("WebSocket连接关闭", zap.String("sessionID", sessionID), zap.Error(err))
			tcpConn.Close()
			break
		}

		if messageType == websocket.TextMessage {
			// 尝试解析 JSON（resize 命令）
			var msg map[string]interface{}
			if err := json.Unmarshal(data, &msg); err == nil {
				if msgType, ok := msg["type"].(string); ok && msgType == "resize" {
					newCols, colsOk := msg["cols"].(float64)
					newRows, rowsOk := msg["rows"].(float64)
					if colsOk && rowsOk {
						ntm.sendTelnetNAWS(tcpConn, uint16(newCols), uint16(newRows))
						continue
					}
				}
			}
			// 转换退格键: xterm.js 发送 DEL (0x7f)，Telnet 设备需要 BS (0x08)
			data = telnetFixBackspace(data)
			if recorder != nil {
				recorder.RecordInput(data)
			}
			tcpConn.Write(data)
		} else if messageType == websocket.BinaryMessage {
			data = telnetFixBackspace(data)
			if recorder != nil {
				recorder.RecordInput(data)
			}
			tcpConn.Write(data)
		}
	}

	wg.Wait()
	appLogger.Info("Telnet会话结束", zap.String("sessionID", sessionID))
}

// handleSSHSession 处理网络设备的 SSH 会话
func (ntm *NetworkTerminalManager) handleSSHSession(wsConn *websocket.Conn, device *assetbiz.NetworkDevice, credential *assetbiz.Credential, userID uint, username string, cols, rows uint16) {
	if credential == nil {
		wsConn.WriteMessage(websocket.TextMessage, []byte("未配置凭证\r\n"))
		return
	}

	// 构建 SSH 认证
	var authMethods []ssh.AuthMethod
	var err error
	switch credential.Type {
	case "password":
		authMethods = append(authMethods, ssh.Password(credential.Password))
	case "key", "private_key":
		var signer ssh.Signer
		if credential.Passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(credential.PrivateKey), []byte(credential.Passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(credential.PrivateKey))
		}
		if err != nil {
			wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("解析私钥失败: %s\r\n", err.Error())))
			return
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	default:
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("不支持的凭证类型: %s\r\n", credential.Type)))
		return
	}

	config := &ssh.ClientConfig{
		Config: ssh.Config{
			// 兼容旧设备的密钥交换算法（如 Cisco 2960 等老型号）
			KeyExchanges: []string{
				"curve25519-sha256",
				"curve25519-sha256@libssh.org",
				"ecdh-sha2-nistp256",
				"ecdh-sha2-nistp384",
				"ecdh-sha2-nistp521",
				"diffie-hellman-group14-sha256",
				"diffie-hellman-group14-sha1",
				"diffie-hellman-group1-sha1", // 旧设备兼容
			},
			Ciphers: []string{
				"aes128-gcm@openssh.com",
				"aes256-gcm@openssh.com",
				"chacha20-poly1305@openssh.com",
				"aes128-ctr",
				"aes192-ctr",
				"aes256-ctr",
				"aes128-cbc", // 旧设备兼容
				"aes192-cbc",
				"aes256-cbc",
				"3des-cbc",
			},
		},
		User:            credential.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", device.IP, device.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		_ = ntm.deviceUseCase.UpdateStatus(context.Background(), device.ID, 0)
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("SSH连接失败: %s\r\n", err.Error())))
		return
	}
	defer client.Close()

	// 连接成功，更新设备状态为在线
	_ = ntm.deviceUseCase.UpdateStatus(context.Background(), device.ID, 1)

	sshSession, err := client.NewSession()
	if err != nil {
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("创建SSH会话失败: %s\r\n", err.Error())))
		return
	}
	defer sshSession.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	if err := sshSession.RequestPty("xterm-256color", int(rows), int(cols), modes); err != nil {
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("请求伪终端失败: %s\r\n", err.Error())))
		return
	}

	stdinPipe, err := sshSession.StdinPipe()
	if err != nil {
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("获取stdin管道失败: %s\r\n", err.Error())))
		return
	}

	stdoutPipe, err := sshSession.StdoutPipe()
	if err != nil {
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("获取stdout管道失败: %s\r\n", err.Error())))
		return
	}

	if err := sshSession.Shell(); err != nil {
		wsConn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("启动shell失败: %s\r\n", err.Error())))
		return
	}

	// 创建录制器
	recordingDir := "./data/terminal-recordings"
	recorder, err := NewAsciinemaRecorder(recordingDir, int(cols), int(rows))
	if err != nil {
		recorder = nil
	}

	sessionID := fmt.Sprintf("net-%d-%d", device.ID, time.Now().Unix())

	session := &NetworkTerminalSession{
		ID:         sessionID,
		DeviceID:   device.ID,
		DeviceName: device.Name,
		DeviceIP:   device.IP,
		Protocol:   "ssh",
		UserID:     userID,
		Username:   username,
		SSHClient:  client,
		SSHSession: sshSession,
		StdinPipe:  stdinPipe,
		Recorder:   recorder,
		CreatedAt:  time.Now(),
	}

	ntm.mu.Lock()
	ntm.sessions[sessionID] = session
	ntm.mu.Unlock()

	defer func() {
		ntm.closeNetworkSession(sessionID)
	}()

	appLogger.Info("网络设备SSH会话已创建", zap.String("sessionID", sessionID))

	var wg sync.WaitGroup
	wg.Add(1)

	// 从 SSH 读取并发送到 WebSocket
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				if recorder != nil {
					recorder.RecordOutput(buf[:n])
				}
				wsConn.WriteMessage(websocket.BinaryMessage, buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	// 从 WebSocket 读取并发送到 SSH
	for {
		wsConn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		messageType, data, err := wsConn.ReadMessage()
		if err != nil {
			appLogger.Info("WebSocket连接关闭", zap.String("sessionID", sessionID), zap.Error(err))
			sshSession.Close()
			client.Close()
			break
		}

		if messageType == websocket.TextMessage {
			var msg map[string]interface{}
			if err := json.Unmarshal(data, &msg); err == nil {
				if msgType, ok := msg["type"].(string); ok && msgType == "resize" {
					newCols, colsOk := msg["cols"].(float64)
					newRows, rowsOk := msg["rows"].(float64)
					if colsOk && rowsOk {
						sshSession.WindowChange(int(newRows), int(newCols))
						continue
					}
				}
			}
			if recorder != nil {
				recorder.RecordInput(data)
			}
			stdinPipe.Write(data)
		} else if messageType == websocket.BinaryMessage {
			if recorder != nil {
				recorder.RecordInput(data)
			}
			stdinPipe.Write(data)
		}
	}

	wg.Wait()
	appLogger.Info("网络设备SSH会话结束", zap.String("sessionID", sessionID))
}

// closeNetworkSession 关闭网络设备终端会话
func (ntm *NetworkTerminalManager) closeNetworkSession(sessionID string) {
	ntm.mu.Lock()
	defer ntm.mu.Unlock()

	session, ok := ntm.sessions[sessionID]
	if !ok {
		return
	}

	// 关闭录制器并保存
	if session.Recorder != nil {
		session.Recorder.Close()

		duration := session.Recorder.GetDuration()
		fileSize := session.Recorder.GetFileSize()
		recordingPath := session.Recorder.GetRecordingPath()

		terminalSession := &assetbiz.TerminalSession{
			SessionType:   session.Protocol,
			HostID:        session.DeviceID,
			HostName:      session.DeviceName,
			HostIP:        session.DeviceIP,
			UserID:        session.UserID,
			Username:      session.Username,
			RecordingPath: recordingPath,
			Duration:      duration,
			FileSize:      fileSize,
			Status:        "completed",
		}

		if err := ntm.db.Create(terminalSession).Error; err != nil {
			appLogger.Error("保存网络设备终端会话记录失败", zap.Error(err))
		}
	}

	// 关闭连接
	if session.Conn != nil {
		session.Conn.Close()
	}
	if session.SSHSession != nil {
		session.SSHSession.Close()
	}
	if session.SSHClient != nil {
		session.SSHClient.Close()
	}

	delete(ntm.sessions, sessionID)
	appLogger.Info("网络设备终端会话已关闭", zap.String("sessionID", sessionID))
}

// telnetFixBackspace 转换退格键: xterm.js 发送 DEL(0x7f)，但 Telnet 设备通常需要 BS(0x08)
func telnetFixBackspace(data []byte) []byte {
	result := make([]byte, len(data))
	copy(result, data)
	for i, b := range result {
		if b == 0x7f {
			result[i] = 0x08
		}
	}
	return result
}

// processTelnetData 处理 Telnet IAC 协商数据，返回过滤后的纯数据
func (ntm *NetworkTerminalManager) processTelnetData(conn net.Conn, data []byte, cols, rows uint16) []byte {
	var cleaned []byte
	i := 0
	for i < len(data) {
		if data[i] == telnetIAC && i+1 < len(data) {
			cmd := data[i+1]
			switch cmd {
			case telnetDO:
				if i+2 < len(data) {
					option := data[i+2]
					ntm.handleTelnetDO(conn, option, cols, rows)
					i += 3
				} else {
					i += 2
				}
			case telnetDONT:
				if i+2 < len(data) {
					option := data[i+2]
					// 响应 WONT
					conn.Write([]byte{telnetIAC, telnetWONT, option})
					i += 3
				} else {
					i += 2
				}
			case telnetWILL:
				if i+2 < len(data) {
					option := data[i+2]
					ntm.handleTelnetWILL(conn, option)
					i += 3
				} else {
					i += 2
				}
			case telnetWONT:
				if i+2 < len(data) {
					// 对方拒绝，跳过
					i += 3
				} else {
					i += 2
				}
			case telnetSB:
				// 子协商，找到 SE 结束
				end := i + 2
				for end < len(data)-1 {
					if data[end] == telnetIAC && data[end+1] == telnetSE {
						end += 2
						break
					}
					end++
				}
				i = end
			case telnetIAC:
				// 转义的 255 字节
				cleaned = append(cleaned, 255)
				i += 2
			default:
				i += 2
			}
		} else {
			cleaned = append(cleaned, data[i])
			i++
		}
	}
	return cleaned
}

// handleTelnetDO 处理 Telnet DO 请求
func (ntm *NetworkTerminalManager) handleTelnetDO(conn net.Conn, option byte, cols, rows uint16) {
	switch option {
	case telnetNAWS:
		// 同意 NAWS 并发送窗口大小
		conn.Write([]byte{telnetIAC, telnetWILL, telnetNAWS})
		ntm.sendTelnetNAWS(conn, cols, rows)
	case telnetTTYP:
		// 同意终端类型并发送
		conn.Write([]byte{telnetIAC, telnetWILL, telnetTTYP})
		// 发送终端类型子协商
		termType := "xterm-256color"
		sb := []byte{telnetIAC, telnetSB, telnetTTYP, 0} // 0 = IS
		sb = append(sb, []byte(termType)...)
		sb = append(sb, telnetIAC, telnetSE)
		conn.Write(sb)
	case telnetSGA:
		conn.Write([]byte{telnetIAC, telnetWILL, telnetSGA})
	default:
		// 默认拒绝
		conn.Write([]byte{telnetIAC, telnetWONT, option})
	}
}

// handleTelnetWILL 处理 Telnet WILL 请求
func (ntm *NetworkTerminalManager) handleTelnetWILL(conn net.Conn, option byte) {
	switch option {
	case telnetECHO, telnetSGA:
		conn.Write([]byte{telnetIAC, telnetDO, option})
	default:
		conn.Write([]byte{telnetIAC, telnetDONT, option})
	}
}

// sendTelnetNAWS 发送 Telnet 窗口大小
func (ntm *NetworkTerminalManager) sendTelnetNAWS(conn net.Conn, cols, rows uint16) {
	sb := []byte{
		telnetIAC, telnetSB, telnetNAWS,
		byte(cols >> 8), byte(cols & 0xff),
		byte(rows >> 8), byte(rows & 0xff),
		telnetIAC, telnetSE,
	}
	conn.Write(sb)
}
