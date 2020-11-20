package postgres

import (
	"github.com/crossplane-contrib/provider-in-cluster/pkg/controller/utils"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/crossplane-contrib/provider-in-cluster/apis/database/v1alpha1"
)

const (
	errGetPasswordSecretFailed = "cannot get password secret"
	NamespacePrefixOpenShift   = "openshift-"
	DefaultPostgresPort        = 5432
)

type Client interface {
	CreateOrUpdate(ctx context.Context, obj runtime.Object) (controllerutil.OperationResult, error)
	ParseInputSecret(ctx context.Context, postgres v1alpha1.Postgres) (string, error)
	DeleteBucketPVC(ctx context.Context, postgres *v1alpha1.Postgres) error
	DeleteBucketDeployment(ctx context.Context, postgres *v1alpha1.Postgres) error
	DeleteBucketService(ctx context.Context, postgres *v1alpha1.Postgres) error
}

type postgresClient struct {
	kube client.Client
}

func (c postgresClient) DeleteBucketPVC(ctx context.Context, postgres *v1alpha1.Postgres) error {
	pvc := v1.PersistentVolumeClaim{}
	err := c.kube.Get(ctx, client.ObjectKey{
		Namespace: postgres.Namespace,
		Name:      postgres.Name,
	}, &pvc)
	if err != nil {
		return nil
	}
	return c.kube.Delete(ctx, &pvc)
}

func (c postgresClient) DeleteBucketDeployment(ctx context.Context, postgres *v1alpha1.Postgres) error {
	dpl := appsv1.Deployment{}
	err := c.kube.Get(ctx, client.ObjectKey{
		Name:      postgres.Name,
		Namespace: postgres.Namespace,
	}, &dpl)
	if err != nil {
		return nil
	}
	return c.kube.Delete(ctx, &dpl)
}

func (c postgresClient) DeleteBucketService(ctx context.Context, postgres *v1alpha1.Postgres) error {
	svc := v1.Service{}
	err := c.kube.Get(ctx, client.ObjectKey{
		Name:      postgres.Name,
		Namespace: postgres.Namespace,
	}, &svc)
	if err != nil {
		return nil
	}
	return c.kube.Delete(ctx, &svc)
}

func NewRoleClient(kube client.Client) Client {
	return postgresClient{kube: kube}
}

func (c postgresClient) CreateOrUpdate(ctx context.Context, obj runtime.Object) (controllerutil.OperationResult, error) {
	return controllerutil.CreateOrUpdate(ctx, c.kube, obj, func() error {
		return nil
	})
}

func (c postgresClient) ParseInputSecret(ctx context.Context, postgres v1alpha1.Postgres) (string, error) {
	if postgres.Spec.ForProvider.MasterPasswordSecretRef == nil {
		return "", errors.New(errGetPasswordSecretFailed)
	}
	nn := types.NamespacedName{
		Name:      postgres.Spec.ForProvider.MasterPasswordSecretRef.Name,
		Namespace: postgres.Spec.ForProvider.MasterPasswordSecretRef.Namespace,
	}
	s := &v1.Secret{}
	if err := c.kube.Get(ctx, nn, s); err != nil {
		return "", errors.Wrap(err, errGetPasswordSecretFailed)
	}
	return string(s.Data[postgres.Spec.ForProvider.MasterPasswordSecretRef.Key]), nil
}

func MakePVCPostgres(postgres *v1alpha1.Postgres) *v1.PersistentVolumeClaim {
	fs := v1.PersistentVolumeFilesystem
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      postgres.Name,
			Namespace: postgres.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolumeClaim",
			APIVersion: "v1",
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes:      []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			VolumeMode:       &fs,
			StorageClassName: postgres.Spec.ForProvider.StorageClass,
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceStorage: resource.MustParse(postgres.Spec.ForProvider.DatabaseSize),
				},
			},
		},
	}
}

func int32Ptr(i int32) *int32 { return &i }
func StringPtrToVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func MakePostgresDeployment(ps *v1alpha1.Postgres, pw string) *appsv1.Deployment {
	depl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.Name,
			Namespace: ps.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"deployment": ps.Name,
				},
			},
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Volumes: []v1.Volume{
						{
							Name: ps.Name,
							VolumeSource: v1.VolumeSource{
								PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
									ClaimName: ps.Name,
								},
							},
						},
					},
					Containers: MakeDefaultPostgresPodContainers(ps, pw),
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment": ps.Name,
					},
				},
			},
		},
	}
	// required for restricted namespace
	if strings.HasPrefix(ps.Namespace, NamespacePrefixOpenShift) {
		userGroupId := int64(26)
		depl.Spec.Template.Spec.SecurityContext = &v1.PodSecurityContext{
			FSGroup:            &userGroupId,
			SupplementalGroups: []int64{userGroupId},
		}
	}
	return depl
}

func MakeDefaultPostgresPodContainers(ps *v1alpha1.Postgres, pw string) []v1.Container {
	return []v1.Container{
		{
			Name:  ps.Name,
			Image: "postgres:13.0",
			Ports: []v1.ContainerPort{
				{
					ContainerPort: DefaultPostgresPort,
					Protocol:      v1.ProtocolTCP,
				},
			},
			Env: []v1.EnvVar{
				envVarFromValue("POSTGRES_USER", StringPtrToVal(ps.Spec.ForProvider.MasterUsername)),
				envVarFromValue("POSTGRES_PASSWORD", pw),
				envVarFromValue("POSTGRES_DB", StringPtrToVal(ps.Spec.ForProvider.Database)),
			},
			Resources: v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("250m"),
					v1.ResourceMemory: resource.MustParse("2Gi"),
				},
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("50m"),
					v1.ResourceMemory: resource.MustParse("512Mi"),
				},
			},
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      ps.Name,
					MountPath: "/var/lib/pgsql/data",
				},
			},
			LivenessProbe: &v1.Probe{
				Handler: v1.Handler{
					TCPSocket: &v1.TCPSocketAction{
						Port: intstr.IntOrString{
							Type:   intstr.Int,
							IntVal: DefaultPostgresPort,
						},
					},
				},
				InitialDelaySeconds: 30,
				PeriodSeconds:       10,
				TimeoutSeconds:      0,
				SuccessThreshold:    0,
				FailureThreshold:    0,
			},
			ReadinessProbe: &v1.Probe{
				Handler: v1.Handler{
					Exec: &v1.ExecAction{
						Command: []string{"/bin/sh", "-i", "-c", "psql -h 127.0.0.1 -U $POSTGRES_USER -q -d $POSTGRES_DB -c 'SELECT 1'"}},
				},
				InitialDelaySeconds: 10,
				PeriodSeconds:       30,
				TimeoutSeconds:      5,
				SuccessThreshold:    0,
				FailureThreshold:    0,
			},
			ImagePullPolicy: v1.PullIfNotPresent,
		},
	}
}

// create an environment variable referencing a secret
func envVarFromValue(envVarName, value string) v1.EnvVar {
	return v1.EnvVar{
		Name:  envVarName,
		Value: value,
	}
}

func MakeDefaultPostgresService(ps *v1alpha1.Postgres) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ps.Name,
			Namespace: ps.Namespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{
					Name:       "postgresql",
					Protocol:   v1.ProtocolTCP,
					Port:       int32(utils.IntValue(ps.Spec.ForProvider.Port)),
					TargetPort: intstr.FromInt(DefaultPostgresPort),
				},
			},
			Selector: map[string]string{"deployment": ps.Name},
		},
	}
}
