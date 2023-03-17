/*
Copyright 2022. projectsveltos.io. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package evaluation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	logs "github.com/projectsveltos/libsveltos/lib/logsettings"
)

// Those are used only for uts

// Classifier
var (
	IsVersionAMatch            = (*manager).isVersionAMatch
	IsResourceAMatch           = (*manager).isResourceAMatch
	CleanClassifierReport      = (*manager).cleanClassifierReport
	CreateClassifierReport     = (*manager).createClassifierReport
	EvaluateClassifierInstance = (*manager).evaluateClassifierInstance
	BuildList                  = (*manager).buildList
	BuildSortedList            = (*manager).buildSortedList
	GvkInstalled               = (*manager).gvkInstalled
	GetInstalledResources      = (*manager).getInstalledResources
	StartWatcher               = (*manager).startWatcher
	UpdateWatchers             = (*manager).updateWatchers
	GetManamegentClusterClient = (*manager).getManamegentClusterClient
	SendClassifierReport       = (*manager).sendClassifierReport
)

// healthCheck
var (
	GetResourceHealthStatus   = (*manager).getResourceHealthStatus
	FetchHealthCheckResources = (*manager).fetchHealthCheckResources
	GetHealthStatus           = (*manager).getHealthStatus
	CreateHealthCheckReport   = (*manager).createHealthCheckReport
	SendHealthCheckReport     = (*manager).sendHealthCheckReport
	CleanHealthCheckReport    = (*manager).cleanHealthCheckReport
)

// eventSource
var (
	GetEventMatchingResources  = (*manager).getEventMatchingResources
	FetchEventSourceResources  = (*manager).fetchEventSourceResources
	IsMatchForEventSource      = (*manager).isMatchForEventSource
	IsMatchForClassifierScript = (*manager).isMatchForClassifierScript
	CreateEventReport          = (*manager).createEventReport
	SendEventReport            = (*manager).sendEventReport
	CleanEventReport           = (*manager).cleanEventReport
	MarshalSliceOfUnstructured = (*manager).marshalSliceOfUnstructured
)

func Reset() {
	managerInstance = nil
}

func GetWatchers() map[schema.GroupVersionKind]context.CancelFunc {
	return managerInstance.watchers
}

func GetUnknownResourcesToWatch() []schema.GroupVersionKind {
	return managerInstance.unknownResourcesToWatch
}

func InitializeManagerWithSkip(ctx context.Context, l logr.Logger, config *rest.Config, c client.Client,
	intervalInSecond uint) {

	// Used only for testing purposes (so to avoid using testEnv when not required by test)
	if managerInstance == nil {
		getManagerLock.Lock()
		defer getManagerLock.Unlock()
		if managerInstance == nil {
			l.V(logs.LogInfo).Info(fmt.Sprintf("Creating manager now. Interval (in seconds): %d", intervalInSecond))
			managerInstance = &manager{log: l, Client: c, config: config}
			managerInstance.classifierJobQueue = make(map[string]bool)
			managerInstance.healthCheckJobQueue = make(map[string]bool)
			managerInstance.interval = time.Duration(intervalInSecond) * time.Second
			managerInstance.mu = &sync.Mutex{}

			managerInstance.resourcesToWatch = make([]schema.GroupVersionKind, 0)
			managerInstance.rebuildResourceToWatch = 0
			managerInstance.watchMu = &sync.Mutex{}

			managerInstance.unknownResourcesToWatch = make([]schema.GroupVersionKind, 0)

			managerInstance.watchers = make(map[schema.GroupVersionKind]context.CancelFunc)

			go managerInstance.evaluateClassifiers(ctx)
			go managerInstance.evaluateHealthChecks(ctx)
			go managerInstance.buildResourceToWatch(ctx)
			// Do not start a watcher for CustomResourceDefinition. Meant to be used by ut only
			// go managerInstance.watchCustomResourceDefinition(ctx)
		}
	}
}
