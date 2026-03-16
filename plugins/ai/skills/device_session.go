package skills

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	aiDeviceSessionIdleTimeout = 10 * time.Minute
	aiDeviceReadQuietWindow    = 1200 * time.Millisecond
	aiDeviceFirstByteTimeout   = 8 * time.Second
	aiDeviceCommandTimeout     = 20 * time.Second

	telnetIAC  = 255
	telnetDONT = 254
	telnetDO   = 253
	telnetWONT = 252
	telnetWILL = 251
	telnetSB   = 250
	telnetSE   = 240
	telnetNAWS = 31
	telnetECHO = 1
	telnetSGA  = 3
	telnetTTYP = 24
)

var (
	devicePromptLinePattern = regexp.MustCompile(`(?m)[^\n\r]{0,200}(?:>|#|\]|\$|%)\s*$`)
	ansiEscapePattern       = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	aiDeviceShellSessions   = newAIDeviceShellSessionManager()
)

type aiDeviceTarget struct {
	ID       uint
	Name     string
	IP       string
	Protocol string
	Port     int
}

type aiDeviceShellSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*aiDeviceShellSession
}

type aiDeviceShellSession struct {
	key        string
	deviceID   uint
	deviceName string
	deviceIP   string
	chatID     uint
	protocol   string
	conn       net.Conn
	client     *ssh.Client
	session    *ssh.Session
	stdin      io.WriteCloser
	outputCh   chan []byte
	errCh      chan error
	closed     chan struct{}
	closeOnce  sync.Once
	lastActive time.Time
	mu         sync.Mutex
	activityMu sync.RWMutex
}

func newAIDeviceShellSessionManager() *aiDeviceShellSessionManager {
	manager := &aiDeviceShellSessionManager{
		sessions: make(map[string]*aiDeviceShellSession),
	}
	go manager.cleanupLoop()
	return manager
}

func (m *aiDeviceShellSessionManager) ExecuteCommand(chatID uint, device aiDeviceTarget, username string, password string, command string) (string, bool, error) {
	if chatID == 0 {
		return "", false, fmt.Errorf("缺少 AI 会话 ID，无法创建交互式设备会话")
	}

	session, reused, err := m.acquire(chatID, device, username, password)
	if err != nil {
		return "", reused, err
	}

	output, err := session.execute(command)
	if err != nil {
		return output, reused, err
	}
	return output, reused, nil
}

func (m *aiDeviceShellSessionManager) HasActiveSession(chatID uint, deviceID uint) bool {
	if chatID == 0 {
		return false
	}
	key := fmt.Sprintf("%d:%d", chatID, deviceID)
	m.mu.RLock()
	session := m.sessions[key]
	m.mu.RUnlock()
	return session != nil && !session.isClosed()
}

func (m *aiDeviceShellSessionManager) ListSessions(chatID uint, deviceIDs map[uint]bool) []map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]map[string]any, 0, len(m.sessions))
	now := time.Now()
	for _, session := range m.sessions {
		if session.chatID != chatID || session.isClosed() {
			continue
		}
		if len(deviceIDs) > 0 && !deviceIDs[session.deviceID] {
			continue
		}
		lastActive := session.getLastActive()
		result = append(result, map[string]any{
			"deviceId":                  session.deviceID,
			"deviceName":                session.deviceName,
			"deviceIP":                  session.deviceIP,
			"protocol":                  session.protocol,
			"lastActiveAt":              lastActive,
			"idleSeconds":               int(now.Sub(lastActive).Seconds()),
			"sessionIdleTimeoutSeconds": int(aiDeviceSessionIdleTimeout.Seconds()),
		})
	}
	return result
}

func (m *aiDeviceShellSessionManager) CloseSessions(chatID uint, deviceIDs map[uint]bool) int {
	m.mu.RLock()
	targets := make([]*aiDeviceShellSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		if session.chatID != chatID || session.isClosed() {
			continue
		}
		if len(deviceIDs) > 0 && !deviceIDs[session.deviceID] {
			continue
		}
		targets = append(targets, session)
	}
	m.mu.RUnlock()

	closedCount := 0
	for _, session := range targets {
		session.close()
		m.remove(session.key)
		closedCount++
	}
	return closedCount
}

func (m *aiDeviceShellSessionManager) acquire(chatID uint, device aiDeviceTarget, username string, password string) (*aiDeviceShellSession, bool, error) {
	key := fmt.Sprintf("%d:%d", chatID, device.ID)

	m.mu.RLock()
	existing := m.sessions[key]
	m.mu.RUnlock()
	if existing != nil && !existing.isClosed() {
		existing.touch()
		return existing, true, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing = m.sessions[key]; existing != nil && !existing.isClosed() {
		existing.touch()
		return existing, true, nil
	}

	session, err := newAIDeviceShellSession(key, chatID, device, username, password)
	if err != nil {
		return nil, false, err
	}
	m.sessions[key] = session
	return session, false, nil
}

func (m *aiDeviceShellSessionManager) remove(key string) {
	m.mu.Lock()
	delete(m.sessions, key)
	m.mu.Unlock()
}

func (m *aiDeviceShellSessionManager) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		var stale []*aiDeviceShellSession

		m.mu.RLock()
		for _, session := range m.sessions {
			if now.Sub(session.getLastActive()) > aiDeviceSessionIdleTimeout {
				stale = append(stale, session)
			}
		}
		m.mu.RUnlock()

		for _, session := range stale {
			session.close()
			m.remove(session.key)
		}
	}
}

func newAIDeviceShellSession(key string, chatID uint, device aiDeviceTarget, username string, password string) (*aiDeviceShellSession, error) {
	if device.Protocol == "telnet" {
		return newAITelnetDeviceShellSession(key, chatID, device, username, password)
	}

	addr := fmt.Sprintf("%s:%d", device.IP, device.Port)
	client, err := ssh.Dial("tcp", addr, buildDeviceSSHConfig(username, password))
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %v", err)
	}

	sshSession, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("创建会话失败: %v", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sshSession.RequestPty("xterm", 24, 160, modes); err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("请求伪终端失败: %v", err)
	}

	stdinPipe, err := sshSession.StdinPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("获取 stdin 管道失败: %v", err)
	}

	stdoutPipe, err := sshSession.StdoutPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("获取 stdout 管道失败: %v", err)
	}

	if err := sshSession.Shell(); err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("启动交互式 shell 失败: %v", err)
	}

	session := &aiDeviceShellSession{
		key:        key,
		deviceID:   device.ID,
		deviceName: device.Name,
		deviceIP:   device.IP,
		chatID:     chatID,
		protocol:   "ssh",
		client:     client,
		session:    sshSession,
		stdin:      stdinPipe,
		outputCh:   make(chan []byte, 256),
		errCh:      make(chan error, 1),
		closed:     make(chan struct{}),
		lastActive: time.Now(),
	}

	go session.readLoop(stdoutPipe)

	_, _ = io.WriteString(stdinPipe, "\n")
	_, _ = session.collectOutput(aiDeviceFirstByteTimeout, 800*time.Millisecond, 10*time.Second)

	return session, nil
}

func newAITelnetDeviceShellSession(key string, chatID uint, device aiDeviceTarget, username string, password string) (*aiDeviceShellSession, error) {
	addr := fmt.Sprintf("%s:%d", device.IP, device.Port)
	conn, err := net.DialTimeout("tcp", addr, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("Telnet连接失败: %v", err)
	}

	session := &aiDeviceShellSession{
		key:        key,
		deviceID:   device.ID,
		deviceName: device.Name,
		deviceIP:   device.IP,
		chatID:     chatID,
		protocol:   "telnet",
		conn:       conn,
		outputCh:   make(chan []byte, 256),
		errCh:      make(chan error, 1),
		closed:     make(chan struct{}),
		lastActive: time.Now(),
	}

	go session.readTelnetLoop(conn)

	if err := session.performTelnetLogin(username, password); err != nil {
		session.close()
		return nil, err
	}

	return session, nil
}

func (s *aiDeviceShellSession) readLoop(stdout io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := stdout.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			select {
			case s.outputCh <- chunk:
			case <-s.closed:
				return
			}
		}
		if err != nil {
			select {
			case s.errCh <- err:
			default:
			}
			s.close()
			return
		}
	}
}

func (s *aiDeviceShellSession) readTelnetLoop(conn net.Conn) {
	buf := make([]byte, 4096)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(aiDeviceSessionIdleTimeout))
		n, err := conn.Read(buf)
		if n > 0 {
			cleaned := processAITelnetData(conn, buf[:n])
			if len(cleaned) > 0 {
				chunk := make([]byte, len(cleaned))
				copy(chunk, cleaned)
				select {
				case s.outputCh <- chunk:
				case <-s.closed:
					return
				}
			}
		}
		if err != nil {
			select {
			case s.errCh <- err:
			default:
			}
			s.close()
			return
		}
	}
}

func (s *aiDeviceShellSession) performTelnetLogin(username string, password string) error {
	sentUsername := false
	sentPassword := false
	deadline := time.Now().Add(15 * time.Second)

	for time.Now().Before(deadline) {
		output, err := s.collectOutput(2*time.Second, 400*time.Millisecond, 3*time.Second)
		normalized := strings.ToLower(normalizeInteractiveOutput(output))
		if devicePromptLinePattern.MatchString(normalized) {
			s.touch()
			return nil
		}

		if !sentUsername && (strings.Contains(normalized, "username:") || strings.Contains(normalized, "login:") || strings.Contains(normalized, "user name:")) {
			if err := s.writeInput([]byte(username + "\n")); err != nil {
				return fmt.Errorf("发送 Telnet 用户名失败: %v", err)
			}
			sentUsername = true
			continue
		}

		if !sentPassword && strings.Contains(normalized, "password:") {
			if err := s.writeInput([]byte(password + "\n")); err != nil {
				return fmt.Errorf("发送 Telnet 密码失败: %v", err)
			}
			sentPassword = true
			continue
		}

		if err != nil && strings.TrimSpace(output) == "" {
			if !sentUsername {
				if writeErr := s.writeInput([]byte(username + "\n")); writeErr == nil {
					sentUsername = true
					continue
				}
			}
			if sentUsername && !sentPassword {
				if writeErr := s.writeInput([]byte(password + "\n")); writeErr == nil {
					sentPassword = true
					continue
				}
			}
		}
	}

	return fmt.Errorf("Telnet 登录失败：未识别到设备提示符")
}

func (s *aiDeviceShellSession) execute(command string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isClosed() {
		return "", fmt.Errorf("设备交互会话已关闭")
	}

	_, _ = s.collectOutput(100*time.Millisecond, 150*time.Millisecond, 300*time.Millisecond)

	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	isHelpQuery := strings.HasSuffix(trimmed, "?")
	payload := trimmed + "\n"
	if isHelpQuery {
		payload = trimmed
	}

	if err := s.writeInput([]byte(payload)); err != nil {
		s.close()
		return "", fmt.Errorf("发送命令失败: %v", err)
	}

	raw, err := s.collectOutput(aiDeviceFirstByteTimeout, aiDeviceReadQuietWindow, aiDeviceCommandTimeout)
	if isHelpQuery {
		_ = s.writeInput([]byte{0x03})
		_, _ = s.collectOutput(300*time.Millisecond, 300*time.Millisecond, 1200*time.Millisecond)
	}
	if err != nil && strings.TrimSpace(raw) == "" {
		return "", err
	}

	s.touch()
	return cleanupDeviceInteractiveOutput(trimmed, raw), nil
}

func (s *aiDeviceShellSession) writeInput(data []byte) error {
	if s.protocol == "telnet" {
		if s.conn == nil {
			return fmt.Errorf("Telnet 连接不存在")
		}
		data = telnetFixBackspace(data)
		_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		_, err := s.conn.Write(data)
		if err == nil {
			s.touch()
		}
		return err
	}
	if s.stdin == nil {
		return fmt.Errorf("SSH stdin 管道不存在")
	}
	_, err := s.stdin.Write(data)
	if err == nil {
		s.touch()
	}
	return err
}

func (s *aiDeviceShellSession) collectOutput(firstByteTimeout time.Duration, quietWindow time.Duration, maxDuration time.Duration) (string, error) {
	firstTimer := time.NewTimer(firstByteTimeout)
	quietTimer := time.NewTimer(maxDuration)
	maxTimer := time.NewTimer(maxDuration)
	if !quietTimer.Stop() {
		select {
		case <-quietTimer.C:
		default:
		}
	}
	defer firstTimer.Stop()
	defer quietTimer.Stop()
	defer maxTimer.Stop()

	var buf bytes.Buffer
	received := false

	resetQuietTimer := func() {
		if !quietTimer.Stop() {
			select {
			case <-quietTimer.C:
			default:
			}
		}
		quietTimer.Reset(quietWindow)
	}

	for {
		select {
		case chunk := <-s.outputCh:
			if len(chunk) == 0 {
				continue
			}
			received = true
			buf.Write(chunk)
			s.touch()
			if !firstTimer.Stop() {
				select {
				case <-firstTimer.C:
				default:
				}
			}
			resetQuietTimer()
			if devicePromptLinePattern.MatchString(normalizeInteractiveOutput(buf.String())) {
				resetQuietTimer()
			}
		case err := <-s.errCh:
			if strings.TrimSpace(buf.String()) != "" {
				return buf.String(), nil
			}
			if err == io.EOF {
				return "", fmt.Errorf("设备会话已关闭")
			}
			return "", err
		case <-firstTimer.C:
			if !received {
				return buf.String(), fmt.Errorf("等待设备响应超时")
			}
		case <-quietTimer.C:
			if received {
				return buf.String(), nil
			}
		case <-maxTimer.C:
			if strings.TrimSpace(buf.String()) != "" {
				return buf.String(), nil
			}
			return "", fmt.Errorf("设备响应超时")
		}
	}
}

func (s *aiDeviceShellSession) touch() {
	s.activityMu.Lock()
	s.lastActive = time.Now()
	s.activityMu.Unlock()
}

func (s *aiDeviceShellSession) getLastActive() time.Time {
	s.activityMu.RLock()
	defer s.activityMu.RUnlock()
	return s.lastActive
}

func (s *aiDeviceShellSession) isClosed() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}

func (s *aiDeviceShellSession) close() {
	s.closeOnce.Do(func() {
		close(s.closed)
		if s.conn != nil {
			_ = s.conn.Close()
		}
		if s.session != nil {
			_ = s.session.Close()
		}
		if s.client != nil {
			_ = s.client.Close()
		}
	})
}

func normalizeInteractiveOutput(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = ansiEscapePattern.ReplaceAllString(text, "")
	return text
}

func cleanupDeviceInteractiveOutput(command string, raw string) string {
	text := normalizeInteractiveOutput(raw)
	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	trimmedCommand := strings.TrimSpace(command)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			cleaned = append(cleaned, "")
			continue
		}
		if trimmedLine == trimmedCommand {
			continue
		}
		if strings.HasSuffix(trimmedLine, trimmedCommand) && devicePromptLinePattern.MatchString(trimmedLine) {
			continue
		}
		cleaned = append(cleaned, line)
	}

	for len(cleaned) > 0 {
		last := strings.TrimSpace(cleaned[len(cleaned)-1])
		if last == "" {
			cleaned = cleaned[:len(cleaned)-1]
			continue
		}
		if devicePromptLinePattern.MatchString(last) {
			cleaned = cleaned[:len(cleaned)-1]
			continue
		}
		break
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

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

func processAITelnetData(conn net.Conn, data []byte) []byte {
	var cleaned []byte
	i := 0
	for i < len(data) {
		if data[i] == telnetIAC && i+1 < len(data) {
			cmd := data[i+1]
			switch cmd {
			case telnetDO:
				if i+2 < len(data) {
					option := data[i+2]
					handleAITelnetDO(conn, option)
					i += 3
				} else {
					i += 2
				}
			case telnetDONT:
				if i+2 < len(data) {
					_, _ = conn.Write([]byte{telnetIAC, telnetWONT, data[i+2]})
					i += 3
				} else {
					i += 2
				}
			case telnetWILL:
				if i+2 < len(data) {
					handleAITelnetWILL(conn, data[i+2])
					i += 3
				} else {
					i += 2
				}
			case telnetWONT:
				if i+2 < len(data) {
					i += 3
				} else {
					i += 2
				}
			case telnetSB:
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
				cleaned = append(cleaned, telnetIAC)
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

func handleAITelnetDO(conn net.Conn, option byte) {
	switch option {
	case telnetTTYP, telnetNAWS, telnetSGA:
		_, _ = conn.Write([]byte{telnetIAC, telnetWILL, option})
	default:
		_, _ = conn.Write([]byte{telnetIAC, telnetWONT, option})
	}
}

func handleAITelnetWILL(conn net.Conn, option byte) {
	switch option {
	case telnetECHO, telnetSGA:
		_, _ = conn.Write([]byte{telnetIAC, telnetDO, option})
	default:
		_, _ = conn.Write([]byte{telnetIAC, telnetDONT, option})
	}
}
