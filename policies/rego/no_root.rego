package k8s.no_root

deny contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    container.securityContext.runAsUser == 0
    msg := sprintf("container '%v' runs as root (uid=0) — forbidden", [container.name])
}

deny contains msg if {
    input.request.kind.kind == "Pod"
    container := input.request.object.spec.containers[_]
    container.securityContext.privileged == true
    msg := sprintf("container '%v' is privileged — forbidden", [container.name])
}

deny contains msg if {
    input.request.kind.kind == "Pod"
    input.request.object.spec.securityContext.runAsNonRoot == false
    msg := "pod must run as non-root"
}
