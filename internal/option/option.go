package option

import (
	"fmt"
	"github.com/cloudflare/cloudflare-go"
	"k8s.io/api/extensions/v1beta1"
	"strings"
)

type AccessAnnotation struct {
	ZoneName             string
	ApplicationName      string
	ApplicationSubDomain string
	ApplicationPath      string
	SessionDuration      string
	Policies             []cloudflare.AccessPolicy
}

func (a *AccessAnnotation) GetDomain() string {
	return strings.Join([]string{a.ApplicationSubDomain, ".", a.ZoneName, "/", a.stripSlash(a.ApplicationPath)}, "")
}

func (a *AccessAnnotation) stripSlash(s string) string {
	if strings.HasPrefix(s, "/") {
		return s[1:]
	}
	return s
}

type AccessAppOption struct {
	Domain          string
	SessionDuration string
}

type AccessApp struct {
	ClusterUID  string
	Namespace   string
	IngressName string
	CfZoneName  string
	AccessAppOption
	AccessAppPolicies []cloudflare.AccessPolicy
	Source            string
	Ingress           *v1beta1.Ingress
}

func (app *AccessApp) GetName() string {
	return strings.Join([]string{app.ClusterUID, app.Namespace, app.IngressName}, "-")
}

func (app *AccessApp) GetIngressName() string {
	return fmt.Sprintf("(%s) %s", app.Namespace, app.IngressName)
}
