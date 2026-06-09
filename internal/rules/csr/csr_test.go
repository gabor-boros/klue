package csr_test

import (
	"testing"

	certificatesv1 "k8s.io/api/certificates/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/internal/rules/csr"
	"github.com/gabor-boros/klue/pkg/resource"
)

func node(req *certificatesv1.CertificateSigningRequest) *graph.Node {
	return &graph.Node{
		Ref:    resource.NewReference(resource.ReferenceKindCertificateSigningRequest, "certificates.k8s.io/v1", "", req.Name, string(req.UID)),
		Object: req,
	}
}

func TestDeniedRule(t *testing.T) {
	t.Parallel()

	denied := &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "csr-1"},
		Status: certificatesv1.CertificateSigningRequestStatus{
			Conditions: []certificatesv1.CertificateSigningRequestCondition{
				{Type: certificatesv1.CertificateDenied, Reason: "Denied", Message: "not allowed"},
			},
		},
	}
	if got := (csr.DeniedRule{}).Evaluate(diagnose.RuleContext{}, node(denied)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for denied CSR", len(got))
	}
}

func TestPendingRule(t *testing.T) {
	t.Parallel()

	pending := &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "csr-1"},
		Spec:       certificatesv1.CertificateSigningRequestSpec{SignerName: "kubernetes.io/kube-apiserver-client"},
	}
	if got := (csr.PendingRule{}).Evaluate(diagnose.RuleContext{}, node(pending)); len(got) != 1 {
		t.Fatalf("Evaluate() = %d findings, want 1 for pending CSR", len(got))
	}

	approved := &certificatesv1.CertificateSigningRequest{
		ObjectMeta: metav1.ObjectMeta{Name: "csr-1"},
		Status: certificatesv1.CertificateSigningRequestStatus{
			Conditions: []certificatesv1.CertificateSigningRequestCondition{{Type: certificatesv1.CertificateApproved}},
		},
	}
	if got := (csr.PendingRule{}).Evaluate(diagnose.RuleContext{}, node(approved)); len(got) != 0 {
		t.Errorf("Evaluate() = %d findings, want 0 for approved CSR", len(got))
	}
}
