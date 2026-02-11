package model

import "gorm.io/gorm"

type HelmRepo struct {
	gorm.Model
	Name                  string `json:"name" gorm:"uniqueIndex;type:varchar(100);not null;comment:仓库名称"`
	Type                  string `json:"type" gorm:"type:varchar(20);default:'helm';comment:仓库类型(helm/oci)"`
	URL                   string `json:"url" gorm:"type:varchar(255);not null;comment:仓库地址"`
	Username              string `json:"username" gorm:"type:varchar(100);comment:用户名"`
	Password              string `json:"password" gorm:"type:varchar(255);comment:密码"`
	CertFile              string `json:"certFile" gorm:"type:text;comment:客户端证书内容"`
	KeyFile               string `json:"keyFile" gorm:"type:text;comment:客户端密钥内容"`
	CAFile                string `json:"caFile" gorm:"type:text;comment:CA证书内容"`
	InsecureSkipTLSVerify bool   `json:"insecureSkipTLSVerify" gorm:"default:false;comment:跳过TLS验证"`
}

func (HelmRepo) TableName() string {
	return "k8s_helm_repos"
}
