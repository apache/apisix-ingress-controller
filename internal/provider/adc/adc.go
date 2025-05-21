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

package adc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"go.uber.org/zap"
	networkingv1 "k8s.io/api/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	adctypes "github.com/apache/apisix-ingress-controller/api/adc"
	"github.com/apache/apisix-ingress-controller/api/v1alpha1"
	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/controller/label"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/provider/adc/translator"
)

type adcConfig struct {
	Name       string
	ServerAddr string
	Token      string
	TlsVerify  bool
}

type adcClient struct {
	sync.Mutex

	execLock sync.Mutex

	translator *translator.Translator
	// gateway/ingressclass -> adcConfig
	configs map[provider.ResourceKind]adcConfig
	// httproute/consumer/ingress/gateway -> gateway/ingressclass
	parentRefs map[provider.ResourceKind][]provider.ResourceKind

	syncTimeout time.Duration

	store *Store
}

type Task struct {
	Name          string
	Resources     adctypes.Resources
	Labels        map[string]string
	ResourceTypes []string
	configs       []adcConfig
}

func New() (provider.Provider, error) {
	return &adcClient{
		syncTimeout: config.ControllerConfig.ExecADCTimeout.Duration,
		translator:  &translator.Translator{},
		configs:     make(map[provider.ResourceKind]adcConfig),
		parentRefs:  make(map[provider.ResourceKind][]provider.ResourceKind),
		store:       NewStore(),
	}, nil
}

func (d *adcClient) Update(ctx context.Context, tctx *provider.TranslateContext, obj client.Object) error {
	log.Debugw("updating object", zap.Any("object", obj))
	var (
		result        *translator.TranslateResult
		resourceTypes []string
		err           error
	)

	rk := provider.ResourceKind{
		Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}

	switch t := obj.(type) {
	case *gatewayv1.HTTPRoute:
		result, err = d.translator.TranslateHTTPRoute(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "service")
	case *gatewayv1.Gateway:
		result, err = d.translator.TranslateGateway(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "global_rule", "ssl", "plugin_metadata")
	case *networkingv1.Ingress:
		result, err = d.translator.TranslateIngress(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "service", "ssl")
	case *v1alpha1.Consumer:
		result, err = d.translator.TranslateConsumerV1alpha1(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "consumer")
	case *networkingv1.IngressClass:
		result, err = d.translator.TranslateIngressClass(tctx, t.DeepCopy())
		resourceTypes = append(resourceTypes, "global_rule", "plugin_metadata")
	}
	if err != nil {
		return err
	}
	if result == nil {
		return nil
	}

	oldParentRefs := d.getParentRefs(rk)
	if err := d.updateConfigs(rk, tctx); err != nil {
		return err
	}
	newParentRefs := d.getParentRefs(rk)
	deleteConfigs := d.findConfigsToDelete(oldParentRefs, newParentRefs)
	configs := d.getConfigs(rk)

	// sync delete
	if len(deleteConfigs) > 0 {
		err = d.sync(ctx, Task{
			Name:          obj.GetName(),
			Labels:        label.GenLabel(obj),
			ResourceTypes: resourceTypes,
			configs:       deleteConfigs,
		})
		if err != nil {
			return err
		}
		for _, config := range deleteConfigs {
			if err := d.store.Delete(config.Name, resourceTypes, label.GenLabel(obj)); err != nil {
				log.Errorw("failed to delete resources from store",
					zap.String("name", config.Name),
					zap.Error(err),
				)
				return err
			}
		}
	}

	resources := adctypes.Resources{
		GlobalRules:    result.GlobalRules,
		PluginMetadata: result.PluginMetadata,
		Services:       result.Services,
		SSLs:           result.SSL,
		Consumers:      result.Consumers,
	}

	for _, config := range configs {
		if err := d.store.Insert(config.Name, resourceTypes, resources, label.GenLabel(obj)); err != nil {
			log.Errorw("failed to insert resources into store",
				zap.String("name", config.Name),
				zap.Error(err),
			)
			return err
		}
	}

	// sync update
	return d.sync(ctx, Task{
		Name:          obj.GetName(),
		Labels:        label.GenLabel(obj),
		Resources:     resources,
		ResourceTypes: resourceTypes,
		configs:       configs,
	})
}

func (d *adcClient) Delete(ctx context.Context, obj client.Object) error {
	log.Debugw("deleting object", zap.Any("object", obj))

	var resourceTypes []string
	var labels map[string]string
	switch obj.(type) {
	case *gatewayv1.HTTPRoute:
		resourceTypes = append(resourceTypes, "service")
		labels = label.GenLabel(obj)
	case *gatewayv1.Gateway:
		// delete all resources
	case *networkingv1.Ingress:
		resourceTypes = append(resourceTypes, "service", "ssl")
		labels = label.GenLabel(obj)
	case *v1alpha1.Consumer:
		resourceTypes = append(resourceTypes, "consumer")
		labels = label.GenLabel(obj)
	case *networkingv1.IngressClass:
		// delete all resources
	}

	rk := provider.ResourceKind{
		Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}

	configs := d.getConfigs(rk)
	defer d.deleteConfigs(rk)

	for _, config := range configs {
		if err := d.store.Delete(config.Name, resourceTypes, labels); err != nil {
			log.Errorw("failed to delete resources from store",
				zap.String("name", config.Name),
				zap.Error(err),
			)
			return err
		}
	}

	log.Debugw("successfully deleted resources from store", zap.Any("object", obj))

	err := d.sync(ctx, Task{
		Name:          obj.GetName(),
		Labels:        labels,
		ResourceTypes: resourceTypes,
		configs:       configs,
	})
	if err != nil {
		return err
	}

	return nil
}

func (d *adcClient) Sync(ctx context.Context) error {
	log.Debug("syncing all resources")

	if len(d.configs) == 0 {
		return nil
	}

	cfg := map[string]adcConfig{}
	for _, config := range d.configs {
		cfg[config.Name] = config
	}

	for name, config := range cfg {
		resources, err := d.store.GetResources(name)
		if err != nil {
			return err
		}
		if resources == nil {
			continue
		}

		err = d.sync(ctx, Task{
			Name:      name + "-sync",
			configs:   []adcConfig{config},
			Resources: *resources,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *adcClient) sync(ctx context.Context, task Task) error {
	log.Debugw("syncing resources", zap.Any("task", task))

	if len(task.configs) == 0 {
		log.Errorw("no adc configs provided", zap.Any("task", task))
		return errors.New("no adc configs provided")
	}

	data, err := json.Marshal(task.Resources)
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "adc-task-*.json")
	if err != nil {
		return err
	}
	defer func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpFile.Name())
	}()

	log.Debugf("generated adc file, filename: %s, json: %s\n", tmpFile.Name(), string(data))

	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	args := []string{
		"sync",
		"-f", tmpFile.Name(),
	}

	for k, v := range task.Labels {
		args = append(args, "--label-selector", k+"="+v)
	}
	for _, t := range task.ResourceTypes {
		args = append(args, "--include-resource-type", t)
	}

	log.Debugw("syncing resources with multiple configs", zap.Any("configs", task.configs))
	for _, config := range task.configs {
		if err := d.execADC(ctx, config, args); err != nil {
			return err
		}
	}

	return nil
}

func (d *adcClient) execADC(ctx context.Context, config adcConfig, args []string) error {
	d.execLock.Lock()
	defer d.execLock.Unlock()

	ctxWithTimeout, cancel := context.WithTimeout(ctx, d.syncTimeout)
	defer cancel()
	serverAddr := config.ServerAddr
	token := config.Token
	tlsVerify := config.TlsVerify
	if !tlsVerify {
		args = append(args, "--tls-skip-verify")
	}

	adcEnv := []string{
		"ADC_EXPERIMENTAL_FEATURE_FLAGS=remote-state-file,parallel-backend-request",
		"ADC_RUNNING_MODE=ingress",
		"ADC_BACKEND=api7ee",
		"ADC_SERVER=" + serverAddr,
		"ADC_TOKEN=" + token,
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctxWithTimeout, "adc", args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, adcEnv...)

	log.Debug("running adc command", zap.String("command", cmd.String()), zap.Strings("env", adcEnv))

	var result adctypes.SyncResult
	if err := cmd.Run(); err != nil {
		stderrStr := stderr.String()
		stdoutStr := stdout.String()
		errMsg := stderrStr
		if errMsg == "" {
			errMsg = stdoutStr
		}
		log.Errorw("failed to run adc",
			zap.Error(err),
			zap.String("output", stdoutStr),
			zap.String("stderr", stderrStr),
		)
		return errors.New("failed to sync resources: " + errMsg + ", exit err: " + err.Error())
	}

	output := stdout.Bytes()
	if err := json.Unmarshal(output, &result); err != nil {
		log.Errorw("failed to unmarshal adc output",
			zap.Error(err),
			zap.String("stdout", string(output)),
		)
		return errors.New("failed to unmarshal adc result: " + err.Error())
	}

	if result.FailedCount > 0 {
		log.Errorw("adc sync failed", zap.Any("result", result))
		failed := result.Failed
		return errors.New(failed[0].Reason)
	}

	log.Debugw("adc sync success", zap.Any("result", result))
	return nil
}
