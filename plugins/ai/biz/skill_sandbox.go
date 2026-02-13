package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"gorm.io/gorm"
)

// ScriptSandbox 脚本沙箱执行引擎
type ScriptSandbox struct {
	db *gorm.DB
}

// NewScriptSandbox 创建脚本沙箱
func NewScriptSandbox(db *gorm.DB) *ScriptSandbox {
	return &ScriptSandbox{db: db}
}

// ExecuteScript 执行自定义脚本
func (s *ScriptSandbox) ExecuteScript(scriptType, scriptBody string, params map[string]any) (any, error) {
	switch scriptType {
	case "javascript":
		return s.executeJavaScript(scriptBody, params)
	case "python":
		return s.executePython(scriptBody, params)
	default:
		return nil, fmt.Errorf("不支持的脚本类型: %s", scriptType)
	}
}

// executeJavaScript 执行 JavaScript 脚本（通过 Node.js）
// 注意: 生产环境推荐使用 goja (Go 内嵌 JS 引擎)，这里用 Node.js 作为兼容方案
func (s *ScriptSandbox) executeJavaScript(script string, params map[string]any) (any, error) {
	// 构建执行脚本
	paramsJSON, _ := json.Marshal(params)
	wrapper := fmt.Sprintf(`
const params = %s;
const mom = {
  log: (...args) => console.error('[skill]', ...args),
};
async function main() {
  %s
}
main().then(result => {
  console.log(JSON.stringify(result || {}));
}).catch(err => {
  console.log(JSON.stringify({error: err.message}));
});
`, string(paramsJSON), script)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "node", "-e", wrapper)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		if stderrStr != "" {
			return nil, fmt.Errorf("脚本执行失败: %s", stderrStr)
		}
		return nil, fmt.Errorf("脚本执行失败: %v", err)
	}

	// 解析输出
	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return map[string]any{"result": "脚本执行完成，无输出"}, nil
	}

	var result any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return map[string]any{"output": output}, nil
	}

	return result, nil
}

// executePython 执行 Python 脚本
func (s *ScriptSandbox) executePython(script string, params map[string]any) (any, error) {
	paramsJSON, _ := json.Marshal(params)
	wrapper := fmt.Sprintf(`
import json, sys

params = json.loads('%s')

class MomAPI:
    @staticmethod
    def log(*args):
        print('[skill]', *args, file=sys.stderr)

mom = MomAPI()

%s
`, strings.ReplaceAll(string(paramsJSON), "'", "\\'"), script)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "python3", "-c", wrapper)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		// 过滤掉 [skill] 日志
		var errLines []string
		for _, line := range strings.Split(stderrStr, "\n") {
			if !strings.HasPrefix(line, "[skill]") && strings.TrimSpace(line) != "" {
				errLines = append(errLines, line)
			}
		}
		if len(errLines) > 0 {
			return nil, fmt.Errorf("脚本执行失败: %s", strings.Join(errLines, "\n"))
		}
		return nil, fmt.Errorf("脚本执行失败: %v", err)
	}

	output := strings.TrimSpace(stdout.String())
	if output == "" {
		return map[string]any{"result": "脚本执行完成，无输出"}, nil
	}

	var result any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return map[string]any{"output": output}, nil
	}

	return result, nil
}
