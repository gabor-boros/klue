// Package csr contains diagnostic rules for CertificateSigningRequests.
package csr

import (
	"fmt"

	certificatesv1 "k8s.io/api/certificates/v1"

	"github.com/gabor-boros/klue/internal/diagnose"
	"github.com/gabor-boros/klue/internal/graph"
	"github.com/gabor-boros/klue/pkg/resource"
)

// DeniedRule flags CertificateSigningRequests that were denied or failed.
type DeniedRule struct{}

func (DeniedRule) ID() string { return "csr/denied" }

func (DeniedRule) Description() string {
	return "Detects denied or failed CertificateSigningRequests"
}

func (DeniedRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindCertificateSigningRequest}
}

func (r DeniedRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	request, ok := graph.As[*certificatesv1.CertificateSigningRequest](node)
	if !ok {
		return nil
	}

	for _, condition := range request.Status.Conditions {
		if condition.Type != certificatesv1.CertificateDenied && condition.Type != certificatesv1.CertificateFailed {
			continue
		}

		return []diagnose.Finding{
			{
				ID:         r.ID(),
				Title:      fmt.Sprintf("CertificateSigningRequest was %s", condition.Type),
				Severity:   diagnose.SeverityError,
				Confidence: 0.85,
				Resource:   node.Ref,
				Evidence: []diagnose.Evidence{
					diagnose.NewEvidence(node.Ref, "Condition", condition.Message, condition.Reason),
				},
				Explanation: "The signing request will not produce a certificate. The requester must submit a new CSR after addressing the denial reason.",
				Suggestions: []diagnose.Suggestion{
					{
						Title:   "Inspect the CSR conditions",
						Command: fmt.Sprintf("kubectl describe csr %s", request.Name),
					},
				},
			},
		}
	}

	return nil
}

// PendingRule flags CertificateSigningRequests that are still awaiting approval.
type PendingRule struct{}

func (PendingRule) ID() string { return "csr/pending" }

func (PendingRule) Description() string {
	return "Detects CertificateSigningRequests awaiting approval"
}

func (PendingRule) AppliesTo() []resource.Kind {
	return []resource.Kind{resource.ReferenceKindCertificateSigningRequest}
}

func (r PendingRule) Evaluate(_ diagnose.RuleContext, node *graph.Node) []diagnose.Finding {
	request, ok := graph.As[*certificatesv1.CertificateSigningRequest](node)
	if !ok {
		return nil
	}

	for _, condition := range request.Status.Conditions {
		switch condition.Type {
		case certificatesv1.CertificateApproved, certificatesv1.CertificateDenied, certificatesv1.CertificateFailed:
			return nil
		}
	}

	return []diagnose.Finding{
		{
			ID:         r.ID(),
			Title:      "CertificateSigningRequest is pending approval",
			Severity:   diagnose.SeverityWarning,
			Confidence: 0.6,
			Resource:   node.Ref,
			Evidence: []diagnose.Evidence{
				diagnose.NewEvidence(node.Ref, "Status", fmt.Sprintf("signerName=%s", request.Spec.SignerName), "Pending"),
			},
			Explanation: "The CSR has neither been approved nor denied. It needs an approver (manual or controller) to proceed.",
			Suggestions: []diagnose.Suggestion{
				{
					Title:   "Approve the request if it is legitimate",
					Command: fmt.Sprintf("kubectl certificate approve %s", request.Name),
				},
			},
		},
	}
}
