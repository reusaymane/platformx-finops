package k8s.finops_budget

deny contains msg if {
    input.request.kind.kind == "Pod"
    namespace := input.request.namespace
    budget := data.budgets[namespace]
    budget.used_percent >= 95
    msg := sprintf(
        "namespace '%v' has used %.1f%% of its monthly budget ($%.2f / $%.2f) — deployment blocked",
        [namespace, budget.used_percent, budget.current_cost, budget.monthly_limit]
    )
}

deny contains msg if {
    input.request.kind.kind == "Deployment"
    namespace := input.request.namespace
    budget := data.budgets[namespace]
    budget.used_percent >= 95
    msg := sprintf(
        "namespace '%v' budget exceeded (%.1f%%) — scale up blocked by FinOps policy",
        [namespace, budget.used_percent]
    )
}
