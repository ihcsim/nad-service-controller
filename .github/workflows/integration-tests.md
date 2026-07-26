---
emoji: 🧪
description: Execute integration tests for nad-service-controller once a week
on:
  schedule: "weekly on Monday"
  workflow_dispatch: {}
runs-on: ubuntu-latest
permissions:
  contents: read
network:
  allowed:
    - go
    - containers
    - github
    - get.helm.sh
    - registry.k8s.io
    - storage.googleapis.com
steps:
  - name: Set up Go
    uses: actions/setup-go@v7
    with:
      go-version: stable
  - name: Install kubectl
    run: |
      curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
      chmod +x kubectl
      sudo mv kubectl /usr/local/bin/
  - name: Install kind
    run: go install sigs.k8s.io/kind@latest
  - name: Install ko
    run: go install github.com/google/ko@latest
  - name: Install helm
    run: curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
  - name: Read test cases
    run: cat integration/testcases.md
safe-outputs:
  create-issue:
    labels: [bug, integration-test]
---

# Weekly Integration Test Runner

## Task

You are an integration test runner for the `nad-service-controller` project.

The test cases to execute are in `integration/testcases.md` and were printed above in the pre-execution steps.

### Cluster Name

Use `integration-test` as the KinD cluster name for all `make` targets that accept `KIND_CLUSTER_NAME`.

### Setup

Execute the following setup steps in order, stopping immediately if any step fails:

1. Create the KinD cluster with all required components:
   ```
   make KIND_CLUSTER_NAME=integration-test kind
   make KIND_CLUSTER_NAME=integration-test multus
   make KIND_CLUSTER_NAME=integration-test whereabouts
   make KIND_CLUSTER_NAME=integration-test nad
   ```

2. Build the controller image with `ko` and deploy it to the cluster:
   ```
   make KIND_CLUSTER_NAME=integration-test apply-kind
   ```

3. Wait for the controller to be ready:
   ```
   kubectl wait --for condition=Ready po -lapp.kubernetes.io/name=nad-service-controller --timeout=120s
   ```

4. Deploy the test data:
   ```
   make testdata
   ```

### Test Execution

Execute each test case from `integration/testcases.md` in sequence (TC-001 through TC-006).

For each test case:
- Run every step in the test case using `bash`.
- Record whether it **passed** or **failed**, including any error output.
- Continue to the next test case even if the current one fails.

### Teardown

After all test cases have been executed (regardless of pass/fail), always clean up:
```
make KIND_CLUSTER_NAME=integration-test purge
```

### Reporting

After teardown:
- If **all test cases passed**: call `noop` with a one-line summary such as
  `All 6 integration test cases passed.`
- If **any test case failed**: use the `create-issue` safe output with a body that includes:
  - A table summarising which test cases passed and which failed
  - The error output and failure reason for each failing test case
  - A link to this workflow run:
    [§${{ github.run_id }}](https://github.com/${{ github.repository }}/actions/runs/${{ github.run_id }})
