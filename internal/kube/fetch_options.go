package kube

import (
	"fmt"
	"strings"
)

// CRDFetchMode controls how dynamic custom resources are listed.
type CRDFetchMode string

const (
	CRDFetchAll     CRDFetchMode = "all"
	CRDFetchRelated CRDFetchMode = "related"
	CRDFetchNone    CRDFetchMode = "none"
)

// DefaultCRDFetchMode returns the default custom-resource fetch behavior.
func DefaultCRDFetchMode() string {
	return string(CRDFetchRelated)
}

// ParseCRDFetchMode validates and normalizes a CRD fetch mode.
func ParseCRDFetchMode(raw string) (CRDFetchMode, error) {
	mode := CRDFetchMode(strings.ToLower(strings.TrimSpace(raw)))
	switch mode {
	case "":
		return CRDFetchRelated, nil
	case CRDFetchAll, CRDFetchRelated, CRDFetchNone:
		return mode, nil
	default:
		return "", fmt.Errorf("invalid CRD fetch mode %q (valid values: all, related, none)", raw)
	}
}

// SnapshotFetchOptions controls how cluster objects are fetched for diagnosis.
type SnapshotFetchOptions struct {
	Resources      []APIResource
	TargetResource APIResource
	TargetName     string
	FullSnapshot   bool
	CRDFetchMode   CRDFetchMode
}
