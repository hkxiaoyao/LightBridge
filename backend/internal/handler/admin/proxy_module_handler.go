package admin

import (
	"net/http"
	"strconv"

	proxymodule "github.com/WilliamWang1721/LightBridge/internal/modules/proxy"
	"github.com/gin-gonic/gin"
)

type ProxyModuleHandler struct {
	service *proxymodule.Service
}

func NewProxyModuleHandler(service *proxymodule.Service) *ProxyModuleHandler {
	return &ProxyModuleHandler{service: service}
}

func (h *ProxyModuleHandler) ListNodes(c *gin.Context) {
	items, err := h.service.ListNodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nodes": items})
}

func (h *ProxyModuleHandler) CreateNode(c *gin.Context) {
	var req proxymodule.CreateNodeInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.service.CreateManualNode(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProxyModuleHandler) ImportNodes(c *gin.Context) {
	var req proxymodule.ImportNodesInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	items, err := h.service.ImportNodes(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"nodes": items})
}

func (h *ProxyModuleHandler) DeleteNode(c *gin.Context) {
	id, err := parseProxyModuleID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid node id"})
		return
	}
	if err := h.service.DeleteNode(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ProxyModuleHandler) ListProfiles(c *gin.Context) {
	items, err := h.service.ListProfiles(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"profiles": items})
}

func (h *ProxyModuleHandler) CreateProfile(c *gin.Context) {
	var req proxymodule.CreateProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.service.CreateProfile(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProxyModuleHandler) UpdateProfile(c *gin.Context) {
	id, err := parseProxyModuleID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}
	var req proxymodule.UpdateProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.service.UpdateProfile(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProxyModuleHandler) StartProfile(c *gin.Context) {
	id, err := parseProxyModuleID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}
	item, err := h.service.StartProfile(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProxyModuleHandler) StopProfile(c *gin.Context) {
	id, err := parseProxyModuleID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}
	item, err := h.service.StopProfile(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProxyModuleHandler) RestartProfile(c *gin.Context) {
	id, err := parseProxyModuleID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}
	item, err := h.service.RestartProfile(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProxyModuleHandler) TestProfile(c *gin.Context) {
	id, err := parseProxyModuleID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}
	item, err := h.service.TestProfile(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProxyModuleHandler) GetProfileRuntime(c *gin.Context) {
	id, err := parseProxyModuleID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}
	item, err := h.service.GetRuntime(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProxyModuleHandler) GetRuntimeStatus(c *gin.Context) {
	item, err := h.service.GetRuntimeStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProxyModuleHandler) ListBindings(c *gin.Context) {
	items, err := h.service.ListBindings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"bindings": items})
}

func (h *ProxyModuleHandler) CreateBinding(c *gin.Context) {
	var req proxymodule.CreateBindingInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.service.CreateBinding(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *ProxyModuleHandler) DeleteBinding(c *gin.Context) {
	id, err := parseProxyModuleID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid binding id"})
		return
	}
	if err := h.service.DeleteBinding(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ProxyModuleHandler) MigrateLegacy(c *gin.Context) {
	report, err := h.service.MigrateLegacyProxies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func parseProxyModuleID(c *gin.Context) (int64, error) {
	return strconv.ParseInt(c.Param("id"), 10, 64)
}
