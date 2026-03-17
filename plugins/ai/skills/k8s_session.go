package skills

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	aiK8sSessionIdleTimeout = 10 * time.Minute
	aiK8sReadQuietWindow    = 1200 * time.Millisecond
	aiK8sFirstByteTimeout   = 8 * time.Second
	aiK8sCommandTimeout     = 20 * time.Second
)

var (
	k8sPromptLinePattern = regexp.MustCompile(`(?m)[^\n\r]{0,200}(?:>|#|\]|\$|%)\s*$`)
	aiK8sShellSessions   = newAIK8sShellSessionManager()
)

type aiK8sExecTarget struct {
	ClusterID   uint
	ClusterName string
	Namespace   string
	PodName     string
	Container   string
}

func (t aiK8sExecTarget) key(chatID uint) string {
	return fmt.Sprintf("%d:%d:%s:%s:%s", chatID, t.ClusterID, t.Namespace, t.PodName, t.Container)
}

type aiK8sSessionFilter struct {
	ClusterID uint
	Namespace string
	PodName   string
	Container string
}

type aiK8sShellSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*aiK8sShellSession
}

type aiK8sShellSession struct {
	key         string
	chatID      uint
	clusterID   uint
	clusterName string
	namespace   string
	podName     string
	container   string
	stdin       *io.PipeWriter
	cancel      context.CancelFunc
	outputCh    chan []byte
	errCh       chan error
	closed      chan struct{}
	closeOnce   sync.Once
	lastActive  time.Time
	mu          sync.Mutex
	activityMu  sync.RWMutex
}

func newAIK8sShellSessionManager() *aiK8sShellSessionManager {
	manager := &aiK8sShellSessionManager{
		sessions: make(map[string]*aiK8sShellSession),
	}
	go manager.cleanupLoop()
	return manager
}

func (m *aiK8sShellSessionManager) ExecuteCommand(chatID uint, restConfig *rest.Config, target aiK8sExecTarget, command string) (string, bool, error) {
	if chatID == 0 {
		return "", false, fmt.Errorf("缺少 AI 会话 ID，无法创建 Pod 交互式会话")
	}
	session, reused, err := m.acquire(chatID, restConfig, target)
	if err != nil {
		return "", reused, err
	}
	output, err := session.execute(command)
	if err != nil {
		return output, reused, err
	}
	return output, reused, nil
}

func (m *aiK8sShellSessionManager) HasActiveSession(chatID uint, target aiK8sExecTarget) bool {
	if chatID == 0 {
		return false
	}
	key := target.key(chatID)
	m.mu.RLock()
	session := m.sessions[key]
	m.mu.RUnlock()
	return session != nil && !session.isClosed()
}

func (m *aiK8sShellSessionManager) ListSessions(chatID uint, filter aiK8sSessionFilter) []map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	result := make([]map[string]any, 0, len(m.sessions))
	for _, session := range m.sessions {
		if session.isClosed() || session.clusterID == 0 || chatID != 0 && chatID != session.chatID {
			continue
		}
		if !filter.matches(session) {
			continue
		}
		lastActive := session.getLastActive()
		result = append(result, map[string]any{
			"clusterId":                  session.clusterID,
			"clusterName":                session.clusterName,
			"namespace":                  session.namespace,
			"podName":                    session.podName,
			"container":                  session.container,
			"lastActiveAt":               lastActive,
			"idleSeconds":                int(now.Sub(lastActive).Seconds()),
			"sessionIdleTimeoutSeconds":  int(aiK8sSessionIdleTimeout.Seconds()),
		})
	}
	return result
}

func (m *aiK8sShellSessionManager) CloseSessions(chatID uint, filter aiK8sSessionFilter) int {
	m.mu.RLock()
	targets := make([]*aiK8sShellSession, 0, len(m.sessions))
	for _, session := range m.sessions {
		if session.isClosed() || chatID != 0 && chatID != session.chatID {
			continue
		}
		if !filter.matches(session) {
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

func (m *aiK8sShellSessionManager) acquire(chatID uint, restConfig *rest.Config, target aiK8sExecTarget) (*aiK8sShellSession, bool, error) {
	key := target.key(chatID)

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

	session, err := newAIK8sShellSession(key, restConfig, target)
	if err != nil {
		return nil, false, err
	}
	m.sessions[key] = session
	return session, false, nil
}

func (m *aiK8sShellSessionManager) remove(key string) {
	m.mu.Lock()
	delete(m.sessions, key)
	m.mu.Unlock()
}

func (m *aiK8sShellSessionManager) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		var stale []*aiK8sShellSession

		m.mu.RLock()
		for _, session := range m.sessions {
			if now.Sub(session.getLastActive()) > aiK8sSessionIdleTimeout {
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

func newAIK8sShellSession(key string, restConfig *rest.Config, target aiK8sExecTarget) (*aiK8sShellSession, error) {
	execURL, err := buildK8sExecURL(restConfig, target, true, []string{
		"/bin/sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh",
	})
	if err != nil {
		return nil, err
	}
	executor, err := remotecommand.NewSPDYExecutor(restConfig, "POST", execURL)
	if err != nil {
		return nil, fmt.Errorf("创建 Pod exec executor 失败: %v", err)
	}

	stdinReader, stdinWriter := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	session := &aiK8sShellSession{
		key:         key,
		chatID:      sessionChatIDFromKey(key),
		clusterID:   target.ClusterID,
		clusterName: target.ClusterName,
		namespace:   target.Namespace,
		podName:     target.PodName,
		container:   target.Container,
		stdin:       stdinWriter,
		cancel:      cancel,
		outputCh:    make(chan []byte, 256),
		errCh:       make(chan error, 1),
		closed:      make(chan struct{}),
		lastActive:  time.Now(),
	}

	go session.streamShell(ctx, executor, stdinReader)

	_, _ = io.WriteString(stdinWriter, "\n")
	initialOutput, initialErr := session.collectOutput(aiK8sFirstByteTimeout, 800*time.Millisecond, 10*time.Second)
	if initialErr != nil && strings.TrimSpace(initialOutput) == "" {
		session.close()
		return nil, normalizeK8sExecError(initialErr, true)
	}
	if session.isClosed() && strings.TrimSpace(initialOutput) == "" {
		return nil, normalizeK8sExecError(fmt.Errorf("pod shell 初始化失败"), true)
	}

	return session, nil
}

func (s *aiK8sShellSession) streamShell(ctx context.Context, executor remotecommand.Executor, stdinReader *io.PipeReader) {
	defer stdinReader.Close()
	writer := &k8sSessionChannelWriter{outputCh: s.outputCh, closed: s.closed}
	err := executor.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdinReader,
		Stdout: writer,
		Stderr: writer,
		Tty:    true,
	})
	if err != nil {
		select {
		case s.errCh <- err:
		default:
		}
	}
	s.close()
}

func (s *aiK8sShellSession) execute(command string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isClosed() {
		return "", fmt.Errorf("Pod 交互会话已关闭")
	}

	_, _ = s.collectOutput(100*time.Millisecond, 150*time.Millisecond, 300*time.Millisecond)

	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "", fmt.Errorf("命令不能为空")
	}
	if err := s.writeInput([]byte(trimmed + "\n")); err != nil {
		s.close()
		return "", fmt.Errorf("发送 Pod 命令失败: %v", err)
	}

	raw, err := s.collectOutput(aiK8sFirstByteTimeout, aiK8sReadQuietWindow, aiK8sCommandTimeout)
	if err != nil && strings.TrimSpace(raw) == "" {
		return "", err
	}

	s.touch()
	return cleanupK8sInteractiveOutput(trimmed, raw), nil
}

func (s *aiK8sShellSession) writeInput(data []byte) error {
	if s.stdin == nil {
		return fmt.Errorf("Pod stdin 管道不存在")
	}
	if _, err := s.stdin.Write(data); err != nil {
		return err
	}
	s.touch()
	return nil
}

func (s *aiK8sShellSession) collectOutput(firstByteTimeout time.Duration, quietWindow time.Duration, maxDuration time.Duration) (string, error) {
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
			if k8sPromptLinePattern.MatchString(normalizeInteractiveOutput(buf.String())) {
				resetQuietTimer()
			}
		case err := <-s.errCh:
			if strings.TrimSpace(buf.String()) != "" {
				return buf.String(), nil
			}
			if err == io.EOF {
				return "", fmt.Errorf("Pod 会话已关闭")
			}
			return "", err
		case <-firstTimer.C:
			if !received {
				return buf.String(), fmt.Errorf("等待 Pod 响应超时")
			}
		case <-quietTimer.C:
			if received {
				return buf.String(), nil
			}
		case <-maxTimer.C:
			if strings.TrimSpace(buf.String()) != "" {
				return buf.String(), nil
			}
			return "", fmt.Errorf("Pod 响应超时")
		case <-s.closed:
			if strings.TrimSpace(buf.String()) != "" {
				return buf.String(), nil
			}
			return "", fmt.Errorf("Pod 会话已关闭")
		}
	}
}

func (s *aiK8sShellSession) touch() {
	s.activityMu.Lock()
	s.lastActive = time.Now()
	s.activityMu.Unlock()
}

func (s *aiK8sShellSession) getLastActive() time.Time {
	s.activityMu.RLock()
	defer s.activityMu.RUnlock()
	return s.lastActive
}

func (s *aiK8sShellSession) isClosed() bool {
	select {
	case <-s.closed:
		return true
	default:
		return false
	}
}

func (s *aiK8sShellSession) close() {
	s.closeOnce.Do(func() {
		close(s.closed)
		if s.cancel != nil {
			s.cancel()
		}
		if s.stdin != nil {
			_ = s.stdin.Close()
		}
	})
}

func (f aiK8sSessionFilter) matches(session *aiK8sShellSession) bool {
	if f.ClusterID > 0 && session.clusterID != f.ClusterID {
		return false
	}
	if strings.TrimSpace(f.Namespace) != "" && session.namespace != f.Namespace {
		return false
	}
	if strings.TrimSpace(f.PodName) != "" && session.podName != f.PodName {
		return false
	}
	if strings.TrimSpace(f.Container) != "" && session.container != f.Container {
		return false
	}
	return true
}

func cleanupK8sInteractiveOutput(command string, raw string) string {
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
		if strings.HasSuffix(trimmedLine, trimmedCommand) && k8sPromptLinePattern.MatchString(trimmedLine) {
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
		if k8sPromptLinePattern.MatchString(last) {
			cleaned = cleaned[:len(cleaned)-1]
			continue
		}
		break
	}

	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

type k8sSessionChannelWriter struct {
	outputCh chan []byte
	closed   chan struct{}
}

func (w *k8sSessionChannelWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	chunk := make([]byte, len(p))
	copy(chunk, p)
	select {
	case w.outputCh <- chunk:
		return len(p), nil
	case <-w.closed:
		return 0, io.EOF
	}
}

func buildK8sExecURL(restConfig *rest.Config, target aiK8sExecTarget, tty bool, command []string) (*url.URL, error) {
	serverURL, err := url.Parse(restConfig.Host)
	if err != nil {
		return nil, fmt.Errorf("解析集群 URL 失败: %v", err)
	}
	query := url.Values{}
	query.Set("container", target.Container)
	query.Set("stdin", fmt.Sprintf("%t", tty))
	query.Set("stdout", "true")
	query.Set("stderr", "true")
	query.Set("tty", fmt.Sprintf("%t", tty))
	for _, item := range command {
		query.Add("command", item)
	}
	return &url.URL{
		Scheme:   serverURL.Scheme,
		Host:     serverURL.Host,
		Path:     fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/exec", target.Namespace, target.PodName),
		RawQuery: query.Encode(),
	}, nil
}

func sessionChatIDFromKey(key string) uint {
	var chatID uint
	_, _ = fmt.Sscanf(key, "%d:", &chatID)
	return chatID
}
