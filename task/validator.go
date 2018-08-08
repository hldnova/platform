package task

import (
	"context"
	"errors"

	"github.com/influxdata/platform"
	platcontext "github.com/influxdata/platform/context"
)

func NewValidator(ts platform.TaskService) platform.TaskService {
	return &TaskServiceValidator{
		TaskService: ts,
	}
}

type TaskServiceValidator struct {
	platform.TaskService
}

func (ts *TaskServiceValidator) CreateTask(ctx context.Context, t *platform.Task) error {
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
		return errors.New("unauthorized")
	}
}
