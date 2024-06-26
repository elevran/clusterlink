// Copyright (c) The ClusterLink Authors.
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

package authz

import (
	"context"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/clusterlink-net/clusterlink/pkg/apis/clusterlink.net/v1alpha1"
	"github.com/clusterlink-net/clusterlink/pkg/controlplane/authz/connectivitypdp"
	"github.com/clusterlink-net/clusterlink/pkg/util/controller"
)

// CreateControllers creates the various k8s controllers used to update the xDS manager.
func CreateControllers(mgr *Manager, controllerManager ctrl.Manager) error {
	err := controller.AddToManager(controllerManager, &controller.Spec{
		Name:   "authz.access-policy",
		Object: &v1alpha1.AccessPolicy{},
		AddHandler: func(ctx context.Context, object any) error {
			accPolicy := connectivitypdp.PolicyFromCR(object.(*v1alpha1.AccessPolicy))
			return mgr.AddAccessPolicy(accPolicy)
		},
		DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
			return mgr.DeleteAccessPolicy(name, false)
		},
	})
	if err != nil {
		return err
	}

	err = controller.AddToManager(controllerManager, &controller.Spec{
		Name:   "authz.privileged-access-policy",
		Object: &v1alpha1.PrivilegedAccessPolicy{},
		AddHandler: func(_ context.Context, object any) error {
			accPolicy := connectivitypdp.PolicyFromPrivilegedCR(object.(*v1alpha1.PrivilegedAccessPolicy))
			return mgr.AddAccessPolicy(accPolicy)
		},
		DeleteHandler: func(_ context.Context, name types.NamespacedName) error {
			return mgr.DeleteAccessPolicy(name, true)
		},
	})
	if err != nil {
		return err
	}

	err = controller.AddToManager(controllerManager, &controller.Spec{
		Name:   "authz.peer",
		Object: &v1alpha1.Peer{},
		AddHandler: func(ctx context.Context, object any) error {
			mgr.AddPeer(object.(*v1alpha1.Peer))
			return nil
		},
		DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
			mgr.DeletePeer(name.Name)
			return nil
		},
	})
	if err != nil {
		return err
	}

	err = controller.AddToManager(controllerManager, &controller.Spec{
		Name:   "authz.import",
		Object: &v1alpha1.Import{},
		AddHandler: func(ctx context.Context, object any) error {
			return nil
		},
		DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
			return nil
		},
	})
	if err != nil {
		return err
	}

	err = controller.AddToManager(controllerManager, &controller.Spec{
		Name:   "authz.export",
		Object: &v1alpha1.Export{},
		AddHandler: func(ctx context.Context, object any) error {
			return nil
		},
		DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
			return nil
		},
	})
	if err != nil {
		return err
	}

	return controller.AddToManager(controllerManager, &controller.Spec{
		Name:   "authz.pod",
		Object: &v1.Pod{},
		AddHandler: func(ctx context.Context, object any) error {
			mgr.addPod(object.(*v1.Pod))
			return nil
		},
		DeleteHandler: func(ctx context.Context, name types.NamespacedName) error {
			mgr.deletePod(name)
			return nil
		},
	})
}
