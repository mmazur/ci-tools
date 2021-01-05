// +build e2e

package lease

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/openshift/ci-tools/test/e2e/framework"
)

const (
	defaultJobSpec = `JOB_SPEC={"type":"presubmit","job":"pull-ci-test-test-master-success","buildid":"0","prowjobid":"uuid","refs":{"org":"test","repo":"test","base_ref":"master","base_sha":"6d231cc37652e85e0f0e25c21088b73d644d89ad","pulls":[{"number":1234,"author":"droslean","sha":"538680dfd2f6cff3b3506c80ca182dcb0dd22a58"}]}}`
)

func TestLeasesWithoutBoskos(t *testing.T) {
	framework.Run(t, "without boskos", func(t *framework.T, cmd *framework.CiOperatorCommand) {
		cmd.AddArgs("--registry=step-registry", "--config=config.yaml", "--target=success")
		cmd.AddEnv(defaultJobSpec)
		output, err := cmd.Run()
		if err == nil {
			t.Fatalf("without boskos: expected an error from ci-operator: %v; output:\n%v", err, string(output))
		}
		for _, line := range []string{"a lease client was required but none was provided, add the --lease-... arguments"} {
			if !bytes.Contains(output, []byte(line)) {
				t.Errorf("without boskos: could not find line %q in output; output:\n%v", line, string(output))
			}
		}
	})
}

func TestLeases(t *testing.T) {
	successClusterProfileDir := filepath.Join(t.TempDir(), "success-cluster-profile")
	if err := os.MkdirAll(successClusterProfileDir, 0755); err != nil {
		t.Fatalf("failed to create dummy secret dir: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(successClusterProfileDir, "data"), []byte("nothing"), 0644); err != nil {
		t.Fatalf("failed to create dummy secret data: %v", err)
	}

	invalidLeaseClusterProfileDir := filepath.Join(t.TempDir(), "invalid-lease-cluster-profile")
	if err := os.MkdirAll(invalidLeaseClusterProfileDir, 0755); err != nil {
		t.Fatalf("failed to create dummy secret dir: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(invalidLeaseClusterProfileDir, "data"), []byte("nothing"), 0644); err != nil {
		t.Fatalf("failed to create dummy secret data: %v", err)
	}

	var testCases = []struct {
		name    string
		args    []string
		env     []string
		success bool
		output  []string
	}{
		{
			name:    "passing lease info when needed",
			args:    []string{"--target=success", "--secret-dir=" + successClusterProfileDir},
			env:     []string{defaultJobSpec},
			success: true,
			output: []string{
				`Acquiring 1 lease(s) for "aws-quota-slice"`,
				`Acquired lease(s) for "aws-quota-slice"`,
				`Releasing leases for "success"`,
				`Releasing lease for "aws-quota-slice"`,
			},
		},
		{
			name:    "invalid lease fails",
			args:    []string{"--target=invalid-lease", "--secret-dir=" + invalidLeaseClusterProfileDir},
			env:     []string{defaultJobSpec},
			success: false,
			output: []string{
				`Acquiring 1 lease(s) for "azure4-quota-slice"`,
				`step invalid-lease failed: failed to acquire lease: resources not found`,
			},
		},
		{
			name:    "configurable leases",
			args:    []string{"--target=configurable-leases", "--secret-dir=" + successClusterProfileDir},
			env:     []string{defaultJobSpec},
			success: true,
			output: []string{
				`Acquiring 1 lease(s) for "aws-quota-slice"`,
				`Acquired lease(s) for "aws-quota-slice"`,
				`Acquiring 1 lease(s) for "gcp-quota-slice"`,
				`Acquired lease(s) for "gcp-quota-slice"`,
				`Releasing leases for "configurable-leases"`,
				`Releasing lease for "aws-quota-slice"`,
				`Releasing lease for "gcp-quota-slice"`,
			},
		},
		{
			name:    "configurable leases in the registry",
			args:    []string{"--target=configurable-leases-registry", "--secret-dir=" + successClusterProfileDir},
			env:     []string{defaultJobSpec},
			success: true,
			output: []string{
				`Acquiring 1 lease(s) for "aws-quota-slice"`,
				`Acquired lease(s) for "aws-quota-slice"`,
				`Acquiring 1 lease(s) for "gcp-quota-slice"`,
				`Acquired lease(s) for "gcp-quota-slice"`,
				`Releasing leases for "configurable-leases-registry"`,
				`Releasing lease for "aws-quota-slice"`,
				`Releasing lease for "gcp-quota-slice"`,
			},
		},
		{
			name:    "plural configurable leases",
			args:    []string{"--target=configurable-leases-count", "--secret-dir=" + successClusterProfileDir},
			env:     []string{defaultJobSpec},
			success: true,
			output: []string{
				`Acquiring 3 lease(s) for "aws-quota-slice"`,
				`Acquired lease(s) for "aws-quota-slice"`,
				`Acquiring 5 lease(s) for "gcp-quota-slice"`,
				`Acquired lease(s) for "gcp-quota-slice"`,
				`Releasing leases for "configurable-leases-count"`,
				`Releasing lease for "aws-quota-slice"`,
				`Releasing lease for "gcp-quota-slice"`,
			},
		},
		{
			name:    "leases for template",
			args:    []string{"--target=template", "--template=template.yaml"},
			env:     []string{defaultJobSpec, "CLUSTER_TYPE=aws"},
			success: true,
			output:  []string{"Container test in pod template completed successfully"},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		framework.Run(t, testCase.name, func(t *framework.T, cmd *framework.CiOperatorCommand) {
			cmd.AddArgs(
				framework.RemotePullSecretFlag(t),
				"--registry=step-registry",
				"--config=config.yaml",
			)
			cmd.AddArgs(testCase.args...)
			cmd.AddEnv(testCase.env...)
			output, err := cmd.Run()
			if testCase.success != (err == nil) {
				t.Fatalf("%s: didn't expect an error from ci-operator: %v; output:\n%v", testCase.name, err, string(output))
			}
			for _, line := range testCase.output {
				if !bytes.Contains(output, []byte(line)) {
					t.Errorf("%s: could not find line %q in output; output:\n%v", testCase.name, line, string(output))
				}
			}
		}, framework.Boskos(framework.BoskosOptions{ConfigPath: "boskos.yaml"}))
	}
}