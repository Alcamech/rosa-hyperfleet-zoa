package labels

import "testing"

func TestJobsNamespace(t *testing.T) {
	t.Run("When env unset it should default to zoa-jobs", func(t *testing.T) {
		t.Setenv(envJobsNamespace, "")
		if ns := JobsNamespace(); ns != DefaultJobsNamespace {
			t.Fatalf("expected %q, got %q", DefaultJobsNamespace, ns)
		}
	})

	t.Run("When env set it should use override", func(t *testing.T) {
		t.Setenv(envJobsNamespace, "custom-jobs")
		if ns := JobsNamespace(); ns != "custom-jobs" {
			t.Fatalf("expected custom-jobs, got %q", ns)
		}
	})
}

func TestMustGatherPodSelector(t *testing.T) {
	got := MustGatherPodSelector()
	want := KeyManagedBy + "=" + ValueManagedByZOA + "," + KeyComponent + "=" + ComponentMustGather
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestZOAManagedSelector(t *testing.T) {
	got := ZOAManagedSelector()
	want := KeyManagedBy + "=" + ValueManagedByZOA
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestClusterNamespacePrefix(t *testing.T) {
	if ClusterNamespacePrefix != "cluster-" {
		t.Fatalf("expected cluster- prefix, got %q", ClusterNamespacePrefix)
	}
}
