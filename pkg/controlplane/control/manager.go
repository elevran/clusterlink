// Copyright 2023 The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package control

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dpapp "github.com/clusterlink-net/clusterlink/cmd/cl-dataplane/app"
	"github.com/clusterlink-net/clusterlink/pkg/api"
	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/util/net"
)

// Manager is responsible for handling control operations,
// which needs to be coordinated across all dataplane/controlplane instances.
// This includes target port generation for imported services, as well as
// k8s service creation per imported service.
type Manager struct {
	client client.Client
	ports  *portManager

	logger *logrus.Entry
}

// AddLegacyExport defines a new route target for ingress dataplane connections.
func (m *Manager) AddLegacyExport(name, namespace string, eSpec *api.ExportSpec) error {
	m.logger.Infof("Adding export '%s'.", name)

	if eSpec.ExternalService != "" && !net.IsIP(eSpec.ExternalService) && !net.IsDNS(eSpec.ExternalService) {
		return fmt.Errorf("the external service %s is not a hostname or an IP address", eSpec.ExternalService)
	}

	// create a k8s external service.
	extName := eSpec.ExternalService
	if extName != "" {
		if net.IsIP(extName) {
			extName += ".nip.io" // Convert IP to DNS address.
		}

		m.logger.Infof("Creating Kubernetes service %s of type ExternalName linked to %s.", eSpec.Service.Host, extName)

		err := m.client.Create(
			context.Background(),
			&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      eSpec.Service.Host,
					Namespace: namespace,
				},
				Spec: v1.ServiceSpec{
					Type:         v1.ServiceTypeExternalName,
					ExternalName: extName,
				},
			})
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteLegacyExport removes the possibility for ingress dataplane connections to access a given service.
func (m *Manager) DeleteLegacyExport(namespace string, exportSpec *api.ExportSpec) error {
	// Deleting a k8s external service.
	if exportSpec.ExternalService != "" {
		err := m.client.Delete(
			context.Background(),
			&v1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      exportSpec.Service.Host,
					Namespace: namespace,
				},
			})
		if err != nil {
			return err
		}
	}

	return nil
}

// AddImport adds a listening socket for an imported remote service.
func (m *Manager) AddImport(ctx context.Context, imp *v1alpha1.Import) error {
	m.logger.Infof("Adding import '%s/%s'.", imp.Namespace, imp.Name)

	// TODO: port manager should map ports to imports, and be able to detect conflicts
	port, err := m.ports.Lease(imp.Spec.TargetPort)
	if err != nil {
		return fmt.Errorf("cannot generate listening port: %w", err)
	}

	imp.Spec.TargetPort = port

	newService := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      imp.Name,
			Namespace: imp.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Protocol:   v1.ProtocolTCP,
					Port:       int32(imp.Spec.Port),
					TargetPort: intstr.FromInt32(int32(imp.Spec.TargetPort)),
				},
			},
			Type:     v1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": dpapp.Name},
		},
	}

	var oldService v1.Service
	err = m.client.Get(
		ctx,
		types.NamespacedName{
			Name:      imp.Name,
			Namespace: imp.Namespace,
		},
		&oldService)
	if err != nil {
		if errors.IsNotFound(err) {
			return m.client.Create(ctx, newService)
		}

		return err
	}

	if serviceChanged(&oldService, newService) {
		return m.client.Update(ctx, newService)
	}

	return nil
}

// DeleteImport removes the listening socket of a previously imported service.
func (m *Manager) DeleteImport(ctx context.Context, imp *v1alpha1.Import) error {
	m.logger.Infof("Deleting import '%s/%s'.", imp.Namespace, imp.Name)
	m.ports.Release(imp.Spec.TargetPort)
	return m.client.Delete(
		ctx,
		&v1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      imp.Name,
				Namespace: imp.Namespace,
			},
		})
}

func serviceChanged(svc1, svc2 *v1.Service) bool {
	if svc1.Spec.Type != svc2.Spec.Type {
		return true
	}

	if len(svc1.Spec.Ports) != len(svc2.Spec.Ports) {
		return true
	}

	for i := 0; i < len(svc1.Spec.Ports); i++ {
		if svc1.Spec.Ports[i].Protocol != svc2.Spec.Ports[i].Protocol {
			return true
		}

		if svc1.Spec.Ports[i].Port != svc2.Spec.Ports[i].Port {
			return true
		}

		if svc1.Spec.Ports[i].TargetPort != svc2.Spec.Ports[i].TargetPort {
			return true
		}
	}

	return false
}

// NewManager returns a new control manager.
func NewManager(cl client.Client) *Manager {
	logger := logrus.WithField("component", "controlplane.control.manager")

	return &Manager{
		client: cl,
		ports:  newPortManager(),
		logger: logger,
	}
}