package task

import (
	"context"
	"errors"

	"github.com/influxdata/platform"
	platcontext "github.com/influxdata/platform/context"
)

var ErrUnauthorized = errors.New("unauthorized")

type taskServiceValidator struct {
	platform.TaskService
}

func NewValidator(ts platform.TaskService) platform.TaskService {
	return &taskServiceValidator{
		TaskService: ts,
	}
}

func (ts *taskServiceValidator) CreateTask(ctx context.Context, t *platform.Task) error {
	if err := checkPermission(ctx, platform.Permission{Action: platform.CreateAction, Resource: platform.TaskResource}); err != nil {
		return err
	}

	return ts.TaskService.CreateTask(ctx, t)
}

// TODO(lh): add permission checking for the all the platform.TaskService functions.

func checkPermission(ctx context.Context, perm platform.Permission) error {
	auth, err := platcontext.GetAuthorization(ctx)
	if err != nil {
		return err
	}

	if !platform.Allowed(perm, auth.Permissions) {
		return ErrUnauthorized
	}

	return nil
}
