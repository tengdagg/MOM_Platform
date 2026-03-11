package security

import (
	"fmt"
	"os"
	"strings"

	"github.com/ydcloud-dy/mom/internal/conf"
)

const (
	EnvCredentialEncryptionKey = "MOM_CREDENTIAL_ENCRYPTION_KEY"
	EnvK8sEncryptionKey        = "MOM_K8S_ENCRYPTION_KEY"
)

func MustCredentialEncryptionKey() []byte {
	return mustRead32ByteKey(EnvCredentialEncryptionKey)
}

func MustK8sEncryptionKey() []byte {
	return mustRead32ByteKey(EnvK8sEncryptionKey)
}

func mustRead32ByteKey(envName string) []byte {
	value := readEnvValue(envName)
	if value == "" {
		value = readConfigValue(envName)
	}
	if value == "" {
		value = readEnvValue(legacyEnvName(envName))
	}
	if value == "" {
		value = readLegacyConfigValue(envName)
	}
	if value == "" {
		panic(fmt.Sprintf("%s 未配置，必须提供 32 字节加密密钥", envName))
	}
	if len(value) != 32 {
		panic(fmt.Sprintf("%s 长度无效：当前 %d 字节，必须为 32 字节", envName, len(value)))
	}
	return []byte(value)
}

func readEnvValue(envName string) string {
	value := strings.TrimSpace(os.Getenv(envName))
	if value == "" {
		value = strings.TrimSpace(os.Getenv(legacyEnvName(envName)))
	}
	return value
}

func readConfigValue(envName string) string {
	cfg := conf.Get()
	if cfg == nil {
		return ""
	}
	switch envName {
	case EnvCredentialEncryptionKey:
		return strings.TrimSpace(cfg.Security.CredentialEncryptionKey)
	case EnvK8sEncryptionKey:
		return strings.TrimSpace(cfg.Security.K8sEncryptionKey)
	default:
		return ""
	}
}

func legacyEnvName(envName string) string {
	const upperPrefix = "MOM_"
	if strings.HasPrefix(envName, upperPrefix) {
		return "mom_" + strings.TrimPrefix(envName, upperPrefix)
	}
	return envName
}

func readLegacyConfigValue(envName string) string {
	return readConfigValue(envName)
}
