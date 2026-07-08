// Package indexer provides functionality for indexing resources by their
// predefined fields. The index values can be used as field selectors in "list"
// and "get" calls to efficiently retrieve resources based on their indexed fields.
//
// The FieldIndexer will automatically take care of indexing over namespace and
// supporting efficient all-namespace queries.
//
// See description of how the indexer interface works at
// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.24.1/pkg/client#FieldIndexer
package indexer
