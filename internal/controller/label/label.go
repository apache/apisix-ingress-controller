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

package label

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
)

type Label map[string]string

const (
	LabelKind           = "k8s/kind"
	LabelName           = "k8s/name"
	LabelNamespace      = "k8s/namespace"
	LabelControllerName = "k8s/controller-name"
	LabelManagedBy      = "manager-by"
)

func GenLabel(client client.Object, args ...string) Label {
	label := make(Label)
	label[LabelKind] = client.GetObjectKind().GroupVersionKind().Kind
	label[LabelNamespace] = client.GetNamespace()
	label[LabelName] = client.GetName()
	label[LabelControllerName] = config.ControllerConfig.ControllerName
	label[LabelManagedBy] = "apisix-ingress-controller"
	for i := 0; i < len(args); i += 2 {
		label[args[i]] = args[i+1]
	}
	return label
}
