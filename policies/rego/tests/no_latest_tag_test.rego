package k8s.no_latest_tag

test_deny_latest_tag if {
    deny[_] with input as {
        "request": {
            "kind": {"kind": "Pod"},
            "object": {
                "spec": {
                    "containers": [{"name": "app", "image": "nginx:latest"}]
                }
            }
        }
    }
}

test_deny_no_tag if {
    deny[_] with input as {
        "request": {
            "kind": {"kind": "Pod"},
            "object": {
                "spec": {
                    "containers": [{"name": "app", "image": "nginx"}]
                }
            }
        }
    }
}

test_allow_pinned_tag if {
    count(deny) == 0 with input as {
        "request": {
            "kind": {"kind": "Pod"},
            "object": {
                "spec": {
                    "containers": [{"name": "app", "image": "nginx:1.25.3"}]
                }
            }
        }
    }
}
