package controller

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"github.com/DingGGu/cloudflare-access-controller/internal/option"
	"github.com/DingGGu/cloudflare-access-controller/internal/provider/cf"
	"github.com/DingGGu/cloudflare-access-controller/internal/store"
	"github.com/cloudflare/cloudflare-go"
	"github.com/iancoleman/strcase"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"strings"
)

type Controller struct {
	KubeClient       *kubernetes.Clientset
	CfClient         *cf.CloudFlareProvider
	ClusterName      string
	AnnotationPrefix string
	ZoneNames        []string
}

func (c Controller) Run() {
	clusterUID := c.getClusterUID()
	ingresses, err := c.KubeClient.ExtensionsV1beta1().Ingresses("").List(v1.ListOptions{})

	if err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	var k8sApps []option.AccessApp

	for _, ingress := range ingresses.Items {
		parsed := false
		annotations := ingress.Annotations
		ann := option.AccessAnnotation{}

		if val, ok := annotations[c.getAnnotationKey("ZoneName")]; ok {
			ann.ZoneName = val
			parsed = true
		}

		if val, ok := annotations[c.getAnnotationKey("ApplicationName")]; ok {
			ann.ApplicationName = val
		}

		if val, ok := annotations[c.getAnnotationKey("ApplicationSubDomain")]; ok {
			ann.ApplicationSubDomain = val
		}

		if val, ok := annotations[c.getAnnotationKey("ApplicationPath")]; ok {
			ann.ApplicationPath = val
		}

		if val, ok := annotations[c.getAnnotationKey("SessionDuration")]; ok {
			ann.SessionDuration = val
		}

		if val, ok := annotations[c.getAnnotationKey("Policies")]; ok {
			var accessPolicy []cloudflare.AccessPolicy
			err := json.Unmarshal([]byte(val), &accessPolicy)
			if err != nil {
				logrus.Errorf("%s - %s", ingress.Name, err)
			} else {
				ann.Policies = accessPolicy
			}
		}

		if parsed {
			app := option.AccessApp{
				ClusterUID:  clusterUID,
				IngressName: ingress.Name,
				Namespace:   ingress.Namespace,
				Source:      "k8s",
				Ingress:     &ingress,
			}
			app.CfZoneName = ann.ZoneName
			app.AccessAppOption = option.AccessAppOption{
				Domain:          ann.GetDomain(),
				SessionDuration: ann.SessionDuration,
			}
			app.AccessAppPolicies = ann.Policies
			k8sApps = append(k8sApps, app)
		}
	}

	s := store.Store{
		ClusterUID:         clusterUID,
		CloudFlareProvider: c.CfClient,
		KubeClient:         c.KubeClient,
	}
	s.Run(k8sApps, c.ZoneNames)
}

func (c Controller) getClusterUID() string {
	clusterHash := md5.Sum([]byte(c.ClusterName))
	clusterUID := hex.EncodeToString(clusterHash[:])[:8]
	return clusterUID
}

func (c Controller) parse(annotation string) (key string, ok bool) {
	if strings.HasPrefix(annotation, c.AnnotationPrefix) {
		return annotation[len(c.AnnotationPrefix):], true
	}
	return "", false
}

func (c Controller) getAnnotationKey(key string) string {
	return strings.Join([]string{c.AnnotationPrefix, strcase.ToKebab(key)}, "")
}

func NewController(
	KubeClient *kubernetes.Clientset,
	CfClient *cf.CloudFlareProvider,
	ClusterName string,
	AnnotationPrefix string,
	ZoneNames []string) *Controller {
	ctrl := Controller{}

	ctrl.KubeClient = KubeClient
	ctrl.CfClient = CfClient
	ctrl.ClusterName = ClusterName

	if AnnotationPrefix == "" {
		ctrl.AnnotationPrefix = "access.cloudflare.com/"
	}
	ctrl.ZoneNames = ZoneNames

	return &ctrl
}
