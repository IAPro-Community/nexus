package computer

import (
	"context"
	"errors"
)

var ErrUnavailable = errors.New("computer-use provider unavailable")

type Provider interface {
	Capabilities(context.Context) (Capabilities, error)
	Permissions(context.Context) (PermissionStatus, error)
	ListApps(context.Context) ([]AppInfo, error)
	ListWindows(context.Context, string) ([]WindowInfo, error)
	Snapshot(context.Context, SnapshotRequest) (Snapshot, error)
	Act(context.Context, ActionRequest) (ActionResult, error)
}

type UnavailableProvider struct {
	Platform string
	Reason   string
}

func (p UnavailableProvider) Capabilities(context.Context) (Capabilities, error) {
	return Capabilities{
		Provider:        "unavailable",
		ProtocolVersion: 1,
		Platform:        p.Platform,
	}, nil
}

func (p UnavailableProvider) Permissions(context.Context) (PermissionStatus, error) {
	return PermissionStatus{
		Available:        false,
		Accessibility:   "UNKNOWN",
		ScreenCapture:   "UNKNOWN",
		InputMonitoring: "UNKNOWN",
		Reason:          p.Reason,
	}, nil
}

func (p UnavailableProvider) ListApps(context.Context) ([]AppInfo, error) {
	return nil, ErrUnavailable
}

func (p UnavailableProvider) ListWindows(context.Context, string) ([]WindowInfo, error) {
	return nil, ErrUnavailable
}

func (p UnavailableProvider) Snapshot(context.Context, SnapshotRequest) (Snapshot, error) {
	return Snapshot{}, ErrUnavailable
}

func (p UnavailableProvider) Act(context.Context, ActionRequest) (ActionResult, error) {
	return ActionResult{}, ErrUnavailable
}
