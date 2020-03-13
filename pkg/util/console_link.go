package util

import (
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	consolev1 "github.com/openshift/api/console/v1"
)


// creates a NamespaceDashboard ConsoleLink instance
func CreateNamespaceDashboardLinK(name string, namespace string, route *routev1.Route) *consolev1.ConsoleLink {
	consoleLink := &consolev1.ConsoleLink{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{"app": "atlasmap"},
		},
		Spec: consolev1.ConsoleLinkSpec{
			Location: consolev1.NamespaceDashboard,
			NamespaceDashboard: &consolev1.NamespaceDashboardSpec{
				Namespaces: []string{namespace},
			},
		},
	}

	setNamespaceDashboardLink(consoleLink, route)

	return consoleLink
}

func setNamespaceDashboardLink(consoleLink *consolev1.ConsoleLink, route *routev1.Route) {
	consoleLink.Spec.Link.Text = "atlasmap"
	consoleLink.Spec.Link.Href = "https://" + route.Spec.Host
}
