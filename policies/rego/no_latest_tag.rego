package k8s.no_latest_tag

deny contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    endswith(container.image, ":latest")
    msg := sprintf("container '%v' uses ':latest' tag — pin a specific version", [container.name])
}

deny contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    not contains(container.image, ":")
    msg := sprintf("container '%v' has no image tag — pin a specific version", [container.name])
}
