/*
 * Copyright (c) 2017, MegaEase
 * All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"context"
	"fmt"

	"go.etcd.io/etcd/api/v3/mvccpb"
	"gopkg.in/yaml.v2"

	"github.com/megaease/easegress/pkg/api"
	"github.com/megaease/easegress/pkg/logger"
	"github.com/megaease/easegress/pkg/object/meshcontroller/layout"
	"github.com/megaease/easegress/pkg/object/meshcontroller/spec"
	"github.com/megaease/easegress/pkg/object/meshcontroller/storage"
	"github.com/megaease/easegress/pkg/supervisor"
)

type (
	// Service is the business layer between mesh and store.
	// It is not concurrently safe, the users need to do it by themselves.
	Service struct {
		superSpec *supervisor.Spec
		spec      *spec.Admin

		store storage.Storage
	}
)

// New creates a service with spec
func New(superSpec *supervisor.Spec) *Service {
	s := &Service{
		superSpec: superSpec,
		spec:      superSpec.ObjectSpec().(*spec.Admin),
		store:     storage.New(superSpec.Name(), superSpec.Super().Cluster()),
	}

	return s
}

// Lock locks all store, it will do cluster panic if failed.
func (s *Service) Lock() {
	err := s.store.Lock()
	if err != nil {
		api.ClusterPanic(err)
	}
}

// Unlock unlocks all store, it will do cluster panic if failed.
func (s *Service) Unlock() {
	err := s.store.Unlock()
	if err != nil {
		api.ClusterPanic(err)
	}
}

// PutServiceSpec writes the service spec
func (s *Service) PutServiceSpec(serviceSpec *spec.Service) {
	buff, err := yaml.Marshal(serviceSpec)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", serviceSpec, err))
	}

	err = s.store.Put(layout.ServiceSpecKey(serviceSpec.Name), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// GetServiceSpec gets the service spec by its name
func (s *Service) GetServiceSpec(serviceName string) *spec.Service {
	serviceSpec, _ := s.GetServiceSpecWithInfo(serviceName)
	return serviceSpec
}

// GetServiceSpecWithInfo gets the service spec by its name
func (s *Service) GetServiceSpecWithInfo(serviceName string) (*spec.Service, *mvccpb.KeyValue) {
	kv, err := s.store.GetRaw(layout.ServiceSpecKey(serviceName))
	if err != nil {
		api.ClusterPanic(err)
	}

	if kv == nil {
		return nil, nil
	}

	serviceSpec := &spec.Service{}
	err = yaml.Unmarshal(kv.Value, serviceSpec)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", string(kv.Value), err))
	}

	return serviceSpec, kv
}

// GetGlobalCanaryHeaders gets the global canary headers
func (s *Service) GetGlobalCanaryHeaders() *spec.GlobalCanaryHeaders {
	globalCanaryHeaders, _ := s.GetGlobalCanaryHeadersWithInfo()
	return globalCanaryHeaders
}

// GetGlobalCanaryHeadersWithInfo gets the global canary headers with information
func (s *Service) GetGlobalCanaryHeadersWithInfo() (*spec.GlobalCanaryHeaders, *mvccpb.KeyValue) {
	kv, err := s.store.GetRaw(layout.GlobalCanaryHeaders())
	if err != nil {
		api.ClusterPanic(err)
	}

	if kv == nil {
		return nil, nil
	}

	globalCanaryHeaders := &spec.GlobalCanaryHeaders{}
	err = yaml.Unmarshal([]byte(kv.Value), globalCanaryHeaders)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", string(kv.Value), err))
	}

	return globalCanaryHeaders, kv
}

// PutGlobalCanaryHeaders puts the global canary headers
func (s *Service) PutGlobalCanaryHeaders(globalCanaryHeaders *spec.GlobalCanaryHeaders) {
	buff, err := yaml.Marshal(globalCanaryHeaders)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", globalCanaryHeaders, err))
	}

	err = s.store.Put(layout.GlobalCanaryHeaders(), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// DeleteServiceSpec deletes service spec by its name
func (s *Service) DeleteServiceSpec(serviceName string) {
	err := s.store.Delete(layout.ServiceSpecKey(serviceName))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// ListServiceSpecs lists services specs
func (s *Service) ListServiceSpecs() []*spec.Service {
	services := []*spec.Service{}
	kvs, err := s.store.GetRawPrefix(layout.ServiceSpecPrefix())
	if err != nil {
		api.ClusterPanic(err)
	}

	for _, v := range kvs {
		serviceSpec := &spec.Service{}
		err := yaml.Unmarshal(v.Value, serviceSpec)
		if err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}
		services = append(services, serviceSpec)
	}

	return services
}

// GetServiceInstanceCert gets one specified service instance's cert
func (s *Service) GetServiceInstanceCert(serviceName, instanceID string) *spec.Certificate {
	value, err := s.store.Get(layout.ServiceInstanceCertKey(serviceName, instanceID))
	if err != nil {
		api.ClusterPanic(err)
	}

	if value == nil {
		return nil
	}

	cert := &spec.Certificate{}
	err = yaml.Unmarshal([]byte(*value), cert)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", *value, err))
	}

	return cert
}

// PutServiceInstanceCert puts one service's instance cert.
func (s *Service) PutServiceInstanceCert(serviceName, instaceID string, cert *spec.Certificate) {
	buff, err := yaml.Marshal(cert)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", cert, err))
	}

	err = s.store.Put(layout.ServiceInstanceCertKey(serviceName, instaceID), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// DelServiceInstanceCert deletes one service's cert.
func (s *Service) DelServiceInstanceCert(serviceName, instanceID string) {
	err := s.store.Delete(layout.ServiceInstanceCertKey(serviceName, instanceID))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// ListServiceCerts lists services certs.
func (s *Service) ListServiceCerts() []*spec.Certificate {
	certs := []*spec.Certificate{}
	values, err := s.store.GetPrefix(layout.AllServiceCertPrefix())
	if err != nil {
		api.ClusterPanic(err)
	}

	for _, v := range values {
		cert := &spec.Certificate{}
		err := yaml.Unmarshal([]byte(v), cert)
		if err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}
		certs = append(certs, cert)
	}

	return certs
}

// ListAllIngressControllerInstanceCerts  gets the ingress controller cert.
func (s *Service) ListAllIngressControllerInstanceCerts() []*spec.Certificate {
	var certs []*spec.Certificate
	values, err := s.store.GetPrefix(layout.AllIngressControllerInstanceCertPrefix())
	if err != nil {
		api.ClusterPanic(err)
	}

	for _, v := range values {
		cert := &spec.Certificate{}
		if err = yaml.Unmarshal([]byte(v), cert); err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}

		certs = append(certs, cert)

	}
	return certs
}

// PutIngressControllerInstanceCert puts the root cert.
func (s *Service) PutIngressControllerInstanceCert(instaceID string, cert *spec.Certificate) {
	buff, err := yaml.Marshal(cert)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", cert, err))
	}

	err = s.store.Put(layout.IngressControllerInstanceCertKey(instaceID), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
	return
}

// DelIngressControllerInstanceCert deletes root cert.
func (s *Service) DelIngressControllerInstanceCert(instanceID string) {
	err := s.store.Delete(layout.IngressControllerInstanceCertKey(instanceID))
	if err != nil {
		api.ClusterPanic(err)
	}
	return
}

// DelAllIngressControllerInstanceCert deletes all ingress controller certs.
func (s *Service) DelAllIngressControllerInstanceCert() {
	err := s.store.DeletePrefix(layout.AllIngressControllerInstanceCertPrefix())
	if err != nil {
		api.ClusterPanic(err)
	}
}

// PutIngressControllerInstanceSpec puts ingress controller's spec
func (s *Service) PutIngressControllerInstanceSpec(instance *spec.ServiceInstanceSpec) {
	buff, err := yaml.Marshal(instance)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", instance, err))
	}

	err = s.store.Put(layout.IngressControllerInstanceSpecKey(instance.InstanceID), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
	return
}

// GetRootCert  gets the root cert.
func (s *Service) GetRootCert() *spec.Certificate {
	value, err := s.store.Get(layout.RootCertKey())
	if err != nil {
		api.ClusterPanic(err)
	}

	if value == nil {
		return nil
	}

	cert := &spec.Certificate{}
	err = yaml.Unmarshal([]byte(*value), cert)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", *value, err))
	}

	return cert
}

// PutRootCert puts the root cert.
func (s *Service) PutRootCert(cert *spec.Certificate) {
	buff, err := yaml.Marshal(cert)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", cert, err))
	}

	err = s.store.Put(layout.RootCertKey(), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
	return
}

//  DelRootCert deletes root cert.
func (s *Service) DelRootCert() {
	err := s.store.Delete(layout.RootCertKey())
	if err != nil {
		api.ClusterPanic(err)
	}
}

// GetTenantSpec gets tenant spec with its name
func (s *Service) GetTenantSpec(tenantName string) *spec.Tenant {
	tenant, _ := s.GetTenantSpecWithInfo(tenantName)
	return tenant
}

// GetTenantSpecWithInfo gets tenant spec with information
func (s *Service) GetTenantSpecWithInfo(tenantName string) (*spec.Tenant, *mvccpb.KeyValue) {
	kvs, err := s.store.GetRaw(layout.TenantSpecKey(tenantName))
	if err != nil {
		api.ClusterPanic(err)
	}

	if kvs == nil {
		return nil, nil
	}

	tenant := &spec.Tenant{}
	err = yaml.Unmarshal(kvs.Value, tenant)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", string(kvs.Value), err))
	}

	return tenant, kvs
}

// PutTenantSpec writes the tenant spec.
func (s *Service) PutTenantSpec(tenantSpec *spec.Tenant) {
	buff, err := yaml.Marshal(tenantSpec)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", tenantSpec, err))
	}

	err = s.store.Put(layout.TenantSpecKey(tenantSpec.Name), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// ListAllServiceInstanceStatuses lists all service instance statuses.
func (s *Service) ListAllServiceInstanceStatuses() []*spec.ServiceInstanceStatus {
	return s.listServiceInstanceStatuses(true, "")
}

// ListServiceInstanceStatuses lists service instance statuses
func (s *Service) ListServiceInstanceStatuses(serviceName string) []*spec.ServiceInstanceStatus {
	return s.listServiceInstanceStatuses(false, serviceName)
}

func (s *Service) listServiceInstanceStatuses(all bool, serviceName string) []*spec.ServiceInstanceStatus {
	statuses := []*spec.ServiceInstanceStatus{}
	var prefix string
	if all {
		prefix = layout.AllServiceInstanceStatusPrefix()
	} else {
		prefix = layout.ServiceInstanceSpecPrefix(serviceName)
	}

	kvs, err := s.store.GetRawPrefix(prefix)
	if err != nil {
		api.ClusterPanic(err)
	}

	for _, v := range kvs {
		status := &spec.ServiceInstanceStatus{}
		if err = yaml.Unmarshal(v.Value, status); err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// ListAllServiceInstanceSpecs lists all service instance specs.
func (s *Service) ListAllServiceInstanceSpecs() []*spec.ServiceInstanceSpec {
	return s.listServiceInstanceSpecs(true, "")
}

// ListServiceInstanceSpecs lists service instance specs.
func (s *Service) ListServiceInstanceSpecs(serviceName string) []*spec.ServiceInstanceSpec {
	return s.listServiceInstanceSpecs(false, serviceName)
}

func (s *Service) listServiceInstanceSpecs(all bool, serviceName string) []*spec.ServiceInstanceSpec {
	specs := []*spec.ServiceInstanceSpec{}
	var prefix string
	if all {
		prefix = layout.AllServiceInstanceSpecPrefix()
	} else {
		prefix = layout.ServiceInstanceSpecPrefix(serviceName)
	}

	kvs, err := s.store.GetRawPrefix(prefix)
	if err != nil {
		api.ClusterPanic(err)
	}

	for _, v := range kvs {
		_spec := &spec.ServiceInstanceSpec{}
		if err = yaml.Unmarshal(v.Value, _spec); err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}

		specs = append(specs, _spec)
	}

	return specs
}

// GetServiceInstanceSpec gets the service instance spec
func (s *Service) GetServiceInstanceSpec(serviceName, instanceID string) *spec.ServiceInstanceSpec {
	value, err := s.store.Get(layout.ServiceInstanceSpecKey(serviceName, instanceID))
	if err != nil {
		api.ClusterPanic(err)
	}

	if value == nil {
		return nil
	}

	instanceSpec := &spec.ServiceInstanceSpec{}
	err = yaml.Unmarshal([]byte(*value), instanceSpec)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", *value, err))
	}

	return instanceSpec
}

// PutServiceInstanceSpec writes the service instance spec
func (s *Service) PutServiceInstanceSpec(_spec *spec.ServiceInstanceSpec) {
	buff, err := yaml.Marshal(_spec)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", _spec, err))
	}

	err = s.store.Put(layout.ServiceInstanceSpecKey(_spec.ServiceName, _spec.InstanceID), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// DeleteServiceInstanceSpec deletes the service instance spec.
func (s *Service) DeleteServiceInstanceSpec(serviceName, instanceID string) {
	err := s.store.Delete(layout.ServiceInstanceSpecKey(serviceName, instanceID))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// ListTenantSpecs lists tenant specs
func (s *Service) ListTenantSpecs() []*spec.Tenant {
	tenants := []*spec.Tenant{}
	kvs, err := s.store.GetRawPrefix(layout.TenantPrefix())
	if err != nil {
		api.ClusterPanic(err)
	}

	for _, v := range kvs {
		tenantSpec := &spec.Tenant{}
		err := yaml.Unmarshal(v.Value, tenantSpec)
		if err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}
		tenants = append(tenants, tenantSpec)
	}

	return tenants
}

// DeleteTenantSpec deletes tenant spec
func (s *Service) DeleteTenantSpec(tenantName string) {
	err := s.store.Delete(layout.TenantSpecKey(tenantName))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// GetIngressSpec gets the ingress spec
func (s *Service) GetIngressSpec(ingressName string) *spec.Ingress {
	ingress, _ := s.GetIngressSpecWithInfo(ingressName)
	return ingress
}

// GetIngressSpecWithInfo gets ingress spec with information.
func (s *Service) GetIngressSpecWithInfo(ingressName string) (*spec.Ingress, *mvccpb.KeyValue) {
	kvs, err := s.store.GetRaw(layout.IngressSpecKey(ingressName))
	if err != nil {
		api.ClusterPanic(err)
	}

	if kvs == nil {
		return nil, nil
	}

	ingress := &spec.Ingress{}
	err = yaml.Unmarshal(kvs.Value, ingress)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", string(kvs.Value), err))
	}

	return ingress, kvs
}

// PutIngressSpec writes the ingress spec
func (s *Service) PutIngressSpec(ingressSpec *spec.Ingress) {
	buff, err := yaml.Marshal(ingressSpec)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", ingressSpec, err))
	}

	err = s.store.Put(layout.IngressSpecKey(ingressSpec.Name), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// GetIngressControllerInstanceSpec gets one ingress controller's spec
func (s *Service) GetIngressControllerInstanceSpec(instaceID string) *spec.ServiceInstanceSpec {
	instance := &spec.ServiceInstanceSpec{}
	value, err := s.store.Get(layout.IngressControllerInstanceSpecKey(instaceID))
	if err != nil {
		api.ClusterPanic(err)
	}

	if value == nil {
		return nil
	}

	err = yaml.Unmarshal([]byte(*value), instance)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", *value, err))
	}
	return instance
}

// GetIngressControllerInstanceCert gets one ingress controller's cert
func (s *Service) GetIngressControllerInstanceCert(instaceID string) *spec.Certificate {
	cert := &spec.Certificate{}
	value, err := s.store.Get(layout.IngressControllerInstanceCertKey(instaceID))
	if err != nil {
		api.ClusterPanic(err)
	}

	if value == nil {
		return nil
	}

	err = yaml.Unmarshal([]byte(*value), cert)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", *value, err))
	}
	return cert
}

// ListAllIngressControllerInstanceSpecs lists all IngressController's instances specs
func (s *Service) ListAllIngressControllerInstanceSpecs() []*spec.ServiceInstanceSpec {
	specs := []*spec.ServiceInstanceSpec{}

	kvs, err := s.store.GetPrefix(layout.AllIngressControllerInstanceCertPrefix())
	if err != nil {
		api.ClusterPanic(err)
	}

	for _, v := range kvs {
		_spec := &spec.ServiceInstanceSpec{}
		if err = yaml.Unmarshal([]byte(v), _spec); err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}

		specs = append(specs, _spec)
	}

	return specs
}

// ListIngressSpecs lists the ingress specs
func (s *Service) ListIngressSpecs() []*spec.Ingress {
	ingresses := []*spec.Ingress{}
	kvs, err := s.store.GetRawPrefix(layout.IngressPrefix())
	if err != nil {
		api.ClusterPanic(err)
	}

	for _, v := range kvs {
		ingressSpec := &spec.Ingress{}
		err := yaml.Unmarshal(v.Value, ingressSpec)
		if err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}
		ingresses = append(ingresses, ingressSpec)
	}

	return ingresses
}

// DeleteIngressSpec deletes the ingress spec
func (s *Service) DeleteIngressSpec(ingressName string) {
	err := s.store.Delete(layout.IngressSpecKey(ingressName))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// ListCustomResourceKinds lists custom resource kinds
func (s *Service) ListCustomResourceKinds() []*spec.CustomResourceKind {
	kvs, err := s.store.GetRawPrefix(layout.CustomResourceKindPrefix())
	if err != nil {
		api.ClusterPanic(err)
	}

	kinds := []*spec.CustomResourceKind{}
	for _, v := range kvs {
		kind := &spec.CustomResourceKind{}
		err := yaml.Unmarshal(v.Value, kind)
		if err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}
		kinds = append(kinds, kind)
	}

	return kinds
}

// DeleteCustomResourceKind deletes a custom resource kind
func (s *Service) DeleteCustomResourceKind(kind string) {
	err := s.store.Delete(layout.CustomResourceKindKey(kind))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// GetCustomResourceKind gets custom resource kind with its name
func (s *Service) GetCustomResourceKind(name string) *spec.CustomResourceKind {
	kvs, err := s.store.GetRaw(layout.CustomResourceKindKey(name))
	if err != nil {
		api.ClusterPanic(err)
	}

	if kvs == nil {
		return nil
	}

	kind := &spec.CustomResourceKind{}
	err = yaml.Unmarshal(kvs.Value, kind)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", string(kvs.Value), err))
	}

	return kind
}

// PutCustomResourceKind writes the custom resource kind to storage.
func (s *Service) PutCustomResourceKind(kind *spec.CustomResourceKind) {
	buff, err := yaml.Marshal(kind)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", kind, err))
	}

	err = s.store.Put(layout.CustomResourceKindKey(kind.Name), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// ListCustomResources lists custom resources of specified kind.
// if kind is empty, it returns custom objects of all kinds.
func (s *Service) ListCustomResources(kind string) []*spec.CustomResource {
	prefix := layout.AllCustomResourcePrefix()
	if kind != "" {
		prefix = layout.CustomResourcePrefix(kind)
	}
	kvs, err := s.store.GetRawPrefix(prefix)
	if err != nil {
		api.ClusterPanic(err)
	}

	resources := []*spec.CustomResource{}
	for _, v := range kvs {
		resource := &spec.CustomResource{}
		err := yaml.Unmarshal(v.Value, resource)
		if err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}
		resources = append(resources, resource)
	}

	return resources
}

// DeleteCustomResource deletes a custom resource
func (s *Service) DeleteCustomResource(kind, name string) {
	err := s.store.Delete(layout.CustomResourceKey(kind, name))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// GetCustomResource gets custom resource with its kind & name
func (s *Service) GetCustomResource(kind, name string) *spec.CustomResource {
	kvs, err := s.store.GetRaw(layout.CustomResourceKey(kind, name))
	if err != nil {
		api.ClusterPanic(err)
	}

	if kvs == nil {
		return nil
	}

	resource := &spec.CustomResource{}
	err = yaml.Unmarshal(kvs.Value, resource)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", string(kvs.Value), err))
	}

	return resource
}

// PutCustomResource writes the custom resource kind to storage.
func (s *Service) PutCustomResource(obj *spec.CustomResource) {
	buff, err := yaml.Marshal(obj)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", obj, err))
	}

	err = s.store.Put(layout.CustomResourceKey(obj.Kind(), obj.Name()), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// WatchCustomResource watches custom resources of the specified kind
func (s *Service) WatchCustomResource(ctx context.Context, kind string, onChange func([]*spec.CustomResource)) error {
	syncer, err := s.store.Syncer()
	if err != nil {
		return err
	}

	prefix := layout.CustomResourcePrefix(kind)
	ch, err := syncer.SyncRawPrefix(prefix)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			syncer.Close()
			return nil
		case m := <-ch:
			resources := make([]*spec.CustomResource, 0, len(m))
			for _, v := range m {
				resource := &spec.CustomResource{}
				err = yaml.Unmarshal(v.Value, resource)
				if err == nil {
					resources = append(resources, resource)
				}
			}
			onChange(resources)
		}
	}
}

// ListHTTPRouteGroups lists HTTP route groups
func (s *Service) ListHTTPRouteGroups() []*spec.HTTPRouteGroup {
	kvs, err := s.store.GetRawPrefix(layout.HTTPRouteGroupPrefix())
	if err != nil {
		api.ClusterPanic(err)
	}

	groups := []*spec.HTTPRouteGroup{}
	for _, v := range kvs {
		group := &spec.HTTPRouteGroup{}
		err := yaml.Unmarshal(v.Value, group)
		if err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}
		groups = append(groups, group)
	}

	return groups
}

// DeleteHTTPRouteGroup deletes a HTTP route group
func (s *Service) DeleteHTTPRouteGroup(name string) {
	err := s.store.Delete(layout.HTTPRouteGroupKey(name))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// GetHTTPRouteGroup gets HTTP route group with its name
func (s *Service) GetHTTPRouteGroup(name string) *spec.HTTPRouteGroup {
	kvs, err := s.store.GetRaw(layout.HTTPRouteGroupKey(name))
	if err != nil {
		api.ClusterPanic(err)
	}

	if kvs == nil {
		return nil
	}

	group := &spec.HTTPRouteGroup{}
	err = yaml.Unmarshal(kvs.Value, group)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", string(kvs.Value), err))
	}

	return group
}

// PutHTTPRouteGroup writes the HTTP route group to storage.
func (s *Service) PutHTTPRouteGroup(group *spec.HTTPRouteGroup) {
	buff, err := yaml.Marshal(group)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", group, err))
	}

	err = s.store.Put(layout.HTTPRouteGroupKey(group.Name), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// ListTrafficTargets lists traffic targets
func (s *Service) ListTrafficTargets() []*spec.TrafficTarget {
	kvs, err := s.store.GetRawPrefix(layout.TrafficTargetPrefix())
	if err != nil {
		api.ClusterPanic(err)
	}

	tts := []*spec.TrafficTarget{}
	for _, v := range kvs {
		tt := &spec.TrafficTarget{}
		err := yaml.Unmarshal(v.Value, tt)
		if err != nil {
			logger.Errorf("BUG: unmarshal %s to yaml failed: %v", v, err)
			continue
		}
		tts = append(tts, tt)
	}

	return tts
}

// DeleteTrafficTarget deletes a traffic target
func (s *Service) DeleteTrafficTarget(name string) {
	err := s.store.Delete(layout.TrafficTargetKey(name))
	if err != nil {
		api.ClusterPanic(err)
	}
}

// GetTrafficTarget gets traffic target with its name
func (s *Service) GetTrafficTarget(name string) *spec.TrafficTarget {
	kvs, err := s.store.GetRaw(layout.TrafficTargetKey(name))
	if err != nil {
		api.ClusterPanic(err)
	}

	if kvs == nil {
		return nil
	}

	tt := &spec.TrafficTarget{}
	err = yaml.Unmarshal(kvs.Value, tt)
	if err != nil {
		panic(fmt.Errorf("BUG: unmarshal %s to yaml failed: %v", string(kvs.Value), err))
	}

	return tt
}

// PutTrafficTarget writes the traffic target to storage.
func (s *Service) PutTrafficTarget(tt *spec.TrafficTarget) {
	buff, err := yaml.Marshal(tt)
	if err != nil {
		panic(fmt.Errorf("BUG: marshal %#v to yaml failed: %v", tt, err))
	}

	err = s.store.Put(layout.TrafficTargetKey(tt.Name), string(buff))
	if err != nil {
		api.ClusterPanic(err)
	}
}
