//go:build infra_openclaw
// +build infra_openclaw

package openclaw

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/JustSebNL/Spiderweb/pkg/bus"
	"github.com/JustSebNL/Spiderweb/pkg/config"
	"github.com/JustSebNL/Spiderweb/pkg/logger"
	"github.com/JustSebNL/Spiderweb/pkg/skills"
)

// OpenClawBridge provides integration with the OpenClaw ecosystem
// for enhanced model capabilities and external service connectivity
type OpenClawBridge struct {
	config     *config.OpenClawConfig
	httpClient *http.Client
	eventBus   *bus.Bus
	logger     logger.Logger
	mu         sync.RWMutex
	connected  bool
	services   map[string]*ServiceInfo
}

// ServiceInfo represents an OpenClaw service
type ServiceInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Endpoint    string    `json:"endpoint"`
	AuthType    string    `json:"auth_type"`
	AuthConfig  AuthConfig `json:"auth_config"`
	Capabilities []string  `json:"capabilities"`
	Status      string    `json:"status"`
	LastSeen    time.Time `json:"last_seen"`
}

// AuthConfig represents authentication configuration for a service
type AuthConfig struct {
	APIKey     string `json:"api_key,omitempty"`
	Username   string `json:"username,omitempty"`
	Password   string `json:"password,omitempty"`
	Token      string `json:"token,omitempty"`
	PrivateKey string `json:"private_key,omitempty"`
}

// ServiceRequest represents a request to an OpenClaw service
type ServiceRequest struct {
	ServiceID string                 `json:"service_id"`
	Action    string                 `json:"action"`
	Params    map[string]interface{} `json:"params"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

// ServiceResponse represents a response from an OpenClaw service
type ServiceResponse struct {
	Success   bool                   `json:"success"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewOpenClawBridge creates a new OpenClaw bridge instance
func NewOpenClawBridge(cfg *config.OpenClawConfig, eventBus *bus.Bus, logger logger.Logger) *OpenClawBridge {
	return &OpenClawBridge{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		eventBus:   eventBus,
		logger:     logger,
		services:   make(map[string]*ServiceInfo),
	}
}

// Start initializes the OpenClaw bridge
func (b *OpenClawBridge) Start(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.config.Enabled {
		b.logger.Info("OpenClaw bridge disabled in configuration")
		return nil
	}

	b.logger.Info("Starting OpenClaw bridge")
	
	// Register for relevant events
	b.eventBus.Subscribe("skill_execution", b.handleSkillExecution)
	b.eventBus.Subscribe("model_request", b.handleModelRequest)
	
	// Start service discovery
	go b.discoverServices(ctx)
	
	// Start health monitoring
	go b.monitorServices(ctx)
	
	b.connected = true
	b.logger.Info("OpenClaw bridge started successfully")
	
	return nil
}

// Stop gracefully shuts down the OpenClaw bridge
func (b *OpenClawBridge) Stop() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return nil
	}

	b.logger.Info("Stopping OpenClaw bridge")
	
	// Unsubscribe from events
	b.eventBus.Unsubscribe("skill_execution", b.handleSkillExecution)
	b.eventBus.Unsubscribe("model_request", b.handleModelRequest)
	
	b.connected = false
	b.logger.Info("OpenClaw bridge stopped")
	
	return nil
}

// discoverServices performs service discovery in the OpenClaw ecosystem
func (b *OpenClawBridge) discoverServices(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(b.config.DiscoveryInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := b.performServiceDiscovery(ctx); err != nil {
				b.logger.Error("Service discovery failed", "error", err)
			}
		}
	}
}

// performServiceDiscovery discovers available OpenClaw services
func (b *OpenClawBridge) performServiceDiscovery(ctx context.Context) error {
	b.logger.Debug("Performing service discovery")
	
	discoveryURL := fmt.Sprintf("%s/discover", b.config.Endpoint)
	
	req, err := http.NewRequestWithContext(ctx, "GET", discoveryURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create discovery request: %w", err)
	}
	
	req.Header.Set("User-Agent", "Spiderweb-OpenClaw-Bridge/1.0")
	
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform discovery request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discovery request failed with status: %d", resp.StatusCode)
	}
	
	var services []ServiceInfo
	if err := json.NewDecoder(resp.Body).Decode(&services); err != nil {
		return fmt.Errorf("failed to decode discovery response: %w", err)
	}
	
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Update service registry
	for _, service := range services {
		b.services[service.ID] = &service
		b.logger.Info("Discovered service", "id", service.ID, "name", service.Name, "endpoint", service.Endpoint)
	}
	
	return nil
}

// monitorServices monitors the health of registered services
func (b *OpenClawBridge) monitorServices(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(b.config.HealthCheckInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.checkServiceHealth(ctx)
		}
	}
}

// checkServiceHealth checks the health of all registered services
func (b *OpenClawBridge) checkServiceHealth(ctx context.Context) {
	b.mu.RLock()
	services := make([]*ServiceInfo, 0, len(b.services))
	for _, service := range b.services {
		services = append(services, service)
	}
	b.mu.RUnlock()

	for _, service := range services {
		if err := b.checkServiceHealthSingle(ctx, service); err != nil {
			b.logger.Error("Service health check failed", "service", service.ID, "error", err)
			b.updateServiceStatus(service.ID, "unhealthy")
		} else {
			b.updateServiceStatus(service.ID, "healthy")
		}
	}
}

// checkServiceHealthSingle checks the health of a single service
func (b *OpenClawBridge) checkServiceHealthSingle(ctx context.Context, service *ServiceInfo) error {
	healthURL := fmt.Sprintf("%s/health", service.Endpoint)
	
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}
	
	req.Header.Set("User-Agent", "Spiderweb-OpenClaw-Bridge/1.0")
	
	resp, err := b.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to perform health check: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}
	
	return nil
}

// updateServiceStatus updates the status of a service
func (b *OpenClawBridge) updateServiceStatus(serviceID, status string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if service, exists := b.services[serviceID]; exists {
		service.Status = status
		service.LastSeen = time.Now()
	}
}

// executeServiceRequest executes a request to an OpenClaw service
func (b *OpenClawBridge) executeServiceRequest(ctx context.Context, request *ServiceRequest) (*ServiceResponse, error) {
	b.mu.RLock()
	service, exists := b.services[request.ServiceID]
	b.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("service %s not found", request.ServiceID)
	}
	
	if service.Status != "healthy" {
		return nil, fmt.Errorf("service %s is not healthy (status: %s)", request.ServiceID, service.Status)
	}
	
	// Build request URL
	url := fmt.Sprintf("%s/%s", service.Endpoint, request.Action)
	
	// Marshal request body
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	
	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Spiderweb-OpenClaw-Bridge/1.0")
	
	// Add authentication headers
	if err := b.addAuthHeaders(httpReq, service); err != nil {
		return nil, fmt.Errorf("failed to add authentication headers: %w", err)
	}
	
	// Execute request
	resp, err := b.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute service request: %w", err)
	}
	defer resp.Body.Close()
	
	// Parse response
	var serviceResp ServiceResponse
	if err := json.NewDecoder(resp.Body).Decode(&serviceResp); err != nil {
		return nil, fmt.Errorf("failed to decode service response: %w", err)
	}
	
	serviceResp.Timestamp = time.Now()
	
	return &serviceResp, nil
}

// addAuthHeaders adds authentication headers to the HTTP request
func (b *OpenClawBridge) addAuthHeaders(req *http.Request, service *ServiceInfo) error {
	switch service.AuthType {
	case "api_key":
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", service.AuthConfig.APIKey))
	case "basic":
		req.SetBasicAuth(service.AuthConfig.Username, service.AuthConfig.Password)
	case "token":
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", service.AuthConfig.Token))
	case "none":
		// No authentication required
	default:
		return fmt.Errorf("unsupported authentication type: %s", service.AuthType)
	}
	
	return nil
}

// handleSkillExecution handles skill execution events
func (b *OpenClawBridge) handleSkillExecution(event bus.Event) {
	if !b.config.SkillExecutionEnabled {
		return
	}
	
	skillEvent, ok := event.Data.(*skills.SkillExecutionEvent)
	if !ok {
		b.logger.Error("Invalid skill execution event type")
		return
	}
	
	b.logger.Debug("Handling skill execution", "skill", skillEvent.SkillName)
	
	// Check if this skill can be enhanced by OpenClaw services
	if b.canEnhanceSkill(skillEvent.SkillName) {
		go b.enhanceSkillExecution(skillEvent)
	}
}

// handleModelRequest handles model request events
func (b *OpenClawBridge) handleModelRequest(event bus.Event) {
	if !b.config.ModelEnhancementEnabled {
		return
	}
	
	modelEvent, ok := event.Data.(*ModelRequestEvent)
	if !ok {
		b.logger.Error("Invalid model request event type")
		return
	}
	
	b.logger.Debug("Handling model request", "model", modelEvent.ModelID)
	
	// Check if this model request can be enhanced by OpenClaw services
	if b.canEnhanceModelRequest(modelEvent) {
		go b.enhanceModelRequest(modelEvent)
	}
}

// canEnhanceSkill checks if a skill can be enhanced by OpenClaw services
func (b *OpenClawBridge) canEnhanceSkill(skillName string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	for _, service := range b.services {
		if service.Status == "healthy" {
			for _, capability := range service.Capabilities {
				if capability == skillName {
					return true
				}
			}
		}
	}
	
	return false
}

// canEnhanceModelRequest checks if a model request can be enhanced by OpenClaw services
func (b *OpenClawBridge) canEnhanceModelRequest(event *ModelRequestEvent) bool {
	// Check if any service can enhance this type of request
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	for _, service := range b.services {
		if service.Status == "healthy" {
			// Check if service can enhance based on request type or content
			if b.serviceCanEnhanceRequest(service, event) {
				return true
			}
		}
	}
	
	return false
}

// serviceCanEnhanceRequest checks if a service can enhance a specific model request
func (b *OpenClawBridge) serviceCanEnhanceRequest(service *ServiceInfo, event *ModelRequestEvent) bool {
	// Implement logic to determine if service can enhance this request
	// This could be based on request content, model type, or other criteria
	return true // Placeholder - implement actual logic
}

// enhanceSkillExecution enhances a skill execution with OpenClaw services
func (b *OpenClawBridge) enhanceSkillExecution(event *skills.SkillExecutionEvent) {
	// Find services that can enhance this skill
	services := b.getServicesForSkill(event.SkillName)
	
	for _, service := range services {
		request := &ServiceRequest{
			ServiceID: service.ID,
			Action:    "enhance",
			Params: map[string]interface{}{
				"skill_name": event.SkillName,
				"input":      event.Input,
				"context":    event.Context,
			},
			Metadata: map[string]string{
				"request_id": event.RequestID,
				"timestamp":  time.Now().Format(time.RFC3339),
			},
		}
		
		resp, err := b.executeServiceRequest(context.Background(), request)
		if err != nil {
			b.logger.Error("Failed to enhance skill execution", "skill", event.SkillName, "service", service.ID, "error", err)
			continue
		}
		
		if resp.Success {
			b.logger.Info("Skill execution enhanced", "skill", event.SkillName, "service", service.ID)
			// Process enhanced response
			b.processEnhancedResponse(event, resp)
		}
	}
}

// enhanceModelRequest enhances a model request with OpenClaw services
func (b *OpenClawBridge) enhanceModelRequest(event *ModelRequestEvent) {
	// Find services that can enhance this model request
	services := b.getServicesForModelRequest(event)
	
	for _, service := range services {
		request := &ServiceRequest{
			ServiceID: service.ID,
			Action:    "enhance",
			Params: map[string]interface{}{
				"model_id": event.ModelID,
				"prompt":   event.Prompt,
				"params":   event.Params,
			},
			Metadata: map[string]string{
				"request_id": event.RequestID,
				"timestamp":  time.Now().Format(time.RFC3339),
			},
		}
		
		resp, err := b.executeServiceRequest(context.Background(), request)
		if err != nil {
			b.logger.Error("Failed to enhance model request", "model", event.ModelID, "service", service.ID, "error", err)
			continue
		}
		
		if resp.Success {
			b.logger.Info("Model request enhanced", "model", event.ModelID, "service", service.ID)
			// Process enhanced response
			b.processEnhancedModelResponse(event, resp)
		}
	}
}

// getServicesForSkill returns services that can enhance a specific skill
func (b *OpenClawBridge) getServicesForSkill(skillName string) []*ServiceInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	var services []*ServiceInfo
	for _, service := range b.services {
		if service.Status == "healthy" {
			for _, capability := range service.Capabilities {
				if capability == skillName {
					services = append(services, service)
					break
				}
			}
		}
	}
	
	return services
}

// getServicesForModelRequest returns services that can enhance a model request
func (b *OpenClawBridge) getServicesForModelRequest(event *ModelRequestEvent) []*ServiceInfo {
	// Implement logic to find relevant services for this model request
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	var services []*ServiceInfo
	for _, service := range b.services {
		if service.Status == "healthy" && b.serviceCanEnhanceRequest(service, event) {
			services = append(services, service)
		}
	}
	
	return services
}

// processEnhancedResponse processes an enhanced skill execution response
func (b *OpenClawBridge) processEnhancedResponse(event *skills.SkillExecutionEvent, resp *ServiceResponse) {
	// Implement logic to process the enhanced response
	// This could involve updating the skill execution result or triggering additional actions
	b.logger.Debug("Processing enhanced skill response", "skill", event.SkillName, "data", resp.Data)
}

// processEnhancedModelResponse processes an enhanced model request response
func (b *OpenClawBridge) processEnhancedModelResponse(event *ModelRequestEvent, resp *ServiceResponse) {
	// Implement logic to process the enhanced response
	// This could involve updating the model response or triggering additional actions
	b.logger.Debug("Processing enhanced model response", "model", event.ModelID, "data", resp.Data)
}

// GetServiceInfo returns information about a specific service
func (b *OpenClawBridge) GetServiceInfo(serviceID string) (*ServiceInfo, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	service, exists := b.services[serviceID]
	return service, exists
}

// ListServices returns a list of all available services
func (b *OpenClawBridge) ListServices() []*ServiceInfo {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	services := make([]*ServiceInfo, 0, len(b.services))
	for _, service := range b.services {
		services = append(services, service)
	}
	
	return services
}

// ModelRequestEvent represents a model request event
type ModelRequestEvent struct {
	RequestID string                 `json:"request_id"`
	ModelID   string                 `json:"model_id"`
	Prompt    string                 `json:"prompt"`
	Params    map[string]interface{} `json:"params"`
	Timestamp time.Time              `json:"timestamp"`
}
