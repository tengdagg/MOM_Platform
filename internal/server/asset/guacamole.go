// Copyright (c) 2026 DYCloud J.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package asset

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	assetbiz "github.com/ydcloud-dy/mom/internal/biz/asset"
	"github.com/ydcloud-dy/mom/internal/conf"
	appLogger "github.com/ydcloud-dy/mom/pkg/logger"
	"go.uber.org/zap"
)

// getGuacdAddr 获取guacd地址，优先级：config.yaml > 默认值(127.0.0.1:4822)
func getGuacdAddr() string {
	if cfg := conf.Get(); cfg != nil {
		return cfg.Guacd.GetGuacdAddr()
	}
	return "127.0.0.1:4822"
}

// GuacamoleProxy handle connection between WebSocket and Guacd
type GuacamoleProxy struct {
	wsConn      *websocket.Conn
	guacdConn   net.Conn
	guacdReader *bufio.Reader // persistent reader to avoid losing buffered data
	recording   *assetbiz.TerminalSession
}

func NewGuacamoleProxy(ws *websocket.Conn) *GuacamoleProxy {
	return &GuacamoleProxy{
		wsConn: ws,
	}
}

// Connect connects to guacd
func (p *GuacamoleProxy) Connect() error {
	addr := getGuacdAddr()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to guacd at %s (is guacd running?): %v", addr, err)
	}
	p.guacdConn = conn
	p.guacdReader = bufio.NewReader(conn) // single persistent reader
	return nil
}

// Close closes connections
func (p *GuacamoleProxy) Close() {
	if p.wsConn != nil {
		p.wsConn.Close()
	}
	if p.guacdConn != nil {
		p.guacdConn.Close()
	}
}

// Handshake performs the Guacamole protocol handshake
func (p *GuacamoleProxy) Handshake(config map[string]string) error {
	// 1. Send "select" instruction
	if err := p.writeGuacamoleInstruction("select", "rdp"); err != nil {
		return fmt.Errorf("failed to send select instruction: %v", err)
	}

	// 2. Read "args" instruction (using persistent reader)
	instruction, err := p.readGuacamoleInstruction()
	if err != nil {
		return fmt.Errorf("failed to read args instruction: %v", err)
	}
	if len(instruction) == 0 || instruction[0] != "args" {
		return fmt.Errorf("expected 'args' instruction, got: %v", instruction)
	}

	appLogger.Info("Guacamole args received", zap.Strings("args", instruction[1:]))

	// 3. Prepare "connect" arguments based on "args"
	argNames := instruction[1:]
	argValues := make([]string, len(argNames))

	for i, name := range argNames {
		// Echo back the protocol version tag (e.g., VERSION_1_5_0)
		// guacd 1.5.0+ expects the client to confirm the protocol version
		if strings.HasPrefix(name, "VERSION_") {
			argValues[i] = name
		} else if val, ok := config[name]; ok {
			argValues[i] = val
		} else {
			argValues[i] = ""
		}
	}

	// 4. Send client capability instructions
	if err := p.writeGuacamoleInstruction("size", config["width"], config["height"], config["dpi"]); err != nil {
		return fmt.Errorf("failed to send size instruction: %v", err)
	}

	if err := p.writeGuacamoleInstruction("audio", "audio/L16"); err != nil {
		return fmt.Errorf("failed to send audio instruction: %v", err)
	}

	if err := p.writeGuacamoleInstruction("video"); err != nil {
		return fmt.Errorf("failed to send video instruction: %v", err)
	}

	if err := p.writeGuacamoleInstruction("image", "image/png", "image/jpeg"); err != nil {
		return fmt.Errorf("failed to send image instruction: %v", err)
	}

	// 5. Send "connect" with argument values
	if err := p.writeGuacamoleInstruction("connect", argValues...); err != nil {
		return fmt.Errorf("failed to send connect instruction: %v", err)
	}

	// 6. Read "ready" instruction (using persistent reader - avoids losing buffered data)
	content, err := p.readOneInstruction(p.guacdReader)
	if err != nil {
		return fmt.Errorf("failed to read ready instruction: %v", err)
	}

	appLogger.Info("Guacamole handshake complete", zap.String("response", content))

	// Forward the ready instruction to the WebSocket client
	if err := p.wsConn.WriteMessage(websocket.TextMessage, []byte(content)); err != nil {
		return fmt.Errorf("failed to forward ready instruction to WebSocket: %v", err)
	}

	return nil
}

// Proxy starts proxying data between guacd and WebSocket
func (p *GuacamoleProxy) Proxy() error {
	var wg sync.WaitGroup
	wg.Add(2)

	// Guacd -> WebSocket (use the persistent reader to not lose any buffered data)
	go func() {
		defer wg.Done()
		defer p.guacdConn.Close() // Close guacd when this goroutine exits
		msgCount := 0
		for {
			content, err := p.readOneInstruction(p.guacdReader)
			if err != nil {
				appLogger.Debug("Guacd read error (connection closing)", zap.Error(err))
				p.wsConn.Close() // Signal WebSocket goroutine to exit
				return
			}
			msgCount++
			if msgCount <= 10 {
				preview := content
				if len(preview) > 200 {
					preview = preview[:200] + "...(truncated)"
				}
				appLogger.Info("Proxy guacd→ws",
					zap.Int("msg#", msgCount),
					zap.Int("len", len(content)),
					zap.String("data", preview))
			}
			if err := p.wsConn.WriteMessage(websocket.TextMessage, []byte(content)); err != nil {
				appLogger.Debug("WebSocket write error (connection closing)", zap.Error(err))
				return
			}
		}
	}()

	// WebSocket -> Guacd
	go func() {
		defer wg.Done()
		msgCount := 0
		for {
			_, msg, err := p.wsConn.ReadMessage()
			if err != nil {
				appLogger.Debug("WebSocket read error (connection closing)", zap.Error(err))
				p.guacdConn.Close() // Signal guacd goroutine to exit
				return
			}
			msgCount++
			if msgCount <= 5 {
				preview := string(msg)
				if len(preview) > 200 {
					preview = preview[:200] + "...(truncated)"
				}
				appLogger.Info("Proxy ws→guacd",
					zap.Int("msg#", msgCount),
					zap.String("data", preview))
			}
			if _, err := p.guacdConn.Write(msg); err != nil {
				appLogger.Debug("Guacd write error (connection closing)", zap.Error(err))
				return
			}
		}
	}()

	wg.Wait()
	return nil
}

// sendGuacamoleError sends a properly formatted Guacamole error instruction to the WebSocket client
// Uses utf8.RuneCountInString for message length to match JavaScript's string indexing
func (p *GuacamoleProxy) sendGuacamoleError(message string, statusCode int) {
	// Guacamole error format: error instruction followed by disconnect
	// 5.error,<len>.<message>,<len>.<statusCode>;
	statusStr := strconv.Itoa(statusCode)
	errorInstruction := fmt.Sprintf("%d.%s,%d.%s,%d.%s;", len("error"), "error", utf8.RuneCountInString(message), message, len(statusStr), statusStr)
	p.wsConn.WriteMessage(websocket.TextMessage, []byte(errorInstruction))
	// Also send disconnect
	disconnectInstruction := fmt.Sprintf("%d.%s;", len("disconnect"), "disconnect")
	p.wsConn.WriteMessage(websocket.TextMessage, []byte(disconnectInstruction))
}

func (p *GuacamoleProxy) writeGuacamoleInstruction(opcode string, args ...string) error {
	var sb strings.Builder

	// Opcode
	sb.WriteString(fmt.Sprintf("%d.%s", len(opcode), opcode))

	// Args
	for _, arg := range args {
		sb.WriteString(fmt.Sprintf(",%d.%s", len(arg), arg))
	}

	// Terminator
	sb.WriteString(";")

	_, err := p.guacdConn.Write([]byte(sb.String()))
	return err
}

func (p *GuacamoleProxy) readGuacamoleInstruction() ([]string, error) {
	content, err := p.readOneInstruction(p.guacdReader)
	if err != nil {
		return nil, err
	}

	// Parse instruction
	// Format: len.val,len.val;
	parts := strings.Split(content[:len(content)-1], ",") // remove trailing ;
	var elements []string

	for _, part := range parts {
		dotIndex := strings.Index(part, ".")
		if dotIndex == -1 {
			continue
		}
		val := part[dotIndex+1:]
		elements = append(elements, val)
	}

	return elements, nil
}

// readOneInstruction reads a full Guacamole instruction using protocol-aware parsing.
// The Guacamole protocol format is: length.value,length.value,...,length.value;
// Element values can contain ANY byte (including ';' and ','), so we must parse
// length prefixes and read exact byte counts rather than scanning for delimiters.
func (p *GuacamoleProxy) readOneInstruction(reader *bufio.Reader) (string, error) {
	var buf []byte
	for {
		// 1. Read the length prefix: digits terminated by '.'
		var lengthBytes []byte
		for {
			b, err := reader.ReadByte()
			if err != nil {
				return "", err
			}
			buf = append(buf, b)
			if b == '.' {
				break
			}
			lengthBytes = append(lengthBytes, b)
		}

		// 2. Parse the length prefix
		elemLen, err := strconv.Atoi(string(lengthBytes))
		if err != nil {
			return "", fmt.Errorf("invalid Guacamole instruction: bad length prefix '%s'", string(lengthBytes))
		}

		// 3. Read exactly elemLen bytes for the element value
		if elemLen > 0 {
			value := make([]byte, elemLen)
			if _, err := io.ReadFull(reader, value); err != nil {
				return "", fmt.Errorf("failed to read element value of length %d: %v", elemLen, err)
			}
			buf = append(buf, value...)
		}

		// 4. Read the element terminator: ',' (more elements) or ';' (end of instruction)
		term, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		buf = append(buf, term)

		if term == ';' {
			return string(buf), nil
		}
		if term != ',' {
			return "", fmt.Errorf("invalid Guacamole instruction: expected ',' or ';' but got '%c' (0x%02x)", term, term)
		}
	}
}

// HandleRDPConnection handles RDP connection requests
func (s *HTTPServer) HandleRDPConnection(c *gin.Context, host *assetbiz.HostInfoVO, credential *assetbiz.Credential) {
	appLogger.Info("Starting RDP connection",
		zap.String("host", host.IP),
		zap.Int("rdpPort", host.RDPPort),
		zap.String("username", credential.Username))

	// Upgrade to WebSocket with guacamole subprotocol
	wsParams := websocket.Upgrader{
		CheckOrigin:  func(r *http.Request) bool { return true },
		Subprotocols: []string{"guacamole"},
	}

	conn, err := wsParams.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		appLogger.Error("WebSocket upgrade failed for RDP", zap.Error(err))
		return
	}
	defer conn.Close()

	proxy := NewGuacamoleProxy(conn)

	// Connect to guacd daemon
	if err := proxy.Connect(); err != nil {
		appLogger.Error("Failed to connect to guacd", zap.Error(err),
			zap.String("guacdAddr", getGuacdAddr()))
		// Send a properly formatted Guacamole error so the frontend client can display it
		proxy.sendGuacamoleError("无法连接到RDP代理服务(guacd)，请确保guacd服务已启动", 519)
		return
	}
	defer proxy.Close()

	// Parse display dimensions from query parameters
	width := c.DefaultQuery("width", "1024")
	height := c.DefaultQuery("height", "768")
	dpi := c.DefaultQuery("dpi", "96")

	// Determine RDP port
	rdpPort := host.RDPPort
	if rdpPort == 0 {
		rdpPort = 3389
	}

	// Build RDP connection config for guacd
	config := map[string]string{
		"hostname":       host.IP,
		"port":           strconv.Itoa(rdpPort),
		"width":          width,
		"height":         height,
		"dpi":            dpi,
		"color-depth":    "24",
		"security":       "any",
		"ignore-cert":    "true",
		"resize-method":  "display-update",
		"disable-gfx":    "true",  // Disable GFX pipeline - prevents black screen on many Windows versions
		"enable-font-smoothing":      "true",
		"enable-wallpaper":           "true",
		"enable-desktop-composition": "true",
		"enable-theming":             "true",
		"client-name":                "MOM-Platform",
	}

	if credential != nil {
		config["username"] = credential.Username
		config["password"] = credential.Password
		// Support DOMAIN\User format
		if strings.Contains(credential.Username, "\\") {
			parts := strings.SplitN(credential.Username, "\\", 2)
			config["domain"] = parts[0]
			config["username"] = parts[1]
		}
	}

	appLogger.Info("RDP config prepared",
		zap.String("hostname", config["hostname"]),
		zap.String("port", config["port"]),
		zap.String("width", width),
		zap.String("height", height),
		zap.String("username", config["username"]))

	// Perform Guacamole protocol handshake with guacd
	if err := proxy.Handshake(config); err != nil {
		appLogger.Error("Guacamole handshake failed", zap.Error(err))
		proxy.sendGuacamoleError("RDP连接握手失败: "+err.Error(), 519)
		return
	}

	appLogger.Info("RDP connection established, starting proxy",
		zap.String("host", host.IP))

	// Start bidirectional proxying between WebSocket and guacd
	proxy.Proxy()

	appLogger.Info("RDP connection closed", zap.String("host", host.IP))
}
