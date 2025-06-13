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

package utils

import (
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/apisix-ingress-controller/internal/types"
)

func NamespacedName(obj client.Object) k8stypes.NamespacedName {
	return k8stypes.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

func NamespacedNameKind(obj client.Object) types.NamespacedNameKind {
	return types.NamespacedNameKind{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
		Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
	}
}
