package k8s.resource_limits

test_deny_no_cpu_limit if {
    deny[_] with input as {
        "request": {
            "kind": {"kind": "Pod"},
            "object": {
                "spec": {
                    "containers": [{
                        "name": "app",
                        "image": "nginx:1.25.3",
                        "resources": {
                            "requests": {"cpu": "100m", "memory": "128Mi"},
                            "limits": {"memory": "256Mi"}
                        }
                    }]
                }
            }
        }
    }
}

test_allow_with_limits if {
    count(deny) == 0 with input as {
        "request": {
            "kind": {"kind": "Pod"},
            "object": {
                "spec": {
                    "containers": [{
                        "name": "app",
                        "image": "nginx:1.25.3",
                        "resources": {
                            "requests": {"cpu": "100m", "memory": "128Mi"},
                            "limits": {"cpu": "500m", "memory": "256Mi"}
                        }
                    }]
                }
            }
        }
    }
}
