//go:build unit

package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type registrationAbuseStoreStub struct {
	finding     *registrationAbuseFinding
	findingErr  error
	resetErr    error
	markErr     error
	disableErr  error
	events      []registrationAbuseEvent
	resetIDs    []int64
	markedIDs   []int64
	disabledIDs []int64
}

func (s *registrationAbuseStoreStub) RecordAndEvaluate(_ context.Context, event registrationAbuseEvent) (*registrationAbuseFinding, error) {
	s.events = append(s.events, event)
	return s.finding, s.findingErr
}

func (s *registrationAbuseStoreStub) ResetInviterQuota(_ context.Context, inviterID int64) error {
	s.resetIDs = append(s.resetIDs, inviterID)
	return s.resetErr
}

func (s *registrationAbuseStoreStub) MarkAccountsDeleted(_ context.Context, userIDs []int64) error {
	s.markedIDs = append(s.markedIDs, userIDs...)
	return s.markErr
}

func (s *registrationAbuseStoreStub) DisableAccounts(_ context.Context, userIDs []int64) error {
	s.disabledIDs = append(s.disabledIDs, userIDs...)
	return s.disableErr
}

type registrationAbuseDeleterStub struct {
	deleted []int64
	errByID map[int64]error
}

func (s *registrationAbuseDeleterStub) DeleteUser(_ context.Context, id int64) error {
	s.deleted = append(s.deleted, id)
	return s.errByID[id]
}

func TestRegistrationAbuseServiceBlocksAndCleansConfirmedBatch(t *testing.T) {
	inviterID := int64(90)
	store := &registrationAbuseStoreStub{
		finding: &registrationAbuseFinding{
			Blocked:           true,
			Signals:           []string{"inviter_burst_10m"},
			OffendingUserIDs:  []int64{3, 2, 2, 1},
			InviterID:         &inviterID,
			InviterQuotaReset: true,
		},
	}
	deleter := &registrationAbuseDeleterStub{}
	svc := &RegistrationAbuseService{store: store, userDeleter: deleter}

	decision, err := svc.ProcessSuccessfulRegistration(context.Background(), RegistrationAbuseEventInput{
		UserID:      3,
		Email:       "abuse@example.com",
		ClientIP:    "203.0.113.7",
		Fingerprint: "browser-fingerprint-123",
		InviterID:   &inviterID,
	})

	require.NoError(t, err)
	require.True(t, decision.Blocked)
	require.Equal(t, []string{"inviter_burst_10m"}, decision.Signals)
	require.Equal(t, []int64{3, 2, 1}, decision.DeletedUserIDs)
	require.Equal(t, []int64{3, 2, 1}, deleter.deleted)
	require.Equal(t, []int64{90, 90}, store.resetIDs)
	require.Equal(t, []int64{3, 2, 1}, store.markedIDs)
	require.Empty(t, store.disabledIDs)
	require.True(t, decision.InviterQuotaReset)
	require.False(t, decision.CleanupDegraded)
	require.Len(t, store.events, 1)
	require.NotEmpty(t, store.events[0].ClientIPHash)
	require.NotEqual(t, "203.0.113.7", store.events[0].ClientIPHash)
}

func TestRegistrationAbuseServiceLeavesNormalRegistrationUntouched(t *testing.T) {
	store := &registrationAbuseStoreStub{finding: &registrationAbuseFinding{}}
	deleter := &registrationAbuseDeleterStub{}
	svc := &RegistrationAbuseService{store: store, userDeleter: deleter}

	decision, err := svc.ProcessSuccessfulRegistration(context.Background(), RegistrationAbuseEventInput{
		UserID: 7,
		Email:  "normal@example.com",
	})

	require.NoError(t, err)
	require.False(t, decision.Blocked)
	require.Empty(t, deleter.deleted)
	require.Empty(t, store.resetIDs)
	require.Empty(t, store.markedIDs)
}

func TestRegistrationAbuseServiceDisablesAccountsWhenDeletionFails(t *testing.T) {
	store := &registrationAbuseStoreStub{
		finding: &registrationAbuseFinding{
			Blocked:          true,
			Signals:          []string{"ip_burst_10m"},
			OffendingUserIDs: []int64{8, 9},
		},
	}
	deleter := &registrationAbuseDeleterStub{
		errByID: map[int64]error{8: errors.New("database unavailable")},
	}
	svc := &RegistrationAbuseService{store: store, userDeleter: deleter}

	decision, err := svc.ProcessSuccessfulRegistration(context.Background(), RegistrationAbuseEventInput{UserID: 8})

	require.NoError(t, err)
	require.True(t, decision.Blocked)
	require.True(t, decision.CleanupDegraded)
	require.Equal(t, []int64{9}, decision.DeletedUserIDs)
	require.Equal(t, []int64{8}, store.disabledIDs)
}

func TestRegistrationAbuseServiceReportsQuotaResetFailure(t *testing.T) {
	inviterID := int64(42)
	store := &registrationAbuseStoreStub{
		finding: &registrationAbuseFinding{
			Blocked:           true,
			Signals:           []string{"inviter_burst_1h"},
			OffendingUserIDs:  []int64{11},
			InviterID:         &inviterID,
			InviterQuotaReset: true,
		},
		resetErr: errors.New("reset failed"),
	}
	svc := &RegistrationAbuseService{
		store:       store,
		userDeleter: &registrationAbuseDeleterStub{},
	}

	decision, err := svc.ProcessSuccessfulRegistration(context.Background(), RegistrationAbuseEventInput{UserID: 11})

	require.NoError(t, err)
	require.True(t, decision.Blocked)
	require.True(t, decision.CleanupDegraded)
	require.False(t, decision.InviterQuotaReset)
}

func TestRegistrationAbuseServiceCleanupRejectedRegistrationDisablesOnDeleteFailure(t *testing.T) {
	store := &registrationAbuseStoreStub{}
	deleter := &registrationAbuseDeleterStub{
		errByID: map[int64]error{12: errors.New("delete failed")},
	}
	svc := &RegistrationAbuseService{store: store, userDeleter: deleter}

	err := svc.CleanupRejectedRegistration(context.Background(), 12)

	require.Error(t, err)
	require.Equal(t, []int64{12}, store.disabledIDs)
}
