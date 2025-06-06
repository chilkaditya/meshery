package kubernetes

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/meshery/meshery/server/machines"
	"github.com/meshery/meshery/server/models"
	"github.com/meshery/meshery/server/models/meshmodel/core"
	"github.com/meshery/meshkit/models/events"
)

type RegisterAction struct{}

// Execute On Entry and Exit should not return next eventtype i suppose, look again.
func (ra *RegisterAction) ExecuteOnEntry(ctx context.Context, machineCtx interface{}, data interface{}) (machines.EventType, *events.Event, error) {
	return machines.NoOp, nil, nil
}

func (ra *RegisterAction) Execute(ctx context.Context, machineCtx interface{}, data interface{}) (machines.EventType, *events.Event, error) {
	user, _ := ctx.Value(models.UserCtxKey).(*models.User)
	sysID, _ := ctx.Value(models.SystemIDKey).(*uuid.UUID)
	userUUID := uuid.FromStringOrNil(user.ID)
	provider, _ := ctx.Value(models.ProviderCtxKey).(models.Provider)
	eventBuilder := events.NewEvent().ActedUpon(uuid.Nil).WithCategory("connection").WithAction("register").FromSystem(*sysID).FromUser(userUUID).WithDescription("Failed to interact with the connection.")

	machinectx, err := GetMachineCtx(machineCtx, eventBuilder)
	if err != nil {
		eventBuilder.WithMetadata(map[string]interface{}{"error": err})
		return machines.NoOp, eventBuilder.Build(), err
	}

	machinectx.log.Debug("executing ping test for connection", machinectx.K8sContext.ConnectionID)
	err = machinectx.K8sContext.PingTest()

	if err != nil {
		eventBuilder.WithDescription(fmt.Sprintf("Unable to ping kubernetes context %s at %s", machinectx.K8sContext.Name, machinectx.K8sContext.Server)).WithMetadata(map[string]interface{}{"error": err})
		machinectx.log.Error(err)
		return machines.NotFound, eventBuilder.Build(), err
	}

	context := []*models.K8sContext{&machinectx.K8sContext}

	machinectx.K8sCompRegHelper.UpdateContexts(context).RegisterComponents(context, []models.K8sRegistrationFunction{core.RegisterK8sMeshModelComponents}, machinectx.RegistryManager, machinectx.EventBroadcaster, provider, user.ID, true)

	return machines.Connect, nil, nil
}

func (ra *RegisterAction) ExecuteOnExit(ctx context.Context, machineCtx interface{}, data interface{}) (machines.EventType, *events.Event, error) {
	return machines.NoOp, nil, nil
}
