// Package provider implements cloud provider interfaces for temporary IAM role management
package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/yckao/gta/pkg/logger"
	resourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

const (
	// gcpBindingTitlePrefix is used to identify bindings created by this tool
	gcpBindingTitlePrefix = "gta_temporary_access"
	// policyVersion is required for using conditions in IAM policies
	policyVersion = 3
	// rolePrefix is the standard prefix for GCP IAM roles
	rolePrefix = "roles/"
)

// temporaryBinding represents a binding that will be cleaned up
type temporaryBinding struct {
	Role      string
	Member    string
	BindingID string
	Index     int
}

// GrantedRole represents a successfully granted role and its binding ID
type GrantedRole struct {
	Role      string
	BindingID string
}

// GCPProvider implements the Provider interface for Google Cloud Platform
type GCPProvider struct {
	ctx          context.Context
	service      *resourcemanager.Service
	dryRun       bool
	grantedRoles []GrantedRole // Track successfully granted roles and their binding IDs
}

// GCPOptions contains GCP-specific options for granting temporary access
type GCPOptions struct {
	Project string
	Roles   []string
	User    string
	TTL     time.Duration
}

// IsOptions implements provider.Options interface
func (o *GCPOptions) IsOptions() {}

// formatRole ensures the role has the proper prefix
func formatRole(role string) string {
	if strings.HasPrefix(role, rolePrefix) {
		return role
	}
	return rolePrefix + role
}

// formatMember formats a user email into a GCP member string
func formatMember(email string) string {
	return fmt.Sprintf("user:%s", email)
}

// NewGCPProvider creates a new GCP provider instance
func NewGCPProvider(ctx context.Context, dryRun bool) (*GCPProvider, error) {
	service, err := resourcemanager.NewService(ctx, option.WithScopes(resourcemanager.CloudPlatformScope))
	if err != nil {
		return nil, fmt.Errorf("failed to create Cloud Resource Manager service: %v", err)
	}

	return &GCPProvider{
		ctx:          ctx,
		service:      service,
		dryRun:       dryRun,
		grantedRoles: make([]GrantedRole, 0),
	}, nil
}

// getCurrentUser gets the email of the currently authenticated user
func (p *GCPProvider) getCurrentUser() (string, error) {
	oauth2Service, err := oauth2.NewService(p.ctx, option.WithScopes("https://www.googleapis.com/auth/userinfo.email"))
	if err != nil {
		return "", fmt.Errorf("failed to create OAuth2 service: %v", err)
	}

	userInfo, err := oauth2Service.Userinfo.Get().Do()
	if err != nil {
		return "", fmt.Errorf("failed to get user info: %v", err)
	}

	if userInfo.Email == "" {
		return "", fmt.Errorf("no email found in credentials")
	}

	return userInfo.Email, nil
}

// getIAMPolicy gets the IAM policy for a project with the required version
func (p *GCPProvider) getIAMPolicy(project string) (*resourcemanager.Policy, error) {
	getRequest := &resourcemanager.GetIamPolicyRequest{
		Options: &resourcemanager.GetPolicyOptions{
			RequestedPolicyVersion: policyVersion,
		},
	}
	policy, err := p.service.Projects.GetIamPolicy(project, getRequest).Context(p.ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get IAM policy: %v", err)
	}

	// Set the policy version to support conditions
	policy.Version = policyVersion
	return policy, nil
}

// setIAMPolicy updates the IAM policy for a project
func (p *GCPProvider) setIAMPolicy(project string, policy *resourcemanager.Policy) error {
	setRequest := &resourcemanager.SetIamPolicyRequest{
		Policy: policy,
	}
	_, err := p.service.Projects.SetIamPolicy(project, setRequest).Context(p.ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to set IAM policy: %v", err)
	}
	return nil
}

// createBinding creates a new IAM binding with the specified role, member, and expiration
func (p *GCPProvider) createBinding(role, member string, ttl time.Duration) *resourcemanager.Binding {
	expireTime := time.Now().Add(ttl).Format(time.RFC3339)
	bindingID := fmt.Sprintf("%s_%d", gcpBindingTitlePrefix, time.Now().UnixNano())

	return &resourcemanager.Binding{
		Role:    role,
		Members: []string{member},
		Condition: &resourcemanager.Expr{
			Title:       bindingID,
			Description: fmt.Sprintf("Temporary access granted by GTA tool at %s", time.Now().Format(time.RFC3339)),
			Expression:  fmt.Sprintf("request.time < timestamp('%s')", expireTime),
		},
	}
}

// Grant grants temporary access to the specified roles in the specified project
func (p *GCPProvider) Grant(opts Options) error {
	gcpOpts, ok := opts.(*GCPOptions)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	if gcpOpts.User == "" {
		user, err := p.getCurrentUser()
		if err != nil {
			return fmt.Errorf("failed to get current user: %v", err)
		}
		gcpOpts.User = user
		logger.Debug("Using current user: %s", user)
	}

	var grantErrors []string
	member := formatMember(gcpOpts.User)

	for _, role := range gcpOpts.Roles {
		formattedRole := formatRole(role)
		logger.Info("Granting role %s to %s in project %s for %v", formattedRole, gcpOpts.User, gcpOpts.Project, gcpOpts.TTL)
		if p.dryRun {
			logger.Info("[DRY-RUN] Would grant role %s to %s in project %s", formattedRole, gcpOpts.User, gcpOpts.Project)
			continue
		}

		policy, err := p.getIAMPolicy(gcpOpts.Project)
		if err != nil {
			logger.Warn("Failed to get IAM policy for role %s: %v", formattedRole, err)
			grantErrors = append(grantErrors, fmt.Sprintf("role %s: %v", formattedRole, err))
			continue
		}

		binding := p.createBinding(formattedRole, member, gcpOpts.TTL)
		policy.Bindings = append(policy.Bindings, binding)

		if err := p.setIAMPolicy(gcpOpts.Project, policy); err != nil {
			logger.Warn("Failed to set IAM policy for role %s: %v", formattedRole, err)
			grantErrors = append(grantErrors, fmt.Sprintf("role %s: %v", formattedRole, err))
			continue
		}

		// Track successfully granted roles and their binding IDs
		p.grantedRoles = append(p.grantedRoles, GrantedRole{
			Role:      formattedRole,
			BindingID: binding.Condition.Title,
		})
	}

	if len(grantErrors) > 0 {
		if len(p.grantedRoles) == 0 {
			// If no roles were granted, return an error
			return fmt.Errorf("failed to grant any roles: %s", strings.Join(grantErrors, "; "))
		}
		// If some roles were granted, just log the errors
		logger.Warn("Failed to grant some roles: %s", strings.Join(grantErrors, "; "))
	}

	return nil
}

// Revoke revokes temporary access from the specified roles in the specified project
func (p *GCPProvider) Revoke(opts Options) error {
	gcpOpts, ok := opts.(*GCPOptions)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	// Use only the successfully granted roles for revocation
	if len(p.grantedRoles) == 0 {
		logger.Info("No roles to revoke")
		return nil
	}

	var revokeErrors []string
	member := formatMember(gcpOpts.User)

	for _, grantedRole := range p.grantedRoles {
		logger.Info("Revoking role %s from %s in project %s", grantedRole.Role, gcpOpts.User, gcpOpts.Project)
		if p.dryRun {
			logger.Info("[DRY-RUN] Would revoke role %s from %s in project %s", grantedRole.Role, gcpOpts.User, gcpOpts.Project)
			continue
		}

		policy, err := p.getIAMPolicy(gcpOpts.Project)
		if err != nil {
			logger.Warn("Failed to get IAM policy for role %s: %v", grantedRole.Role, err)
			revokeErrors = append(revokeErrors, fmt.Sprintf("role %s: %v", grantedRole.Role, err))
			continue
		}

		for i, binding := range policy.Bindings {
			// Only remove bindings that match both the role and the binding ID from this execution
			if binding.Role == grantedRole.Role && binding.Condition != nil && binding.Condition.Title == grantedRole.BindingID {
				newMembers := make([]string, 0)
				for _, m := range binding.Members {
					if m != member {
						newMembers = append(newMembers, m)
					}
				}
				if len(newMembers) == 0 {
					// Remove the entire binding if there are no members left
					policy.Bindings = append(policy.Bindings[:i], policy.Bindings[i+1:]...)
				} else {
					binding.Members = newMembers
				}
				break
			}
		}

		if err := p.setIAMPolicy(gcpOpts.Project, policy); err != nil {
			logger.Warn("Failed to set IAM policy for role %s: %v", grantedRole.Role, err)
			revokeErrors = append(revokeErrors, fmt.Sprintf("role %s: %v", grantedRole.Role, err))
			continue
		}
	}

	if len(revokeErrors) > 0 {
		logger.Warn("Failed to revoke some roles: %s", strings.Join(revokeErrors, "; "))
	}

	return nil
}

// ListTemporaryBindings lists temporary bindings for the specified project
func (p *GCPProvider) ListTemporaryBindings(opts Options) error {
	gcpOpts, ok := opts.(*GCPOptions)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	policy, err := p.getIAMPolicy(gcpOpts.Project)
	if err != nil {
		return fmt.Errorf("failed to get IAM policy: %v", err)
	}

	found := false
	for _, binding := range policy.Bindings {
		// Only show bindings with our condition title prefix
		if binding.Condition == nil || !strings.HasPrefix(binding.Condition.Title, gcpBindingTitlePrefix) {
			continue
		}

		for _, member := range binding.Members {
			if strings.HasPrefix(member, "user:") && (gcpOpts.User == "" || member == formatMember(gcpOpts.User)) {
				found = true
				logger.Info("Found temporary binding: Role=%s, Member=%s, Expires=%s, ID=%s",
					binding.Role,
					member,
					strings.TrimPrefix(strings.TrimPrefix(binding.Condition.Expression, "request.time < timestamp('"), "')"),
					binding.Condition.Title,
				)
			}
		}
	}

	if !found {
		logger.Info("No temporary bindings found")
	}

	return nil
}

// CleanTemporaryBindings lists and optionally removes temporary bindings for the specified project
func (p *GCPProvider) CleanTemporaryBindings(opts Options) error {
	gcpOpts, ok := opts.(*GCPOptions)
	if !ok {
		return fmt.Errorf("invalid options type")
	}

	policy, err := p.getIAMPolicy(gcpOpts.Project)
	if err != nil {
		return fmt.Errorf("failed to get IAM policy: %v", err)
	}

	// First, find all temporary bindings
	var bindings []temporaryBinding

	for i, binding := range policy.Bindings {
		// Only process bindings with our condition title prefix
		if binding.Condition == nil || !strings.HasPrefix(binding.Condition.Title, gcpBindingTitlePrefix) {
			continue
		}

		for _, member := range binding.Members {
			if strings.HasPrefix(member, "user:") && (gcpOpts.User == "" || member == formatMember(gcpOpts.User)) {
				bindings = append(bindings, temporaryBinding{
					Role:      binding.Role,
					Member:    member,
					BindingID: binding.Condition.Title,
					Index:     i,
				})
			}
		}
	}

	if len(bindings) == 0 {
		logger.Info("No temporary bindings found")
		return nil
	}

	// List all bindings that will be affected
	for _, binding := range bindings {
		if p.dryRun {
			logger.Info("[DRY-RUN] Would remove binding: Role=%s, Member=%s, ID=%s",
				binding.Role,
				binding.Member,
				binding.BindingID,
			)
		} else {
			logger.Info("Found binding to remove: Role=%s, Member=%s, ID=%s",
				binding.Role,
				binding.Member,
				binding.BindingID,
			)
		}
	}

	if p.dryRun {
		return nil
	}

	// Remove the bindings
	// We need to process them in reverse order to avoid index shifting
	for i := len(bindings) - 1; i >= 0; i-- {
		binding := bindings[i]
		logger.Info("Removing binding: Role=%s, Member=%s", binding.Role, binding.Member)

		// Get the binding from the policy
		policyBinding := policy.Bindings[binding.Index]

		// Remove the member from the binding
		newMembers := make([]string, 0)
		for _, m := range policyBinding.Members {
			if m != binding.Member {
				newMembers = append(newMembers, m)
			}
		}

		if len(newMembers) == 0 {
			// Remove the entire binding if there are no members left
			policy.Bindings = append(policy.Bindings[:binding.Index], policy.Bindings[binding.Index+1:]...)
		} else {
			policyBinding.Members = newMembers
		}
	}

	if err := p.setIAMPolicy(gcpOpts.Project, policy); err != nil {
		return fmt.Errorf("failed to update IAM policy: %v", err)
	}

	logger.Info("Successfully cleaned up %d temporary binding(s)", len(bindings))
	return nil
}
