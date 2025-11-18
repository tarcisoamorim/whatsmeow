package whatsapp

import (
	"context"
	"fmt"
	"time"

	"go.mau.fi/whatsmeow/types"
	"go.uber.org/zap"

	pkglogger "github.com/tarcisoamorim/whatsmeow/meta-adapter/pkg/logger"
)

// CreateGroup creates a new WhatsApp group
func (m *Manager) CreateGroup(ctx context.Context, tenantID, instanceID, name string, participants []string) (*types.GroupInfo, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return nil, fmt.Errorf("instance is not connected")
	}

	// Parse participant JIDs
	participantJIDs := make([]types.JID, len(participants))
	for i, participant := range participants {
		jid, err := m.parseJID(participant)
		if err != nil {
			return nil, fmt.Errorf("invalid participant %s: %w", participant, err)
		}
		participantJIDs[i] = jid
	}

	// Create group using whatsmeow
	groupInfo, err := clientInstance.Client.CreateGroup(types.GroupCreateRequest{
		Name:         name,
		Participants: participantJIDs,
	})
	if err != nil {
		logger.Error("Failed to create group", zap.Error(err))
		return nil, fmt.Errorf("failed to create group: %w", err)
	}

	logger.Info("Group created",
		zap.String("group_jid", groupInfo.JID.String()),
		zap.String("name", name),
		zap.Int("participants", len(participants)),
	)

	return groupInfo, nil
}

// GetGroupInfo retrieves information about a group
func (m *Manager) GetGroupInfo(ctx context.Context, tenantID, instanceID, groupJID string) (*types.GroupInfo, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return nil, fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return nil, err
	}

	// Get group info using whatsmeow
	groupInfo, err := clientInstance.Client.GetGroupInfo(jid)
	if err != nil {
		logger.Error("Failed to get group info", zap.Error(err), zap.String("group_jid", groupJID))
		return nil, fmt.Errorf("failed to get group info: %w", err)
	}

	logger.Info("Retrieved group info",
		zap.String("group_jid", groupJID),
		zap.String("name", groupInfo.Name),
	)

	return groupInfo, nil
}

// GetJoinedGroups retrieves all groups the user is a member of
func (m *Manager) GetJoinedGroups(ctx context.Context, tenantID, instanceID string) ([]*types.GroupInfo, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return nil, fmt.Errorf("instance is not connected")
	}

	// Get joined groups using whatsmeow
	groups, err := clientInstance.Client.GetJoinedGroups()
	if err != nil {
		logger.Error("Failed to get joined groups", zap.Error(err))
		return nil, fmt.Errorf("failed to get joined groups: %w", err)
	}

	logger.Info("Retrieved joined groups",
		zap.Int("count", len(groups)),
	)

	return groups, nil
}

// LeaveGroup leaves a group
func (m *Manager) LeaveGroup(ctx context.Context, tenantID, instanceID, groupJID string) error {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return err
	}

	// Leave group using whatsmeow
	err = clientInstance.Client.LeaveGroup(jid)
	if err != nil {
		logger.Error("Failed to leave group", zap.Error(err), zap.String("group_jid", groupJID))
		return fmt.Errorf("failed to leave group: %w", err)
	}

	logger.Info("Left group",
		zap.String("group_jid", groupJID),
	)

	return nil
}

// JoinGroupWithLink joins a group using an invite link
func (m *Manager) JoinGroupWithLink(ctx context.Context, tenantID, instanceID, inviteCode string) (types.JID, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return types.EmptyJID, fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return types.EmptyJID, fmt.Errorf("instance is not connected")
	}

	// Join group using invite link
	groupJID, err := clientInstance.Client.JoinGroupWithLink(inviteCode)
	if err != nil {
		logger.Error("Failed to join group with link", zap.Error(err), zap.String("invite_code", inviteCode))
		return types.EmptyJID, fmt.Errorf("failed to join group: %w", err)
	}

	logger.Info("Joined group with link",
		zap.String("group_jid", groupJID.String()),
		zap.String("invite_code", inviteCode),
	)

	return groupJID, nil
}

// AddParticipants adds participants to a group
func (m *Manager) AddParticipants(ctx context.Context, tenantID, instanceID, groupJID string, participants []string) ([]types.GroupParticipant, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return nil, fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return nil, err
	}

	// Parse participant JIDs
	participantJIDs := make([]types.JID, len(participants))
	for i, participant := range participants {
		participantJID, err := m.parseJID(participant)
		if err != nil {
			return nil, fmt.Errorf("invalid participant %s: %w", participant, err)
		}
		participantJIDs[i] = participantJID
	}

	// Add participants using whatsmeow
	results, err := clientInstance.Client.UpdateGroupParticipants(jid, participantJIDs, types.ParticipantChangeAdd)
	if err != nil {
		logger.Error("Failed to add participants", zap.Error(err), zap.String("group_jid", groupJID))
		return nil, fmt.Errorf("failed to add participants: %w", err)
	}

	logger.Info("Added participants",
		zap.String("group_jid", groupJID),
		zap.Int("count", len(participants)),
	)

	return results, nil
}

// RemoveParticipants removes participants from a group
func (m *Manager) RemoveParticipants(ctx context.Context, tenantID, instanceID, groupJID string, participants []string) ([]types.GroupParticipant, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return nil, fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return nil, err
	}

	// Parse participant JIDs
	participantJIDs := make([]types.JID, len(participants))
	for i, participant := range participants {
		participantJID, err := m.parseJID(participant)
		if err != nil {
			return nil, fmt.Errorf("invalid participant %s: %w", participant, err)
		}
		participantJIDs[i] = participantJID
	}

	// Remove participants using whatsmeow
	results, err := clientInstance.Client.UpdateGroupParticipants(jid, participantJIDs, types.ParticipantChangeRemove)
	if err != nil {
		logger.Error("Failed to remove participants", zap.Error(err), zap.String("group_jid", groupJID))
		return nil, fmt.Errorf("failed to remove participants: %w", err)
	}

	logger.Info("Removed participants",
		zap.String("group_jid", groupJID),
		zap.Int("count", len(participants)),
	)

	return results, nil
}

// PromoteParticipants promotes participants to admin status
func (m *Manager) PromoteParticipants(ctx context.Context, tenantID, instanceID, groupJID string, participants []string) ([]types.GroupParticipant, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return nil, fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return nil, err
	}

	// Parse participant JIDs
	participantJIDs := make([]types.JID, len(participants))
	for i, participant := range participants {
		participantJID, err := m.parseJID(participant)
		if err != nil {
			return nil, fmt.Errorf("invalid participant %s: %w", participant, err)
		}
		participantJIDs[i] = participantJID
	}

	// Promote participants using whatsmeow
	results, err := clientInstance.Client.UpdateGroupParticipants(jid, participantJIDs, types.ParticipantChangePromote)
	if err != nil {
		logger.Error("Failed to promote participants", zap.Error(err), zap.String("group_jid", groupJID))
		return nil, fmt.Errorf("failed to promote participants: %w", err)
	}

	logger.Info("Promoted participants",
		zap.String("group_jid", groupJID),
		zap.Int("count", len(participants)),
	)

	return results, nil
}

// DemoteParticipants demotes participants from admin status
func (m *Manager) DemoteParticipants(ctx context.Context, tenantID, instanceID, groupJID string, participants []string) ([]types.GroupParticipant, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return nil, fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return nil, err
	}

	// Parse participant JIDs
	participantJIDs := make([]types.JID, len(participants))
	for i, participant := range participants {
		participantJID, err := m.parseJID(participant)
		if err != nil {
			return nil, fmt.Errorf("invalid participant %s: %w", participant, err)
		}
		participantJIDs[i] = participantJID
	}

	// Demote participants using whatsmeow
	results, err := clientInstance.Client.UpdateGroupParticipants(jid, participantJIDs, types.ParticipantChangeDemote)
	if err != nil {
		logger.Error("Failed to demote participants", zap.Error(err), zap.String("group_jid", groupJID))
		return nil, fmt.Errorf("failed to demote participants: %w", err)
	}

	logger.Info("Demoted participants",
		zap.String("group_jid", groupJID),
		zap.Int("count", len(participants)),
	)

	return results, nil
}

// SetGroupName updates the group name
func (m *Manager) SetGroupName(ctx context.Context, tenantID, instanceID, groupJID, name string) error {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return err
	}

	// Set group name using whatsmeow
	err = clientInstance.Client.SetGroupName(jid, name)
	if err != nil {
		logger.Error("Failed to set group name", zap.Error(err), zap.String("group_jid", groupJID))
		return fmt.Errorf("failed to set group name: %w", err)
	}

	logger.Info("Set group name",
		zap.String("group_jid", groupJID),
		zap.String("name", name),
	)

	return nil
}

// SetGroupTopic updates the group description/topic
func (m *Manager) SetGroupTopic(ctx context.Context, tenantID, instanceID, groupJID, topic string) error {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return err
	}

	// Set group topic using whatsmeow
	err = clientInstance.Client.SetGroupTopic(jid, "", "", topic)
	if err != nil {
		logger.Error("Failed to set group topic", zap.Error(err), zap.String("group_jid", groupJID))
		return fmt.Errorf("failed to set group topic: %w", err)
	}

	logger.Info("Set group topic",
		zap.String("group_jid", groupJID),
		zap.String("topic", topic),
	)

	return nil
}

// SetGroupPhoto updates the group photo
func (m *Manager) SetGroupPhoto(ctx context.Context, tenantID, instanceID, groupJID string, imageData []byte) (string, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return "", fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return "", err
	}

	// Set group photo using whatsmeow
	pictureID, err := clientInstance.Client.SetGroupPhoto(jid, imageData)
	if err != nil {
		logger.Error("Failed to set group photo", zap.Error(err), zap.String("group_jid", groupJID))
		return "", fmt.Errorf("failed to set group photo: %w", err)
	}

	logger.Info("Set group photo",
		zap.String("group_jid", groupJID),
		zap.String("picture_id", pictureID),
	)

	return pictureID, nil
}

// SetGroupAnnounce sets whether only admins can send messages
func (m *Manager) SetGroupAnnounce(ctx context.Context, tenantID, instanceID, groupJID string, announce bool) error {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return err
	}

	// Set group announce setting using whatsmeow
	err = clientInstance.Client.SetGroupAnnounce(jid, announce)
	if err != nil {
		logger.Error("Failed to set group announce", zap.Error(err), zap.String("group_jid", groupJID))
		return fmt.Errorf("failed to set group announce: %w", err)
	}

	logger.Info("Set group announce",
		zap.String("group_jid", groupJID),
		zap.Bool("announce", announce),
	)

	return nil
}

// SetGroupLocked sets whether only admins can edit group info
func (m *Manager) SetGroupLocked(ctx context.Context, tenantID, instanceID, groupJID string, locked bool) error {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return err
	}

	// Set group locked setting using whatsmeow
	err = clientInstance.Client.SetGroupLocked(jid, locked)
	if err != nil {
		logger.Error("Failed to set group locked", zap.Error(err), zap.String("group_jid", groupJID))
		return fmt.Errorf("failed to set group locked: %w", err)
	}

	logger.Info("Set group locked",
		zap.String("group_jid", groupJID),
		zap.Bool("locked", locked),
	)

	return nil
}

// GetGroupInviteLink gets or resets the group invite link
func (m *Manager) GetGroupInviteLink(ctx context.Context, tenantID, instanceID, groupJID string, reset bool) (string, error) {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return "", fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return "", fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return "", err
	}

	var inviteLink string
	if reset {
		// Reset (revoke and create new) invite link
		inviteLink, err = clientInstance.Client.RevokeGroupInviteLink(jid)
	} else {
		// Get current invite link
		inviteLink, err = clientInstance.Client.GetGroupInviteLink(jid, reset)
	}

	if err != nil {
		logger.Error("Failed to get group invite link", zap.Error(err), zap.String("group_jid", groupJID))
		return "", fmt.Errorf("failed to get group invite link: %w", err)
	}

	logger.Info("Retrieved group invite link",
		zap.String("group_jid", groupJID),
		zap.Bool("reset", reset),
	)

	return inviteLink, nil
}

// SetGroupDisappearingTimer sets the disappearing message timer for a group
func (m *Manager) SetGroupDisappearingTimer(ctx context.Context, tenantID, instanceID, groupJID string, timer time.Duration) error {
	logger := pkglogger.Get()

	clientInstance, err := m.GetOrCreateClient(ctx, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to get client: %w", err)
	}

	if !clientInstance.Client.IsConnected() {
		return fmt.Errorf("instance is not connected")
	}

	jid, err := m.parseJID(groupJID)
	if err != nil {
		return err
	}

	// Set disappearing timer for group
	err = clientInstance.Client.SetDisappearingTimer(ctx, jid, timer, time.Now())
	if err != nil {
		logger.Error("Failed to set group disappearing timer", zap.Error(err), zap.String("group_jid", groupJID))
		return fmt.Errorf("failed to set group disappearing timer: %w", err)
	}

	logger.Info("Set group disappearing timer",
		zap.String("group_jid", groupJID),
		zap.Duration("timer", timer),
	)

	return nil
}
