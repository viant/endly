package ios

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/viant/endly"
)

func (s *service) destinationRegister(ctx *endly.Context, request *DestinationRegisterRequest) (*DestinationRegisterResponse, error) {
	if err := request.Validate(); err != nil {
		return nil, err
	}
	key := "ios:external:" + strings.ToLower(strings.TrimSpace(request.Provider)) + ":" + strings.TrimSpace(request.DeviceID)
	processLease, err := s.leaseStore.Acquire(ctx.Background(), key)
	if err != nil {
		return nil, err
	}
	lease := DestinationLease{
		ID: uuid.NewString(), Fence: processLease.Fence, UDID: request.DeviceID,
		Name: request.Provider + ":" + request.DeviceID, Runtime: request.PlatformVersion,
		Kind: "external", PreserveOnRelease: true, ProcessLease: processLease,
	}
	s.storeLease(lease)
	s.cleanupStack(ctx).Push("external-destination:"+lease.ID, func(cleanupCtx context.Context) error {
		_, err := s.releaseExternalDestination(cleanupCtx, lease)
		return err
	})
	return &DestinationRegisterResponse{Lease: lease}, nil
}

func (s *service) destinationRelease(ctx *endly.Context, request *DestinationReleaseRequest) (*DestinationReleaseResponse, error) {
	return s.releaseExternalDestination(ctx.Background(), request.Lease)
}

func (s *service) releaseExternalDestination(_ context.Context, lease DestinationLease) (*DestinationReleaseResponse, error) {
	s.mu.Lock()
	stored, ok := s.leases[lease.ID]
	s.mu.Unlock()
	if !ok {
		return &DestinationReleaseResponse{Warning: "external destination already released or unknown"}, nil
	}
	if !stored.IsExternal() || stored.Fence != lease.Fence || stored.UDID != lease.UDID {
		return nil, fmt.Errorf("external iOS destination lease fence mismatch")
	}
	if stored.ProcessLease != nil {
		if lease.ProcessLease == nil || stored.ProcessLease.Token != lease.ProcessLease.Token {
			return nil, fmt.Errorf("external iOS destination persistent lease token mismatch")
		}
		if err := s.leaseStore.Validate(stored.ProcessLease); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	delete(s.leases, lease.ID)
	s.mu.Unlock()
	if err := s.leaseStore.Release(stored.ProcessLease); err != nil {
		return nil, err
	}
	return &DestinationReleaseResponse{Released: true}, nil
}
