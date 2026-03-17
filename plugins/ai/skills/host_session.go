package skills

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	aiHostSessionIdleTimeout = 10 * time.Minute
	aiHostReadQuietWindow    = 1200 * time.Millisecond
	aiHostFirstByteTimeout   = 8 * time.Second
	aiHostCommandTimeout     = 20 * time.Second
)

var (
	hostPromptLinePattern = regexp.MustCompile(`(?m)[^\n\r]{0,200}(?:>|#|\]|\$|%)\s*$`)
	aiHostShellSessions   = newAIHostShellSessionManager()
)

type aiHostTarget struct {
	ID           uint
	Name         string
	IP           string
	Port         int
	SSHUser      string
	CredentialID uint
}

type aiHostShellSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*aiHostShellSession
}

type aiHostShellSession struct {
	key        string
	hostID     uint
	hostName   string
	hostIP     string
	chatID     uint
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

func newAIHostShellSessionManager() *aiHostShellSessionManager {
	manager := &aiHostShellSessionManager{
		sessions: make(map[string]*aiHostShellSession),
	}
	go manager.cleanupLoop()
	return manager
}

func (m *aiHostShellSessionManager) ExecuteCommand(chatID uint, host aiHostTarget, password string, privateKey string, passphrase string, command string) (string, bool, error) {
	if chatID == 0 {
		return "", false, fmt.Errorf("缺少 AI 会话 ID，无法创建主机交互式会话")
	}

	session, reused, err := m.acquire(chatID, host, password, privateKey, passphrase)
	if err != nil {
		return "", reused, err
	}
	output, err := session.execute(command)
	if err != nil {
		return output, reused, err
	}
	return output, reused, nil
}

func (m *aiHostShellSessionManager) HasActiveSession(chatID uint, hostID uint) bool {
	if chatID == 0 {
		return false
	}
	key := fmt.Sprintf("%d:%d", chatID, hostID)
	m.mu.RLock()
	session := m.sessions[key]
	m.mu.RUnlock()
	return session != nil && !session.isClosed()
}

func (m *aiHostShellSessionManager) ListSessions(chatID uint, hostIDs map[uint]bool) []map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	result := make([]map[string]any, 0, len(m.sessions))
	for _, session := range m.sessions {
		if session.chatID != chatID || session.isClosed() {
			continue
		}
		if len(hostIDs) > 0 && !hostIDs[session.hostID] {
			continue
		}
		lastActive := session.getLastActive()
		result = append(result, map[string]any{
			"hostId":                    session.hostID,
			"hostName":                  session.hostName,
			"hostIP":                    session.hostIP,
			"lastActiveAt":              lastActive,
			"idleSeconds":               int(now.Sub(lastActive).Seconds()),
			"sessionIdleTimeoutSeconds": int(aiHostSessionIdleTimeout.Seconds()),
		})
	}
	return result
}

func (m *aiHostShellSessionManager) CloseSessions(chatID uint, hostIDs map[uint]bool) int {
	m.mu.RLock()
	targets := make([]*aiHostShellSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		if session.chatID != chatID || session.isClosed() {
			continue
		}
		if len(hostIDs) > 0 && !hostIDs[session.hostID] {
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

func (m *aiHostShellSessionManager) acquire(chatID uint, host aiHostTarget, password string, privateKey string, passphrase string) (*aiHostShellSession, bool, error) {
	key := fmt.Sprintf("%d:%d", chatID, host.ID)

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

	session, err := newAIHostShellSession(key, chatID, host, password, privateKey, passphrase)
	if err != nil {
		return nil, false, err
	}
	m.sessions[key] = session
	return session, false, nil
}

func (m *aiHostShellSessionManager) remove(key string) {
	m.mu.Lock()
	delete(m.sessions, key)
	m.mu.Unlock()
}

func (m *aiHostShellSessionManager) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		var stale []*aiHostShellSession

		m.mu.RLock()
		for _, session := range m.sessions {
			if now.Sub(session.getLastActive()) > aiHostSessionIdleTimeout {
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

func newAIHostShellSession(key string, chatID uint, host aiHostTarget, password string, privateKey string, passphrase string) (*aiHostShellSession, error) {
	if host.Port == 0 {
		host.Port = 22
	}
	if strings.TrimSpace(host.SSHUser) == "" {
		host.SSHUser = "root"
	}

	var authMethods []ssh.AuthMethod
	if strings.TrimSpace(privateKey) != "" {
		var signer ssh.Signer
		var err error
		if passphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(privateKey), []byte(passphrase))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(privateKey))
		}
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if strings.TrimSpace(password) != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}
	if len(authMethods) == 0 {
		return nil, fmt.Errorf("主机 %s(%s) 未配置可用的 SSH 凭证", host.Name, host.IP)
	}

	config := &ssh.ClientConfig{
		User:            host.SSHUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host.IP, host.Port), config)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %v", err)
	}

	sshSession, err := client.NewSession()
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("创建主机会话失败: %v", err)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := sshSession.RequestPty("xterm", 24, 160, modes); err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("请求主机伪终端失败: %v", err)
	}

	stdinPipe, err := sshSession.StdinPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("获取主机 stdin 管道失败: %v", err)
	}
	stdoutPipe, err := sshSession.StdoutPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("获取主机 stdout 管道失败: %v", err)
	}
	stderrPipe, err := sshSession.StderrPipe()
	if err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("获取主机 stderr 管道失败: %v", err)
	}
	if err := sshSession.Shell(); err != nil {
		sshSession.Close()
		client.Close()
		return nil, fmt.Errorf("启动主机交互式 shell 失败: %v", err)
	}

	session := &aiHostShellSession{
		key:        key,
		hostID:     host.ID,
		hostName:   host.Name,
		hostIP:     host.IP,
		chatID:     chatID,
		client:     client,
		session:    sshSession,
		stdin:      stdinPipe,
		outputCh:   make(chan []byte, 256),
		errCh:      make(chan error, 1),
		closed:     make(chan struct{}),
		lastActive: time.Now(),
	}

	go session.readLoop(stdoutPipe)
	go session.readLoop(stderrPipe)

	_, _ = io.WriteString(stdinPipe, "\n")
	_, _ = session.collectOutput(aiHostFirstByteTimeout, 800*time.Millisecond, 10*time.Second)

	return session, nil
}

func (s *aiHostShellSession) readLoop(reader io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
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

func (s *aiHostShellSession) execute(command string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isClosed() {
		return "", fmt.Errorf("主机交互会话已关闭")
	}

	_, _ = s.collectOutput(100*time.Millisecond, 150*time.Millisecond, 300*time.Millisecond)
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "", fmt.Errorf("命令不能为空")
	}
	if _, err := io.WriteString(s.stdin, trimmed+"\n"); err != nil {
		s.close()
		return "", fmt.Errorf("发送主机命令失败: %v", err)
	}

	raw, err := s.collectOutput(aiHostFirstByteTimeout, aiHostReadQuietWindow, aiHostCommandTimeout)
	if err != nil && strings.TrimSpace(raw) == "" {
		return "", err
	}

	s.touch()
	return cleanupHostInteractiveOutput(trimmed, raw), nil
}

func (s *aiHostShellSession) collectOutput(firstByteTimeout time.Duration, quietWindow time.Duration, maxDuration time.Duration) (string, error) {
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
			if hostPromptLinePattern.MatchString(normalizeInteractiveOutput(buf.String())) {
				resetQuietTimer()
			}
		case err := <-s.errCh:
			if strings.TrimSpace(buf.String()) != "" {
				return buf.String(), nil
			}
			if err == io.EOF {
				return "", fmt.Errorf("主机会话已关闭")
			}
			return "", err
		case <-firstTimer.C:
			if !received {
				return buf.String(), fmt.Errorf("等待主机响应超时")
			}
		case <-quietTimer.C:
			if received {
				return buf.String(), nil
			}
		case <-maxTimer.C:
			if strings.TrimSpace(buf.String()) != "" {
				return buf.String(), nil
			}
			return "", fmt.Errorf("主机响应超时")
		}
	}
}

func (s *aiHostShellSession) touch() {
	s.activityMu.Lock()
	s.lastActive = time.Now()
	s.activityMu.Unlock()
}

func (s *aiHostShellSession) getLastActive() time.Time {
	s.activityMu.RLock()
	defer s.activityMu.RUnlock()
	return s.lastActive
}

func (s *aiHostShellSession) isClosed() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}

func (s *aiHostShellSession) close() {
	s.closeOnce.Do(func() {
		close(s.closed)
		if s.session != nil {
			_ = s.session.Close()
		}
		if s.client != nil {
			_ = s.client.Close()
		}
	})
}

func cleanupHostInteractiveOutput(command string, raw string) string {
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
		if strings.HasSuffix(trimmedLine, trimmedCommand) && hostPromptLinePattern.MatchString(trimmedLine) {
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
		if hostPromptLinePattern.MatchString(last) {
			cleaned = cleaned[:len(cleaned)-1]
			continue
		}
		break
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}
