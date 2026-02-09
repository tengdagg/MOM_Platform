package service

import (
	"context"
	"fmt"
	"os"
	"time"

	"gorm.io/gorm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/ydcloud-dy/mom/plugins/kubernetes/model"
	"helm.sh/helm/v3/pkg/chart/loader"
	"sigs.k8s.io/yaml"
)

// HelmService Helm 服务
type HelmService struct {
	db             *gorm.DB
	clusterService *ClusterService
	settings       *cli.EnvSettings
}

// NewHelmService 创建 Helm 服务
func NewHelmService(db *gorm.DB, clusterService *ClusterService) *HelmService {
	return &HelmService{
		db:             db,
		clusterService: clusterService,
		settings:       cli.New(),
	}
}

// ==================== 仓库管理 ====================

// ListRepos 获取 Helm 仓库列表
func (s *HelmService) ListRepos(ctx context.Context) ([]model.HelmRepo, error) {
	var repos []model.HelmRepo
	err := s.db.Find(&repos).Error
	return repos, err
}

// AddRepo 添加 Helm 仓库
func (s *HelmService) AddRepo(ctx context.Context, repo *model.HelmRepo) error {
	// 检查是否存在同名仓库
	var count int64
	s.db.Unscoped().Model(&model.HelmRepo{}).Where("name = ?", repo.Name).Count(&count)
	if count > 0 {
		return fmt.Errorf("仓库名称 '%s' 已存在", repo.Name)
	}
	// 验证仓库连接
	if err := s.validateRepo(repo); err != nil {
		return fmt.Errorf("验证仓库失败: %v", err)
	}
	return s.db.Create(repo).Error
}

// DeleteRepo 删除 Helm 仓库 (硬删除)
func (s *HelmService) DeleteRepo(ctx context.Context, id uint) error {
	result := s.db.Unscoped().Where("id = ?", id).Delete(&model.HelmRepo{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("仓库不存在或已被删除")
	}
	return nil
}

// UpdateRepo 更新 Helm 仓库
func (s *HelmService) UpdateRepo(ctx context.Context, repo *model.HelmRepo) error {
	// 验证仓库连接
	if err := s.validateRepo(repo); err != nil {
		return fmt.Errorf("验证仓库失败: %v", err)
	}
	updates := map[string]interface{}{
		"url":                      repo.URL,
		"username":                 repo.Username,
		"cert_file":                repo.CertFile,
		"key_file":                 repo.KeyFile,
		"ca_file":                  repo.CAFile,
		"insecure_skip_tls_verify": repo.InsecureSkipTLSVerify,
	}
	// 仅在密码非空时更新密码
	if repo.Password != "" {
		updates["password"] = repo.Password
	}
	return s.db.Model(&model.HelmRepo{}).Where("id = ?", repo.ID).Updates(updates).Error
}

// validateRepo 验证仓库是否可用
func (s *HelmService) validateRepo(r *model.HelmRepo) error {
	entry, cleanup, err := s.getRepoEntry(r)
	if err != nil {
		return err
	}
	defer cleanup()

	// 创建 ChartRepository
	cr, err := repo.NewChartRepository(entry, getter.All(s.settings))
	if err != nil {
		return err
	}

	// 尝试下载 index.yaml
	_, err = cr.DownloadIndexFile()
	return err
}

func (s *HelmService) getRepoEntry(r *model.HelmRepo) (*repo.Entry, func(), error) {
	entry := &repo.Entry{
		Name:                  r.Name,
		URL:                   r.URL,
		Username:              r.Username,
		Password:              r.Password,
		InsecureSkipTLSverify: r.InsecureSkipTLSVerify,
	}

	var filesToRemove []string
	cleanup := func() {
		for _, f := range filesToRemove {
			os.Remove(f)
		}
	}

	if r.CertFile != "" {
		f, err := os.CreateTemp("", "helm-cert-*")
		if err != nil {
			cleanup()
			return nil, nil, err
		}
		f.WriteString(r.CertFile)
		f.Close()
		entry.CertFile = f.Name()
		filesToRemove = append(filesToRemove, f.Name())
	}

	if r.KeyFile != "" {
		f, err := os.CreateTemp("", "helm-key-*")
		if err != nil {
			cleanup()
			return nil, nil, err
		}
		f.WriteString(r.KeyFile)
		f.Close()
		entry.KeyFile = f.Name()
		filesToRemove = append(filesToRemove, f.Name())
	}

	if r.CAFile != "" {
		f, err := os.CreateTemp("", "helm-ca-*")
		if err != nil {
			cleanup()
			return nil, nil, err
		}
		f.WriteString(r.CAFile)
		f.Close()
		entry.CAFile = f.Name()
		filesToRemove = append(filesToRemove, f.Name())
	}

	return entry, cleanup, nil
}

// ==================== Chart 管理 ====================

// ChartInfo Chart 信息
type ChartInfo struct {
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	AppVersion  string   `json:"appVersion"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Keywords    []string `json:"keywords"`
	Home        string   `json:"home"`
	RepoName    string   `json:"repoName"`
}

// ListCharts 获取指定仓库的 Chart 列表
func (s *HelmService) ListCharts(ctx context.Context, repoID uint) ([]ChartInfo, error) {
	var r model.HelmRepo
	if err := s.db.First(&r, repoID).Error; err != nil {
		return nil, err
	}

	entry, cleanup, err := s.getRepoEntry(&r)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	cr, err := repo.NewChartRepository(entry, getter.All(s.settings))
	if err != nil {
		return nil, err
	}

	indexFilePath, err := cr.DownloadIndexFile()
	if err != nil {
		return nil, err
	}

	indexFile, err := repo.LoadIndexFile(indexFilePath)
	if err != nil {
		return nil, err
	}

	var charts []ChartInfo
	for name, versions := range indexFile.Entries {
		if len(versions) == 0 {
			continue
		}
		latest := versions[0]
		charts = append(charts, ChartInfo{
			Name:        name,
			Version:     latest.Version,
			AppVersion:  latest.AppVersion,
			Description: latest.Description,
			Icon:        latest.Icon,
			Keywords:    latest.Keywords,
			Home:        latest.Home,
			RepoName:    r.Name,
		})
	}

	return charts, nil
}

// ChartVersion represents a specific version of a chart
type ChartVersion struct {
	Version    string `json:"version"`
	AppVersion string `json:"appVersion"`
	Created    string `json:"created"`
}

// GetChartVersions 获取指定 Chart 的所有版本
func (s *HelmService) GetChartVersions(ctx context.Context, repoID uint, chartName string) ([]ChartVersion, error) {
	var r model.HelmRepo
	if err := s.db.First(&r, repoID).Error; err != nil {
		return nil, err
	}

	entry, cleanup, err := s.getRepoEntry(&r)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	cr, err := repo.NewChartRepository(entry, getter.All(s.settings))
	if err != nil {
		return nil, err
	}

	indexFilePath, err := cr.DownloadIndexFile()
	if err != nil {
		return nil, err
	}

	indexFile, err := repo.LoadIndexFile(indexFilePath)
	if err != nil {
		return nil, err
	}

	versions, ok := indexFile.Entries[chartName]
	if !ok {
		return nil, fmt.Errorf("chart %s not found in repo", chartName)
	}

	var result []ChartVersion
	for _, v := range versions {
		result = append(result, ChartVersion{
			Version:    v.Version,
			AppVersion: v.AppVersion,
			Created:    v.Created.Format(time.RFC3339),
		})
	}

	return result, nil
}

// InstallReleaseRequest 安装 Release 请求
type InstallReleaseRequest struct {
	ClusterID   uint   `json:"clusterId"`
	RepoID      uint   `json:"repoId"`
	ChartName   string `json:"chartName"`
	Version     string `json:"version"`
	ReleaseName string `json:"releaseName"`
	Namespace   string `json:"namespace"`
	Values      string `json:"values"` // YAML format values
}

// InstallRelease 安装 Chart 到集群
func (s *HelmService) InstallRelease(ctx context.Context, req *InstallReleaseRequest) (*ReleaseInfo, error) {
	// Get repo info
	var r model.HelmRepo
	if err := s.db.First(&r, req.RepoID).Error; err != nil {
		return nil, fmt.Errorf("获取仓库失败: %w", err)
	}

	// Get kubeconfig
	kubeConfigPath, cleanup, err := s.getTempKubeConfig(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	actionConfig, err := s.getActionConfig(kubeConfigPath, req.Namespace)
	if err != nil {
		return nil, err
	}

	// Create install action
	client := action.NewInstall(actionConfig)
	client.Namespace = req.Namespace
	client.ReleaseName = req.ReleaseName
	client.CreateNamespace = true
	client.Wait = false // Don't wait for resources to be ready

	// Build chart URL
	chartURL := fmt.Sprintf("%s/%s-%s.tgz", r.URL, req.ChartName, req.Version)

	// Pull and load chart
	chartPath, err := client.ChartPathOptions.LocateChart(chartURL, s.settings)
	if err != nil {
		return nil, fmt.Errorf("定位 Chart 失败: %w", err)
	}

	chart, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("加载 Chart 失败: %w", err)
	}

	// Parse values
	vals := map[string]interface{}{}
	if req.Values != "" {
		if err := yaml.Unmarshal([]byte(req.Values), &vals); err != nil {
			return nil, fmt.Errorf("解析 Values 失败: %w", err)
		}
	}

	// Install
	rel, err := client.Run(chart, vals)
	if err != nil {
		return nil, fmt.Errorf("安装失败: %w", err)
	}

	return &ReleaseInfo{
		Name:       rel.Name,
		Namespace:  rel.Namespace,
		Revision:   rel.Version,
		Updated:    rel.Info.LastDeployed.Format(time.RFC3339),
		Status:     rel.Info.Status.String(),
		Chart:      rel.Chart.Metadata.Name + "-" + rel.Chart.Metadata.Version,
		AppVersion: rel.Chart.Metadata.AppVersion,
	}, nil
}

// ==================== Release 管理 ====================

// ReleaseInfo Release 简要信息
type ReleaseInfo struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Revision   int    `json:"revision"`
	Updated    string `json:"updated"`
	Status     string `json:"status"`
	Chart      string `json:"chart"`
	AppVersion string `json:"appVersion"`
}

// ReleaseDetail Release 详细信息
type ReleaseDetail struct {
	ReleaseInfo
	Values   string `json:"values"`
	Manifest string `json:"manifest"`
	Notes    string `json:"notes"`
}

// ListReleases 获取集群中的所有 Release
func (s *HelmService) ListReleases(ctx context.Context, clusterID uint) ([]ReleaseInfo, error) {
	// 获取 kubeconfig 临时文件
	kubeConfigPath, cleanup, err := s.getTempKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	actionConfig, err := s.getActionConfig(kubeConfigPath, "")
	if err != nil {
		return nil, err
	}

	client := action.NewList(actionConfig)
	client.AllNamespaces = true
	// client.Deployed = true

	results, err := client.Run()
	if err != nil {
		return nil, err
	}

	var releases []ReleaseInfo
	for _, rel := range results {
		releases = append(releases, ReleaseInfo{
			Name:       rel.Name,
			Namespace:  rel.Namespace,
			Revision:   rel.Version,
			Updated:    rel.Info.LastDeployed.Format(time.RFC3339),
			Status:     rel.Info.Status.String(),
			Chart:      rel.Chart.Metadata.Name + "-" + rel.Chart.Metadata.Version,
			AppVersion: rel.Chart.Metadata.AppVersion,
		})
	}

	return releases, nil
}

// GetRelease 获取 Release 详情
func (s *HelmService) GetRelease(ctx context.Context, clusterID uint, namespace, name string) (*ReleaseDetail, error) {
	kubeConfigPath, cleanup, err := s.getTempKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	actionConfig, err := s.getActionConfig(kubeConfigPath, namespace)
	if err != nil {
		return nil, err
	}

	client := action.NewGet(actionConfig)
	rel, err := client.Run(name)
	if err != nil {
		return nil, err
	}

	values, _ := yaml.Marshal(rel.Config)

	return &ReleaseDetail{
		ReleaseInfo: ReleaseInfo{
			Name:       rel.Name,
			Namespace:  rel.Namespace,
			Revision:   rel.Version,
			Updated:    rel.Info.LastDeployed.Format(time.RFC3339),
			Status:     rel.Info.Status.String(),
			Chart:      rel.Chart.Metadata.Name + "-" + rel.Chart.Metadata.Version,
			AppVersion: rel.Chart.Metadata.AppVersion,
		},
		Values:   string(values),
		Manifest: rel.Manifest,
		Notes:    rel.Info.Notes,
	}, nil
}

// UninstallRelease 卸载 Release
func (s *HelmService) UninstallRelease(ctx context.Context, clusterID uint, namespace, name string) error {
	kubeConfigPath, cleanup, err := s.getTempKubeConfig(ctx, clusterID)
	if err != nil {
		return err
	}
	defer cleanup()

	actionConfig, err := s.getActionConfig(kubeConfigPath, namespace)
	if err != nil {
		return err
	}

	client := action.NewUninstall(actionConfig)
	_, err = client.Run(name)
	return err
}

// ==================== 辅助方法 ====================

// getTempKubeConfig 获取临时 KubeConfig 文件路径
func (s *HelmService) getTempKubeConfig(ctx context.Context, clusterID uint) (string, func(), error) {
	kubeConfigContent, err := s.clusterService.GetClusterKubeConfig(ctx, clusterID)
	if err != nil {
		return "", nil, fmt.Errorf("获取集群配置失败: %w", err)
	}

	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "kubeconfig-*.yaml")
	if err != nil {
		return "", nil, fmt.Errorf("创建临时文件失败: %w", err)
	}

	if _, err := tmpFile.WriteString(kubeConfigContent); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("写入临时文件失败: %w", err)
	}
	tmpFile.Close()

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return tmpFile.Name(), cleanup, nil
}

// getActionConfig 获取 Helm Action Configuration
func (s *HelmService) getActionConfig(kubeConfigPath string, namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)

	// 使用 genericclioptions 构造 RESTClientGetter
	cf := genericclioptions.NewConfigFlags(true)
	cf.KubeConfig = &kubeConfigPath
	if namespace != "" {
		cf.Namespace = &namespace
	}

	// 如果 namespace 为空，helm 可能会报错，所以尽量提供 default
	if namespace == "" {
		ns := "default"
		cf.Namespace = &ns
	}

	debugLog := func(format string, v ...interface{}) {
		// fmt.Printf(format, v...)
	}

	if err := actionConfig.Init(cf, namespace, "secrets", debugLog); err != nil {
		return nil, err
	}

	return actionConfig, nil
}
