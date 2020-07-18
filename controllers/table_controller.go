/*


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

package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	apix "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dbv1 "op/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

var (
	ownerKey = ".metadata.controller"
	apiGVStr = apix.SchemeGroupVersion.String()
)

// TableReconciler reconciles a Table object
type TableReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=db.k8sasdb.org,resources=tables,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=db.k8sasdb.org,resources=tables/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions/status,verbs=get;update;patch
func (r *TableReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("table", req.NamespacedName)

	constructCRDForTable := func(table *dbv1.Table) (*apix.CustomResourceDefinition, error) {
		// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
		name := fmt.Sprintf("%s", table.Name)
		group := "user.k8sasdb.org"
		pluralName := name + "s"
		crdName := pluralName + "." + group // CRD naming rule

		crd := &apix.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        crdName,
				Namespace:   table.Namespace,
			},
		}
		crd.Spec = apix.CustomResourceDefinitionSpec{
			Group: group,
			Names: apix.CustomResourceDefinitionNames{
				Plural:   pluralName,
				Singular: name,
				Kind:     strings.Title(name),
			},
			Scope: "Namespaced",
		}
		constructProps := func(columns []dbv1.ColumnSpec, props map[string]apix.JSONSchemaProps) map[string]apix.JSONSchemaProps {
			rtn := map[string]apix.JSONSchemaProps{}
			for _, columnSpec := range columns {
				rtn[columnSpec.Name] = apix.JSONSchemaProps{
					Type:       columnSpec.Type,
					Properties: props,
				}
			}
			return rtn
		}
		constructmSimpleProp := func(field string, _type string, props map[string]apix.JSONSchemaProps) map[string]apix.JSONSchemaProps {
			return constructProps([]dbv1.ColumnSpec{{Name: field, Type: _type}}, props)
		}
		schema := apix.CustomResourceValidation{
			OpenAPIV3Schema: &apix.JSONSchemaProps{
				Type: "object",
				Properties: constructmSimpleProp("spec", "object",
					constructProps(table.Spec.Columns, nil),
				),
			},
		}
		additionalPrinterColumns := []apix.CustomResourceColumnDefinition{}
		constructPrinterColumn := func(columnSpec dbv1.ColumnSpec) apix.CustomResourceColumnDefinition {
			return apix.CustomResourceColumnDefinition{
				Name:     columnSpec.Name,
				Type:     columnSpec.Type,
				JSONPath: ".spec." + columnSpec.Name,
			}
		}
		for _, columnSpec := range table.Spec.Columns {
			additionalPrinterColumns = append(additionalPrinterColumns, constructPrinterColumn(columnSpec))
		}
		crd.Spec.Versions = append(crd.Spec.Versions,
			apix.CustomResourceDefinitionVersion{
				Name:                     "v1",
				Served:                   true,
				Storage:                  true,
				Schema:                   &schema,
				AdditionalPrinterColumns: additionalPrinterColumns,
			},
		)

		// for k, v := range table.Spec.JobTemplate.Annotations {
		// 		job.Annotations[k] = v
		// }
		// job.Annotations[scheduledTimeAnnotation] = scheduledTime.Format(time.RFC3339)
		// for k, v := range cronJob.Spec.JobTemplate.Labels {
		// 		job.Labels[k] = v
		// }
		if err := ctrl.SetControllerReference(table, crd, r.Scheme); err != nil {
			return nil, err
		}

		return crd, nil
	}
	// your logic here

	var table dbv1.Table
	if err := r.Get(ctx, req.NamespacedName, &table); err != nil {
		log.Error(err, "unable to fetch Table")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	var crds apix.CustomResourceDefinitionList
	// if err := r.List(ctx, &crds, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
	if err := r.List(ctx, &crds, client.MatchingFields{ownerKey: req.Name}); err != nil {
		log.Error(err, "unable to list CRD")
		return ctrl.Result{}, err
	}

	crd, err := constructCRDForTable(&table)
	if err != nil {
		log.Error(err, "unable to construct crd from template")
		// don't bother requeuing until we get a change to the spec
		return ctrl.Result{}, nil
	}

	// ...and create it on the cluster
	if err := r.Create(ctx, crd); err != nil {
		log.Error(err, "unable to create CRD for Table", "crd", crd)
		return ctrl.Result{}, err
	}

	log.V(1).Info("created CRD for Table run", "crd", crd)
	return ctrl.Result{}, nil
}

func (r *TableReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// At first, apiextensions isn't loaded in manager scheme. So need to load
	if err := apix.AddToScheme(mgr.GetScheme()); err != nil {
		return nil
	}

	if err := mgr.GetFieldIndexer().IndexField(&apix.CustomResourceDefinition{}, ownerKey, func(rawObj runtime.Object) []string {
		// grab the CRD object, extract the owner...
		crd := rawObj.(*apix.CustomResourceDefinition)
		owner := metav1.GetControllerOf(crd)

		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiGVStr || owner.Kind != "Table" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&dbv1.Table{}).
		Owns(&apix.CustomResourceDefinition{}).
		Complete(r)
}
