package etcd

import "fmt"

const (
	RegistryPrefix = "/registry"
)

func Key(group, version, resources, namespace, name string) string {
	if namespace != "" {
		return fmt.Sprintf("%s/%s/%s/%s/%s/%s", RegistryPrefix, group, version, resources, namespace, name)
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s", RegistryPrefix, group, version, resources, name)
}

func PodKey(namespace, name string) string {
	return Key("core", "v1", "pods", "", name)
}

func NamespaceKey(namespace, name string) string {
	return Key("core", "v1", "namespaces", "", name)
}

func PodPrefix(namespace string) string {
	if namespace != "" {
		return fmt.Sprintf("%s/core/v1/pods/%s", RegistryPrefix, namespace)
	}
	return fmt.Sprintf("%s/core/v1/pods", RegistryPrefix)
}

func NodePrefix() string {
	return fmt.Sprintf("%s/core/v1/nodes", RegistryPrefix)
}

func NamespacePrefix() string {
	return fmt.Sprintf("%s/core/v1/namespaces", RegistryPrefix)
}
