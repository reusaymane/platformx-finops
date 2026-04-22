package k8s.resource_limits

deny contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    not container.resources.limits.cpu
    msg := sprintf("container '%v' has no CPU limit — required for FinOps cost control", [container.name])
}

deny contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    not container.resources.limits.memory
    msg := sprintf("container '%v' has no memory limit — required for FinOps cost control", [container.name])
}

deny contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    not container.resources.requests.cpu
    msg := sprintf("container '%v' has no CPU request", [container.name])
}
