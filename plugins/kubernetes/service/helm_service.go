package service

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/gorm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"sigs.k8s.io/yaml"

	"github.com/ydcloud-dy/mom/plugins/kubernetes/model"
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
		"type":                     repo.Type,
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
	if r.Type == "oci" {
		return s.validateOCIRepo(r)
	}
	// 传统 Helm 仓库：先用 HTTP 请求预检查，获取更友好的错误信息
	indexURL := r.URL + "/index.yaml"

	transport := &http.Transport{}
	if r.InsecureSkipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	req, err := http.NewRequest("GET", indexURL, nil)
	if err != nil {
		return fmt.Errorf("无法创建请求: %v", err)
	}

	if r.Username != "" && r.Password != "" {
		req.SetBasicAuth(r.Username, r.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("无法连接仓库 (%s): %v", r.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("认证失败 (HTTP %d): 请检查用户名和密码", resp.StatusCode)
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("仓库地址无效 (HTTP 404): 请检查 URL 是否正确，Harbor 格式为 https://harbor地址/chartrepo/项目名")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("仓库响应异常 (HTTP %d)", resp.StatusCode)
	}

	// 检查返回内容类型是否为 YAML
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "yaml") && !strings.Contains(contentType, "octet-stream") && !strings.Contains(contentType, "text/plain") {
		return fmt.Errorf("仓库返回了非预期的内容类型 (%s)，请确认 URL 指向有效的 Helm Chart 仓库", contentType)
	}

	return nil
}

// validateOCIRepo 验证 OCI 仓库是否可用
func (s *HelmService) validateOCIRepo(r *model.HelmRepo) error {
	hostURL := s.getOCIHostURL(r)
	checkURL := hostURL + "/v2/"

	client := s.getHTTPClient(r)
	req, err := http.NewRequest("GET", checkURL, nil)
	if err != nil {
		return fmt.Errorf("无法创建请求: %v", err)
	}
	if r.Username != "" && r.Password != "" {
		req.SetBasicAuth(r.Username, r.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("无法连接 OCI 仓库 (%s): %v", r.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("OCI 认证失败 (HTTP %d): 请检查用户名和密码", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("OCI 仓库响应异常 (HTTP %d)", resp.StatusCode)
	}
	return nil
}

// getOCIHostURL 从 OCI URL 提取 HTTPS 主机地址
// oci://harbor.xxx.com/project -> https://harbor.xxx.com
func (s *HelmService) getOCIHostURL(r *model.HelmRepo) string {
	url := strings.TrimPrefix(r.URL, "oci://")
	parts := strings.SplitN(url, "/", 2)
	host := parts[0]
	scheme := "https"
	if r.InsecureSkipTLSVerify {
		// 仍然使用 https，但 TLS 验证已跳过
	}
	return scheme + "://" + host
}

// getOCIProject 从 OCI URL 提取项目路径
// oci://harbor.xxx.com/chartrepo -> chartrepo
func (s *HelmService) getOCIProject(r *model.HelmRepo) string {
	url := strings.TrimPrefix(r.URL, "oci://")
	parts := strings.SplitN(url, "/", 2)
	if len(parts) > 1 {
		return strings.TrimSuffix(parts[1], "/")
	}
	return ""
}

// getHTTPClient 创建带 TLS 配置的 HTTP 客户端
func (s *HelmService) getHTTPClient(r *model.HelmRepo) *http.Client {
	transport := &http.Transport{}
	if r.InsecureSkipTLSVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
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

	if r.Type == "oci" {
		return s.listOCICharts(ctx, &r)
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

// listOCICharts 通过 Docker Registry API 列出 OCI 仓库的 Charts
func (s *HelmService) listOCICharts(ctx context.Context, r *model.HelmRepo) ([]ChartInfo, error) {
	hostURL := s.getOCIHostURL(r)
	project := s.getOCIProject(r)

	// 使用 Docker Registry API V2 列出仓库
	catalogURL := hostURL + "/v2/_catalog?n=1000"
	client := s.getHTTPClient(r)

	req, err := http.NewRequest("GET", catalogURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	if r.Username != "" && r.Password != "" {
		req.SetBasicAuth(r.Username, r.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 OCI catalog 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OCI catalog 请求失败 (HTTP %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var catalog struct {
		Repositories []string `json:"repositories"`
	}
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("解析 catalog 响应失败: %w", err)
	}

	var charts []ChartInfo
	for _, repoName := range catalog.Repositories {
		// 只显示当前项目下的 charts
		if project != "" && !strings.HasPrefix(repoName, project+"/") {
			continue
		}

		// 提取 chart 名称（去掉项目前缀）
		chartName := repoName
		if project != "" {
			chartName = strings.TrimPrefix(repoName, project+"/")
		}

		// 获取最新版本（最新 tag）
		latestVersion := s.getOCILatestTag(r, repoName)

		charts = append(charts, ChartInfo{
			Name:        chartName,
			Version:     latestVersion,
			Description: "",
			RepoName:    r.Name,
		})
	}

	return charts, nil
}

// getOCILatestTag 获取 OCI 仓库中某个 chart 的最新 tag
func (s *HelmService) getOCILatestTag(r *model.HelmRepo, repoName string) string {
	hostURL := s.getOCIHostURL(r)
	tagsURL := fmt.Sprintf("%s/v2/%s/tags/list", hostURL, repoName)

	client := s.getHTTPClient(r)
	req, _ := http.NewRequest("GET", tagsURL, nil)
	if r.Username != "" && r.Password != "" {
		req.SetBasicAuth(r.Username, r.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "unknown"
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "unknown"
	}

	body, _ := io.ReadAll(resp.Body)
	var tagsResp struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(body, &tagsResp); err != nil || len(tagsResp.Tags) == 0 {
		return "unknown"
	}

	// 返回最后一个 tag（通常是最新版本）
	return tagsResp.Tags[len(tagsResp.Tags)-1]
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

	if r.Type == "oci" {
		return s.getOCIChartVersions(ctx, &r, chartName)
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

// getOCIChartVersions 通过 Docker Registry API 获取 OCI chart 的所有版本
func (s *HelmService) getOCIChartVersions(ctx context.Context, r *model.HelmRepo, chartName string) ([]ChartVersion, error) {
	hostURL := s.getOCIHostURL(r)
	project := s.getOCIProject(r)

	repoPath := chartName
	if project != "" {
		repoPath = project + "/" + chartName
	}

	tagsURL := fmt.Sprintf("%s/v2/%s/tags/list", hostURL, repoPath)
	client := s.getHTTPClient(r)

	req, err := http.NewRequest("GET", tagsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	if r.Username != "" && r.Password != "" {
		req.SetBasicAuth(r.Username, r.Password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 OCI tags 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OCI tags 请求失败 (HTTP %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var tagsResp struct {
		Tags []string `json:"tags"`
	}
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return nil, fmt.Errorf("解析 tags 响应失败: %w", err)
	}

	var result []ChartVersion
	// 倒序输出，最新版本在前
	for i := len(tagsResp.Tags) - 1; i >= 0; i-- {
		result = append(result, ChartVersion{
			Version: tagsResp.Tags[i],
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

	// Create install action
	actionConfig, err := s.getActionConfig(kubeConfigPath, req.Namespace, r.InsecureSkipTLSVerify)
	if err != nil {
		return nil, err
	}
	client := action.NewInstall(actionConfig)
	client.Namespace = req.Namespace
	client.ReleaseName = req.ReleaseName
	client.CreateNamespace = true
	client.Wait = false // Don't wait for resources to be ready

	// Determine chart URL based on repo type
	var chartURL string

	if r.Type == "oci" {
		// OCI repo: construct URL directly (e.g., oci://harbor.xxx.com/chartrepo/nginx)
		chartURL = strings.TrimSuffix(r.URL, "/") + "/" + req.ChartName
	} else {
		// Traditional repo: look up chart URL from index.yaml
		entry, entryCleanup, err := s.getRepoEntry(&r)
		if err != nil {
			return nil, fmt.Errorf("获取仓库配置失败: %w", err)
		}
		defer entryCleanup()

		cr, err := repo.NewChartRepository(entry, getter.All(s.settings))
		if err != nil {
			return nil, fmt.Errorf("创建仓库客户端失败: %w", err)
		}

		indexFilePath, err := cr.DownloadIndexFile()
		if err != nil {
			return nil, fmt.Errorf("下载仓库索引失败: %w", err)
		}

		indexFile, err := repo.LoadIndexFile(indexFilePath)
		if err != nil {
			return nil, fmt.Errorf("加载仓库索引失败: %w", err)
		}

		// Find the chart URL from the index
		chartVersions, ok := indexFile.Entries[req.ChartName]
		if !ok || len(chartVersions) == 0 {
			return nil, fmt.Errorf("Chart %s 在仓库中不存在", req.ChartName)
		}

		for _, cv := range chartVersions {
			if cv.Version == req.Version {
				if len(cv.URLs) > 0 {
					chartURL = cv.URLs[0]
					if strings.HasPrefix(chartURL, "oci://") {
						// For OCI URLs, strip the version tag
						if idx := strings.LastIndex(chartURL, ":"); idx > strings.LastIndex(chartURL, "/") {
							chartURL = chartURL[:idx]
						}
					} else if !strings.HasPrefix(chartURL, "http://") && !strings.HasPrefix(chartURL, "https://") {
						// If the URL is relative, prepend the repo URL
						chartURL = strings.TrimSuffix(r.URL, "/") + "/" + chartURL
					}
				}
				break
			}
		}

		if chartURL == "" {
			return nil, fmt.Errorf("Chart %s 版本 %s 未找到下载地址", req.ChartName, req.Version)
		}

		// Set cert/key/ca from entry
		if entry.CertFile != "" {
			client.ChartPathOptions.CertFile = entry.CertFile
		}
		if entry.KeyFile != "" {
			client.ChartPathOptions.KeyFile = entry.KeyFile
		}
		if entry.CAFile != "" {
			client.ChartPathOptions.CaFile = entry.CAFile
		}
	}

	// Configure chart path options for auth and TLS
	client.ChartPathOptions.Version = req.Version
	if r.Username != "" {
		client.ChartPathOptions.Username = r.Username
		client.ChartPathOptions.Password = r.Password
	}
	if r.InsecureSkipTLSVerify {
		client.ChartPathOptions.InsecureSkipTLSverify = true
	}

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

// ResourceItem 从 manifest 解析出的资源条目
type ResourceItem struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// ReleaseDetail Release 详细信息
type ReleaseDetail struct {
	ReleaseInfo
	Values     string                    `json:"values"`     // 所有合并后的 values (chart defaults + user overrides)
	UserValues string                    `json:"userValues"` // 仅用户自定义的 values (用于 upgrade 编辑)
	Manifest   string                    `json:"manifest"`
	Notes      string                    `json:"notes"`
	Resources  map[string][]ResourceItem `json:"resources"` // 按 Kind 分组的资源列表
}

// ListReleases 获取集群中的所有 Release
func (s *HelmService) ListReleases(ctx context.Context, clusterID uint) ([]ReleaseInfo, error) {
	// 获取 kubeconfig 临时文件
	kubeConfigPath, cleanup, err := s.getTempKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	actionConfig, err := s.getActionConfig(kubeConfigPath, "", false)
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

	actionConfig, err := s.getActionConfig(kubeConfigPath, namespace, false)
	if err != nil {
		return nil, err
	}

	client := action.NewGet(actionConfig)
	rel, err := client.Run(name)
	if err != nil {
		return nil, err
	}

	// 合并所有 values (chart defaults + user overrides)
	computedValues, err := chartutil.CoalesceValues(rel.Chart, rel.Config)
	var allValuesYAML []byte
	if err == nil {
		allValuesYAML, _ = yaml.Marshal(computedValues)
	} else {
		// fallback: 仅用 Config
		allValuesYAML, _ = yaml.Marshal(rel.Config)
	}

	// 用户自定义 values
	userConfig := rel.Config
	if userConfig == nil {
		userConfig = map[string]interface{}{}
	}
	userValuesYAML, _ := yaml.Marshal(userConfig)

	// 解析 manifest 中的资源
	resources := parseManifestResources(rel.Manifest)

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
		Values:     string(allValuesYAML),
		UserValues: string(userValuesYAML),
		Manifest:   rel.Manifest,
		Notes:      rel.Info.Notes,
		Resources:  resources,
	}, nil
}

// parseManifestResources 从 manifest 解析出资源列表，按 Kind 分组
func parseManifestResources(manifest string) map[string][]ResourceItem {
	resources := make(map[string][]ResourceItem)
	docs := strings.Split(manifest, "---")
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		var obj map[string]interface{}
		if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
			continue
		}
		kind, _ := obj["kind"].(string)
		metadata, _ := obj["metadata"].(map[string]interface{})
		if kind == "" || metadata == nil {
			continue
		}
		name, _ := metadata["name"].(string)
		namespace, _ := metadata["namespace"].(string)
		resources[kind] = append(resources[kind], ResourceItem{
			Kind:      kind,
			Name:      name,
			Namespace: namespace,
		})
	}
	return resources
}

// UpgradeReleaseRequest 升级 Release 请求
type UpgradeReleaseRequest struct {
	Values string `json:"values"` // YAML 格式的 values
}

// UpgradeRelease 升级 Release (使用当前 chart + 新 values)
func (s *HelmService) UpgradeRelease(ctx context.Context, clusterID uint, namespace, name string, values string) (*ReleaseInfo, error) {
	kubeConfigPath, cleanup, err := s.getTempKubeConfig(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	actionConfig, err := s.getActionConfig(kubeConfigPath, namespace, false)
	if err != nil {
		return nil, err
	}

	// 获取当前 release 以复用 chart
	getClient := action.NewGet(actionConfig)
	rel, err := getClient.Run(name)
	if err != nil {
		return nil, fmt.Errorf("获取当前 Release 失败: %w", err)
	}

	// 解析新 values
	vals := map[string]interface{}{}
	if values != "" {
		if err := yaml.Unmarshal([]byte(values), &vals); err != nil {
			return nil, fmt.Errorf("解析 Values 失败: %w", err)
		}
	}

	// 执行升级
	upgradeClient := action.NewUpgrade(actionConfig)
	upgradeClient.Namespace = namespace
	upgradeClient.ReuseValues = false
	upgradeClient.Wait = false

	upgraded, err := upgradeClient.Run(name, rel.Chart, vals)
	if err != nil {
		return nil, fmt.Errorf("升级失败: %w", err)
	}

	return &ReleaseInfo{
		Name:       upgraded.Name,
		Namespace:  upgraded.Namespace,
		Revision:   upgraded.Version,
		Updated:    upgraded.Info.LastDeployed.Format(time.RFC3339),
		Status:     upgraded.Info.Status.String(),
		Chart:      upgraded.Chart.Metadata.Name + "-" + upgraded.Chart.Metadata.Version,
		AppVersion: upgraded.Chart.Metadata.AppVersion,
	}, nil
}

// UninstallRelease 卸载 Release
func (s *HelmService) UninstallRelease(ctx context.Context, clusterID uint, namespace, name string) error {
	kubeConfigPath, cleanup, err := s.getTempKubeConfig(ctx, clusterID)
	if err != nil {
		return err
	}
	defer cleanup()

	actionConfig, err := s.getActionConfig(kubeConfigPath, namespace, false)
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
func (s *HelmService) getActionConfig(kubeConfigPath string, namespace string, insecureSkipTLS bool) (*action.Configuration, error) {
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

	// Initialize OCI registry client for pulling oci:// charts
	var opts []registry.ClientOption
	opts = append(opts, registry.ClientOptDebug(false))
	opts = append(opts, registry.ClientOptEnableCache(true))

	if insecureSkipTLS {
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
		opts = append(opts, registry.ClientOptHTTPClient(httpClient))
	}

	registryClient, err := registry.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("创建 OCI registry 客户端失败: %w", err)
	}
	actionConfig.RegistryClient = registryClient

	return actionConfig, nil
}
