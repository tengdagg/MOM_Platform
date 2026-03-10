package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/mom/pkg/response"
	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	"gorm.io/datatypes"
)

type channelRequest struct {
	Name            string         `json:"name" binding:"required"`
	ChannelType     string         `json:"channelType" binding:"required"`
	Enabled         bool           `json:"enabled"`
	AppID           string         `json:"appId" binding:"required"`
	AppSecret       string         `json:"appSecret"`
	DefaultModelID  uint           `json:"defaultModelId"`
	ExecuteAsUserID uint           `json:"executeAsUserId" binding:"required"`
	Config          map[string]any `json:"config"`
}

func (h *Handler) ListChannels(c *gin.Context) {
	channels, err := h.channelSvc.ListChannels()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取聊天渠道列表失败")
		return
	}
	response.Success(c, channels)
}

func (h *Handler) GetChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	channel, err := h.channelSvc.GetChannel(uint(id))
	if err != nil {
		response.ErrorCode(c, http.StatusNotFound, "聊天渠道不存在")
		return
	}
	response.Success(c, channel)
}

func (h *Handler) CreateChannel(c *gin.Context) {
	var req channelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	channel, err := buildChannelConfigFromRequest(req)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.channelSvc.CreateChannel(channel); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if channel.Enabled {
		if err := h.channelRuntime.StartChannel(channel.ID); err != nil {
			response.ErrorCode(c, http.StatusBadRequest, "创建成功，但启动失败: "+err.Error())
			return
		}
	}
	channel.AppSecretSet = channel.AppSecret != ""
	response.Success(c, channel)
}

func (h *Handler) UpdateChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req channelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	channel, err := buildChannelConfigFromRequest(req)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	updated, err := h.channelSvc.UpdateChannel(uint(id), channel)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, err.Error())
		return
	}
	if updated.Enabled {
		if err := h.channelRuntime.StartChannel(updated.ID); err != nil {
			response.ErrorCode(c, http.StatusBadRequest, "保存成功，但启动失败: "+err.Error())
			return
		}
	} else {
		_ = h.channelRuntime.StopChannel(updated.ID)
	}
	response.Success(c, updated)
}

func (h *Handler) DeleteChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := h.channelRuntime.StopChannel(uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "停止聊天渠道失败")
		return
	}
	if err := h.channelSvc.DeleteChannel(uint(id)); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "删除聊天渠道失败")
		return
	}
	response.Success(c, nil)
}

func (h *Handler) TestChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	ctx, cancel := context.WithTimeout(context.Background(), h.channelTestTimeout)
	defer cancel()
	if err := h.channelRuntime.TestChannel(ctx, uint(id)); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "测试失败: "+err.Error())
		return
	}
	response.SuccessWithMessage(c, "测试成功", gin.H{"id": uint(id)})
}

func (h *Handler) StartChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	_ = h.channelSvc.SetChannelEnabled(uint(id), true)
	if err := h.channelRuntime.StartChannel(uint(id)); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "启动失败: "+err.Error())
		return
	}
	response.SuccessWithMessage(c, "聊天渠道已启动", gin.H{"id": uint(id)})
}

func (h *Handler) StopChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	_ = h.channelSvc.SetChannelEnabled(uint(id), false)
	if err := h.channelRuntime.StopChannel(uint(id)); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "停止失败: "+err.Error())
		return
	}
	response.SuccessWithMessage(c, "聊天渠道已停止", gin.H{"id": uint(id)})
}

func (h *Handler) ReconnectChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	_ = h.channelSvc.SetChannelEnabled(uint(id), true)
	if err := h.channelRuntime.ReconnectChannel(uint(id)); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "重连失败: "+err.Error())
		return
	}
	response.SuccessWithMessage(c, "聊天渠道已重连", gin.H{"id": uint(id)})
}

func buildChannelConfigFromRequest(req channelRequest) (*biz.AIChannelConfig, error) {
	configBytes, err := json.Marshal(req.Config)
	if err != nil {
		return nil, err
	}
	if len(configBytes) == 0 {
		configBytes = []byte(`{}`)
	}
	return &biz.AIChannelConfig{
		Name:            req.Name,
		ChannelType:     req.ChannelType,
		Enabled:         req.Enabled,
		AppID:           req.AppID,
		AppSecret:       req.AppSecret,
		DefaultModelID:  req.DefaultModelID,
		ExecuteAsUserID: req.ExecuteAsUserID,
		ConfigJSON:      datatypes.JSON(configBytes),
	}, nil
}
